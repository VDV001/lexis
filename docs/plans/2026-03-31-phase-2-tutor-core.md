# Phase 2 — Tutor Core + Multi-model AI

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Implement all 5 learning modes (Chat, Quiz, Translate, Gap, Scramble) with multi-model AI backend, SSE streaming for chat, and pixel-perfect UI from Pencil mockups.

**Architecture:** AIProvider interface with 4 implementations (Claude, Qwen, OpenAI, Gemini). System prompts built dynamically from user settings. SSE streaming for chat mode. Each mode has: usecase (TDD) → handler → frontend page.

**Tech Stack:** Go 1.26.1 (chi, SSE via net/http flusher), Next.js 16 + React 19.2 (EventSource for SSE, Zustand for state).

---

## Task 1: AIProvider Interface + Claude Implementation

**Files:**
- Create: `lexis-api/internal/modules/tutor/domain/ai_provider.go`
- Create: `lexis-api/internal/modules/tutor/domain/models.go`
- Create: `lexis-api/internal/modules/tutor/infra/claude_provider.go`
- Create: `lexis-api/internal/modules/tutor/domain/prompts.go`

Domain types: ChatRequest, ChatDelta, ExerciseRequest, Exercise, CheckRequest, CheckResult.
AIProvider interface: Chat(ctx, req) → (<-chan ChatDelta, error), GenerateExercise(ctx, req), CheckAnswer(ctx, req).
System prompt builder: BuildSystemPrompt(settings, mode) from spec section 12.
Claude implementation using Anthropic API with streaming.

---

## Task 2: OpenAI + Qwen + Gemini Providers

**Files:**
- Create: `lexis-api/internal/modules/tutor/infra/openai_provider.go`
- Create: `lexis-api/internal/modules/tutor/infra/qwen_provider.go`
- Create: `lexis-api/internal/modules/tutor/infra/gemini_provider.go`
- Create: `lexis-api/internal/modules/tutor/infra/provider_registry.go`

ProviderRegistry: maps model ID → AIProvider. Used by usecases to get the right provider based on user's ai_model setting.

---

## Task 3: Chat Usecase + SSE Handler (TDD)

**Files:**
- Create: `lexis-api/internal/modules/tutor/usecase/chat_service.go`
- Create: `lexis-api/internal/modules/tutor/usecase/chat_service_test.go`
- Create: `lexis-api/internal/modules/tutor/handler/tutor_handler.go`

Chat usecase: get user settings, build prompt, call AIProvider.Chat, stream deltas.
SSE handler: POST /api/v1/tutor/chat → Content-Type: text/event-stream.
SSE format from spec 7.3: delta, correction, feedback, words, done events.

---

## Task 4: Quiz Usecase + Handler (TDD)

**Files:**
- Create: `lexis-api/internal/modules/tutor/usecase/quiz_service.go`
- Create: `lexis-api/internal/modules/tutor/usecase/quiz_service_test.go`

POST /tutor/quiz/generate → generate question via AI.
POST /tutor/quiz/answer → check answer, return explanation.

---

## Task 5: Translate + Gap + Scramble Usecases (TDD)

**Files:**
- Create: `lexis-api/internal/modules/tutor/usecase/translate_service.go`
- Create: `lexis-api/internal/modules/tutor/usecase/gap_service.go`
- Create: `lexis-api/internal/modules/tutor/usecase/scramble_service.go`

Each: generate exercise via AI, check answer. Endpoints from spec 7.2.

---

## Task 6: Wire Tutor Module in main.go

**Files:**
- Modify: `lexis-api/cmd/api/main.go`

Create provider registry, tutor services, tutor handler. Mount /api/v1/tutor/* routes (protected by auth + 20 req/min rate limit).

---

## Task 7: Frontend — useSSE Hook + Chat Page

**Files:**
- Create: `lexis-web/lib/hooks/useSSE.ts`
- Modify: `lexis-web/app/(app)/chat/page.tsx`

useSSE hook: EventSource wrapper, parses SSE events, returns stream state.
Chat page: message list, correction blocks, typing indicator, input bar — matching Pencil screen 01 exactly.

---

## Task 8: Frontend — Quiz Page

**Files:**
- Modify: `lexis-web/app/(app)/quiz/page.tsx`
- Create: `lexis-web/components/tutor/QuizMode.tsx`

Quiz card with question, confidence bar, 4 options, result block, next button — matching Pencil screen 02.

---

## Task 9: Frontend — Translate + Gap + Scramble Pages

**Files:**
- Modify: `lexis-web/app/(app)/translate/page.tsx`
- Modify: `lexis-web/app/(app)/gap/page.tsx`
- Modify: `lexis-web/app/(app)/scramble/page.tsx`
- Create: `lexis-web/components/tutor/TranslateMode.tsx`
- Create: `lexis-web/components/tutor/GapMode.tsx`
- Create: `lexis-web/components/tutor/ScrambleMode.tsx`

Each mode: exercise card, input/options, result, next — matching spec section 5.8.

---

## Task 10: Frontend — BFF Route Handlers

**Files:**
- Create: `lexis-web/app/api/v1/tutor/chat/route.ts`
- Create: `lexis-web/app/api/v1/tutor/quiz/route.ts`
- Create: `lexis-web/app/api/v1/tutor/translate/route.ts`
- Create: `lexis-web/app/api/v1/tutor/gap/route.ts`
- Create: `lexis-web/app/api/v1/tutor/scramble/route.ts`

BFF proxies: forward requests to Go API, hide AI keys from client. Chat route streams SSE through.

---

## Task 11: ModelSelector Component + Integration

**Files:**
- Create: `lexis-web/components/layout/ModelSelector.tsx`

ModelSelector in ConfigStrip header (compact) — shows current AI model icon.
Used in SettingsModal (full list) — already done in Phase 1.
Fetch models from GET /ai/models.

---

## Task 12: Final Integration + Verification

Verify all 5 modes work end-to-end:
- go build + go test + golangci-lint
- npm run build + tsc --noEmit
- Manual test: chat SSE streaming, quiz flow, translate/gap/scramble
