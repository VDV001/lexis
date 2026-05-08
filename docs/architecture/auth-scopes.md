# JWT Scopes — Authorisation Reference

> **Audience:** developers adding new endpoints, frontend authors building
> features, operators investigating 403 responses.
>
> **Refs:** `reflective-agent-defaults` v1.3 Rule 4 (scoped tokens),
> `lexis-api/internal/modules/auth/domain/scope.go` (canonical constants),
> `lexis-api/internal/shared/middleware/scope.go` (`RequireScope` middleware).

---

## Catalogue

The full set of valid scopes lives in `auth/domain/scope.go` as `Scope`
constants. Anything outside this list is silently dropped from the
request context — so a token claiming an unknown scope cannot create
new permissions.

| Scope | Granted at login | What it lets the bearer do |
|---|---|---|
| `chat:read` | yes | (reserved — currently unused on routes) read chat history |
| `chat:write` | yes | invoke the LLM: chat, exercise generation, answer checking |
| `vocab:read` | yes | list / fetch words and review queue |
| `vocab:write` | yes | add / update / delete words |
| `settings:read` | yes | read profile + settings + AI model catalogue |
| `settings:write` | yes | update profile + settings |
| `progress:read` | yes | read summary, sessions, goals, errors, vocab curve |
| `progress:write` | yes | start a session, record a round |
| `account:delete` | yes | delete the user's own account (DESTRUCTIVE) |
| `admin:full` | **no** | reserved for first-party admin sessions, never default |

`DefaultUserScopes()` returns every scope except `admin:full`. The
`/auth/login` and `/auth/register` flows hand out exactly that set.

## Endpoint → required scope

| Method + path | Required scope | Module |
|---|---|---|
| `POST /api/v1/auth/login` | _none — public_ | auth |
| `POST /api/v1/auth/register` | _none — public_ | auth |
| `POST /api/v1/auth/refresh` | _none — public_ | auth |
| `POST /api/v1/auth/logout` | _authenticated only_ | auth |
| `POST /api/v1/auth/logout-all` | _authenticated only_ | auth |
| `GET  /api/v1/users/me` | `settings:read` | auth/settings |
| `PATCH /api/v1/users/me` | `settings:write` | auth/settings |
| `GET  /api/v1/users/me/settings` | `settings:read` | auth/settings |
| `PUT  /api/v1/users/me/settings` | `settings:write` | auth/settings |
| `GET  /api/v1/ai/models` | `settings:read` | auth/handler |
| `GET  /api/v1/vocabulary/` | `vocab:read` | vocabulary |
| `POST /api/v1/vocabulary/` | `vocab:write` | vocabulary |
| `DELETE /api/v1/vocabulary/{id}` | `vocab:write` | vocabulary |
| `PATCH  /api/v1/vocabulary/{id}` | `vocab:write` | vocabulary |
| `GET  /api/v1/vocabulary/due` | `vocab:read` | vocabulary |
| `GET  /api/v1/progress/summary` | `progress:read` | progress |
| `GET  /api/v1/progress/vocabulary` | `progress:read` | progress |
| `GET  /api/v1/progress/vocabulary/curve` | `progress:read` | progress |
| `GET  /api/v1/progress/goals` | `progress:read` | progress |
| `GET  /api/v1/progress/errors` | `progress:read` | progress |
| `GET  /api/v1/progress/sessions` | `progress:read` | progress |
| `GET  /api/v1/progress/sessions/{id}` | `progress:read` | progress |
| `POST /api/v1/progress/sessions` | `progress:write` | progress |
| `POST /api/v1/progress/rounds` | `progress:write` | progress |
| `POST /api/v1/tutor/*` (chat + exercise) | `chat:write` | tutor |

`logout` and `logout-all` deliberately have NO scope: revoking your
**own** sessions is authentication (the bearer is who they say they are
— established by `middleware.Auth`), not authorisation. There is no
ambient capability to add, so `RequireScope` would only add friction.

## JWT shape

A token issued by `/auth/login` decodes to:

```json
{
  "sub": "00000000-0000-0000-0000-000000000000",
  "iat": 1778414400,
  "exp": 1778415300,
  "aud": ["lexis-api"],
  "scope": [
    "chat:read", "chat:write",
    "vocab:read", "vocab:write",
    "settings:read", "settings:write",
    "progress:read", "progress:write",
    "account:delete"
  ]
}
```

- `sub` — user id (UUID).
- `aud` — always `["lexis-api"]` so a token cannot be silently replayed
  against a future admin or MCP service that issues its own audience.
- `scope` — array of strings exactly matching the `Scope` constants.
  Unknown strings are dropped at extract time.

## Adding a new endpoint

1. Decide the smallest scope that fits. Read endpoints get `*:read`,
   write endpoints `*:write`. New module → consider whether an existing
   scope set covers it before introducing a new prefix.
2. Wrap the route declaration:

   ```go
   r.With(middleware.RequireScope(authdomain.ScopeVocabRead)).
       Get("/", h.ListWords)
   ```

3. Add a row to the table above in this doc.
4. In the handler test, use `withUserID()` (which now grants
   `DefaultUserScopes`) for behaviour tests; chain `withScopes(req)` to
   overwrite with an explicit set when testing rejection.
5. Add a row to the module's `*_RequireScope_table` test for explicit
   negative coverage.

## Migration window for legacy tokens

Tokens issued before scopes landed (#7) carry no `scope` claim. By
default the `Auth` middleware grants those tokens `DefaultUserScopes()`
and emits one log line per request:

```
auth: legacy token (sub=…) granted default scopes — refresh to upgrade
```

This is a temporary grace so active sessions keep working through the
rollout.

### Hard cutoff (issue #9, v0.13.0)

Set the env var `LEGACY_TOKEN_CUTOFF` to an RFC3339 timestamp
(e.g. `2026-06-08T00:00:00Z`) to switch the middleware into rejection
mode. Once active:

- Any token without a `scope` claim → 401 (regardless of `iat`).
- Scoped tokens are unaffected (`RequireScope` continues to gate
  per-endpoint).
- The audit log distinguishes the expected migration tail from an
  issuer regression:
  - `iat < cutoff` → `auth: legacy token (sub=…, iat=…) rejected — cutoff … active`
  - `iat ≥ cutoff` → `auth: post-cutoff legacy token (sub=…, iat=…) rejected — issuer regression, no scope claim should be possible`
  - `iat` missing → `auth: legacy token (sub=…, iat=missing) rejected — cutoff … active`

Operators should watch the pre-cutoff log line volume and only flip
`LEGACY_TOKEN_CUTOFF` once it falls to zero — the variable is a
hard, breaking gate. Leave it unset to keep the migration grant.

## Granting `admin:full`

Not yet implemented. When the admin UI lands, a separate flow
(invitation + explicit grant) issues tokens that include `admin:full`
in addition to (or instead of) `DefaultUserScopes`. `admin:full`
**does not bypass** any other scope — granular checks always win, by
design. An admin operating on user-owned data still needs the same
fine-grained scope a regular user would. The reasoning: a leaked admin
token is an even bigger blast radius if it bypasses every gate.
