package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/lexis-app/lexis-api/internal/modules/vocabulary/domain"
	"github.com/lexis-app/lexis-api/internal/modules/vocabulary/usecase"
	"github.com/lexis-app/lexis-api/internal/shared/httputil"
	"github.com/lexis-app/lexis-api/internal/shared/middleware"
)

type VocabHandler struct {
	service *usecase.VocabService
}

func NewVocabHandler(service *usecase.VocabService) *VocabHandler {
	return &VocabHandler{service: service}
}

func (h *VocabHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/", h.ListWords)
	r.Post("/", h.AddWord)
	r.Delete("/{id}", h.DeleteWord)
	r.Patch("/{id}", h.UpdateWord)
	r.Get("/due", h.GetDueForReview)
	return r
}

type AddWordRequest struct {
	Word     string             `json:"word"`
	Language string             `json:"language"`
	Status   domain.VocabStatus `json:"status"`
	Context  string             `json:"context"`
}

type UpdateWordRequest struct {
	Status domain.VocabStatus `json:"status"`
}

func (h *VocabHandler) ListWords(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		httputil.WriteProblem(w, http.StatusUnauthorized, "Unauthorized", "Missing user ID in context")
		return
	}

	limit := 500
	offset := 0
	if v := r.URL.Query().Get("limit"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	if limit > 500 {
		limit = 500
	}
	if v := r.URL.Query().Get("offset"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	words, err := h.service.ListWords(r.Context(), userID, limit, offset)
	if err != nil {
		httputil.WriteProblem(w, http.StatusInternalServerError, "Internal server error", "Failed to fetch vocabulary words")
		return
	}

	httputil.WriteJSON(w, http.StatusOK, words)
}

func (h *VocabHandler) AddWord(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		httputil.WriteProblem(w, http.StatusUnauthorized, "Unauthorized", "Missing user ID in context")
		return
	}

	var req AddWordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteProblem(w, http.StatusBadRequest, "Invalid request body", err.Error())
		return
	}

	if req.Word == "" {
		httputil.WriteProblem(w, http.StatusBadRequest, "Invalid request body", "word is required")
		return
	}

	word, err := h.service.AddWord(r.Context(), usecase.AddWordInput{
		UserID:   userID,
		Word:     req.Word,
		Language: req.Language,
		Status:   req.Status,
		Context:  req.Context,
	})
	if errors.Is(err, usecase.ErrInvalidStatus) {
		httputil.WriteProblem(w, http.StatusBadRequest, "Invalid request body", "status must be one of: unknown, uncertain, confident")
		return
	}
	if err != nil {
		httputil.WriteProblem(w, http.StatusInternalServerError, "Internal server error", "Failed to save word")
		return
	}

	httputil.WriteJSON(w, http.StatusCreated, word)
}

func (h *VocabHandler) DeleteWord(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		httputil.WriteProblem(w, http.StatusUnauthorized, "Unauthorized", "Missing user ID in context")
		return
	}

	wordID := chi.URLParam(r, "id")
	if wordID == "" {
		httputil.WriteProblem(w, http.StatusBadRequest, "Bad request", "Missing word ID")
		return
	}

	if err := h.service.DeleteWord(r.Context(), wordID, userID); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			httputil.WriteProblem(w, http.StatusNotFound, "Not found", "Word not found")
			return
		}
		httputil.WriteProblem(w, http.StatusInternalServerError, "Internal server error", "Failed to delete word")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *VocabHandler) UpdateWord(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		httputil.WriteProblem(w, http.StatusUnauthorized, "Unauthorized", "Missing user ID in context")
		return
	}

	wordID := chi.URLParam(r, "id")
	if wordID == "" {
		httputil.WriteProblem(w, http.StatusBadRequest, "Bad request", "Missing word ID")
		return
	}

	var req UpdateWordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteProblem(w, http.StatusBadRequest, "Invalid request body", err.Error())
		return
	}

	if req.Status == "" {
		httputil.WriteProblem(w, http.StatusBadRequest, "Invalid request body", "status is required")
		return
	}

	err := h.service.UpdateStatus(r.Context(), wordID, userID, req.Status)
	if errors.Is(err, usecase.ErrInvalidStatus) {
		httputil.WriteProblem(w, http.StatusBadRequest, "Invalid request body", "status must be one of: unknown, uncertain, confident")
		return
	}
	if errors.Is(err, domain.ErrNotFound) {
		httputil.WriteProblem(w, http.StatusNotFound, "Not found", "Word not found")
		return
	}
	if err != nil {
		httputil.WriteProblem(w, http.StatusInternalServerError, "Internal server error", "Failed to update word")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *VocabHandler) GetDueForReview(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		httputil.WriteProblem(w, http.StatusUnauthorized, "Unauthorized", "Missing user ID in context")
		return
	}

	words, err := h.service.GetDueForReview(r.Context(), userID, 50)
	if err != nil {
		httputil.WriteProblem(w, http.StatusInternalServerError, "Internal server error", "Failed to fetch due words")
		return
	}

	httputil.WriteJSON(w, http.StatusOK, words)
}
