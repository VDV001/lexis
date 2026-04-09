package handler

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/lexis-app/lexis-api/internal/modules/tutor/domain"
	"github.com/lexis-app/lexis-api/internal/modules/tutor/usecase"
	"github.com/lexis-app/lexis-api/internal/shared/eventbus"
	"github.com/lexis-app/lexis-api/internal/shared/httputil"
	"github.com/lexis-app/lexis-api/internal/shared/middleware"
)

type TutorHandler struct {
	chatService     *usecase.ChatService
	exerciseService *usecase.ExerciseService
	bus             eventbus.Publisher
}

func NewTutorHandler(chatService *usecase.ChatService, exerciseService *usecase.ExerciseService, bus eventbus.Publisher) *TutorHandler {
	return &TutorHandler{chatService: chatService, exerciseService: exerciseService, bus: bus}
}

func (h *TutorHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Post("/chat", h.HandleChat)
	r.Post("/quiz/generate", h.HandleGenerateExercise(domain.ModeQuiz))
	r.Post("/quiz/answer", h.HandleCheckAnswer(domain.ModeQuiz))
	r.Post("/translate/generate", h.HandleGenerateExercise(domain.ModeTranslate))
	r.Post("/translate/check", h.HandleCheckAnswer(domain.ModeTranslate))
	r.Post("/gap/generate", h.HandleGenerateExercise(domain.ModeGap))
	r.Post("/gap/check", h.HandleCheckAnswer(domain.ModeGap))
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
		httputil.WriteProblem(w, http.StatusUnauthorized, "Unauthorized", "missing user")
		return
	}

	var req ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteProblem(w, http.StatusBadRequest, "Bad Request", "invalid request body")
		return
	}

	if len(req.Messages) == 0 {
		httputil.WriteProblem(w, http.StatusBadRequest, "Bad Request", "messages must not be empty")
		return
	}
	if len(req.Messages) > 100 {
		httputil.WriteProblem(w, http.StatusBadRequest, "Bad Request", "too many messages (max 100)")
		return
	}
	for _, m := range req.Messages {
		if len(m.Content) > 16000 {
			httputil.WriteProblem(w, http.StatusBadRequest, "Bad Request", "message content too long (max 16000 chars)")
			return
		}
	}

	ch, err := h.chatService.Chat(r.Context(), usecase.ChatInput{
		UserID:   userID,
		Messages: req.Messages,
	})
	if err != nil {
		log.Printf("tutor chat error: %v", err)
		httputil.WriteProblem(w, http.StatusInternalServerError, "Internal Error", "internal server error")
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		httputil.WriteProblem(w, http.StatusInternalServerError, "Internal Error", "streaming not supported")
		return
	}

	// Set SSE headers after confirming flusher support
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

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
			httputil.WriteProblem(w, http.StatusUnauthorized, "Unauthorized", "missing user")
			return
		}

		exercise, err := h.exerciseService.Generate(r.Context(), usecase.GenerateInput{
			UserID: userID,
			Mode:   mode,
		})
		if err != nil {
			log.Printf("tutor generate error: %v", err)
			httputil.WriteProblem(w, http.StatusInternalServerError, "Internal Error", "internal server error")
			return
		}

		if !json.Valid([]byte(exercise.Raw)) {
			log.Printf("tutor: AI returned invalid JSON for exercise generation")
			httputil.WriteProblem(w, http.StatusBadGateway, "Bad Gateway", "AI model returned invalid response")
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
			httputil.WriteProblem(w, http.StatusUnauthorized, "Unauthorized", "missing user")
			return
		}

		var req checkAnswerRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			httputil.WriteProblem(w, http.StatusBadRequest, "Bad Request", "invalid request body")
			return
		}

		if len(req.Answer) > 16000 {
			httputil.WriteProblem(w, http.StatusBadRequest, "Bad Request", "answer too long (max 16000 chars)")
			return
		}
		if len(req.Context) > 32000 {
			httputil.WriteProblem(w, http.StatusBadRequest, "Bad Request", "context too long (max 32000 chars)")
			return
		}

		result, err := h.exerciseService.Check(r.Context(), usecase.CheckInput{
			UserID:     userID,
			Mode:       mode,
			UserAnswer: req.Answer,
			Context:    req.Context,
		})
		if err != nil {
			log.Printf("tutor check error: %v", err)
			httputil.WriteProblem(w, http.StatusInternalServerError, "Internal Error", "internal server error")
			return
		}

		if !json.Valid([]byte(result.Raw)) {
			log.Printf("tutor: AI returned invalid JSON for check answer")
			httputil.WriteProblem(w, http.StatusBadGateway, "Bad Gateway", "AI model returned invalid response")
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(result.Raw))

		var parsed struct {
			Correct  bool     `json:"correct"`
			Word     string   `json:"word"`
			NewWords []string `json:"new_words"`
		}
		if err := json.Unmarshal([]byte(result.Raw), &parsed); err != nil {
			log.Printf("tutor: failed to parse check answer result: %v", err)
			return // don't publish events with zero-value data
		}

		h.bus.Publish(eventbus.Event{
			Type: eventbus.EventRoundCompleted,
			Payload: eventbus.RoundCompletedPayload{
				UserID:     userID,
				Mode:       string(mode),
				IsCorrect:  parsed.Correct,
				Question:   req.Context,
				UserAnswer: req.Answer,
			},
		})

		// Extract discovered words from AI response or exercise context.
		var words []string
		if len(parsed.NewWords) > 0 {
			words = parsed.NewWords
		} else if parsed.Word != "" {
			words = []string{parsed.Word}
		}

		if len(words) > 0 {
			var exerciseCtx struct {
				Language string `json:"language"`
			}
			_ = json.Unmarshal([]byte(req.Context), &exerciseCtx)

			if exerciseCtx.Language != "" {
				h.bus.Publish(eventbus.Event{
					Type: eventbus.EventWordsDiscovered,
					Payload: eventbus.WordsDiscoveredPayload{
						UserID:   userID,
						Language: exerciseCtx.Language,
						Words:    words,
						Context:  req.Context,
					},
				})
			}
		}
	}
}
