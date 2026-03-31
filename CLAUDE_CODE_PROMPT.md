# lexis — Стартовый промпт для Claude Code

## Контекст

Ты реализуешь **lexis** — AI-powered платформу для изучения языков.
Рабочая директория: `~/git/lexis/`
Все решения приняты. Реализуй строго по спецификации.

---

## Файлы проекта

| Файл | Описание |
|------|----------|
| `lang_tutor_spec_v3.docx` | Полная техническая спецификация (13 разделов) — **читать полностью** |
| `english_tutor_seldon_v2.html` | Рабочий HTML-прототип с точным CSS |
| `pencil-new.pen` | Визуальный макет (4 экрана) — справочный материал |

---

## Приоритет источников для UI/UX

```
1. Figma MCP                     ← наивысший приоритет
   Файл:     vh05h7t6sFr4ErGVHgxLoE
   Страница: "English tutor seldon"
   Node:     6053-395
   Инструменты:
     Figma:get_metadata()
     Figma:get_design_context()
     Figma:get_variable_defs()

2. english_tutor_seldon_v2.html  ← точный CSS и поведение компонентов

3. pencil-new.pen                ← справочно, к сведению
   u9R5v (Chat)  k0o5i (Quiz)  4uPUI (Dashboard)  U1ts7 (Settings)
```

При реализации каждого компонента — сначала Figma, потом HTML.

---

## Технический стек

```
Backend:   Go 1.26.1
Frontend:  Next.js 16 + React 19.2 + TypeScript strict
Database:  PostgreSQL 18.3
Cache:     Redis 8.0
Storage:   MinIO (AGPLv3)
AI:        Multi-model (Claude / Qwen / GPT / Gemini)
```

Полный список зависимостей — раздел 2 документа.

---

## Архитектура (не нарушать)

- **Modular Monolith** — модули `auth/` `tutor/` `progress/` `vocabulary/`,
  каждый: `domain/` `usecase/` `handler/` `infra/`
- **TDD** — failing test пишется ДО кода
- **DDD** — агрегаты, value objects, domain events (раздел 3.2)
- **Clean Architecture** — зависимости только внутрь
- Модули общаются только через публичные интерфейсы

---

## CSS (строго)

```css
/* Только переменные — никаких хардкодных hex */
--bg:#0d1117  --bg2:#161b22  --bg3:#1c2128  --bg4:#21262d
--green:#3fb950  --cyan:#58a6ff  --amber:#e3b341  --red:#f85149
--font-mono:'JetBrains Mono','Courier New',monospace

/* 5 анимаций — обязательны все */
@keyframes blink  { 0%,100%{opacity:1} 50%{opacity:0} }
@keyframes pulse  { 0%,100%{opacity:1} 50%{opacity:0.4} }
@keyframes spin   { to{transform:rotate(360deg)} }
@keyframes fadeUp { from{opacity:0;transform:translateY(4px)} to{opacity:1;transform:translateY(0)} }
@keyframes tdot   { 0%,60%,100%{transform:translateY(0)} 30%{transform:translateY(-3px)} }
```

Полный CSS reference — раздел 5 документа.

---

## Фазы реализации

### Фаза 0 — Инфраструктура

```bash
cd ~/git/lexis

# Backend
mkdir lexis-api && cd lexis-api
go mod init github.com/yourusername/lexis-api
mkdir -p cmd/api
mkdir -p internal/modules/{auth,tutor,progress,vocabulary}/{domain,usecase,handler,infra}
mkdir -p internal/shared/{middleware,config,eventbus}
mkdir -p migrations

# Frontend
cd ~/git/lexis
npx create-next-app@latest lexis-web \
  --typescript --tailwind --app --no-src-dir --import-alias "@/*"

# Инфраструктура
docker compose up -d        # раздел 10 документа
# DB миграции               # раздел 6 документа
```

**Не переходи к Фазе 1 без завершения Фазы 0.**

### Фаза 1 — Auth + Settings
### Фаза 2 — Tutor Core + Multi-model AI
### Фаза 3 — Progress + Vocabulary
### Фаза 4 — Polish + Tests
### Фаза 5 — Deploy

Детали каждой фазы — раздел 11 документа.

---

## Ключевые детали реализации

**Multi-model AI** (раздел 2.6)
```go
type AIProvider interface {
    Chat(ctx context.Context, req ChatRequest) (<-chan ChatDelta, error)
    GenerateExercise(ctx context.Context, req ExerciseRequest) (Exercise, error)
    CheckAnswer(ctx context.Context, req CheckRequest) (CheckResult, error)
}
```

**SSE стриминг** (раздел 7.3)
```
data: {"type":"delta","content":"Hello"}
data: {"type":"correction","correction":{...}}
data: {"type":"feedback","feedback":{...}}
data: {"type":"words","words":["goroutine"]}
data: {"type":"done"}
```

**AI-ключи** — только на сервере, через BFF Route Handlers (`lexis-web/app/api/v1/tutor/*/route.ts`)

**Промпты** — только raw JSON, объяснения только на русском (раздел 12)

**Кривая роста словаря** — `vocabulary_daily_snapshots`, снимок 00:00 UTC, endpoint `/progress/vocabulary/curve` (раздел 6.4)

---

## Definition of Done

**Каждая задача:**
- [ ] Failing test написан ДО кода
- [ ] `golangci-lint` / `tsc --noEmit` — чисто
- [ ] UI соответствует Figma / HTML reference

**MVP Release** — раздел 8.4 документа.

---

## Env

```bash
# lexis-api/.env
ANTHROPIC_API_KEY=
QWEN_API_KEY=
OPENAI_API_KEY=
GEMINI_API_KEY=

# lexis-web/.env.local
NEXT_PUBLIC_API_URL=http://localhost:8080/api/v1
```

Полный список — раздел 9 документа.

---

## Стоп-правила

❌ Не меняй стек
❌ Не пропускай тесты
❌ Не хардкодь цвета — только `var(--xxx)`
❌ Не переходи к следующей фазе без завершения текущей
❌ Не используй `any` в TypeScript
❌ Не помещай AI-ключи в клиентский код

---

## Начало работы

1. Прочитай `lang_tutor_spec_v3.docx` полностью
2. Начни **Фазу 0** в `~/git/lexis/`
3. Доложи о результате перед переходом к Фазе 1
