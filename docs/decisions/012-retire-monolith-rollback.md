# ADR 012 - Retire the automated monolith rollback workflow

*   **Status**: Accepted
*   **Decision Date**: 2026-10-05
*   **Decision Maker**: repo owner
*   **Supersedes**: ADR 010 (phần rollback monolith)

## 1. Context

`deploy-production.yml` was the emergency monolith rollback path (spec 020:
FR-043/FR-056, SC-030): `workflow_dispatch`, build from the frozen
`monolith-legacy` tag, push to GHCR, then `docker compose --profile rollback up -d
backend`. The owner no longer wants monolith rollback: the 3-service VPS topology
has been the runtime since the 020 migration, the workflow last ran on
2026-09-12, and the rollback image was never pre-built/pushed to GHCR (T097a is
still open). The constitution §II only requires the frozen tag
`monolith-legacy` to be preserved "for emergency reference only" — it does not
require a workflow.

## 2. Decision

1. Archive `.github/workflows/deploy-production.yml` to
   `docs/archive/deploy-production.yml` (GitHub only executes files under
   `.github/workflows/`). To reactivate, move it back into `.github/workflows/`.
2. Keep the frozen tag `monolith-legacy` untouched (constitution §II).
3. Keep the manual fallback materials dormant: `deployments/monolith-legacy/Dockerfile`,
   the compose `rollback` profile + `MONOLITH_IMAGE`, `deployments/nginx/templates/rollback/`,
   and `make image-build-monolith-rollback`. An operator who explicitly chooses the
   rollback path can still build and run it by hand.
4. Spec 020 marks FR-043, the monolith bullet of FR-056, SC-030 and task T097a as
   superseded/cancelled by this ADR; runbooks stop instructing operators to run
   the archived workflow.
5. `deploy-vps-services.yml` remains the only automated VPS deploy path.

## 3. Consequences

Positive: one fewer deploy path to maintain and audit; no operator is pointed at
a workflow that is no longer in the Actions list; docs no longer promise an image
that was never built.

Negative: emergency rollback to the monolith becomes fully manual — build the
image from the tag, push it to GHCR, then
`docker compose --profile rollback up -d backend`; recovery time is longer if
that path is ever needed. The supported route back to a previous release stays the
service-digest pinning described in the VPS runbook.
