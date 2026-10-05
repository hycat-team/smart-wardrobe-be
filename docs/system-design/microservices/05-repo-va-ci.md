# 05 — Repository, CI và triển khai độc lập

## Repository & module

- Multi-module Go workspace (`go.work`): `services/account`, `services/commerce`,
  `services/ai`, `shared`, `contracts` (+ `deployments/token-helper`).
- Mỗi service: module riêng, container riêng, healthcheck riêng (`/api/v1/health`),
  config riêng, scale riêng — deploy service này KHÔNG cần redeploy service khác.
- Migration: mỗi service sở hữu (`services/<svc>/database/migrations`), tự chạy
  khi boot (Goose) với **bảng version riêng** + allow-out-of-order + advisory lock
  dùng chung; 24→toàn bộ migration lịch sử do Account giữ (FR-035).
- Dockerfile riêng từng service (multi-stage, context = repo root, distroless +
  tzdata nhúng); image chỉ chứa binary + migration của chính service (FR-042).

## CI (`.github/workflows/ci.yml`, FR-020)

- Cache `**/go.sum` + `go.work.sum`; `buf lint` + proto drift.
- Build matrix 3 service (`make build svc=`/`build-all`); `image-build` matrix.
- Parity swagger 9 section; KHÔNG build monolith.
- Gate: `make wire && make lint && make test-race`.

## Registry & deploy độc lập (FR-042/FR-051/FR-052)

- Cloud Run ← **Artifact Registry** (`asia-southeast1`, tag `<svc>:<git-sha>` immutable).
- VPS (rollback + repatriation) ← **GHCR** (token sẵn có). Không public image private.
- Deploy độc lập từng service (CI matrix; 1 service fail không chặn 2 còn lại);
  version pin theo SHA từng service, không all-or-nothing (SC-017).
- Secrets tiêm qua env từ GitHub Actions secrets lúc deploy; rotate = redeploy
  service đó; KHÔNG commit/bake/log giá trị (SC-026).
- Rollback monolith đã ngừng tự động hoá (ADR 012): tag `monolith-legacy` giữ làm
  tham chiếu khẩn cấp (constitution §II); compose profile `rollback` chỉ dùng thủ công.
