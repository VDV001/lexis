package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lexis-app/lexis-api/internal/modules/auth/domain"
	"github.com/lexis-app/lexis-api/internal/modules/auth/handler"
	"github.com/lexis-app/lexis-api/internal/modules/auth/usecase"
	"github.com/lexis-app/lexis-api/internal/shared/httputil"
	"github.com/lexis-app/lexis-api/internal/shared/middleware"
)

// ---------------------------------------------------------------------------
// In-memory repository implementations
// ---------------------------------------------------------------------------

// InMemoryUserRepo stores users in a map keyed by email (for uniqueness) and
// by ID (for lookups).
type InMemoryUserRepo struct {
	mu      sync.RWMutex
	byID    map[string]*domain.User
	byEmail map[string]*domain.User
}

func NewInMemoryUserRepo() *InMemoryUserRepo {
	return &InMemoryUserRepo{
		byID:    make(map[string]*domain.User),
		byEmail: make(map[string]*domain.User),
	}
}

func (r *InMemoryUserRepo) Create(_ context.Context, user *domain.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.byEmail[user.Email]; exists {
		return domain.ErrEmailTaken
	}

	user.ID = uuid.NewString()
	user.CreatedAt = time.Now()

	// Store a copy so mutations don't leak.
	clone := *user
	r.byID[user.ID] = &clone
	r.byEmail[user.Email] = &clone
	return nil
}

func (r *InMemoryUserRepo) GetByID(_ context.Context, id string) (*domain.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	u, ok := r.byID[id]
	if !ok {
		return nil, domain.ErrUserNotFound
	}
	clone := *u
	return &clone, nil
}

func (r *InMemoryUserRepo) GetByEmail(_ context.Context, email string) (*domain.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	u, ok := r.byEmail[email]
	if !ok {
		return nil, domain.ErrUserNotFound
	}
	clone := *u
	return &clone, nil
}

func (r *InMemoryUserRepo) Update(_ context.Context, user *domain.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.byID[user.ID]; !ok {
		return domain.ErrUserNotFound
	}
	clone := *user
	r.byID[user.ID] = &clone
	r.byEmail[user.Email] = &clone
	return nil
}

// InMemoryTokenRepo stores refresh tokens keyed by hash.
type InMemoryTokenRepo struct {
	mu     sync.RWMutex
	byHash map[string]*domain.RefreshToken
}

func NewInMemoryTokenRepo() *InMemoryTokenRepo {
	return &InMemoryTokenRepo{
		byHash: make(map[string]*domain.RefreshToken),
	}
}

func (r *InMemoryTokenRepo) CreateRefreshToken(_ context.Context, token *domain.RefreshToken) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	token.ID = uuid.NewString()
	token.CreatedAt = time.Now()

	clone := *token
	r.byHash[token.TokenHash] = &clone
	return nil
}

func (r *InMemoryTokenRepo) GetByHash(_ context.Context, hash string) (*domain.RefreshToken, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	t, ok := r.byHash[hash]
	if !ok {
		return nil, domain.ErrTokenNotFound
	}
	clone := *t
	return &clone, nil
}

func (r *InMemoryTokenRepo) RevokeByHash(_ context.Context, hash string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	t, ok := r.byHash[hash]
	if !ok {
		return domain.ErrTokenNotFound
	}
	// Atomic: only succeed if not already revoked (matches Postgres WHERE revoked_at IS NULL)
	if t.RevokedAt != nil {
		return domain.ErrTokenNotFound
	}
	now := time.Now()
	t.RevokedAt = &now
	return nil
}

func (r *InMemoryTokenRepo) RevokeAllForUser(_ context.Context, userID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	for _, t := range r.byHash {
		if t.UserID == userID {
			t.RevokedAt = &now
		}
	}
	return nil
}

// InMemorySettingsRepo stores user settings keyed by user ID.
type InMemorySettingsRepo struct {
	mu       sync.RWMutex
	byUserID map[string]*domain.UserSettings
}

func NewInMemorySettingsRepo() *InMemorySettingsRepo {
	return &InMemorySettingsRepo{
		byUserID: make(map[string]*domain.UserSettings),
	}
}

func (r *InMemorySettingsRepo) GetByUserID(_ context.Context, userID string) (*domain.UserSettings, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	s, ok := r.byUserID[userID]
	if !ok {
		defaults := domain.DefaultSettings(userID)
		return &defaults, nil
	}
	clone := *s
	return &clone, nil
}

func (r *InMemorySettingsRepo) Upsert(_ context.Context, settings *domain.UserSettings) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	settings.UpdatedAt = time.Now()
	clone := *settings
	r.byUserID[settings.UserID] = &clone
	return nil
}

// InMemoryBlacklist stores blacklisted token hashes.
type InMemoryBlacklist struct {
	mu     sync.RWMutex
	hashes map[string]struct{}
}

func NewInMemoryBlacklist() *InMemoryBlacklist {
	return &InMemoryBlacklist{
		hashes: make(map[string]struct{}),
	}
}

func (b *InMemoryBlacklist) Add(_ context.Context, tokenHash string, _ time.Duration) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.hashes[tokenHash] = struct{}{}
	return nil
}

func (b *InMemoryBlacklist) IsBlacklisted(_ context.Context, tokenHash string) (bool, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	_, ok := b.hashes[tokenHash]
	return ok, nil
}

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

const testJWTSecret = "test-secret-key-for-handler-tests"

// newTestHandler builds a fully-wired AuthHandler backed by in-memory repos.
func newTestHandler(t *testing.T) *handler.AuthHandler {
	t.Helper()

	svc := usecase.NewAuthService(
		NewInMemoryUserRepo(),
		NewInMemoryTokenRepo(),
		NewInMemorySettingsRepo(),
		NewInMemoryBlacklist(),
		testJWTSecret,
		15*time.Minute, // access TTL
		720*time.Hour,  // refresh TTL
	)

	return handler.NewAuthHandler(svc, false, 720*time.Hour)
}

// newJSONRequest creates an *http.Request with context.Background(), the given
// method/target, and a JSON body. It satisfies the noctx linter.
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

// registerUser registers a user via the handler and returns the parsed
// AuthResponse so subsequent tests can reuse the tokens.
func registerUser(t *testing.T, h *handler.AuthHandler, email, password, displayName string) handler.AuthResponse {
	t.Helper()

	req := newJSONRequest(t, http.MethodPost, "/register", handler.RegisterRequest{
		Email:       email,
		Password:    password,
		DisplayName: displayName,
	})
	rec := httptest.NewRecorder()

	h.Register(rec, req)
	require.Equal(t, http.StatusCreated, rec.Code, "register helper must succeed")

	var resp handler.AuthResponse
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	return resp
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestRegister_Success(t *testing.T) {
	h := newTestHandler(t)

	req := newJSONRequest(t, http.MethodPost, "/register", handler.RegisterRequest{
		Email:       "alice@example.com",
		Password:    "securePass1!",
		DisplayName: "Alice",
	})
	rec := httptest.NewRecorder()

	h.Register(rec, req)

	assert.Equal(t, http.StatusCreated, rec.Code)
	assert.Contains(t, rec.Header().Get("Content-Type"), "application/json")

	var resp handler.AuthResponse
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))

	assert.NotEmpty(t, resp.User.ID)
	assert.Equal(t, "alice@example.com", resp.User.Email)
	assert.Equal(t, "Alice", resp.User.DisplayName)
	assert.NotEmpty(t, resp.AccessToken)
	assert.NotEmpty(t, resp.RefreshToken)

	// A refresh_token cookie must be set.
	cookies := rec.Result().Cookies()
	var found bool
	for _, c := range cookies {
		if c.Name == "refresh_token" {
			found = true
			assert.Equal(t, resp.RefreshToken, c.Value)
			assert.True(t, c.HttpOnly)
		}
	}
	assert.True(t, found, "refresh_token cookie should be set")
}

func TestRegister_InvalidEmail(t *testing.T) {
	h := newTestHandler(t)

	req := newJSONRequest(t, http.MethodPost, "/register", handler.RegisterRequest{
		Email:       "not-an-email",
		Password:    "securePass1!",
		DisplayName: "Alice",
	})
	rec := httptest.NewRecorder()

	h.Register(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var problem httputil.ProblemDetail
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&problem))
	assert.Equal(t, http.StatusBadRequest, problem.Status)
	assert.Equal(t, "Validation failed", problem.Title)
}

func TestRegister_DuplicateEmail(t *testing.T) {
	h := newTestHandler(t)

	// First registration succeeds.
	registerUser(t, h, "dup@example.com", "password123", "First")

	// Second registration with same email should fail.
	req := newJSONRequest(t, http.MethodPost, "/register", handler.RegisterRequest{
		Email:       "dup@example.com",
		Password:    "anotherPass1!",
		DisplayName: "Second",
	})
	rec := httptest.NewRecorder()

	h.Register(rec, req)

	assert.Equal(t, http.StatusConflict, rec.Code)

	var problem httputil.ProblemDetail
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&problem))
	assert.Equal(t, http.StatusConflict, problem.Status)
	assert.Equal(t, "Email already taken", problem.Title)
}

func TestLogin_Success(t *testing.T) {
	h := newTestHandler(t)

	// Register first.
	registerUser(t, h, "bob@example.com", "bobPassword1!", "Bob")

	// Now login.
	req := newJSONRequest(t, http.MethodPost, "/login", handler.LoginRequest{
		Email:    "bob@example.com",
		Password: "bobPassword1!",
	})
	rec := httptest.NewRecorder()

	h.Login(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp handler.AuthResponse
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))

	assert.Equal(t, "bob@example.com", resp.User.Email)
	assert.Equal(t, "Bob", resp.User.DisplayName)
	assert.NotEmpty(t, resp.AccessToken)
	assert.NotEmpty(t, resp.RefreshToken)
}

func TestLogin_WrongPassword(t *testing.T) {
	h := newTestHandler(t)

	registerUser(t, h, "carol@example.com", "correctPass1!", "Carol")

	req := newJSONRequest(t, http.MethodPost, "/login", handler.LoginRequest{
		Email:    "carol@example.com",
		Password: "wrongPassword",
	})
	rec := httptest.NewRecorder()

	h.Login(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)

	var problem httputil.ProblemDetail
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&problem))
	assert.Equal(t, http.StatusUnauthorized, problem.Status)
	assert.Equal(t, "Invalid credentials", problem.Title)
}

func TestRefresh_Success(t *testing.T) {
	h := newTestHandler(t)

	// Register to get tokens.
	auth := registerUser(t, h, "dave@example.com", "davePass123!", "Dave")

	// Refresh using the refresh token.
	req := newJSONRequest(t, http.MethodPost, "/refresh", handler.RefreshRequest{
		RefreshToken: auth.RefreshToken,
	})
	rec := httptest.NewRecorder()

	h.Refresh(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp handler.TokenResponse
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))

	assert.NotEmpty(t, resp.AccessToken)
	assert.NotEmpty(t, resp.RefreshToken)
	// The new refresh token must differ from the old one (rotation).
	assert.NotEqual(t, auth.RefreshToken, resp.RefreshToken)
}

func TestLogout_Success(t *testing.T) {
	h := newTestHandler(t)

	// Register to get tokens.
	auth := registerUser(t, h, "eve@example.com", "evePass1234!", "Eve")

	// Logout — requires userID in context and a refresh token.
	req := newJSONRequest(t, http.MethodPost, "/logout", handler.RefreshRequest{
		RefreshToken: auth.RefreshToken,
	})

	// Inject userID into context the way middleware would.
	ctx := context.WithValue(req.Context(), middleware.UserIDKey, auth.User.ID)
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()

	h.Logout(rec, req)

	assert.Equal(t, http.StatusNoContent, rec.Code)

	// Verify the old refresh token is now unusable (revoked).
	req2 := newJSONRequest(t, http.MethodPost, "/refresh", handler.RefreshRequest{
		RefreshToken: auth.RefreshToken,
	})
	rec2 := httptest.NewRecorder()

	h.Refresh(rec2, req2)

	assert.Equal(t, http.StatusUnauthorized, rec2.Code,
		"refresh with revoked token should fail")
}

// ---------------------------------------------------------------------------
// Additional auth handler tests — error paths & edge cases
// ---------------------------------------------------------------------------

func TestRegister_BadJSON(t *testing.T) {
	h := newTestHandler(t)

	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, "/register",
		bytes.NewBufferString(`{bad json`))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.Register(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	var problem httputil.ProblemDetail
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&problem))
	assert.Equal(t, "Invalid request body", problem.Title)
}

func TestRegister_MissingFields(t *testing.T) {
	h := newTestHandler(t)

	// Missing password and display_name.
	req := newJSONRequest(t, http.MethodPost, "/register", handler.RegisterRequest{
		Email: "a@b.com",
	})
	rec := httptest.NewRecorder()

	h.Register(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	var problem httputil.ProblemDetail
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&problem))
	assert.Equal(t, "Validation failed", problem.Title)
}

func TestRegister_ShortPassword(t *testing.T) {
	h := newTestHandler(t)

	req := newJSONRequest(t, http.MethodPost, "/register", handler.RegisterRequest{
		Email:       "short@example.com",
		Password:    "short",
		DisplayName: "Short",
	})
	rec := httptest.NewRecorder()

	h.Register(rec, req)

	// Validator catches min=8 before service; should be 400.
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestLogin_BadJSON(t *testing.T) {
	h := newTestHandler(t)

	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, "/login",
		bytes.NewBufferString(`not json`))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.Login(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	var problem httputil.ProblemDetail
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&problem))
	assert.Equal(t, "Invalid request body", problem.Title)
}

func TestLogin_ValidationError(t *testing.T) {
	h := newTestHandler(t)

	// Missing email.
	req := newJSONRequest(t, http.MethodPost, "/login", handler.LoginRequest{
		Password: "somepassword",
	})
	rec := httptest.NewRecorder()

	h.Login(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	var problem httputil.ProblemDetail
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&problem))
	assert.Equal(t, "Validation failed", problem.Title)
}

func TestLogin_UserNotFound(t *testing.T) {
	h := newTestHandler(t)

	req := newJSONRequest(t, http.MethodPost, "/login", handler.LoginRequest{
		Email:    "nobody@example.com",
		Password: "password123",
	})
	rec := httptest.NewRecorder()

	h.Login(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	var problem httputil.ProblemDetail
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&problem))
	assert.Equal(t, "Invalid credentials", problem.Title)
}

func TestRefresh_MissingToken(t *testing.T) {
	h := newTestHandler(t)

	// Empty body, no cookie.
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, "/refresh",
		bytes.NewBufferString(`{}`))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.Refresh(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	var problem httputil.ProblemDetail
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&problem))
	assert.Equal(t, "Missing refresh token", problem.Title)
}

func TestRefresh_FromCookie(t *testing.T) {
	h := newTestHandler(t)

	auth := registerUser(t, h, "cookie@example.com", "cookiePass1!", "Cookie")

	// Send refresh token via cookie, not body.
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, "/refresh",
		bytes.NewBufferString(`{}`))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: "refresh_token", Value: auth.RefreshToken})
	rec := httptest.NewRecorder()

	h.Refresh(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	var resp handler.TokenResponse
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	assert.NotEmpty(t, resp.AccessToken)
	assert.NotEmpty(t, resp.RefreshToken)
}

func TestRefresh_InvalidToken(t *testing.T) {
	h := newTestHandler(t)

	req := newJSONRequest(t, http.MethodPost, "/refresh", handler.RefreshRequest{
		RefreshToken: "totally-bogus-token",
	})
	rec := httptest.NewRecorder()

	h.Refresh(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestLogout_MissingUserID(t *testing.T) {
	h := newTestHandler(t)

	// No userID in context.
	req := newJSONRequest(t, http.MethodPost, "/logout", handler.RefreshRequest{
		RefreshToken: "some-token",
	})
	rec := httptest.NewRecorder()

	h.Logout(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	var problem httputil.ProblemDetail
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&problem))
	assert.Equal(t, "Unauthorized", problem.Title)
}

func TestLogout_MissingRefreshToken(t *testing.T) {
	h := newTestHandler(t)

	auth := registerUser(t, h, "logoutmissing@example.com", "password1234!", "LM")

	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, "/logout",
		bytes.NewBufferString(`{}`))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	ctx := context.WithValue(req.Context(), middleware.UserIDKey, auth.User.ID)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	h.Logout(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	var problem httputil.ProblemDetail
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&problem))
	assert.Equal(t, "Missing refresh token", problem.Title)
}

func TestLogout_FromCookie(t *testing.T) {
	h := newTestHandler(t)

	auth := registerUser(t, h, "logoutcookie@example.com", "password1234!", "LC")

	// Send refresh token via cookie.
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, "/logout",
		bytes.NewBufferString(`{}`))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: "refresh_token", Value: auth.RefreshToken})
	ctx := context.WithValue(req.Context(), middleware.UserIDKey, auth.User.ID)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	h.Logout(rec, req)

	assert.Equal(t, http.StatusNoContent, rec.Code)

	// Cookie should be cleared.
	cookies := rec.Result().Cookies()
	for _, c := range cookies {
		if c.Name == "refresh_token" {
			assert.Equal(t, -1, c.MaxAge, "cookie should be cleared")
		}
	}
}

func TestLogout_InvalidToken(t *testing.T) {
	h := newTestHandler(t)

	auth := registerUser(t, h, "logoutinvalid@example.com", "password1234!", "LI")

	req := newJSONRequest(t, http.MethodPost, "/logout", handler.RefreshRequest{
		RefreshToken: "bogus-token",
	})
	ctx := context.WithValue(req.Context(), middleware.UserIDKey, auth.User.ID)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	h.Logout(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestLogoutAll_Success(t *testing.T) {
	h := newTestHandler(t)

	auth := registerUser(t, h, "logoutall@example.com", "password1234!", "LA")

	req := newJSONRequest(t, http.MethodPost, "/logout-all", nil)
	ctx := context.WithValue(req.Context(), middleware.UserIDKey, auth.User.ID)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	h.LogoutAll(rec, req)

	assert.Equal(t, http.StatusNoContent, rec.Code)

	// Verify old refresh token is revoked.
	req2 := newJSONRequest(t, http.MethodPost, "/refresh", handler.RefreshRequest{
		RefreshToken: auth.RefreshToken,
	})
	rec2 := httptest.NewRecorder()
	h.Refresh(rec2, req2)
	assert.Equal(t, http.StatusUnauthorized, rec2.Code)
}

func TestLogoutAll_MissingUserID(t *testing.T) {
	h := newTestHandler(t)

	req := newJSONRequest(t, http.MethodPost, "/logout-all", nil)
	rec := httptest.NewRecorder()

	h.LogoutAll(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	var problem httputil.ProblemDetail
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&problem))
	assert.Equal(t, "Unauthorized", problem.Title)
}

// ---------------------------------------------------------------------------
// HandleGetModels tests
// ---------------------------------------------------------------------------

func TestHandleGetModels(t *testing.T) {
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "/models", nil)
	require.NoError(t, err)
	rec := httptest.NewRecorder()

	handler.HandleGetModels(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Header().Get("Content-Type"), "application/json")

	var resp struct {
		Models []handler.AIModel `json:"models"`
	}
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	assert.NotEmpty(t, resp.Models)
	assert.Len(t, resp.Models, 6)

	// Verify first model has expected fields.
	assert.Equal(t, "claude-sonnet-4-20250514", resp.Models[0].ID)
	assert.True(t, resp.Models[0].Available)
}

// ---------------------------------------------------------------------------
// UserHandler (settings/profile) tests
// ---------------------------------------------------------------------------

// newTestUserHandler builds a UserHandler backed by in-memory repos and returns
// it along with a pre-registered userID.
func newTestUserHandler(t *testing.T) (*handler.UserHandler, string) {
	t.Helper()

	userRepo := NewInMemoryUserRepo()
	settingsRepo := NewInMemorySettingsRepo()

	svc := usecase.NewUserService(userRepo, settingsRepo)
	h := handler.NewUserHandler(svc)

	// Create a user directly in the repo so we have a known ID.
	user := &domain.User{
		Email:        "profile@example.com",
		PasswordHash: "unused-hash",
		DisplayName:  "ProfileUser",
	}
	require.NoError(t, userRepo.Create(context.Background(), user))

	// Also seed default settings.
	defaults := domain.DefaultSettings(user.ID)
	require.NoError(t, settingsRepo.Upsert(context.Background(), &defaults))

	return h, user.ID
}

func TestGetProfile_Success(t *testing.T) {
	h, userID := newTestUserHandler(t)

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "/me", nil)
	require.NoError(t, err)
	ctx := context.WithValue(req.Context(), middleware.UserIDKey, userID)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	h.GetProfile(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	var resp handler.UserResponse
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	assert.Equal(t, userID, resp.ID)
	assert.Equal(t, "profile@example.com", resp.Email)
	assert.Equal(t, "ProfileUser", resp.DisplayName)
}

func TestGetProfile_MissingUserID(t *testing.T) {
	h, _ := newTestUserHandler(t)

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "/me", nil)
	require.NoError(t, err)
	rec := httptest.NewRecorder()

	h.GetProfile(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestGetProfile_UserNotFound(t *testing.T) {
	h, _ := newTestUserHandler(t)

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "/me", nil)
	require.NoError(t, err)
	ctx := context.WithValue(req.Context(), middleware.UserIDKey, "nonexistent-id")
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	h.GetProfile(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestUpdateProfile_Success(t *testing.T) {
	h, userID := newTestUserHandler(t)

	newName := "UpdatedName"
	req := newJSONRequest(t, http.MethodPatch, "/me", handler.ProfileUpdateRequest{
		DisplayName: &newName,
	})
	ctx := context.WithValue(req.Context(), middleware.UserIDKey, userID)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	h.UpdateProfile(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	var resp handler.UserResponse
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	assert.Equal(t, "UpdatedName", resp.DisplayName)
}

func TestUpdateProfile_MissingUserID(t *testing.T) {
	h, _ := newTestUserHandler(t)

	newName := "X"
	req := newJSONRequest(t, http.MethodPatch, "/me", handler.ProfileUpdateRequest{
		DisplayName: &newName,
	})
	rec := httptest.NewRecorder()

	h.UpdateProfile(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestUpdateProfile_BadJSON(t *testing.T) {
	h, userID := newTestUserHandler(t)

	req, err := http.NewRequestWithContext(context.Background(), http.MethodPatch, "/me",
		bytes.NewBufferString(`{bad`))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	ctx := context.WithValue(req.Context(), middleware.UserIDKey, userID)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	h.UpdateProfile(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestUpdateProfile_InvalidDisplayName(t *testing.T) {
	h, userID := newTestUserHandler(t)

	shortName := "X" // too short (min 2)
	req := newJSONRequest(t, http.MethodPatch, "/me", handler.ProfileUpdateRequest{
		DisplayName: &shortName,
	})
	ctx := context.WithValue(req.Context(), middleware.UserIDKey, userID)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	h.UpdateProfile(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestUpdateProfile_AvatarURLTooLong(t *testing.T) {
	h, userID := newTestUserHandler(t)

	longURL := "https://example.com/" + string(make([]byte, 2050))
	req := newJSONRequest(t, http.MethodPatch, "/me", handler.ProfileUpdateRequest{
		AvatarURL: &longURL,
	})
	ctx := context.WithValue(req.Context(), middleware.UserIDKey, userID)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	h.UpdateProfile(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestGetSettings_Success(t *testing.T) {
	h, userID := newTestUserHandler(t)

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "/me/settings", nil)
	require.NoError(t, err)
	ctx := context.WithValue(req.Context(), middleware.UserIDKey, userID)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	h.GetSettings(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	var resp handler.SettingsResponse
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	assert.Equal(t, "en", resp.TargetLanguage)
	assert.Equal(t, "b1", resp.ProficiencyLevel)
	assert.Equal(t, "tech", resp.VocabularyType)
	assert.Equal(t, "claude-sonnet-4-20250514", resp.AIModel)
	assert.Equal(t, 3000, resp.VocabGoal)
	assert.Equal(t, "ru", resp.UILanguage)
}

func TestGetSettings_MissingUserID(t *testing.T) {
	h, _ := newTestUserHandler(t)

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "/me/settings", nil)
	require.NoError(t, err)
	rec := httptest.NewRecorder()

	h.GetSettings(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestUpdateSettings_Success(t *testing.T) {
	h, userID := newTestUserHandler(t)

	body := `{"proficiency_level":"c1","vocab_goal":5000}`
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPut, "/me/settings",
		bytes.NewBufferString(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	ctx := context.WithValue(req.Context(), middleware.UserIDKey, userID)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	h.UpdateSettings(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	var resp handler.SettingsResponse
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	assert.Equal(t, "c1", resp.ProficiencyLevel)
	assert.Equal(t, 5000, resp.VocabGoal)
	// Unchanged fields should retain defaults.
	assert.Equal(t, "en", resp.TargetLanguage)
	assert.Equal(t, "tech", resp.VocabularyType)
}

func TestUpdateSettings_MissingUserID(t *testing.T) {
	h, _ := newTestUserHandler(t)

	req, err := http.NewRequestWithContext(context.Background(), http.MethodPut, "/me/settings",
		bytes.NewBufferString(`{"vocab_goal":5000}`))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.UpdateSettings(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestUpdateSettings_BadJSON(t *testing.T) {
	h, userID := newTestUserHandler(t)

	req, err := http.NewRequestWithContext(context.Background(), http.MethodPut, "/me/settings",
		bytes.NewBufferString(`{bad`))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	ctx := context.WithValue(req.Context(), middleware.UserIDKey, userID)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	h.UpdateSettings(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestUpdateSettings_InvalidValue(t *testing.T) {
	h, userID := newTestUserHandler(t)

	// Invalid proficiency_level.
	body := `{"proficiency_level":"z9"}`
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPut, "/me/settings",
		bytes.NewBufferString(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	ctx := context.WithValue(req.Context(), middleware.UserIDKey, userID)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	h.UpdateSettings(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	var problem httputil.ProblemDetail
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&problem))
	assert.Equal(t, "Invalid settings", problem.Title)
}

func TestUpdateSettings_InvalidVocabGoal(t *testing.T) {
	h, userID := newTestUserHandler(t)

	body := `{"vocab_goal":50}`
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPut, "/me/settings",
		bytes.NewBufferString(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	ctx := context.WithValue(req.Context(), middleware.UserIDKey, userID)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	h.UpdateSettings(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestUpdateSettings_AllFields(t *testing.T) {
	h, userID := newTestUserHandler(t)

	body := `{
		"target_language":"en",
		"proficiency_level":"a2",
		"vocabulary_type":"literary",
		"ai_model":"gpt-4o",
		"vocab_goal":10000,
		"ui_language":"en"
	}`
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPut, "/me/settings",
		bytes.NewBufferString(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	ctx := context.WithValue(req.Context(), middleware.UserIDKey, userID)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	h.UpdateSettings(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	var resp handler.SettingsResponse
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	assert.Equal(t, "en", resp.TargetLanguage)
	assert.Equal(t, "a2", resp.ProficiencyLevel)
	assert.Equal(t, "literary", resp.VocabularyType)
	assert.Equal(t, "gpt-4o", resp.AIModel)
	assert.Equal(t, 10000, resp.VocabGoal)
	assert.Equal(t, "en", resp.UILanguage)
}

func TestUpdateProfile_SetAvatar(t *testing.T) {
	h, userID := newTestUserHandler(t)

	avatarURL := "https://example.com/avatar.png"
	req := newJSONRequest(t, http.MethodPatch, "/me", handler.ProfileUpdateRequest{
		AvatarURL: &avatarURL,
	})
	ctx := context.WithValue(req.Context(), middleware.UserIDKey, userID)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	h.UpdateProfile(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	var resp handler.UserResponse
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	require.NotNil(t, resp.AvatarURL)
	assert.Equal(t, "https://example.com/avatar.png", *resp.AvatarURL)
}

func TestRefresh_EmptyBody(t *testing.T) {
	h := newTestHandler(t)

	// Completely empty body (not even {}).
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, "/refresh",
		bytes.NewBufferString(``))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.Refresh(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	var problem httputil.ProblemDetail
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&problem))
	assert.Equal(t, "Missing refresh token", problem.Title)
}

func TestUpdateSettings_InvalidModel(t *testing.T) {
	h, userID := newTestUserHandler(t)

	body := `{"ai_model":"nonexistent-model"}`
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPut, "/me/settings",
		bytes.NewBufferString(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	ctx := context.WithValue(req.Context(), middleware.UserIDKey, userID)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	h.UpdateSettings(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestUpdateSettings_InvalidUILanguage(t *testing.T) {
	h, userID := newTestUserHandler(t)

	body := `{"ui_language":"fr"}`
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPut, "/me/settings",
		bytes.NewBufferString(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	ctx := context.WithValue(req.Context(), middleware.UserIDKey, userID)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	h.UpdateSettings(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// ---------------------------------------------------------------------------
// Router factory tests (PublicRoutes, ProtectedRoutes, Routes)
// ---------------------------------------------------------------------------

func TestPublicRoutes(t *testing.T) {
	h := newTestHandler(t)
	r := h.PublicRoutes()
	assert.NotNil(t, r)
}

func TestProtectedRoutes(t *testing.T) {
	h := newTestHandler(t)
	r := h.ProtectedRoutes()
	assert.NotNil(t, r)
}

func TestUserHandler_Routes(t *testing.T) {
	h, _ := newTestUserHandler(t)
	r := h.Routes()
	assert.NotNil(t, r)
}

// ---------------------------------------------------------------------------
// Additional edge cases for handleDomainError branches
// ---------------------------------------------------------------------------

func TestRefresh_RevokedToken_DetectsReuse(t *testing.T) {
	h := newTestHandler(t)

	auth := registerUser(t, h, "reuse@example.com", "reusePass123!", "Reuse")
	oldRefresh := auth.RefreshToken

	// First refresh succeeds and rotates the token.
	req := newJSONRequest(t, http.MethodPost, "/refresh", handler.RefreshRequest{
		RefreshToken: oldRefresh,
	})
	rec := httptest.NewRecorder()
	h.Refresh(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)

	// Second refresh with the same (now revoked) token triggers reuse detection.
	req2 := newJSONRequest(t, http.MethodPost, "/refresh", handler.RefreshRequest{
		RefreshToken: oldRefresh,
	})
	rec2 := httptest.NewRecorder()
	h.Refresh(rec2, req2)
	// Should be 401 — either "Token revoked" or "Token not found".
	assert.Equal(t, http.StatusUnauthorized, rec2.Code)
}

func TestUpdateProfile_NonexistentUser(t *testing.T) {
	h, _ := newTestUserHandler(t)

	newName := "Ghost"
	req := newJSONRequest(t, http.MethodPatch, "/me", handler.ProfileUpdateRequest{
		DisplayName: &newName,
	})
	ctx := context.WithValue(req.Context(), middleware.UserIDKey, "nonexistent-user-id")
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	h.UpdateProfile(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestUpdateSettings_InvalidTargetLanguage(t *testing.T) {
	h, userID := newTestUserHandler(t)

	body := `{"target_language":"xx"}`
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPut, "/me/settings",
		bytes.NewBufferString(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	ctx := context.WithValue(req.Context(), middleware.UserIDKey, userID)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	h.UpdateSettings(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestUpdateSettings_InvalidVocabType(t *testing.T) {
	h, userID := newTestUserHandler(t)

	body := `{"vocabulary_type":"slang"}`
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPut, "/me/settings",
		bytes.NewBufferString(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	ctx := context.WithValue(req.Context(), middleware.UserIDKey, userID)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	h.UpdateSettings(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestLogin_EmptyBody(t *testing.T) {
	h := newTestHandler(t)

	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, "/login",
		bytes.NewBufferString(``))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.Login(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestRegister_EmptyBody(t *testing.T) {
	h := newTestHandler(t)

	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, "/register",
		bytes.NewBufferString(``))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.Register(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}
