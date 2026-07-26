# Task Coordination: Technical Debt & Business Logic Remediation

**Branch**: `009-technical-debt-remediation`
**Sprints**: 4 (82 tasks total)

## Dependency Order

Sprint 1 — Foundation Security & Payment Operations (P1)
  └── Sprint 2 — Cross-Module Compliance & N+1 (P1)
        └── Sprint 3 — Production Hardening (P2)
              └── Sprint 4 — Notification Module & Business Logic (P3)

> **⚠️ Sprint 3 là prerequisite cho Sprint 4**: Outbox + RMQ infrastructure
> phải hoàn thành trước khi notification module (Sprint 4) có thể dùng outbox.

## Task Files (per sprint)

| Sprint | File | Tasks | Priority |
|--------|------|-------|----------|
| 1 | [`tasks-sprint1.md`](tasks-sprint1.md) | US1.1–US1.3 (25 tasks) | P1 |
| 2 | [`tasks-sprint2.md`](tasks-sprint2.md) | US2.1–US2.3 (20 tasks) | P1 |
| 3 | [`tasks-sprint3.md`](tasks-sprint3.md) | US3.1 (19 tasks) | P2 |
| 4 | [`tasks-sprint4.md`](tasks-sprint4.md) | US4.0–US4.2 (18 tasks) | P3 |

## FR → Sprint Mapping

| Sprint | FR IDs |
|--------|--------|
| Sprint 1 | FR-001 → FR-007 |
| Sprint 2 | FR-008 → FR-014 |
| Sprint 3 | FR-015 → FR-017 |
| Sprint 4 | FR-022 → FR-029 |

## Milestones

- **M1** (Sprint 1): Auto-refund + password policy + worker lifecycle hoàn thành
- **M2** (Sprint 2): 0 cross-module violation, N+1 eliminated, indexes added
- **M3** (Sprint 3): Outbox pattern active, RMQ reconnect resilient, HTTP pool tuned
- **M4** (Sprint 4): Notification module độc lập, email notification đầy đủ
