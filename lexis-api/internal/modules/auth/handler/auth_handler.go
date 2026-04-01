package handler

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"

	"github.com/lexis-app/lexis-api/internal/modules/auth/domain"
	"github.com/lexis-app/lexis-api/internal/modules/auth/usecase"
	"github.com/lexis-app/lexis-api/internal/shared/middleware"
)

// ---------- Request DTOs ----------

// RegisterRequest is the payload for POST /register.
type RegisterRequest struct {
	Email       string `json:"email" validate:"required,email"`
	Password    string `json:"password" validate:"required,min=8"`
	DisplayName string `json:"display_name" validate:"required,min=2,max=100"`
}

// LoginRequest is the payload for POST /login.
type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

// RefreshRequest is the payload for POST /refresh.
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// ---------- Response DTOs ----------

// AuthResponse is returned by register and login endpoints.
type AuthResponse struct {
	User         UserResponse `json:"user"`
	AccessToken  string       `json:"access_token"`
	RefreshToken string       `json:"refresh_token"`
}

// UserResponse is the public representation of a user.
type UserResponse struct {
	ID          string  `json:"id"`
	Email       string  `json:"email"`
	DisplayName string  `json:"display_name"`
	AvatarURL   *string `json:"avatar_url,omitempty"`
}

// TokenResponse is returned by the refresh endpoint.
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

// ProblemDetail follows RFC 7807 for error responses.
type ProblemDetail struct {
	Type   string `json:"type"`
	Title  string `json:"title"`
	Status int    `json:"status"`
	Detail string `json:"detail"`
}

// ---------- Handler ----------

// AuthHandler handles HTTP requests for the auth module.
type AuthHandler struct {
	service  *usecase.AuthService
	validate *validator.Validate
	secure   bool // whether to set Secure flag on cookies
}

// NewAuthHandler creates a new AuthHandler.
// Set secure to true in production (HTTPS) and false in development.
func NewAuthHandler(service *usecase.AuthService, secure bool) *AuthHandler {
	return &AuthHandler{
		service:  service,
		validate: validator.New(),
		secure:   secure,
	}
}

// PublicRoutes returns a chi.Router with unauthenticated auth routes (register, refresh).
// Login is excluded so callers can mount it separately with rate limiting.
func (h *AuthHandler) PublicRoutes() chi.Router {
	r := chi.NewRouter()
	r.Post("/register", h.Register)
	r.Post("/refresh", h.Refresh)
	return r
}

// ProtectedRoutes returns a chi.Router with authenticated auth routes (logout, logout-all).
func (h *AuthHandler) ProtectedRoutes() chi.Router {
	r := chi.NewRouter()
	r.Post("/logout", h.Logout)
	r.Post("/logout-all", h.LogoutAll)
	return r
}

// Register handles POST /register.
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeProblem(w, http.StatusBadRequest, "Invalid request body", err.Error())
		return
	}
	if err := h.validate.Struct(req); err != nil {
		writeProblem(w, http.StatusBadRequest, "Validation failed", formatValidationErrors(err))
		return
	}

	result, err := h.service.Register(r.Context(), req.Email, req.Password, req.DisplayName)
	if err != nil {
		h.handleDomainError(w, err)
		return
	}

	h.setRefreshCookie(w, result.RefreshToken)
	writeJSON(w, http.StatusCreated, AuthResponse{
		User:         toUserResponse(result.User),
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
	})
}

// Login handles POST /login.
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeProblem(w, http.StatusBadRequest, "Invalid request body", err.Error())
		return
	}
	if err := h.validate.Struct(req); err != nil {
		writeProblem(w, http.StatusBadRequest, "Validation failed", formatValidationErrors(err))
		return
	}

	result, err := h.service.Login(r.Context(), req.Email, req.Password, r.UserAgent(), r.RemoteAddr)
	if err != nil {
		h.handleDomainError(w, err)
		return
	}

	h.setRefreshCookie(w, result.RefreshToken)
	writeJSON(w, http.StatusOK, AuthResponse{
		User:         toUserResponse(result.User),
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
	})
}

// Refresh handles POST /refresh.
func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req RefreshRequest
	// Body may be empty when token comes from cookie — ignore decode errors for
	// empty bodies, but still attempt to read.
	_ = json.NewDecoder(r.Body).Decode(&req)

	refreshToken := req.RefreshToken
	if refreshToken == "" {
		if cookie, err := r.Cookie("refresh_token"); err == nil {
			refreshToken = cookie.Value
		}
	}

	if refreshToken == "" {
		writeProblem(w, http.StatusBadRequest, "Missing refresh token", "Provide refresh_token in body or cookie")
		return
	}

	result, err := h.service.Refresh(r.Context(), refreshToken)
	if err != nil {
		h.handleDomainError(w, err)
		return
	}

	h.setRefreshCookie(w, result.RefreshToken)
	writeJSON(w, http.StatusOK, TokenResponse{
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
	})
}

// Logout handles POST /logout.
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	// Require authenticated user (middleware sets userID in context).
	if _, ok := r.Context().Value(middleware.UserIDKey).(string); !ok {
		writeProblem(w, http.StatusUnauthorized, "Unauthorized", "Missing user ID in context")
		return
	}

	// Get refresh token from body or cookie.
	var req RefreshRequest
	_ = json.NewDecoder(r.Body).Decode(&req)

	refreshToken := req.RefreshToken
	if refreshToken == "" {
		if cookie, err := r.Cookie("refresh_token"); err == nil {
			refreshToken = cookie.Value
		}
	}

	if refreshToken == "" {
		writeProblem(w, http.StatusBadRequest, "Missing refresh token", "Provide refresh_token in body or cookie")
		return
	}

	if err := h.service.Logout(r.Context(), refreshToken); err != nil {
		h.handleDomainError(w, err)
		return
	}

	h.clearRefreshCookie(w)
	w.WriteHeader(http.StatusNoContent)
}

// LogoutAll handles POST /logout-all.
func (h *AuthHandler) LogoutAll(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(string)
	if !ok {
		writeProblem(w, http.StatusUnauthorized, "Unauthorized", "Missing user ID in context")
		return
	}

	if err := h.service.LogoutAll(r.Context(), userID); err != nil {
		h.handleDomainError(w, err)
		return
	}

	h.clearRefreshCookie(w)
	w.WriteHeader(http.StatusNoContent)
}

// ---------- Helpers ----------

func (h *AuthHandler) setRefreshCookie(w http.ResponseWriter, rawRefreshToken string) {
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    rawRefreshToken,
		Path:     "/",
		HttpOnly: true,
		Secure:   h.secure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int((720 * time.Hour).Seconds()),
	})
}

func (h *AuthHandler) clearRefreshCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   h.secure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
}

func (h *AuthHandler) handleDomainError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrEmailTaken):
		writeProblem(w, http.StatusConflict, "Email already taken", err.Error())
	case errors.Is(err, domain.ErrInvalidCredentials):
		writeProblem(w, http.StatusUnauthorized, "Invalid credentials", err.Error())
	case errors.Is(err, domain.ErrTokenExpired):
		writeProblem(w, http.StatusUnauthorized, "Token expired", err.Error())
	case errors.Is(err, domain.ErrTokenRevoked):
		writeProblem(w, http.StatusUnauthorized, "Token revoked", err.Error())
	case errors.Is(err, domain.ErrTokenNotFound):
		writeProblem(w, http.StatusUnauthorized, "Token not found", err.Error())
	default:
		log.Printf("unhandled domain error: %v", err)
		writeProblem(w, http.StatusInternalServerError, "Internal server error", "an unexpected error occurred")
	}
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

func toUserResponse(u *domain.User) UserResponse {
	return UserResponse{
		ID:          u.ID,
		Email:       u.Email,
		DisplayName: u.DisplayName,
		AvatarURL:   u.AvatarURL,
	}
}

func formatValidationErrors(err error) string {
	var ve validator.ValidationErrors
	if errors.As(err, &ve) {
		msg := ""
		for i, fe := range ve {
			if i > 0 {
				msg += "; "
			}
			msg += fe.Field() + " " + fe.Tag()
			if fe.Param() != "" {
				msg += "=" + fe.Param()
			}
		}
		return msg
	}
	return err.Error()
}
