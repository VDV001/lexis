package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/lexis-app/lexis-api/internal/modules/auth/domain"
	"github.com/lexis-app/lexis-api/internal/modules/auth/usecase"
	"github.com/lexis-app/lexis-api/internal/shared/httputil"
	"github.com/lexis-app/lexis-api/internal/shared/middleware"
)

type UserHandler struct {
	service *usecase.UserService
}

func NewUserHandler(service *usecase.UserService) *UserHandler {
	return &UserHandler{service: service}
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
		httputil.WriteProblem(w, http.StatusUnauthorized, "Unauthorized", "Missing user ID in context")
		return
	}

	user, err := h.service.GetProfile(r.Context(), userID)
	if err != nil {
		httputil.WriteProblem(w, http.StatusInternalServerError, "Internal server error", "Failed to fetch user profile")
		return
	}

	httputil.WriteJSON(w, http.StatusOK, toUserResponse(user))
}

// UpdateProfile handles PATCH /api/v1/users/me.
func (h *UserHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		httputil.WriteProblem(w, http.StatusUnauthorized, "Unauthorized", "Missing user ID in context")
		return
	}

	var req ProfileUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteProblem(w, http.StatusBadRequest, "Invalid request body", err.Error())
		return
	}

	user, err := h.service.UpdateProfile(r.Context(), userID, usecase.UpdateProfileInput{
		DisplayName: req.DisplayName,
		AvatarURL:   req.AvatarURL,
	})
	if err != nil {
		if errors.Is(err, domain.ErrInvalidDisplayName) || errors.Is(err, domain.ErrAvatarURLTooLong) {
			httputil.WriteProblem(w, http.StatusBadRequest, "Invalid request body", err.Error())
			return
		}
		httputil.WriteProblem(w, http.StatusInternalServerError, "Internal server error", "Failed to update user profile")
		return
	}

	httputil.WriteJSON(w, http.StatusOK, toUserResponse(user))
}

// GetSettings handles GET /api/v1/users/me/settings.
func (h *UserHandler) GetSettings(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		httputil.WriteProblem(w, http.StatusUnauthorized, "Unauthorized", "Missing user ID in context")
		return
	}

	settings, err := h.service.GetSettings(r.Context(), userID)
	if err != nil {
		httputil.WriteProblem(w, http.StatusInternalServerError, "Internal server error", "Failed to fetch settings")
		return
	}

	httputil.WriteJSON(w, http.StatusOK, toSettingsResponse(settings))
}

// UpdateSettings handles PUT /api/v1/users/me/settings.
func (h *UserHandler) UpdateSettings(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		httputil.WriteProblem(w, http.StatusUnauthorized, "Unauthorized", "Missing user ID in context")
		return
	}

	var patch map[string]json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&patch); err != nil {
		httputil.WriteProblem(w, http.StatusBadRequest, "Invalid request body", err.Error())
		return
	}

	existing, err := h.service.GetSettings(r.Context(), userID)
	if err != nil {
		httputil.WriteProblem(w, http.StatusInternalServerError, "Internal server error", "Failed to fetch settings")
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

	if err := h.service.UpdateSettings(r.Context(), userID, existing); err != nil {
		if errors.Is(err, domain.ErrInvalidSettings) {
			httputil.WriteProblem(w, http.StatusBadRequest, "Invalid settings", err.Error())
			return
		}
		httputil.WriteProblem(w, http.StatusInternalServerError, "Internal server error", "Failed to update settings")
		return
	}

	httputil.WriteJSON(w, http.StatusOK, toSettingsResponse(existing))
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
