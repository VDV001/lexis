package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	tutorDomain "github.com/lexis-app/lexis-api/internal/modules/tutor/domain"
	"github.com/lexis-app/lexis-api/internal/modules/tutor/handler"
	"github.com/lexis-app/lexis-api/internal/modules/tutor/usecase"
	"github.com/lexis-app/lexis-api/internal/shared/eventbus"
	"github.com/lexis-app/lexis-api/internal/shared/httputil"
	"github.com/lexis-app/lexis-api/internal/shared/middleware"
)

// ---------------------------------------------------------------------------
// Mock: AIProvider
// ---------------------------------------------------------------------------

type mockProvider struct {
	chatFn     func(ctx context.Context, req tutorDomain.ChatRequest) (<-chan tutorDomain.ChatDelta, error)
	generateFn func(ctx context.Context, req tutorDomain.ExerciseRequest) (tutorDomain.Exercise, error)
	checkFn    func(ctx context.Context, req tutorDomain.CheckRequest) (tutorDomain.CheckResult, error)
}

func (m *mockProvider) Chat(ctx context.Context, req tutorDomain.ChatRequest) (<-chan tutorDomain.ChatDelta, error) {
	return m.chatFn(ctx, req)
}

func (m *mockProvider) GenerateExercise(ctx context.Context, req tutorDomain.ExerciseRequest) (tutorDomain.Exercise, error) {
	return m.generateFn(ctx, req)
}

func (m *mockProvider) CheckAnswer(ctx context.Context, req tutorDomain.CheckRequest) (tutorDomain.CheckResult, error) {
	return m.checkFn(ctx, req)
}

// ---------------------------------------------------------------------------
// Mock: ProviderRegistry
// ---------------------------------------------------------------------------

type mockRegistry struct {
	provider usecase.AIProvider
	err      error
}

func (m *mockRegistry) Get(_ string) (usecase.AIProvider, error) {
	return m.provider, m.err
}

// ---------------------------------------------------------------------------
// Mock: SettingsReader
// ---------------------------------------------------------------------------

type mockSettingsReader struct {
	settings *usecase.UserSettingsView
	err      error
}

func (m *mockSettingsReader) GetByUserID(_ context.Context, _ string) (*usecase.UserSettingsView, error) {
	return m.settings, m.err
}

// ---------------------------------------------------------------------------
// Mock: UserReader
// ---------------------------------------------------------------------------

type mockUserReader struct {
	user *usecase.UserView
	err  error
}

func (m *mockUserReader) GetByID(_ context.Context, _ string) (*usecase.UserView, error) {
	return m.user, m.err
}

// ---------------------------------------------------------------------------
// Mock: eventbus.Publisher (used by ExerciseService)
// ---------------------------------------------------------------------------

type mockPublisher struct {
	events []eventbus.Event
}

func (m *mockPublisher) Publish(event eventbus.Event) {
	m.events = append(m.events, event)
}

// ---------------------------------------------------------------------------
// Flusher-capable ResponseRecorder for SSE tests
// ---------------------------------------------------------------------------

type flushRecorder struct {
	*httptest.ResponseRecorder
}

func (f *flushRecorder) Flush() {
	// no-op — just satisfies http.Flusher
}

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

const testUserID = "user-123"

func defaultSettings() *usecase.UserSettingsView {
	return &usecase.UserSettingsView{
		TargetLanguage:   "en",
		ProficiencyLevel: "b1",
		VocabularyType:   "tech",
		AIModel:          "test-model",
	}
}

func defaultUser() *usecase.UserView {
	return &usecase.UserView{DisplayName: "Test User"}
}

func defaultProvider() *mockProvider {
	return &mockProvider{
		chatFn: func(_ context.Context, _ tutorDomain.ChatRequest) (<-chan tutorDomain.ChatDelta, error) {
			ch := make(chan tutorDomain.ChatDelta, 2)
			ch <- tutorDomain.ChatDelta{Type: "delta", Content: "Hello"}
			ch <- tutorDomain.ChatDelta{Type: "done"}
			close(ch)
			return ch, nil
		},
		generateFn: func(_ context.Context, _ tutorDomain.ExerciseRequest) (tutorDomain.Exercise, error) {
			return tutorDomain.Exercise{Raw: `{"question":"What is Go?"}`}, nil
		},
		checkFn: func(_ context.Context, _ tutorDomain.CheckRequest) (tutorDomain.CheckResult, error) {
			return tutorDomain.CheckResult{Raw: `{"correct":true,"word":"hello"}`}, nil
		},
	}
}

type testSetup struct {
	handler   *handler.TutorHandler
	provider  *mockProvider
	settings  *mockSettingsReader
	users     *mockUserReader
	publisher *mockPublisher
}

func newTestSetup() *testSetup {
	prov := defaultProvider()
	reg := &mockRegistry{provider: prov}
	settings := &mockSettingsReader{settings: defaultSettings()}
	users := &mockUserReader{user: defaultUser()}
	pub := &mockPublisher{}

	chatSvc := usecase.NewChatService(reg, settings, users)
	exerciseSvc := usecase.NewExerciseService(reg, settings, pub)
	h := handler.NewTutorHandler(chatSvc, exerciseSvc)

	return &testSetup{
		handler:   h,
		provider:  prov,
		settings:  settings,
		users:     users,
		publisher: pub,
	}
}

func reqWithUser(r *http.Request, userID string) *http.Request {
	ctx := context.WithValue(r.Context(), middleware.UserIDKey, userID)
	return r.WithContext(ctx)
}

func newJSONRequest(t *testing.T, method, target string, body any) *http.Request {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		require.NoError(t, json.NewEncoder(&buf).Encode(body))
	}
	req, err := http.NewRequestWithContext(context.Background(), method, target, &buf)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	return req
}

func decodeProblem(t *testing.T, rec *httptest.ResponseRecorder) httputil.ProblemDetail {
	t.Helper()
	var p httputil.ProblemDetail
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&p))
	return p
}

// ---------------------------------------------------------------------------
// HandleChat tests
// ---------------------------------------------------------------------------

func TestHandleChat_Unauthorized(t *testing.T) {
	s := newTestSetup()

	req := newJSONRequest(t, http.MethodPost, "/chat", handler.ChatRequest{
		Messages: []tutorDomain.Message{{Role: "user", Content: "hi"}},
	})
	// No user ID in context
	rec := httptest.NewRecorder()
	s.handler.HandleChat(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	p := decodeProblem(t, rec)
	assert.Equal(t, "Unauthorized", p.Title)
}

func TestHandleChat_BadJSON(t *testing.T) {
	s := newTestSetup()

	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, "/chat", strings.NewReader("{invalid"))
	require.NoError(t, err)
	req = reqWithUser(req, testUserID)
	rec := httptest.NewRecorder()

	s.handler.HandleChat(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	p := decodeProblem(t, rec)
	assert.Equal(t, "Bad Request", p.Title)
	assert.Contains(t, p.Detail, "invalid request body")
}

func TestHandleChat_EmptyMessages(t *testing.T) {
	s := newTestSetup()

	req := newJSONRequest(t, http.MethodPost, "/chat", handler.ChatRequest{
		Messages: []tutorDomain.Message{},
	})
	req = reqWithUser(req, testUserID)
	rec := httptest.NewRecorder()

	s.handler.HandleChat(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	p := decodeProblem(t, rec)
	assert.Contains(t, p.Detail, "messages must not be empty")
}

func TestHandleChat_TooManyMessages(t *testing.T) {
	s := newTestSetup()

	msgs := make([]tutorDomain.Message, 101)
	for i := range msgs {
		msgs[i] = tutorDomain.Message{Role: "user", Content: "msg"}
	}

	req := newJSONRequest(t, http.MethodPost, "/chat", handler.ChatRequest{Messages: msgs})
	req = reqWithUser(req, testUserID)
	rec := httptest.NewRecorder()

	s.handler.HandleChat(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	p := decodeProblem(t, rec)
	assert.Contains(t, p.Detail, "too many messages")
}

func TestHandleChat_MessageTooLong(t *testing.T) {
	s := newTestSetup()

	longContent := strings.Repeat("a", 16001)
	req := newJSONRequest(t, http.MethodPost, "/chat", handler.ChatRequest{
		Messages: []tutorDomain.Message{{Role: "user", Content: longContent}},
	})
	req = reqWithUser(req, testUserID)
	rec := httptest.NewRecorder()

	s.handler.HandleChat(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	p := decodeProblem(t, rec)
	assert.Contains(t, p.Detail, "message content too long")
}

func TestHandleChat_ServiceError(t *testing.T) {
	s := newTestSetup()
	s.provider.chatFn = func(_ context.Context, _ tutorDomain.ChatRequest) (<-chan tutorDomain.ChatDelta, error) {
		return nil, errors.New("provider down")
	}

	req := newJSONRequest(t, http.MethodPost, "/chat", handler.ChatRequest{
		Messages: []tutorDomain.Message{{Role: "user", Content: "hi"}},
	})
	req = reqWithUser(req, testUserID)
	rec := httptest.NewRecorder()

	s.handler.HandleChat(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestHandleChat_SSEStreaming(t *testing.T) {
	s := newTestSetup()
	s.provider.chatFn = func(_ context.Context, _ tutorDomain.ChatRequest) (<-chan tutorDomain.ChatDelta, error) {
		ch := make(chan tutorDomain.ChatDelta, 3)
		ch <- tutorDomain.ChatDelta{Type: "delta", Content: "Hello"}
		ch <- tutorDomain.ChatDelta{Type: "delta", Content: " world"}
		ch <- tutorDomain.ChatDelta{Type: "done"}
		close(ch)
		return ch, nil
	}

	req := newJSONRequest(t, http.MethodPost, "/chat", handler.ChatRequest{
		Messages: []tutorDomain.Message{{Role: "user", Content: "hi"}},
	})
	req = reqWithUser(req, testUserID)

	rec := httptest.NewRecorder()
	w := &flushRecorder{ResponseRecorder: rec}

	s.handler.HandleChat(w, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "text/event-stream", rec.Header().Get("Content-Type"))
	assert.Equal(t, "no-cache", rec.Header().Get("Cache-Control"))
	assert.Equal(t, "keep-alive", rec.Header().Get("Connection"))

	body := rec.Body.String()
	assert.Contains(t, body, "data: ")
	assert.Contains(t, body, `"Hello"`)
	assert.Contains(t, body, `"done"`)

	// Count SSE events — each "data: " line pair
	lines := strings.Split(strings.TrimSpace(body), "\n\n")
	assert.Len(t, lines, 3, "expected 3 SSE events (2 deltas + done)")
}

func TestHandleChat_ExactlyMaxMessages(t *testing.T) {
	s := newTestSetup()

	msgs := make([]tutorDomain.Message, 100)
	for i := range msgs {
		msgs[i] = tutorDomain.Message{Role: "user", Content: "msg"}
	}

	req := newJSONRequest(t, http.MethodPost, "/chat", handler.ChatRequest{Messages: msgs})
	req = reqWithUser(req, testUserID)

	rec := httptest.NewRecorder()
	w := &flushRecorder{ResponseRecorder: rec}

	s.handler.HandleChat(w, req)

	// 100 messages is the boundary — should succeed (not 400)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestHandleChat_ExactlyMaxContentLength(t *testing.T) {
	s := newTestSetup()

	content := strings.Repeat("a", 16000)
	req := newJSONRequest(t, http.MethodPost, "/chat", handler.ChatRequest{
		Messages: []tutorDomain.Message{{Role: "user", Content: content}},
	})
	req = reqWithUser(req, testUserID)

	rec := httptest.NewRecorder()
	w := &flushRecorder{ResponseRecorder: rec}

	s.handler.HandleChat(w, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

// ---------------------------------------------------------------------------
// HandleGenerateExercise tests
// ---------------------------------------------------------------------------

func TestHandleGenerateExercise_Unauthorized(t *testing.T) {
	s := newTestSetup()

	req := newJSONRequest(t, http.MethodPost, "/quiz/generate", nil)
	rec := httptest.NewRecorder()

	s.handler.HandleGenerateExercise(tutorDomain.ModeQuiz)(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestHandleGenerateExercise_ServiceError(t *testing.T) {
	s := newTestSetup()
	s.provider.generateFn = func(_ context.Context, _ tutorDomain.ExerciseRequest) (tutorDomain.Exercise, error) {
		return tutorDomain.Exercise{}, errors.New("AI unavailable")
	}

	req := newJSONRequest(t, http.MethodPost, "/quiz/generate", nil)
	req = reqWithUser(req, testUserID)
	rec := httptest.NewRecorder()

	s.handler.HandleGenerateExercise(tutorDomain.ModeQuiz)(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestHandleGenerateExercise_InvalidJSON(t *testing.T) {
	s := newTestSetup()
	s.provider.generateFn = func(_ context.Context, _ tutorDomain.ExerciseRequest) (tutorDomain.Exercise, error) {
		return tutorDomain.Exercise{Raw: "not valid json {"}, nil
	}

	req := newJSONRequest(t, http.MethodPost, "/quiz/generate", nil)
	req = reqWithUser(req, testUserID)
	rec := httptest.NewRecorder()

	s.handler.HandleGenerateExercise(tutorDomain.ModeQuiz)(rec, req)

	assert.Equal(t, http.StatusBadGateway, rec.Code)
	p := decodeProblem(t, rec)
	assert.Contains(t, p.Detail, "AI model returned invalid response")
}

func TestHandleGenerateExercise_Success(t *testing.T) {
	s := newTestSetup()
	rawJSON := `{"question":"Translate: hello","options":["hola","salut","hallo"],"answer":"hola"}`
	s.provider.generateFn = func(_ context.Context, _ tutorDomain.ExerciseRequest) (tutorDomain.Exercise, error) {
		return tutorDomain.Exercise{Raw: rawJSON}, nil
	}

	req := newJSONRequest(t, http.MethodPost, "/quiz/generate", nil)
	req = reqWithUser(req, testUserID)
	rec := httptest.NewRecorder()

	s.handler.HandleGenerateExercise(tutorDomain.ModeQuiz)(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))
	assert.Equal(t, rawJSON, rec.Body.String())
}

func TestHandleGenerateExercise_AllModes(t *testing.T) {
	modes := []tutorDomain.Mode{
		tutorDomain.ModeQuiz,
		tutorDomain.ModeTranslate,
		tutorDomain.ModeGap,
		tutorDomain.ModeScramble,
	}

	for _, mode := range modes {
		t.Run(string(mode), func(t *testing.T) {
			s := newTestSetup()
			raw := fmt.Sprintf(`{"mode":"%s","ok":true}`, mode)
			s.provider.generateFn = func(_ context.Context, req tutorDomain.ExerciseRequest) (tutorDomain.Exercise, error) {
				assert.Equal(t, mode, req.Mode)
				return tutorDomain.Exercise{Raw: raw}, nil
			}

			req := newJSONRequest(t, http.MethodPost, "/"+string(mode)+"/generate", nil)
			req = reqWithUser(req, testUserID)
			rec := httptest.NewRecorder()

			s.handler.HandleGenerateExercise(mode)(rec, req)

			assert.Equal(t, http.StatusOK, rec.Code)
			assert.Equal(t, raw, rec.Body.String())
		})
	}
}

// ---------------------------------------------------------------------------
// HandleCheckAnswer tests
// ---------------------------------------------------------------------------

func TestHandleCheckAnswer_Unauthorized(t *testing.T) {
	s := newTestSetup()

	req := newJSONRequest(t, http.MethodPost, "/quiz/answer", map[string]string{
		"answer":  "hola",
		"context": "{}",
	})
	rec := httptest.NewRecorder()

	s.handler.HandleCheckAnswer(tutorDomain.ModeQuiz)(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestHandleCheckAnswer_BadJSON(t *testing.T) {
	s := newTestSetup()

	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, "/quiz/answer", strings.NewReader("{bad"))
	require.NoError(t, err)
	req = reqWithUser(req, testUserID)
	rec := httptest.NewRecorder()

	s.handler.HandleCheckAnswer(tutorDomain.ModeQuiz)(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	p := decodeProblem(t, rec)
	assert.Contains(t, p.Detail, "invalid request body")
}

func TestHandleCheckAnswer_AnswerTooLong(t *testing.T) {
	s := newTestSetup()

	req := newJSONRequest(t, http.MethodPost, "/quiz/answer", map[string]string{
		"answer":  strings.Repeat("x", 16001),
		"context": "{}",
	})
	req = reqWithUser(req, testUserID)
	rec := httptest.NewRecorder()

	s.handler.HandleCheckAnswer(tutorDomain.ModeQuiz)(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	p := decodeProblem(t, rec)
	assert.Contains(t, p.Detail, "answer too long")
}

func TestHandleCheckAnswer_ContextTooLong(t *testing.T) {
	s := newTestSetup()

	req := newJSONRequest(t, http.MethodPost, "/quiz/answer", map[string]string{
		"answer":  "hola",
		"context": strings.Repeat("x", 32001),
	})
	req = reqWithUser(req, testUserID)
	rec := httptest.NewRecorder()

	s.handler.HandleCheckAnswer(tutorDomain.ModeQuiz)(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	p := decodeProblem(t, rec)
	assert.Contains(t, p.Detail, "context too long")
}

func TestHandleCheckAnswer_ServiceError(t *testing.T) {
	s := newTestSetup()
	s.provider.checkFn = func(_ context.Context, _ tutorDomain.CheckRequest) (tutorDomain.CheckResult, error) {
		return tutorDomain.CheckResult{}, errors.New("AI error")
	}

	req := newJSONRequest(t, http.MethodPost, "/quiz/answer", map[string]string{
		"answer":  "hola",
		"context": `{"question":"hello"}`,
	})
	req = reqWithUser(req, testUserID)
	rec := httptest.NewRecorder()

	s.handler.HandleCheckAnswer(tutorDomain.ModeQuiz)(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestHandleCheckAnswer_InvalidJSONFromAI(t *testing.T) {
	s := newTestSetup()
	s.provider.checkFn = func(_ context.Context, _ tutorDomain.CheckRequest) (tutorDomain.CheckResult, error) {
		return tutorDomain.CheckResult{Raw: "not json"}, nil
	}

	req := newJSONRequest(t, http.MethodPost, "/quiz/answer", map[string]string{
		"answer":  "hola",
		"context": "{}",
	})
	req = reqWithUser(req, testUserID)
	rec := httptest.NewRecorder()

	s.handler.HandleCheckAnswer(tutorDomain.ModeQuiz)(rec, req)

	assert.Equal(t, http.StatusBadGateway, rec.Code)
	p := decodeProblem(t, rec)
	assert.Contains(t, p.Detail, "AI model returned invalid response")
}

func TestHandleCheckAnswer_Success(t *testing.T) {
	s := newTestSetup()
	s.provider.checkFn = func(_ context.Context, _ tutorDomain.CheckRequest) (tutorDomain.CheckResult, error) {
		return tutorDomain.CheckResult{Raw: `{"correct":true,"word":"hello"}`}, nil
	}

	ctxJSON := `{"question":"translate hello","language":"en"}`
	req := newJSONRequest(t, http.MethodPost, "/quiz/answer", map[string]string{
		"answer":  "hola",
		"context": ctxJSON,
	})
	req = reqWithUser(req, testUserID)
	rec := httptest.NewRecorder()

	s.handler.HandleCheckAnswer(tutorDomain.ModeQuiz)(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))
	assert.Equal(t, `{"correct":true,"word":"hello"}`, rec.Body.String())
}

func TestHandleCheckAnswer_AnswerBoundary(t *testing.T) {
	s := newTestSetup()

	// Exactly 16000 chars — should pass
	req := newJSONRequest(t, http.MethodPost, "/quiz/answer", map[string]string{
		"answer":  strings.Repeat("x", 16000),
		"context": "{}",
	})
	req = reqWithUser(req, testUserID)
	rec := httptest.NewRecorder()

	s.handler.HandleCheckAnswer(tutorDomain.ModeQuiz)(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestHandleCheckAnswer_ContextBoundary(t *testing.T) {
	s := newTestSetup()

	// Exactly 32000 chars — should pass
	req := newJSONRequest(t, http.MethodPost, "/quiz/answer", map[string]string{
		"answer":  "x",
		"context": strings.Repeat("x", 32000),
	})
	req = reqWithUser(req, testUserID)
	rec := httptest.NewRecorder()

	s.handler.HandleCheckAnswer(tutorDomain.ModeQuiz)(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

// ---------------------------------------------------------------------------
// Routes smoke test
// ---------------------------------------------------------------------------

// nonFlushWriter is an http.ResponseWriter that does NOT implement http.Flusher.
type nonFlushWriter struct {
	code   int
	header http.Header
	body   bytes.Buffer
}

func (w *nonFlushWriter) Header() http.Header         { return w.header }
func (w *nonFlushWriter) Write(b []byte) (int, error)  { return w.body.Write(b) }
func (w *nonFlushWriter) WriteHeader(statusCode int)   { w.code = statusCode }

func TestHandleChat_NoFlusherSupport(t *testing.T) {
	s := newTestSetup()

	req := newJSONRequest(t, http.MethodPost, "/chat", handler.ChatRequest{
		Messages: []tutorDomain.Message{{Role: "user", Content: "hi"}},
	})
	req = reqWithUser(req, testUserID)

	w := &nonFlushWriter{header: make(http.Header)}
	s.handler.HandleChat(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.code)
	assert.Contains(t, w.body.String(), "streaming not supported")
}

func TestHandleCheckAnswer_ValidJSONPassedThrough(t *testing.T) {
	s := newTestSetup()
	s.provider.checkFn = func(_ context.Context, _ tutorDomain.CheckRequest) (tutorDomain.CheckResult, error) {
		return tutorDomain.CheckResult{Raw: `{"correct":false,"explanation":"nope"}`}, nil
	}

	req := newJSONRequest(t, http.MethodPost, "/quiz/answer", map[string]string{
		"answer":  "x",
		"context": `{"language":"en"}`,
	})
	req = reqWithUser(req, testUserID)
	rec := httptest.NewRecorder()

	s.handler.HandleCheckAnswer(tutorDomain.ModeQuiz)(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, `{"correct":false,"explanation":"nope"}`, rec.Body.String())
}

func TestRoutes_RegistersExpectedPaths(t *testing.T) {
	s := newTestSetup()
	router := s.handler.Routes()

	// Verify that the router is not nil and is functional
	require.NotNil(t, router)

	// Smoke-test one route through the chi router
	req := newJSONRequest(t, http.MethodPost, "/chat", handler.ChatRequest{
		Messages: []tutorDomain.Message{{Role: "user", Content: "hi"}},
	})
	req = reqWithUser(req, testUserID)

	rec := httptest.NewRecorder()
	w := &flushRecorder{ResponseRecorder: rec}

	router.ServeHTTP(w, req)

	// Should reach the handler and succeed (SSE)
	assert.Equal(t, http.StatusOK, rec.Code)
}
