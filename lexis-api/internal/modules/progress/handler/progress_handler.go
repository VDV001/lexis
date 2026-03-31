package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/lexis-app/lexis-api/internal/modules/progress/usecase"
	"github.com/lexis-app/lexis-api/internal/shared/middleware"
)

type ProblemDetail struct {
	Type   string `json:"type"`
	Title  string `json:"title"`
	Status int    `json:"status"`
	Detail string `json:"detail"`
}

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
	return r
}

// HandleSummary handles GET /progress/summary.
func (h *ProgressHandler) HandleSummary(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		writeProblem(w, http.StatusUnauthorized, "Unauthorized", "Missing user ID in context")
		return
	}

	summary, err := h.service.GetSummary(r.Context(), userID)
	if err != nil {
		writeProblem(w, http.StatusInternalServerError, "Internal server error", "Failed to fetch progress summary")
		return
	}

	writeJSON(w, http.StatusOK, summary)
}

// HandleVocabulary handles GET /progress/vocabulary.
func (h *ProgressHandler) HandleVocabulary(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		writeProblem(w, http.StatusUnauthorized, "Unauthorized", "Missing user ID in context")
		return
	}

	stats, err := h.service.GetVocabulary(r.Context(), userID)
	if err != nil {
		writeProblem(w, http.StatusInternalServerError, "Internal server error", "Failed to fetch vocabulary stats")
		return
	}

	writeJSON(w, http.StatusOK, stats)
}

// HandleVocabCurve handles GET /progress/vocabulary/curve.
func (h *ProgressHandler) HandleVocabCurve(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		writeProblem(w, http.StatusUnauthorized, "Unauthorized", "Missing user ID in context")
		return
	}

	curve, err := h.service.GetVocabCurve(r.Context(), userID)
	if err != nil {
		writeProblem(w, http.StatusInternalServerError, "Internal server error", "Failed to fetch vocabulary curve")
		return
	}

	writeJSON(w, http.StatusOK, curve)
}

// HandleGoals handles GET /progress/goals.
func (h *ProgressHandler) HandleGoals(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		writeProblem(w, http.StatusUnauthorized, "Unauthorized", "Missing user ID in context")
		return
	}

	goals, err := h.service.GetGoals(r.Context(), userID)
	if err != nil {
		writeProblem(w, http.StatusInternalServerError, "Internal server error", "Failed to fetch goals")
		return
	}

	writeJSON(w, http.StatusOK, goals)
}

// HandleErrors handles GET /progress/errors.
func (h *ProgressHandler) HandleErrors(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		writeProblem(w, http.StatusUnauthorized, "Unauthorized", "Missing user ID in context")
		return
	}

	errors, err := h.service.GetErrors(r.Context(), userID)
	if err != nil {
		writeProblem(w, http.StatusInternalServerError, "Internal server error", "Failed to fetch error categories")
		return
	}

	writeJSON(w, http.StatusOK, errors)
}

// HandleSessions handles GET /progress/sessions?limit=20&offset=0.
func (h *ProgressHandler) HandleSessions(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		writeProblem(w, http.StatusUnauthorized, "Unauthorized", "Missing user ID in context")
		return
	}

	limit := 20
	offset := 0

	if v := r.URL.Query().Get("limit"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	if v := r.URL.Query().Get("offset"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	sessions, err := h.service.GetSessions(r.Context(), userID, limit, offset)
	if err != nil {
		writeProblem(w, http.StatusInternalServerError, "Internal server error", "Failed to fetch sessions")
		return
	}

	writeJSON(w, http.StatusOK, sessions)
}

// HandleSession handles GET /progress/sessions/{id}.
func (h *ProgressHandler) HandleSession(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "id")
	if sessionID == "" {
		writeProblem(w, http.StatusBadRequest, "Bad request", "Missing session ID")
		return
	}

	session, err := h.service.GetSession(r.Context(), sessionID)
	if err != nil {
		writeProblem(w, http.StatusInternalServerError, "Internal server error", "Failed to fetch session")
		return
	}
	if session == nil {
		writeProblem(w, http.StatusNotFound, "Not found", "Session not found")
		return
	}

	writeJSON(w, http.StatusOK, session)
}

func writeProblem(w http.ResponseWriter, status int, title, detail string) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(ProblemDetail{
		Type:   "about:blank",
		Title:  title,
		Status: status,
		Detail: detail,
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
