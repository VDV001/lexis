# Phase 0 — Infrastructure Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Set up the complete project skeleton — Go backend (Modular Monolith), Next.js frontend, Docker Compose, DB migrations, CSS design system, CI — so that Phase 1 (Auth + Settings) can start immediately.

**Architecture:** Modular Monolith Go backend (`lexis-api/`) with 4 domain modules (auth, tutor, progress, vocabulary), each split into domain/usecase/handler/infra layers. Next.js 16 App Router frontend (`lexis-web/`) with Tailwind v4, Zustand, and the full Seldon Vault CSS design system. PostgreSQL 18, Redis 8, MinIO via Docker Compose.

**Tech Stack:** Go 1.26.1, Next.js 16.2 + React 19.2 + TypeScript strict, PostgreSQL 18.3, Redis 8.0, MinIO, Docker Compose v2, golangci-lint, sqlc v2, golang-migrate v4.

---

## Task 1: Initialize Git Repository

**Files:**
- Create: `.gitignore`
- Create: `README.md`

**Step 1: Initialize git repo**

```bash
cd ~/git/lexis
git init
```

**Step 2: Create .gitignore**

```gitignore
# Go
lexis-api/bin/
lexis-api/tmp/
*.exe
*.test
*.out

# Node
lexis-web/node_modules/
lexis-web/.next/
lexis-web/out/

# Environment
.env
.env.local
.env.*.local
*.env

# IDE
.idea/
.vscode/
*.swp
*.swo

# OS
.DS_Store
Thumbs.db

# Docker volumes (local)
pgdata/
redisdata/
miniodata/

# Secrets
*.pem
*.key
credentials.json
```

**Step 3: Create minimal README**

```markdown
# lexis (lang.tutor)

AI-powered language learning platform.

## Stack

- Backend: Go 1.26.1 (Modular Monolith)
- Frontend: Next.js 16 + React 19.2 + TypeScript
- Database: PostgreSQL 18.3
- Cache: Redis 8.0
- Storage: MinIO

## Quick Start

```bash
docker compose up -d
cd lexis-api && go run ./cmd/api
cd lexis-web && npm run dev
```
```

**Step 4: Initial commit**

```bash
git add .gitignore README.md CLAUDE_CODE_PROMPT.md DESIGN.md english_tutor_seldon_v2.html lang_tutor_spec_v3.docx pencil-new.pen docs/
git commit -m "chore: initial project setup with spec files"
```

---

## Task 2: Go Backend Skeleton (lexis-api)

**Files:**
- Create: `lexis-api/go.mod`
- Create: `lexis-api/cmd/api/main.go`
- Create: `lexis-api/internal/modules/auth/domain/.gitkeep`
- Create: `lexis-api/internal/modules/auth/usecase/.gitkeep`
- Create: `lexis-api/internal/modules/auth/handler/.gitkeep`
- Create: `lexis-api/internal/modules/auth/infra/.gitkeep`
- Create: (same 4 dirs for tutor, progress, vocabulary)
- Create: `lexis-api/internal/shared/middleware/.gitkeep`
- Create: `lexis-api/internal/shared/config/config.go`
- Create: `lexis-api/internal/shared/eventbus/.gitkeep`
- Create: `lexis-api/migrations/.gitkeep`

**Step 1: Create directory structure**

```bash
cd ~/git/lexis
mkdir -p lexis-api/cmd/api
mkdir -p lexis-api/internal/modules/{auth,tutor,progress,vocabulary}/{domain,usecase,handler,infra}
mkdir -p lexis-api/internal/shared/{middleware,config,eventbus}
mkdir -p lexis-api/migrations
```

**Step 2: Initialize Go module**

```bash
cd ~/git/lexis/lexis-api
go mod init github.com/lexis-app/lexis-api
```

**Step 3: Create cmd/api/main.go**

```go
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/lexis-app/lexis-api/internal/shared/config"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	})

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.AppPort),
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("lexis-api listening on :%d", cfg.AppPort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("shutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("forced shutdown: %v", err)
	}
	log.Println("server stopped")
}
```

**Step 4: Create internal/shared/config/config.go**

```go
package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	AppEnv  string
	AppPort int

	DatabaseURL      string
	DatabaseMaxConns int

	RedisURL      string
	RedisPassword string

	MinioEndpoint  string
	MinioAccessKey string
	MinioSecretKey string
	MinioUseSSL    bool

	AnthropicAPIKey string
	QwenAPIKey      string
	OpenAIAPIKey    string
	GeminiAPIKey    string

	JWTSecret     string
	JWTAccessTTL  time.Duration
	JWTRefreshTTL time.Duration

	CORSAllowedOrigins string
	LogLevel            string
}

func Load() (*Config, error) {
	port, _ := strconv.Atoi(getEnv("APP_PORT", "8080"))
	maxConns, _ := strconv.Atoi(getEnv("DATABASE_MAX_CONNS", "25"))

	accessTTL, err := time.ParseDuration(getEnv("JWT_ACCESS_TTL", "15m"))
	if err != nil {
		return nil, fmt.Errorf("invalid JWT_ACCESS_TTL: %w", err)
	}

	refreshTTL, err := time.ParseDuration(getEnv("JWT_REFRESH_TTL", "720h"))
	if err != nil {
		return nil, fmt.Errorf("invalid JWT_REFRESH_TTL: %w", err)
	}

	return &Config{
		AppEnv:              getEnv("APP_ENV", "development"),
		AppPort:             port,
		DatabaseURL:         getEnv("DATABASE_URL", "postgres://langtutor:langtutor@localhost:5432/langtutor?sslmode=disable"),
		DatabaseMaxConns:    maxConns,
		RedisURL:            getEnv("REDIS_URL", "redis://localhost:6379"),
		RedisPassword:       getEnv("REDIS_PASSWORD", ""),
		MinioEndpoint:       getEnv("MINIO_ENDPOINT", "localhost:9000"),
		MinioAccessKey:      getEnv("MINIO_ACCESS_KEY", "minioadmin"),
		MinioSecretKey:      getEnv("MINIO_SECRET_KEY", "minioadmin"),
		MinioUseSSL:         getEnv("MINIO_USE_SSL", "false") == "true",
		AnthropicAPIKey:     getEnv("ANTHROPIC_API_KEY", ""),
		QwenAPIKey:          getEnv("QWEN_API_KEY", ""),
		OpenAIAPIKey:        getEnv("OPENAI_API_KEY", ""),
		GeminiAPIKey:        getEnv("GEMINI_API_KEY", ""),
		JWTSecret:           getEnv("APP_SECRET", "dev-secret-change-in-production-32ch"),
		JWTAccessTTL:        accessTTL,
		JWTRefreshTTL:       refreshTTL,
		CORSAllowedOrigins:  getEnv("CORS_ALLOWED_ORIGINS", "http://localhost:3000"),
		LogLevel:            getEnv("LOG_LEVEL", "info"),
	}, nil
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}
```

**Step 5: Add .gitkeep files to empty directories**

```bash
for dir in lexis-api/internal/modules/{auth,tutor,progress,vocabulary}/{domain,usecase,handler,infra} \
           lexis-api/internal/shared/{middleware,eventbus} \
           lexis-api/migrations; do
  touch "$dir/.gitkeep"
done
```

**Step 6: Install Go dependencies**

```bash
cd ~/git/lexis/lexis-api
go mod tidy
```

**Step 7: Verify it compiles**

```bash
cd ~/git/lexis/lexis-api
go build ./cmd/api
```

Expected: no errors, binary created.

**Step 8: Commit**

```bash
cd ~/git/lexis
git add lexis-api/
git commit -m "feat: Go backend skeleton with modular monolith structure"
```

---

## Task 3: Install Go Dependencies

**Files:**
- Modify: `lexis-api/go.mod`
- Modify: `lexis-api/go.sum`

**Step 1: Install all required dependencies from spec section 2.2**

```bash
cd ~/git/lexis/lexis-api
go get github.com/go-chi/chi/v5@latest
go get github.com/golang-jwt/jwt/v5@latest
go get github.com/jackc/pgx/v5@latest
go get github.com/redis/go-redis/v9@latest
go get github.com/minio/minio-go/v7@latest
go get github.com/go-playground/validator/v10@latest
go get github.com/rs/zerolog@latest
go get github.com/spf13/viper@latest
go get golang.org/x/crypto@latest
```

**Step 2: Install dev/test dependencies**

```bash
cd ~/git/lexis/lexis-api
go get github.com/stretchr/testify@latest
go get go.uber.org/mock/gomock@latest
go get go.uber.org/mock/mockgen@latest
```

**Step 3: Install golang-migrate CLI**

```bash
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
```

**Step 4: Install sqlc**

```bash
go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
```

**Step 5: Tidy modules**

```bash
cd ~/git/lexis/lexis-api
go mod tidy
```

**Step 6: Verify**

```bash
cd ~/git/lexis/lexis-api
go build ./cmd/api
```

Expected: compiles cleanly.

**Step 7: Commit**

```bash
cd ~/git/lexis
git add lexis-api/go.mod lexis-api/go.sum
git commit -m "chore: add Go dependencies (chi, pgx, redis, jwt, zerolog, testify)"
```

---

## Task 4: Docker Compose

**Files:**
- Create: `docker-compose.yml`
- Create: `lexis-api/.env.example`

**Step 1: Create docker-compose.yml**

```yaml
services:
  postgres:
    image: postgres:18-alpine
    environment:
      POSTGRES_DB: langtutor
      POSTGRES_USER: langtutor
      POSTGRES_PASSWORD: langtutor
    ports:
      - "5432:5432"
    volumes:
      - pg_data:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U langtutor"]
      interval: 10s
      retries: 5

  redis:
    image: redis:8.0-alpine
    ports:
      - "6379:6379"
    command: redis-server --appendonly yes
    volumes:
      - redis_data:/data
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 10s
      retries: 5

  minio:
    image: minio/minio:latest
    ports:
      - "9000:9000"
      - "9001:9001"
    environment:
      MINIO_ROOT_USER: minioadmin
      MINIO_ROOT_PASSWORD: minioadmin
    command: server /data --console-address ':9001'
    volumes:
      - minio_data:/data

volumes:
  pg_data:
  redis_data:
  minio_data:
```

**Step 2: Create lexis-api/.env.example**

```bash
APP_ENV=development
APP_PORT=8080
APP_SECRET=dev-secret-change-in-production-32ch

DATABASE_URL=postgres://langtutor:langtutor@localhost:5432/langtutor?sslmode=disable
DATABASE_MAX_CONNS=25

REDIS_URL=redis://localhost:6379
REDIS_PASSWORD=

MINIO_ENDPOINT=localhost:9000
MINIO_ACCESS_KEY=minioadmin
MINIO_SECRET_KEY=minioadmin
MINIO_USE_SSL=false

ANTHROPIC_API_KEY=
QWEN_API_KEY=
OPENAI_API_KEY=
GEMINI_API_KEY=

JWT_ACCESS_TTL=15m
JWT_REFRESH_TTL=720h

CORS_ALLOWED_ORIGINS=http://localhost:3000
LOG_LEVEL=info
VOCAB_SNAPSHOT_CRON=0 0 * * *
```

**Step 3: Start infrastructure**

```bash
cd ~/git/lexis
docker compose up -d
```

Expected: postgres, redis, minio all healthy.

**Step 4: Verify connectivity**

```bash
psql "postgres://langtutor:langtutor@localhost:5432/langtutor" -c "SELECT 1"
docker exec $(docker compose ps -q redis) redis-cli ping
curl -s http://localhost:9001 | head -1
```

Expected: `1`, `PONG`, HTML response.

**Step 5: Commit**

```bash
cd ~/git/lexis
git add docker-compose.yml lexis-api/.env.example
git commit -m "infra: Docker Compose with PostgreSQL 18, Redis 8, MinIO"
```

---

## Task 5: Database Migrations

**Files:**
- Create: `lexis-api/migrations/000001_init_schema.up.sql`
- Create: `lexis-api/migrations/000001_init_schema.down.sql`
- Create: `lexis-api/sqlc.yaml`

**Step 1: Create up migration (all tables from spec sections 6.1-6.5)**

```sql
-- 000001_init_schema.up.sql

CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- 6.1 users
CREATE TABLE users (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    email         varchar(255) UNIQUE NOT NULL,
    password_hash varchar(255) NOT NULL,
    display_name  varchar(100) NOT NULL,
    avatar_url    text,
    created_at    timestamptz NOT NULL DEFAULT now(),
    deleted_at    timestamptz
);

CREATE INDEX idx_users_email ON users (email) WHERE deleted_at IS NULL;

-- 6.2 user_settings
CREATE TABLE user_settings (
    user_id           uuid PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    target_language   varchar(10)  NOT NULL DEFAULT 'en',
    proficiency_level varchar(5)   NOT NULL DEFAULT 'b1',
    vocabulary_type   varchar(20)  NOT NULL DEFAULT 'tech',
    ai_model          varchar(60)  NOT NULL DEFAULT 'claude-sonnet-4-20250514',
    vocab_goal        int          NOT NULL DEFAULT 3000,
    ui_language       varchar(10)  NOT NULL DEFAULT 'ru',
    updated_at        timestamptz  NOT NULL DEFAULT now()
);

-- 6.3 vocabulary_words
CREATE TABLE vocabulary_words (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    word        varchar(100) NOT NULL,
    language    varchar(10)  NOT NULL,
    status      varchar(20)  NOT NULL DEFAULT 'unknown',
    ease_factor float        NOT NULL DEFAULT 2.5,
    next_review timestamptz  NOT NULL DEFAULT now(),
    context     text,
    last_seen   timestamptz  NOT NULL DEFAULT now(),

    CONSTRAINT chk_vocab_status CHECK (status IN ('unknown', 'uncertain', 'confident'))
);

CREATE INDEX idx_vocab_user_lang ON vocabulary_words (user_id, language);
CREATE INDEX idx_vocab_next_review ON vocabulary_words (user_id, next_review) WHERE status != 'confident';

-- 6.4 vocabulary_daily_snapshots
CREATE TABLE vocabulary_daily_snapshots (
    user_id       uuid       NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    language      varchar(10) NOT NULL,
    snapshot_date date       NOT NULL,
    total_words   int        NOT NULL DEFAULT 0,
    confident     int        NOT NULL DEFAULT 0,
    uncertain     int        NOT NULL DEFAULT 0,
    unknown       int        NOT NULL DEFAULT 0,

    PRIMARY KEY (user_id, language, snapshot_date)
);

-- 6.5 sessions
CREATE TABLE sessions (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id       uuid        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    mode          varchar(20) NOT NULL,
    language      varchar(10) NOT NULL,
    level         varchar(5)  NOT NULL,
    ai_model      varchar(60) NOT NULL,
    started_at    timestamptz NOT NULL DEFAULT now(),
    ended_at      timestamptz,
    round_count   int         NOT NULL DEFAULT 0,
    correct_count int         NOT NULL DEFAULT 0,

    CONSTRAINT chk_session_mode CHECK (mode IN ('chat', 'quiz', 'translate', 'gap', 'scramble'))
);

CREATE INDEX idx_sessions_user ON sessions (user_id, started_at DESC);

-- 6.5 rounds
CREATE TABLE rounds (
    id             uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id     uuid        NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
    user_id        uuid        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    mode           varchar(20) NOT NULL,
    is_correct     boolean     NOT NULL,
    error_type     varchar(30),
    question       text        NOT NULL,
    user_answer    text        NOT NULL,
    correct_answer text,
    explanation    text,
    created_at     timestamptz NOT NULL DEFAULT now(),

    CONSTRAINT chk_error_type CHECK (
        error_type IS NULL OR
        error_type IN ('articles', 'tenses', 'prepositions', 'phrasal', 'vocabulary', 'word_order')
    )
);

CREATE INDEX idx_rounds_session ON rounds (session_id);
CREATE INDEX idx_rounds_user_errors ON rounds (user_id, error_type) WHERE NOT is_correct;

-- 6.5 goals
CREATE TABLE goals (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    uuid        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name       varchar(100) NOT NULL,
    language   varchar(10) NOT NULL DEFAULT 'en',
    progress   int         NOT NULL DEFAULT 0,
    color      varchar(10) NOT NULL DEFAULT 'green',
    is_system  boolean     NOT NULL DEFAULT false,
    updated_at timestamptz NOT NULL DEFAULT now(),

    CONSTRAINT chk_progress CHECK (progress >= 0 AND progress <= 100),
    CONSTRAINT chk_goal_color CHECK (color IN ('green', 'amber', 'red'))
);

CREATE INDEX idx_goals_user ON goals (user_id);

-- 6.5 refresh_tokens
CREATE TABLE refresh_tokens (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    uuid        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash varchar(255) NOT NULL UNIQUE,
    expires_at timestamptz NOT NULL,
    revoked_at timestamptz,
    user_agent text,
    ip_address varchar(45),
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_refresh_tokens_user ON refresh_tokens (user_id) WHERE revoked_at IS NULL;
CREATE INDEX idx_refresh_tokens_hash ON refresh_tokens (token_hash) WHERE revoked_at IS NULL;
```

**Step 2: Create down migration**

```sql
-- 000001_init_schema.down.sql

DROP TABLE IF EXISTS refresh_tokens;
DROP TABLE IF EXISTS goals;
DROP TABLE IF EXISTS rounds;
DROP TABLE IF EXISTS sessions;
DROP TABLE IF EXISTS vocabulary_daily_snapshots;
DROP TABLE IF EXISTS vocabulary_words;
DROP TABLE IF EXISTS user_settings;
DROP TABLE IF EXISTS users;
```

**Step 3: Run migration**

```bash
cd ~/git/lexis/lexis-api
migrate -path migrations -database "postgres://langtutor:langtutor@localhost:5432/langtutor?sslmode=disable" up
```

Expected: `1/u init_schema` applied.

**Step 4: Verify tables exist**

```bash
psql "postgres://langtutor:langtutor@localhost:5432/langtutor" -c "\dt"
```

Expected: 7 tables listed.

**Step 5: Create sqlc.yaml**

```yaml
version: "2"
sql:
  - engine: "postgresql"
    queries: "queries/"
    schema: "migrations/"
    gen:
      go:
        package: "db"
        out: "internal/db"
        sql_package: "pgx/v5"
        emit_json_tags: true
        emit_empty_slices: true
        overrides:
          - db_type: "uuid"
            go_type: "github.com/jackc/pgx/v5/pgtype.UUID"
          - db_type: "timestamptz"
            go_type: "github.com/jackc/pgx/v5/pgtype.Timestamptz"
```

**Step 6: Create queries directory placeholder**

```bash
mkdir -p ~/git/lexis/lexis-api/queries
touch ~/git/lexis/lexis-api/queries/.gitkeep
```

**Step 7: Commit**

```bash
cd ~/git/lexis
git add lexis-api/migrations/ lexis-api/sqlc.yaml lexis-api/queries/
git commit -m "feat: database schema — all 7 tables from spec (users, settings, vocab, sessions, rounds, goals, refresh_tokens)"
```

---

## Task 6: Next.js Frontend Scaffold (lexis-web)

**Files:**
- Create: `lexis-web/` (via create-next-app)

**Step 1: Scaffold Next.js 16 project**

```bash
cd ~/git/lexis
npx create-next-app@latest lexis-web --ts --tailwind --app --import-alias "@/*" --yes
```

**Step 2: Verify it runs**

```bash
cd ~/git/lexis/lexis-web
npm run build
```

Expected: build succeeds.

**Step 3: Create .env.local.example**

```bash
NEXT_PUBLIC_API_URL=http://localhost:8080/api/v1
NEXT_PUBLIC_APP_NAME=lang.tutor
```

**Step 4: Commit**

```bash
cd ~/git/lexis
git add lexis-web/
git commit -m "feat: Next.js 16 frontend scaffold with TypeScript and Tailwind"
```

---

## Task 7: Frontend CSS Design System (globals.css)

**Files:**
- Modify: `lexis-web/app/globals.css`

**Step 1: Replace globals.css with Seldon Vault design system**

This file contains ALL CSS custom properties from spec section 5.1, ALL 5 animations from section 5.3, and base reset styles. Extract exact values from `english_tutor_seldon_v2.html`.

```css
@import url('https://fonts.googleapis.com/css2?family=JetBrains+Mono:wght@300;400;500;600;700&display=swap');

@tailwind base;
@tailwind components;
@tailwind utilities;

:root {
  /* Backgrounds */
  --bg: #0d1117;
  --bg2: #161b22;
  --bg3: #1c2128;
  --bg4: #21262d;

  /* Borders */
  --border: #30363d;
  --border2: #3d444d;

  /* Text */
  --text: #e6edf3;
  --text2: #7d8590;
  --text3: #484f58;

  /* Accents */
  --green: #3fb950;
  --cyan: #58a6ff;
  --amber: #e3b341;
  --red: #f85149;
  --purple: #bc8cff;
  --teal: #39c5cf;

  /* Font */
  --font-mono: 'JetBrains Mono', 'Courier New', monospace;
}

/* ── ANIMATIONS (all 5 required) ── */
@keyframes blink {
  0%, 100% { opacity: 1; }
  50% { opacity: 0; }
}

@keyframes pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.4; }
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

@keyframes fadeUp {
  from { opacity: 0; transform: translateY(4px); }
  to { opacity: 1; transform: translateY(0); }
}

@keyframes tdot {
  0%, 60%, 100% { transform: translateY(0); }
  30% { transform: translateY(-3px); }
}

/* ── BASE RESET ── */
* {
  box-sizing: border-box;
  margin: 0;
  padding: 0;
}

html, body {
  height: 100%;
  overflow: hidden;
}

body {
  font-family: var(--font-mono);
  background: var(--bg);
  color: var(--text);
  font-size: 13px;
}

/* Scrollbar styling */
::-webkit-scrollbar { width: 3px; }
::-webkit-scrollbar-thumb { background: var(--border); }
::-webkit-scrollbar-track { background: transparent; }
```

**Step 2: Update tailwind.config.ts to extend with CSS variables**

Read the existing `tailwind.config.ts` generated by create-next-app and extend it:

```ts
import type { Config } from "tailwindcss";

const config: Config = {
  content: [
    "./pages/**/*.{js,ts,jsx,tsx,mdx}",
    "./components/**/*.{js,ts,jsx,tsx,mdx}",
    "./app/**/*.{js,ts,jsx,tsx,mdx}",
  ],
  theme: {
    extend: {
      colors: {
        bg: "var(--bg)",
        bg2: "var(--bg2)",
        bg3: "var(--bg3)",
        bg4: "var(--bg4)",
        border: "var(--border)",
        border2: "var(--border2)",
        text: "var(--text)",
        text2: "var(--text2)",
        text3: "var(--text3)",
        green: "var(--green)",
        cyan: "var(--cyan)",
        amber: "var(--amber)",
        red: "var(--red)",
        purple: "var(--purple)",
        teal: "var(--teal)",
      },
      fontFamily: {
        mono: ["var(--font-mono)"],
      },
      animation: {
        blink: "blink 1.2s step-end infinite",
        pulse: "pulse 2s ease-in-out infinite",
        spin: "spin 0.7s linear infinite",
        fadeUp: "fadeUp 0.2s ease",
        tdot: "tdot 1.1s infinite",
      },
    },
  },
  plugins: [],
};

export default config;
```

**Step 3: Update app/layout.tsx** — set JetBrains Mono as the base font

Read the existing layout.tsx and update it. Keep the structure but change the font setup and metadata.

**Step 4: Replace app/page.tsx** with a simple landing that shows the design system is working

```tsx
export default function Home() {
  return (
    <div className="flex h-screen items-center justify-center">
      <div className="text-center">
        <h1 className="text-[17px] font-bold text-[var(--green)] tracking-[-0.5px]">
          lang.tutor
          <span className="inline-block w-[9px] h-[16px] bg-[var(--green)] ml-[2px] align-middle animate-blink" />
        </h1>
        <p className="text-[10.5px] text-[var(--text2)] mt-1">
          {'>'} AI-наставник для изучения языков
        </p>
      </div>
    </div>
  );
}
```

**Step 5: Verify build**

```bash
cd ~/git/lexis/lexis-web
npm run build
```

Expected: build succeeds, no TypeScript errors.

**Step 6: Commit**

```bash
cd ~/git/lexis
git add lexis-web/app/globals.css lexis-web/tailwind.config.ts lexis-web/app/layout.tsx lexis-web/app/page.tsx
git commit -m "feat: Seldon Vault CSS design system — variables, animations, JetBrains Mono"
```

---

## Task 8: Frontend Directory Structure

**Files:**
- Create: `lexis-web/components/layout/` (empty, for Phase 1)
- Create: `lexis-web/components/tutor/` (empty, for Phase 2)
- Create: `lexis-web/components/dashboard/` (empty, for Phase 3)
- Create: `lexis-web/lib/stores/` (empty, for Phase 1)
- Create: `lexis-web/types/index.ts`
- Create: `lexis-web/app/api/v1/` (BFF routes dir)

**Step 1: Create directory structure matching spec section 3.4**

```bash
cd ~/git/lexis/lexis-web
mkdir -p components/{layout,tutor,dashboard}
mkdir -p lib/stores
mkdir -p app/api/v1
mkdir -p app/\(auth\)/{login,register}
mkdir -p app/\(app\)/{chat,quiz,translate,gap,scramble,dashboard}
```

**Step 2: Create types/index.ts with core types from spec**

```ts
// Core domain types derived from spec sections 6 + 7

export type ProficiencyLevel = "a2" | "b1" | "b2" | "c1";
export type VocabularyType = "tech" | "literary" | "business";
export type LearningMode = "chat" | "quiz" | "translate" | "gap" | "scramble";
export type VocabStatus = "unknown" | "uncertain" | "confident";
export type ErrorType =
  | "articles"
  | "tenses"
  | "prepositions"
  | "phrasal"
  | "vocabulary"
  | "word_order";
export type FeedbackType = "good" | "note" | "error";
export type GoalColor = "green" | "amber" | "red";

export interface AIModel {
  id: string;
  display_name: string;
  provider: string;
  icon: string;
  description: string;
  available: boolean;
}

export interface UserSettings {
  target_language: string;
  proficiency_level: ProficiencyLevel;
  vocabulary_type: VocabularyType;
  ai_model: string;
  vocab_goal: number;
  ui_language: string;
}

export interface User {
  id: string;
  email: string;
  display_name: string;
  avatar_url: string | null;
}

export interface Goal {
  id: string;
  name: string;
  language: string;
  progress: number;
  color: GoalColor;
  is_system: boolean;
}

export interface VocabWord {
  id: string;
  word: string;
  language: string;
  status: VocabStatus;
  context: string | null;
  last_seen: string;
}

export interface VocabSnapshot {
  date: string;
  total: number;
  confident: number;
  uncertain: number;
  unknown: number;
}

export interface VocabCurveData {
  goal: number;
  current: {
    total: number;
    confident: number;
    uncertain: number;
    unknown: number;
  };
  daily_snapshots: VocabSnapshot[];
}

export interface ChatCorrection {
  original: string;
  fixed: string;
  explanation: string;
}

export interface ChatFeedback {
  type: FeedbackType;
  text: string;
}

export interface ChatResponse {
  reply: string;
  correction: ChatCorrection | null;
  feedback: ChatFeedback;
  error_type: ErrorType | null;
  new_words: string[];
}

export interface QuizQuestion {
  type: string;
  question: string;
  options: string[];
  correct: number;
  explanation: string;
  error_type: ErrorType;
  words: string[];
  confidence: number;
}

export interface ProgressSummary {
  total_rounds: number;
  correct_rounds: number;
  accuracy: number;
  streak: number;
  total_words: number;
}

// SSE event types (spec 7.3)
export type SSEEventType = "delta" | "correction" | "feedback" | "words" | "done";

export interface SSEEvent {
  type: SSEEventType;
  content?: string;
  correction?: ChatCorrection;
  feedback?: ChatFeedback;
  words?: string[];
}
```

**Step 3: Add .gitkeep to empty directories**

```bash
for dir in lexis-web/components/{layout,tutor,dashboard} \
           lexis-web/lib/stores \
           lexis-web/app/api/v1 \
           lexis-web/app/\(auth\)/{login,register} \
           lexis-web/app/\(app\)/{chat,quiz,translate,gap,scramble,dashboard}; do
  touch "$dir/.gitkeep"
done
```

**Step 4: Verify TypeScript**

```bash
cd ~/git/lexis/lexis-web
npx tsc --noEmit
```

Expected: no errors.

**Step 5: Commit**

```bash
cd ~/git/lexis
git add lexis-web/components/ lexis-web/lib/ lexis-web/types/ lexis-web/app/
git commit -m "feat: frontend directory structure and TypeScript types from spec"
```

---

## Task 9: Go Linting & First Test

**Files:**
- Create: `lexis-api/.golangci.yml`
- Create: `lexis-api/internal/shared/config/config_test.go`

**Step 1: Create golangci-lint config**

```yaml
run:
  timeout: 5m

linters:
  enable:
    - errcheck
    - govet
    - staticcheck
    - unused
    - gosimple
    - ineffassign
    - typecheck
    - bodyclose
    - noctx

linters-settings:
  errcheck:
    check-type-assertions: true

issues:
  exclude-dirs:
    - internal/db
```

**Step 2: Run linter**

```bash
cd ~/git/lexis/lexis-api
golangci-lint run ./...
```

Expected: clean (no issues).

**Step 3: Write first test — config loads defaults**

```go
package config_test

import (
	"testing"

	"github.com/lexis-app/lexis-api/internal/shared/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad_Defaults(t *testing.T) {
	cfg, err := config.Load()
	require.NoError(t, err)

	assert.Equal(t, "development", cfg.AppEnv)
	assert.Equal(t, 8080, cfg.AppPort)
	assert.Equal(t, "info", cfg.LogLevel)
	assert.Equal(t, 25, cfg.DatabaseMaxConns)
}
```

**Step 4: Run test — should pass (not TDD red-green since it's config, not domain logic)**

```bash
cd ~/git/lexis/lexis-api
go test ./internal/shared/config/ -v
```

Expected: PASS.

**Step 5: Commit**

```bash
cd ~/git/lexis
git add lexis-api/.golangci.yml lexis-api/internal/shared/config/config_test.go
git commit -m "chore: golangci-lint config and first config test"
```

---

## Task 10: Environment Files & Final Verification

**Files:**
- Create: `lexis-api/.env` (git-ignored, for local dev)
- Create: `lexis-web/.env.local` (git-ignored, for local dev)

**Step 1: Create local .env files (these are gitignored)**

`lexis-api/.env`:
```
APP_ENV=development
APP_PORT=8080
APP_SECRET=dev-secret-change-in-production-32ch
DATABASE_URL=postgres://langtutor:langtutor@localhost:5432/langtutor?sslmode=disable
REDIS_URL=redis://localhost:6379
MINIO_ENDPOINT=localhost:9000
MINIO_ACCESS_KEY=minioadmin
MINIO_SECRET_KEY=minioadmin
ANTHROPIC_API_KEY=
QWEN_API_KEY=
OPENAI_API_KEY=
GEMINI_API_KEY=
```

`lexis-web/.env.local`:
```
NEXT_PUBLIC_API_URL=http://localhost:8080/api/v1
NEXT_PUBLIC_APP_NAME=lang.tutor
```

**Step 2: Final verification checklist**

```bash
# Go compiles
cd ~/git/lexis/lexis-api && go build ./cmd/api

# Go tests pass
cd ~/git/lexis/lexis-api && go test ./...

# Go lint clean
cd ~/git/lexis/lexis-api && golangci-lint run ./...

# Frontend builds
cd ~/git/lexis/lexis-web && npm run build

# Frontend TypeScript clean
cd ~/git/lexis/lexis-web && npx tsc --noEmit

# Docker services running
cd ~/git/lexis && docker compose ps

# DB migration applied
psql "postgres://langtutor:langtutor@localhost:5432/langtutor" -c "\dt"

# API starts and responds
cd ~/git/lexis/lexis-api && timeout 3 go run ./cmd/api &
sleep 2 && curl -s http://localhost:8080/health
kill %1 2>/dev/null
```

Expected: all green.

**Step 3: Commit any remaining files**

```bash
cd ~/git/lexis
git status
# Add anything missed, but NOT .env or .env.local
git add -A
git diff --cached --name-only  # Review what's staged
git commit -m "chore: Phase 0 complete — infrastructure ready"
```

---

## Summary — Phase 0 Deliverables

After completing all 10 tasks:

| Deliverable | Location |
|---|---|
| Git repo initialized | `~/git/lexis/.git` |
| Go backend skeleton | `lexis-api/` with modular monolith structure |
| Go dependencies installed | chi, pgx, redis, jwt, zerolog, testify, gomock |
| Docker Compose running | PG 18 + Redis 8 + MinIO |
| DB schema applied | 7 tables matching spec sections 6.1-6.5 |
| Next.js 16 scaffold | `lexis-web/` with TypeScript + Tailwind |
| CSS design system | globals.css with all variables + 5 animations |
| TypeScript types | `types/index.ts` covering all domain types |
| Frontend dir structure | components/, lib/stores/, app routes |
| Linting configured | golangci-lint + tsc --noEmit |
| First test passing | config_test.go |
| Env files | .env.example committed, .env gitignored |

**Next: Phase 1 — Auth + Settings**
