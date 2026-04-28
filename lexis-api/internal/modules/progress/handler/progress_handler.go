package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	progressDomain "github.com/lexis-app/lexis-api/internal/modules/progress/domain"
	"github.com/lexis-app/lexis-api/internal/modules/progress/usecase"
	"github.com/lexis-app/lexis-api/internal/shared/httputil"
	"github.com/lexis-app/lexis-api/internal/shared/middleware"
)

type ProgressHandler struct {
	service *usecase.ProgressService
}

func NewProgressHandler(service *usecase.ProgressService) *ProgressHandler {
	return &ProgressHandler{service: service}
}

func (h *ProgressHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/summary", h.HandleSummary)
	r.Get("/vocabulary", h.HandleVocabulary)
	r.Get("/vocabulary/curve", h.HandleVocabCurve)
	r.Get("/goals", h.HandleGoals)
	r.Get("/errors", h.HandleErrors)
	r.Get("/sessions", h.HandleSessions)
	r.Get("/sessions/{id}", h.HandleSession)
	r.Post("/sessions", h.HandleStartSession)
	r.Post("/rounds", h.HandleRecordRound)
	return r
}

// HandleSummary handles GET /progress/summary.
func (h *ProgressHandler) HandleSummary(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		httputil.WriteProblem(w, http.StatusUnauthorized, "Unauthorized", "Missing user ID in context")
		return
	}

	summary, err := h.service.GetSummary(r.Context(), userID)
	if err != nil {
		httputil.WriteProblem(w, http.StatusInternalServerError, "Internal server error", "Failed to fetch progress summary")
		return
	}

	httputil.WriteJSON(w, http.StatusOK, summary)
}

// HandleVocabulary handles GET /progress/vocabulary.
func (h *ProgressHandler) HandleVocabulary(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		httputil.WriteProblem(w, http.StatusUnauthorized, "Unauthorized", "Missing user ID in context")
		return
	}

	stats, err := h.service.GetVocabulary(r.Context(), userID)
	if err != nil {
		httputil.WriteProblem(w, http.StatusInternalServerError, "Internal server error", "Failed to fetch vocabulary stats")
		return
	}

	httputil.WriteJSON(w, http.StatusOK, stats)
}

// HandleVocabCurve handles GET /progress/vocabulary/curve.
func (h *ProgressHandler) HandleVocabCurve(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		httputil.WriteProblem(w, http.StatusUnauthorized, "Unauthorized", "Missing user ID in context")
		return
	}

	curve, err := h.service.GetVocabCurve(r.Context(), userID)
	if err != nil {
		httputil.WriteProblem(w, http.StatusInternalServerError, "Internal server error", "Failed to fetch vocabulary curve")
		return
	}

	httputil.WriteJSON(w, http.StatusOK, curve)
}

// HandleGoals handles GET /progress/goals.
func (h *ProgressHandler) HandleGoals(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		httputil.WriteProblem(w, http.StatusUnauthorized, "Unauthorized", "Missing user ID in context")
		return
	}

	goals, err := h.service.GetGoals(r.Context(), userID)
	if err != nil {
		httputil.WriteProblem(w, http.StatusInternalServerError, "Internal server error", "Failed to fetch goals")
		return
	}

	httputil.WriteJSON(w, http.StatusOK, goals)
}

// HandleErrors handles GET /progress/errors.
func (h *ProgressHandler) HandleErrors(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		httputil.WriteProblem(w, http.StatusUnauthorized, "Unauthorized", "Missing user ID in context")
		return
	}

	errors, err := h.service.GetErrors(r.Context(), userID)
	if err != nil {
		httputil.WriteProblem(w, http.StatusInternalServerError, "Internal server error", "Failed to fetch error categories")
		return
	}

	httputil.WriteJSON(w, http.StatusOK, errors)
}

// HandleSessions handles GET /progress/sessions?limit=20&offset=0.
func (h *ProgressHandler) HandleSessions(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		httputil.WriteProblem(w, http.StatusUnauthorized, "Unauthorized", "Missing user ID in context")
		return
	}

	limit := 20
	offset := 0

	if v := r.URL.Query().Get("limit"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	if limit > 100 {
		limit = 100
	}
	if v := r.URL.Query().Get("offset"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	sessions, err := h.service.GetSessions(r.Context(), userID, limit, offset)
	if err != nil {
		httputil.WriteProblem(w, http.StatusInternalServerError, "Internal server error", "Failed to fetch sessions")
		return
	}

	httputil.WriteJSON(w, http.StatusOK, sessions)
}

// HandleSession handles GET /progress/sessions/{id}.
func (h *ProgressHandler) HandleSession(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		httputil.WriteProblem(w, http.StatusUnauthorized, "Unauthorized", "Missing user ID in context")
		return
	}

	sessionID := chi.URLParam(r, "id")
	if sessionID == "" {
		httputil.WriteProblem(w, http.StatusBadRequest, "Bad request", "Missing session ID")
		return
	}

	session, err := h.service.GetSession(r.Context(), sessionID, userID)
	if errors.Is(err, progressDomain.ErrSessionNotFound) {
		httputil.WriteProblem(w, http.StatusNotFound, "Not found", "Session not found")
		return
	}
	if err != nil {
		httputil.WriteProblem(w, http.StatusInternalServerError, "Internal server error", "Failed to fetch session")
		return
	}

	httputil.WriteJSON(w, http.StatusOK, session)
}

// HandleStartSession handles POST /progress/sessions.
func (h *ProgressHandler) HandleStartSession(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		httputil.WriteProblem(w, http.StatusUnauthorized, "Unauthorized", "Missing user ID in context")
		return
	}

	var body struct {
		Mode     string `json:"mode"`
		Language string `json:"language"`
		Level    string `json:"level"`
		AIModel  string `json:"ai_model"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httputil.WriteProblem(w, http.StatusBadRequest, "Bad request", "Invalid JSON body")
		return
	}
	if body.Mode == "" || body.Language == "" || body.Level == "" || body.AIModel == "" {
		httputil.WriteProblem(w, http.StatusBadRequest, "Bad request", "mode, language, level, and ai_model are required")
		return
	}

	if !progressDomain.Mode(body.Mode).IsValid() {
		httputil.WriteProblem(w, http.StatusBadRequest, "Bad request", "invalid mode")
		return
	}

	sessionID, err := h.service.StartSession(r.Context(), userID, body.Mode, body.Language, body.Level, body.AIModel)
	if err != nil {
		httputil.WriteProblem(w, http.StatusInternalServerError, "Internal server error", "Failed to create session")
		return
	}

	httputil.WriteJSON(w, http.StatusCreated, map[string]string{"id": sessionID})
}

// HandleRecordRound handles POST /progress/rounds.
func (h *ProgressHandler) HandleRecordRound(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		httputil.WriteProblem(w, http.StatusUnauthorized, "Unauthorized", "Missing user ID in context")
		return
	}

	var body struct {
		SessionID     string  `json:"session_id"`
		Mode          string  `json:"mode"`
		IsCorrect     bool    `json:"is_correct"`
		ErrorType     *string `json:"error_type"`
		Question      string  `json:"question"`
		UserAnswer    string  `json:"user_answer"`
		CorrectAnswer *string `json:"correct_answer"`
		Explanation   *string `json:"explanation"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httputil.WriteProblem(w, http.StatusBadRequest, "Bad request", "Invalid JSON body")
		return
	}
	if body.SessionID == "" || body.Mode == "" || body.Question == "" || body.UserAnswer == "" {
		httputil.WriteProblem(w, http.StatusBadRequest, "Bad request", "session_id, mode, question, and user_answer are required")
		return
	}

	input := usecase.RecordRoundInput{
		SessionID:     body.SessionID,
		UserID:        userID,
		Mode:          body.Mode,
		IsCorrect:     body.IsCorrect,
		ErrorType:     body.ErrorType,
		Question:      body.Question,
		UserAnswer:    body.UserAnswer,
		CorrectAnswer: body.CorrectAnswer,
		Explanation:   body.Explanation,
	}
	if err := h.service.RecordRound(r.Context(), input); err != nil {
		if errors.Is(err, progressDomain.ErrSessionNotFound) {
			httputil.WriteProblem(w, http.StatusNotFound, "Not found", "Session not found")
			return
		}
		httputil.WriteProblem(w, http.StatusInternalServerError, "Internal server error", "Failed to record round")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

