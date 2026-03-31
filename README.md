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
