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

	return handler.NewAuthHandler(svc, false)
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

	var problem handler.ProblemDetail
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

	var problem handler.ProblemDetail
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

	var problem handler.ProblemDetail
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
	ctx := context.WithValue(req.Context(), handler.UserIDKey, auth.User.ID)
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
