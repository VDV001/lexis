package handler

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/lexis-app/lexis-api/internal/modules/tutor/domain"
	"github.com/lexis-app/lexis-api/internal/modules/tutor/usecase"
	"github.com/lexis-app/lexis-api/internal/shared/middleware"
)

type TutorHandler struct {
	chatService     *usecase.ChatService
	exerciseService *usecase.ExerciseService
}

func NewTutorHandler(chatService *usecase.ChatService, exerciseService *usecase.ExerciseService) *TutorHandler {
	return &TutorHandler{chatService: chatService, exerciseService: exerciseService}
}

func (h *TutorHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Post("/chat", h.HandleChat)
	r.Post("/quiz/generate", h.HandleGenerateExercise(domain.ModeQuiz))
	r.Post("/quiz/answer", h.HandleCheckAnswer(domain.ModeQuiz))
	r.Post("/translate/generate", h.HandleGenerateExercise(domain.ModeTranslate))
	r.Post("/translate/check", h.HandleCheckAnswer(domain.ModeTranslate))
	r.Post("/gap/generate", h.HandleGenerateExercise(domain.ModeGap))
	r.Post("/scramble/generate", h.HandleGenerateExercise(domain.ModeScramble))
	r.Post("/scramble/check", h.HandleCheckAnswer(domain.ModeScramble))
	return r
}

type ChatRequest struct {
	Messages []domain.Message `json:"messages"`
}

func (h *TutorHandler) HandleChat(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		http.Error(w, `{"type":"about:blank","title":"Unauthorized","status":401,"detail":"missing user"}`, http.StatusUnauthorized)
		return
	}

	var req ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"type":"about:blank","title":"Bad Request","status":400,"detail":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	ch, err := h.chatService.Chat(r.Context(), usecase.ChatInput{
		UserID:   userID,
		Messages: req.Messages,
	})
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"type":"about:blank","title":"Internal Error","status":500,"detail":"%s"}`, err.Error()), http.StatusInternalServerError)
		return
	}

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	for delta := range ch {
		data, err := json.Marshal(delta)
		if err != nil {
			continue
		}
		_, _ = fmt.Fprintf(w, "data: %s\n\n", data)
		flusher.Flush()
	}
}

// ---------- Exercise handlers ----------

func (h *TutorHandler) HandleGenerateExercise(mode domain.Mode) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := middleware.GetUserID(r.Context())
		if userID == "" {
			http.Error(w, `{"type":"about:blank","title":"Unauthorized","status":401,"detail":"missing user"}`, http.StatusUnauthorized)
			return
		}

		exercise, err := h.exerciseService.Generate(r.Context(), usecase.GenerateInput{
			UserID: userID,
			Mode:   mode,
		})
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"type":"about:blank","title":"Internal Error","status":500,"detail":"%s"}`, err.Error()), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(exercise.Raw))
	}
}

type checkAnswerRequest struct {
	Answer  string `json:"answer"`
	Context string `json:"context"`
}

func (h *TutorHandler) HandleCheckAnswer(mode domain.Mode) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := middleware.GetUserID(r.Context())
		if userID == "" {
			http.Error(w, `{"type":"about:blank","title":"Unauthorized","status":401,"detail":"missing user"}`, http.StatusUnauthorized)
			return
		}

		var req checkAnswerRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"type":"about:blank","title":"Bad Request","status":400,"detail":"invalid request body"}`, http.StatusBadRequest)
			return
		}

		result, err := h.exerciseService.Check(r.Context(), usecase.CheckInput{
			UserID:     userID,
			Mode:       mode,
			UserAnswer: req.Answer,
			Context:    req.Context,
		})
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"type":"about:blank","title":"Internal Error","status":500,"detail":"%s"}`, err.Error()), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(result.Raw))
	}
}
