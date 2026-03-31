# Phase 1 — Auth + Settings Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Full auth flow (register/login/refresh/logout) with JWT, rate limiting, user settings CRUD, AI models endpoint, and the core layout components (Header, NavTabs, ConfigStrip, SettingsModal, Sidebar).

**Architecture:** TDD — failing tests first, then implementation. Auth module follows Clean Architecture (domain → usecase → handler → infra). Frontend uses Zustand v5 for state, Next.js middleware for route protection.

**Tech Stack:** Go 1.26.1 + chi v5, JWT v5, pgx v5, go-redis v9, bcrypt, testify + gomock. Next.js 16 + React 19.2 + Tailwind v4 + Zustand v5.

---

## Task 1: Auth Domain Layer

**Files:**
- Create: `lexis-api/internal/modules/auth/domain/user.go`
- Create: `lexis-api/internal/modules/auth/domain/token.go`
- Create: `lexis-api/internal/modules/auth/domain/repository.go`
- Create: `lexis-api/internal/modules/auth/domain/errors.go`

Domain entities, value objects, repository interfaces. Pure Go, no external deps.

- User entity (ID uuid, Email, PasswordHash, DisplayName, AvatarURL, CreatedAt, DeletedAt)
- RefreshToken entity (ID, UserID, TokenHash, ExpiresAt, RevokedAt, UserAgent, IPAddress)
- UserSettings value object (all fields from DB)
- UserRepository interface (Create, GetByID, GetByEmail, Update, SoftDelete)
- TokenRepository interface (CreateRefreshToken, GetByHash, RevokeByHash, RevokeAllForUser)
- SettingsRepository interface (GetByUserID, Upsert)
- Domain errors (ErrUserNotFound, ErrEmailTaken, ErrInvalidCredentials, ErrTokenExpired, ErrTokenRevoked)

---

## Task 2: Auth Infra — Postgres Repositories

**Files:**
- Create: `lexis-api/internal/modules/auth/infra/postgres_user_repo.go`
- Create: `lexis-api/internal/modules/auth/infra/postgres_token_repo.go`
- Create: `lexis-api/internal/modules/auth/infra/postgres_settings_repo.go`

Implement repository interfaces using pgx v5. Parameterized queries (no string concat — SQL injection prevention).

---

## Task 3: Auth Infra — Redis Blacklist

**Files:**
- Create: `lexis-api/internal/modules/auth/infra/redis_blacklist.go`

Redis-based JWT blacklist for logout. Key = token hash, value = "1", TTL = remaining refresh token expiry. Interface: `Add(hash string, ttl time.Duration)`, `IsBlacklisted(hash string) bool`.

---

## Task 4: Auth Usecase — Register + Login (TDD)

**Files:**
- Create: `lexis-api/internal/modules/auth/usecase/auth_service.go`
- Create: `lexis-api/internal/modules/auth/usecase/auth_service_test.go`

TDD: write failing tests first, then implement.

AuthService with methods:
- `Register(ctx, email, password, displayName) → (User, AccessToken, RefreshToken, error)`
  - Validate input, check email uniqueness, bcrypt hash (cost 12), create user + settings (defaults), generate JWT pair
- `Login(ctx, email, password, userAgent, ip) → (User, AccessToken, RefreshToken, error)`
  - Find by email, bcrypt compare, generate JWT pair, store refresh token

JWT generation:
- Access token: HS256, claims {sub: userID, exp: now+15m, iat: now}
- Refresh token: random 32 bytes → hex, stored as SHA-256 hash in DB

---

## Task 5: Auth Usecase — Refresh + Logout (TDD)

**Files:**
- Modify: `lexis-api/internal/modules/auth/usecase/auth_service.go`
- Modify: `lexis-api/internal/modules/auth/usecase/auth_service_test.go`

- `Refresh(ctx, rawRefreshToken) → (AccessToken, NewRefreshToken, error)`
  - Find by hash, check not revoked/expired, revoke old, create new (rotation)
- `Logout(ctx, refreshTokenHash) → error`
  - Revoke in DB, add to Redis blacklist
- `LogoutAll(ctx, userID) → error`
  - Revoke all refresh tokens for user

---

## Task 6: Auth Handler — HTTP Endpoints

**Files:**
- Create: `lexis-api/internal/modules/auth/handler/auth_handler.go`
- Create: `lexis-api/internal/modules/auth/handler/auth_handler_test.go`

chi router group `/api/v1/auth`:
- POST /register → 201 {user, access_token, refresh_token}
- POST /login → 200 {user, access_token, refresh_token}
- POST /refresh → 200 {access_token, refresh_token}
- POST /logout → 204 (requires auth middleware)
- POST /logout-all → 204 (requires auth middleware)

Set refresh_token as HttpOnly SameSite=Strict cookie too.
Errors: RFC 7807 format.

---

## Task 7: JWT Middleware + Rate Limiting

**Files:**
- Create: `lexis-api/internal/shared/middleware/auth.go`
- Create: `lexis-api/internal/shared/middleware/auth_test.go`
- Create: `lexis-api/internal/shared/middleware/ratelimit.go`
- Create: `lexis-api/internal/shared/middleware/cors.go`

Auth middleware: extract Bearer token, validate JWT, inject userID into context.
Rate limit: sliding window via Redis — 60 req/min general, 20 req/min for /tutor/*, 5 login attempts/IP/15min.
CORS: from config.CORSAllowedOrigins.

---

## Task 8: User Settings + AI Models Endpoints

**Files:**
- Create: `lexis-api/internal/modules/auth/handler/settings_handler.go`
- Create: `lexis-api/internal/modules/auth/handler/models_handler.go`

- GET /api/v1/users/me → user profile
- PATCH /api/v1/users/me → update profile
- GET /api/v1/users/me/settings → settings JSON
- PUT /api/v1/users/me/settings → update settings
- GET /api/v1/ai/models → static list of 6 models with availability

---

## Task 9: Wire Backend — main.go

**Files:**
- Modify: `lexis-api/cmd/api/main.go`

Connect: pgxpool, redis client, create repos, create services, create handlers, mount routes on chi router. Add middleware chain: CORS → rate limit → routes (auth public + auth-protected).

---

## Task 10: Frontend — Zustand Stores

**Files:**
- Create: `lexis-web/lib/stores/session.ts`
- Create: `lexis-web/lib/stores/settings.ts`
- Create: `lexis-web/lib/api.ts`

session store (in-memory): user, accessToken, isAuthenticated, login/logout actions.
settings store (persisted): all UserSettings fields, hydrate from API, save via PUT.
api.ts: fetch wrapper with auth header injection, base URL from env.

---

## Task 11: Frontend — Auth Pages (Login + Register)

**Files:**
- Create: `lexis-web/app/(auth)/layout.tsx`
- Create: `lexis-web/app/(auth)/login/page.tsx`
- Create: `lexis-web/app/(auth)/register/page.tsx`

Terminal-style forms matching Seldon Vault aesthetic. Email + password + display_name (register). All colors via CSS vars.

---

## Task 12: Frontend — AppHeader + NavTabs + ConfigStrip

**Files:**
- Create: `lexis-web/components/layout/AppHeader.tsx`
- Create: `lexis-web/components/layout/NavTabs.tsx`
- Create: `lexis-web/components/layout/ConfigStrip.tsx`

Exact CSS from spec section 5.4. Logo with blinking cursor, 6 nav tabs, config strip with pills. Match HTML prototype precisely.

---

## Task 13: Frontend — SettingsModal

**Files:**
- Create: `lexis-web/components/layout/SettingsModal.tsx`

520px modal with overlay. Three setting groups: language (flag cards), level (A2-C1 buttons), vocabulary type. AI model selector. Apply button saves via PUT /users/me/settings. Match HTML prototype screen 04.

---

## Task 14: Frontend — AppSidebar + App Layout

**Files:**
- Create: `lexis-web/components/layout/AppSidebar.tsx`
- Create: `lexis-web/app/(app)/layout.tsx`

Sidebar: 220px, goals/feedback/vocab sections. App layout: Header + Sidebar + content area.

---

## Task 15: Frontend — Next.js Middleware + Final Integration

**Files:**
- Create: `lexis-web/middleware.ts`

Protected route middleware. Install Zustand: `npm install zustand`. Wire stores to app layout. Verify full flow: register → login → see header/sidebar/settings modal.
