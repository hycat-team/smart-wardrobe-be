# Architecture Decision Record (ADR) 024

## Title: 024 - Drop Cloud Run, decouple VPS serving from the external box (AWS later)

*   **Status**: Accepted
*   **Decision Date**: 2026-09-19
*   **Decision Maker**: repo owner

## 1. Context

Cloud Run never existed in this project: no GCP project, no `GCP_*`/`INGRESS_*`
secrets, `deploy-platform.yml` never ran, and the trial/billing assumptions behind
FR-044..FR-048 are void. The previous plan coupled "what the VPS serves" with
"whether the cloud exists" into shared Nginx modes (`parallel`/`parallel-v2`),
which blocks the required rollout order: VPS monolith -> 3 services first, external
box later. Supersedes ADR 023 (VPS Nginx ingress + Cloud Run IAM) for the compute
part; the Nginx ingress mechanics stay.

## 2. Decision

1. Delete all Cloud Run remnants: `deploy-platform.yml`, `deployments/token-helper/`,
   `api-v2-routes/`, `make deploy`, Cloud Run IAM docs. No app-code change.
2. VPS serving becomes one independent axis: `services` (api.closy to 3 local
   services, monolith stopped), `services-redirect` (grace redirect only),
   `rollback` (monolith emergency, dormant).
3. The external box becomes a second independent axis, on AWS (Lightsail
   Small-2GB Singapore, ~$12/mo) when GĐ2 runs: 6 containers (http x3 + worker x3)
   + Nginx for `api-v2.closy.hycat.online`, talking only to managed infra
   (Supabase/Redis Cloud/CloudAMQP). x86_64, so existing amd64 images are reused.
4. DNS is the only coordination point between the two axes. Runtime data
   divergence during parallel serving is accepted; reunification is manual
   dump/restore (`10`, `12`).

## 3. Consequences

Positive: VPS migration (GĐ1) proceeds with zero cloud dependency; no dead
Cloud Run/IAM code to confuse operators; AWS box can be switched on/off without
touching the VPS; monolith rollback stays available.
Negative: two datastores diverge while both axes serve writes (manual reunify on
cut); managed accounts (Supabase/Redis/AMQP) must be registered before GĐ2;
Lightsail 2GB needs memory observation (caps are 512m per container).
