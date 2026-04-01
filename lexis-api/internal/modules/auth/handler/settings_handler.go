package handler

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/lexis-app/lexis-api/internal/modules/auth/domain"
	"github.com/lexis-app/lexis-api/internal/shared/middleware"
)

type UserHandler struct {
	users    domain.UserRepository
	settings domain.SettingsRepository
}

func NewUserHandler(users domain.UserRepository, settings domain.SettingsRepository) *UserHandler {
	return &UserHandler{users: users, settings: settings}
}

func (h *UserHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/me", h.GetProfile)
	r.Patch("/me", h.UpdateProfile)
	r.Get("/me/settings", h.GetSettings)
	r.Put("/me/settings", h.UpdateSettings)
	return r
}

// ---------- Request / Response DTOs ----------

type ProfileUpdateRequest struct {
	DisplayName *string `json:"display_name"`
	AvatarURL   *string `json:"avatar_url"`
}

type SettingsResponse struct {
	TargetLanguage   string `json:"target_language"`
	ProficiencyLevel string `json:"proficiency_level"`
	VocabularyType   string `json:"vocabulary_type"`
	AIModel          string `json:"ai_model"`
	VocabGoal        int    `json:"vocab_goal"`
	UILanguage       string `json:"ui_language"`
}

// ---------- Handlers ----------

// GetProfile handles GET /api/v1/users/me.
func (h *UserHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		writeProblem(w, http.StatusUnauthorized, "Unauthorized", "Missing user ID in context")
		return
	}

	user, err := h.users.GetByID(r.Context(), userID)
	if err != nil {
		writeProblem(w, http.StatusInternalServerError, "Internal server error", "Failed to fetch user profile")
		return
	}

	writeJSON(w, http.StatusOK, toUserResponse(user))
}

// UpdateProfile handles PATCH /api/v1/users/me.
func (h *UserHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		writeProblem(w, http.StatusUnauthorized, "Unauthorized", "Missing user ID in context")
		return
	}

	var req ProfileUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeProblem(w, http.StatusBadRequest, "Invalid request body", err.Error())
		return
	}

	user, err := h.users.GetByID(r.Context(), userID)
	if err != nil {
		writeProblem(w, http.StatusInternalServerError, "Internal server error", "Failed to fetch user profile")
		return
	}

	if req.DisplayName != nil {
		user.DisplayName = *req.DisplayName
	}
	if req.AvatarURL != nil {
		user.AvatarURL = req.AvatarURL
	}

	if err := h.users.Update(r.Context(), user); err != nil {
		writeProblem(w, http.StatusInternalServerError, "Internal server error", "Failed to update user profile")
		return
	}

	writeJSON(w, http.StatusOK, toUserResponse(user))
}

// GetSettings handles GET /api/v1/users/me/settings.
func (h *UserHandler) GetSettings(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		writeProblem(w, http.StatusUnauthorized, "Unauthorized", "Missing user ID in context")
		return
	}

	settings, err := h.settings.GetByUserID(r.Context(), userID)
	if err != nil {
		writeProblem(w, http.StatusInternalServerError, "Internal server error", "Failed to fetch settings")
		return
	}

	writeJSON(w, http.StatusOK, toSettingsResponse(settings))
}

// UpdateSettings handles PUT /api/v1/users/me/settings.
// Partial updates are allowed: only fields present in the JSON body are changed.
func (h *UserHandler) UpdateSettings(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		writeProblem(w, http.StatusUnauthorized, "Unauthorized", "Missing user ID in context")
		return
	}

	// Decode into a map so we know which fields the client actually sent.
	var patch map[string]json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&patch); err != nil {
		writeProblem(w, http.StatusBadRequest, "Invalid request body", err.Error())
		return
	}

	existing, err := h.settings.GetByUserID(r.Context(), userID)
	if err != nil {
		writeProblem(w, http.StatusInternalServerError, "Internal server error", "Failed to fetch settings")
		return
	}

	// Apply partial updates.
	if v, ok := patch["target_language"]; ok {
		var s string
		if err := json.Unmarshal(v, &s); err == nil {
			existing.TargetLanguage = s
		}
	}
	if v, ok := patch["proficiency_level"]; ok {
		var s string
		if err := json.Unmarshal(v, &s); err == nil {
			existing.ProficiencyLevel = s
		}
	}
	if v, ok := patch["vocabulary_type"]; ok {
		var s string
		if err := json.Unmarshal(v, &s); err == nil {
			existing.VocabularyType = s
		}
	}
	if v, ok := patch["ai_model"]; ok {
		var s string
		if err := json.Unmarshal(v, &s); err == nil {
			existing.AIModel = s
		}
	}
	if v, ok := patch["vocab_goal"]; ok {
		var i int
		if err := json.Unmarshal(v, &i); err == nil {
			existing.VocabGoal = i
		}
	}
	if v, ok := patch["ui_language"]; ok {
		var s string
		if err := json.Unmarshal(v, &s); err == nil {
			existing.UILanguage = s
		}
	}

	// Validate merged settings.
	if err := validateSettings(existing); err != nil {
		writeProblem(w, http.StatusBadRequest, "Invalid settings", err.Error())
		return
	}

	if err := h.settings.Upsert(r.Context(), existing); err != nil {
		writeProblem(w, http.StatusInternalServerError, "Internal server error", "Failed to update settings")
		return
	}

	writeJSON(w, http.StatusOK, toSettingsResponse(existing))
}

// ---------- Validation ----------

var validLevels = map[string]bool{"a2": true, "b1": true, "b2": true, "c1": true}
var validVocabTypes = map[string]bool{"tech": true, "literary": true, "business": true}
var validModels = map[string]bool{
	"claude-sonnet-4-20250514": true, "claude-haiku-4-20250514": true,
	"qwen-plus": true, "gpt-4o": true, "gpt-4o-mini": true, "gemini-2.0-flash": true,
}
var validLanguages = map[string]bool{"en": true}

func validateSettings(s *domain.UserSettings) error {
	if !validLanguages[s.TargetLanguage] {
		return fmt.Errorf("invalid target_language: %q", s.TargetLanguage)
	}
	if !validLevels[s.ProficiencyLevel] {
		return fmt.Errorf("invalid proficiency_level: %q", s.ProficiencyLevel)
	}
	if !validVocabTypes[s.VocabularyType] {
		return fmt.Errorf("invalid vocabulary_type: %q", s.VocabularyType)
	}
	if !validModels[s.AIModel] {
		return fmt.Errorf("invalid ai_model: %q", s.AIModel)
	}
	if s.VocabGoal < 100 || s.VocabGoal > 50000 {
		return fmt.Errorf("vocab_goal must be between 100 and 50000, got %d", s.VocabGoal)
	}
	return nil
}

// ---------- Helpers ----------

func toSettingsResponse(s *domain.UserSettings) SettingsResponse {
	return SettingsResponse{
		TargetLanguage:   s.TargetLanguage,
		ProficiencyLevel: s.ProficiencyLevel,
		VocabularyType:   s.VocabularyType,
		AIModel:          s.AIModel,
		VocabGoal:        s.VocabGoal,
		UILanguage:       s.UILanguage,
	}
}
