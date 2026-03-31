# Phase 3 — Progress + Vocabulary

**Goal:** Vocabulary tracking (SM-2), daily snapshots job, progress API endpoints, and the Dashboard page with AccuracyRing, VocabCurve, VocabDonut, MiniChart.

---

## Task 1: Vocabulary domain + SM-2 algorithm
Domain entities, repository interfaces, SM-2 implementation.

## Task 2: Progress domain + repositories
Progress repos: sessions, rounds, goals, errors aggregation queries.

## Task 3: Vocabulary daily snapshot job (asynq)
VocabSnapshotWorker — counts words by status per user/language, writes to snapshots table.

## Task 4: Progress usecase + endpoints (TDD)
GET /progress/summary, /vocabulary, /vocabulary/curve, /goals, /errors, /sessions

## Task 5: Wire progress module in main.go

## Task 6: Frontend — install Recharts, create AccuracyRing + VocabDonut
SVG components matching Pencil screen 03.

## Task 7: Frontend — VocabCurve + MiniChart
Recharts LineChart + mini bar chart.

## Task 8: Frontend — Dashboard page
Full dashboard layout, fetch all endpoints, wire components.

## Task 9: Frontend — Sidebar integration (live goals/feedback/words)

## Task 10: Final verification
