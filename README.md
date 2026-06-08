# Lexis

[![CI](https://github.com/VDV001/lexis/actions/workflows/api.yml/badge.svg)](https://github.com/VDV001/lexis/actions/workflows/api.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Version](https://img.shields.io/badge/version-0.16.0-green.svg)](VERSION)

**AI-репетитор иностранных языков**, который адаптируется под ваш уровень, ведет персональный словарь и тренирует через разнообразные упражнения - все на базе выбранной вами AI-модели.

## Зачем Lexis?

Большинство приложений для изучения языков предлагают статичные карточки и заскриптованные диалоги. Lexis генерирует упражнения на лету с помощью AI, подстраивает сложность под ваш уровень владения языком и формирует персональный словарь из каждого взаимодействия. Вы практикуетесь с живым AI-тьютором, а не с записанными скриптами.

## Что умеет

**Разговорный тьютор** - свободный чат с AI, который говорит на целевом языке, исправляет ошибки и вводит новую лексику в контексте.

**4 режима упражнений** - Квиз, Перевод, Заполнение пропусков, Составление слов. Каждое упражнение генерируется AI на основе ваших настроек (язык, уровень, тип лексики).

**Словарь с интервальным повторением** - каждое встреченное слово сохраняется в личный словарь с планированием повторений по алгоритму SM-2. Слова распределяются по статусам *новое*, *неуверен*, *уверен* и повторяются через оптимальные интервалы.

**Аналитика прогресса** - точность ответов, серии правильных ответов, категории ошибок, кривая роста словаря, отслеживание целей с автоматической корректировкой.

**Мульти-модельность** - подключите OpenRouter (один шлюз к GPT / Claude / Gemini / и др.) или нативные API (Claude, GPT-4o, Qwen, Gemini). Модель можно переключить в любой момент в настройках; для OpenRouter список моделей подгружается автоматически.

## Архитектура

```
lexis-api/          Go-бэкенд (модульный монолит, clean architecture)
  cmd/api/          Точка входа HTTP-сервера
  internal/
    modules/
      auth/         Регистрация, логин, JWT, refresh-токены, настройки
      tutor/        AI-чат + генерация упражнений через мульти-провайдерную абстракцию
      vocabulary/   CRUD слов, интервальное повторение, ежедневные снапшоты
      progress/     Сессии, раунды, цели, учет ошибок
    shared/
      eventbus/     Внутрипроцессный pub/sub с типизированными событиями
      httputil/     Общие HTTP-хелперы (RFC 7807)
      middleware/   Авторизация, CORS, rate limiting

lexis-web/          Next.js 16 фронтенд (React 19, TypeScript)
  app/(app)/        Защищенные маршруты: чат, квиз, перевод, пропуски, слова, словарь, аналитика
  app/(auth)/       Логин, регистрация
  components/       Переиспользуемые UI-компоненты
```

Каждый модуль бэкенда следует слоям **domain / usecase / handler / infra**. Зависимости направлены внутрь: обработчики зависят от usecase, usecase - от интерфейсов домена, infra реализует эти интерфейсы.

## Стек технологий

| Слой | Технология |
|------|-----------|
| Бэкенд | Go 1.26, chi, pgx, go-redis |
| Фронтенд | Next.js 16, React 19, TypeScript |
| База данных | PostgreSQL 18 |
| Кэш | Redis 8 |
| AI-провайдеры | OpenRouter (шлюз), Anthropic (Claude), OpenAI (GPT-4o), Alibaba (Qwen), Google (Gemini) |

## Быстрый старт

### Зависимости

- Go 1.26+
- Node.js 22+
- Docker и Docker Compose

### Запуск инфраструктуры

```bash
git clone https://github.com/VDV001/lexis.git
cd lexis
docker compose up -d
```

Запускает PostgreSQL и Redis.

### Запуск бэкенда

```bash
cd lexis-api
cp .env.example .env   # заполните API-ключи
go run ./cmd/api
```

### Запуск фронтенда

```bash
cd lexis-web
npm install
npm run dev
```

Откройте [http://localhost:3000](http://localhost:3000).

## Конфигурация

Все настройки бэкенда задаются через переменные окружения (см. `.env.example`):

- `DATABASE_URL` - строка подключения к PostgreSQL
- `REDIS_URL` - строка подключения к Redis
- `APP_SECRET` - секрет для подписи access/refresh-токенов (обязателен в production, 32+ символов)
- `OPENROUTER_API_KEY` - ключ OpenRouter (рекомендуемый путь, см. «Production deploy via OpenRouter»)
- `ANTHROPIC_API_KEY`, `OPENAI_API_KEY`, `QWEN_API_KEY`, `GEMINI_API_KEY` - ключи нативных AI-провайдеров

Нужен **хотя бы один** ключ AI-провайдера, иначе сервер падает на старте с понятной
ошибкой (fail-fast).

## Production deploy via OpenRouter

Lexis рассчитан на **self-hosting**: вы разворачиваете свою инстанцию. Рекомендуемый
провайдер моделей - [OpenRouter](https://openrouter.ai): один OpenAI-совместимый шлюз
даёт доступ к GPT / Claude / Gemini / и др., и оплачивается там, где нативные API
провайдеров недоступны.

### 1. Подготовьте `.env`

В корне репозитория создайте `.env`:

```bash
# Сгенерируйте надёжный секрет (32+ символов)
APP_SECRET=$(openssl rand -base64 32)

# Ключ OpenRouter: https://openrouter.ai/keys
OPENROUTER_API_KEY=sk-or-v1-...

# Опционально, если фронт открывается не с localhost (укажите ваш домен/хост):
# NEXT_PUBLIC_API_URL=https://example.com/api/v1
# CORS_ALLOWED_ORIGINS=https://example.com
```

`docker compose` подхватит `.env` автоматически. `APP_SECRET` обязателен - запуск
прервётся с понятной ошибкой, если он не задан.

### 2. Поднимите стек

```bash
docker compose -f docker-compose.yml -f docker-compose.prod.yml up -d --build
```

Поднимает PostgreSQL, Redis, API (Go) и web (Next.js, production-сборка). Миграции
применяются на старте API; backup-сервис доступен через профиль `backup`. Проверьте
готовность:

```bash
curl http://localhost:8080/health        # {"status":"ok",...}
docker compose ps                         # все сервисы healthy
```

Откройте [http://localhost:3000](http://localhost:3000).

> Обычный `docker compose up -d` (без `-f docker-compose.prod.yml`) поднимает **dev**-стек
> с hot reload. Production-режим - только через override выше.

### 3. Выберите модель и начните

1. Зарегистрируйтесь на [http://localhost:3000](http://localhost:3000).
2. Откройте настройки → **AI Модель**. Список подгружается из живого каталога OpenRouter
   (отфильтрован до пригодных text-chat моделей). Выберите, например,
   `openai/gpt-4o-mini` (дёшево) или `anthropic/claude-3.5-sonnet` (качество).
3. Ведите разговор, делайте упражнения - встреченные слова попадают в словарь (SM-2).

> **Дефолтная модель** новых пользователей - нативная (`claude-sonnet-4-...`). Если у вас
> сконфигурирован только OpenRouter, выберите OpenRouter-модель в настройках перед первым
> разговором, иначе чат вернёт ошибку «unknown model».

### TLS и обратный прокси

API отдаёт security-заголовки (`X-Frame-Options`, `X-Content-Type-Options`,
`Referrer-Policy`, `Content-Security-Policy`), но **не** ставит `Strict-Transport-Security`:
HSTS и TLS - ответственность обратного прокси (nginx / Caddy / Traefik) перед стеком.
Для публичного деплоя поставьте такой прокси и пропишите домен в `NEXT_PUBLIC_API_URL`
и `CORS_ALLOWED_ORIGINS`.

### Backup и restore

Ежедневный бэкап (`pg_dump → age → S3`) и проверенная процедура восстановления описаны
в [docs/operations/backup-restore.md](docs/operations/backup-restore.md). Recovery target -
30-60 минут.

## Известные ограничения

### CEFR-уровень: scaffolding промпта, не валидация

Lexis использует выбранный пользователем CEFR-уровень (A2/B1/B2/C1) как
**ярлык-scaffolding** в системном промпте AI-модели. Конкретно -
`lexis-api/internal/modules/tutor/domain/prompts.go`, `levelContextMap`
подставляет в промпт строку из ~10 слов («B1 (intermediate, Russian
developer, limited active production)»).

**Это не настоящая CEFR-валидация.** Реальная CEFR-проверка требует:

- лексического контроля по [English Vocabulary Profile](https://englishprofile.org/wordlists/evp) (~15 700 значений с уровнями A1-C2);
- грамматического контроля по [English Grammar Profile](https://englishprofile.org/english-grammar-profile/egp-online) (~1 200 пунктов с уровнями);
- post-generation валидации, что сгенерированный ответ не использует лексику/грамматику выше уровня пользователя.

Ничего из этого Lexis на сегодня не делает. Модель сама решает, что
значит «B1», по training data - и часто промахивается, особенно на
крайних уровнях. Поэтому в Lexis сознательно реализованы только 4
уровня (A2, B1, B2, C1), без A1 и C2 - крайности модель различает
хуже всего без внешней валидации.

### Что работает хорошо

Тематический scaffolding (`vocabulary_type: tech / literary / business`)
- модель действительно отбирает лексику по теме неплохо. Отслеживание
словаря пользователя (`vocab_status: unknown / uncertain / confident`)
с алгоритмом SM-2 - работает как заявлено: уровневой валидации это
не заменяет, но даёт реальный персонализированный словарный рост.

### Если вам важна точная CEFR-валидация

Lexis в текущем виде не подходит. Альтернативы:

- интегрировать EVP/EGP datasets и lexicon-фильтр (большая работа -
  недели);
- использовать продукты с явно заявленной CEFR-валидацией (обычно
  платные, корпоративные).

Issue для отслеживания: см. `Issues` в этом репозитории, метка `cefr`.

## Roadmap

- **Объектное хранилище (MinIO)** - запланировано в плане инфраструктуры
  ([docs/plans/2026-03-31-phase-0-infrastructure.md](docs/plans/2026-03-31-phase-0-infrastructure.md)),
  ещё не интегрировано в `docker-compose.yml`. Будет добавлено когда
  появится фича, которой нужно файловое хранилище (загрузка аватаров,
  экспорт прогресса, и т.п.).

## AgentIgnore

Репо поставляется с `.agentignore` - bounding box для AI-агентов
(Claude Code / Cursor / Veai / etc.). Файл указывает что не подавать
в контекст и что не модифицировать. Стандарт: memory-architecture v1.3.

Ключевые ограничения:

- `.env*`, `secrets/`, `*.key`, `*.pem` - секреты
- `tests/`, `*_test.go`, `*.bats` - тесты как контракт (TDD-флоу,
  предотвращает «починку» тестов под имплементацию)
- `lexis-api/migrations/` - schema as code
- `scripts/backup/restore*.sh`, `retention.sh` - production-критичные скрипты

## Лицензия

Проект распространяется под лицензией [MIT](LICENSE).
