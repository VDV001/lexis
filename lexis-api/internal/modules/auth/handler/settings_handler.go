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

	var req settingsPatchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteProblem(w, http.StatusBadRequest, "Invalid request body", err.Error())
		return
	}

	updated, err := h.service.PatchSettings(r.Context(), userID, usecase.PatchSettingsInput{
		TargetLanguage:   req.TargetLanguage,
		ProficiencyLevel: req.ProficiencyLevel,
		VocabularyType:   req.VocabularyType,
		AIModel:          req.AIModel,
		VocabGoal:        req.VocabGoal,
		UILanguage:       req.UILanguage,
	})
	if err != nil {
		if errors.Is(err, domain.ErrInvalidSettings) {
			httputil.WriteProblem(w, http.StatusBadRequest, "Invalid settings", err.Error())
			return
		}
		httputil.WriteProblem(w, http.StatusInternalServerError, "Internal server error", "Failed to update settings")
		return
	}

	httputil.WriteJSON(w, http.StatusOK, toSettingsResponse(updated))
}

// settingsPatchRequest uses pointers to distinguish "not sent" from "sent as zero".
type settingsPatchRequest struct {
	TargetLanguage   *string `json:"target_language"`
	ProficiencyLevel *string `json:"proficiency_level"`
	VocabularyType   *string `json:"vocabulary_type"`
	AIModel          *string `json:"ai_model"`
	VocabGoal        *int    `json:"vocab_goal"`
	UILanguage       *string `json:"ui_language"`
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
