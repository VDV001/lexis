CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- users
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

-- user_settings
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

-- vocabulary_words
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

-- vocabulary_daily_snapshots
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

-- sessions
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

-- rounds
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

-- goals
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

-- refresh_tokens
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
