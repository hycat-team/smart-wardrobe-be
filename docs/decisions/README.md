# Architecture Decision Records (ADR)

Mỗi file ghi một quyết định kiến trúc/vận hành đã chốt, kèm bối cảnh và hệ quả.
Đọc `template.md` trước khi viết ADR mới.

## Index

| # | Ngày | Tiêu đề | Trạng thái | Ghi chú |
|---|---|---|---|---|
| [001](001-b2b2c-pivot.md) | 2026-06-28 | Pivot to B2B2C business model | Accepted | |
| [002](002-remove-community-resale.md) | 2026-06-28 | Remove Community Posts & Resale features as core | Accepted | |
| [003](003-digital-sample-lab.md) | 2026-06-28 | Implement Digital Sample Lab feature for B2B | Accepted | |
| [004](004-out-of-scope-2026-08.md) | 2026-08-20 | Out of Scope Features (2026-08) | Accepted | 9 hạng mục ngoài phạm vi MVP |
| [005](005-rabbitmq-refactor.md) | 2026-08-27 | Refactor Hệ thống Messaging RabbitMQ | Accepted | bổ sung §4 ngày 2026-10-01 (reconcile topology) |
| [006](006-pre-launch-config-hardening.md) | 2026-09-06 | Hardening Config Pre-launch | Accepted | bị thay một phần bởi 008 (§2.3), 011 (§2.5) |
| [007](007-funded-platforms-and-repatriation.md) | 2026-09-15 | Nền tảng được tài trợ + lộ trình repatriation | Accepted | bị thay một phần bởi 009, 010 |
| [008](008-email-provider-brevo.md) | 2026-09-15 | Email Provider: Brevo thay Resend | Accepted | thay 006 §2.3 |
| [009](009-vps-nginx-ingress-cloud-run-iam.md) | 2026-09-15 | Ingress VPS Nginx + Cloud Run IAM | Accepted | thay 007 §2.1/2.2/2.6/2.7/2.8; bị 010 thay phần compute |
| [010](010-drop-cloud-run-decouple-axes.md) | 2026-09-19 | Drop Cloud Run, decouple VPS serving | Accepted | thay 009 phần compute; bị 012 thay phần rollback monolith |
| [011](011-production-env-semantics.md) | 2026-09-20 | Ngữ nghĩa `SERVER_ENV`: `vps` cũng là production | Accepted | đính chính 006 §2.5 |
| [012](012-retire-monolith-rollback.md) | 2026-10-05 | Retire the automated monolith rollback workflow | Accepted | thay 010 phần rollback monolith |
| [013](013-deferred-lazy-grant-self-heal.md) | 2026-10-07 | Deferred: lazy grant self-heal when an account has no policy grant | Accepted | nợ kỹ thuật; điều kiện mở lại ở §2 |

## Quy ước

- **Tên file**: `NNN-ten-kebab.md` — `NNN` là 3 chữ số, cấp tăng dần.
- **Tiêu đề**: dòng đầu là `# ADR NNN - Tiêu đề`.
- **Metadata bắt buộc**: `Status`, `Decision Date` (YYYY-MM-DD), `Decision Maker`.
  Tuỳ chọn: `Supersedes`, `Superseded by` (ghi rõ số ADR + phạm vi nếu chỉ thay một phần).
- **Status**: `Proposed` / `Accepted` / `Superseded by ADR NNN` / `Deprecated` / `Rejected`.
- **Số ADR là bất biến**: cấp tăng dần, **không tái sử dụng, không đánh số lại** kể cả khi
  ADR bị supersede. Quyết định bị thay thế thì viết ADR mới và thêm dòng `Superseded by`
  ở ADR cũ; không sửa nội dung quyết định cũ (đính chính nhỏ ghi vào mục "Bổ sung (ngày)").
- **Ngôn ngữ**: heading các mục giữ tiếng Anh (`Context` / `Decision` / `Consequences`),
  phần thân viết tiếng Việt. Các ADR lịch sử viết tiếng Anh (001–003, 010, 012) giữ nguyên,
  không dịch lại.

## Lịch sử đánh số (2026-10-05)

Trước ngày này, file dùng số không liền mạch và 4 file thiếu số. Toàn bộ đã được đánh số
lại liền mạch 001–012 theo ngày quyết định; tra cứu tài liệu/PR cũ theo bảng sau:

| Số cũ | Số mới |
|---|---|
| `b2b2c-pivot`, `remove-community-resale`, `digital-sample-lab` | 001, 002, 003 (số đã có sẵn, chỉ thêm prefix tên file) |
| `out-of-scope-2026-08` (không số) | 004 |
| 016 | 005 |
| 020 | 006 |
| 021 | 007 |
| 022 | 008 |
| 023 | 009 |
| 024 | 010 |
| 025 | 011 |
| 026 | 012 |
