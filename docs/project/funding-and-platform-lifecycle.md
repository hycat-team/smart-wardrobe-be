# Kinh phí nền tảng và vòng đời Cloud (3 tháng)

Tài liệu này ghi nhận các hạng mục hạ tầng được đầu tư kinh phí, thời hạn sử dụng và nghĩa vụ
quay về VPS khi hết kinh phí. Đây là quyết định vòng đời dự án (project lifecycle), không phải
quy định code — quy định code nằm ở hiến pháp (`.specify/memory/constitution.md`).

Chi tiết kỹ thuật triển khai: xem ADR `docs/decisions/007-funded-platforms-and-repatriation.md`
và `specs/020-platform-migration/`.

## 1. Hạng mục được đầu tư (duy trì 3 tháng)

| Hạng mục | Gói | Vai trò |
|---|---|---|
| Cloudflare | Pro Plan | Funded, **KHÔNG nằm trong đường API** (theo ADR 009); dùng cho mục đích khác nếu cần |
| Supabase | Pro Plan | Postgres + pgvector (1 project, **1 database dùng chung**, không tách schema staging/prod) |
| CloudAMQP | Tough Tiger | RabbitMQ managed |
| Redis Cloud | Essentials 250MB | Cache managed |

Compute (Cloud Run) **không thuộc diện đầu tư**: chạy bằng trial/free-tier + billing cap cứng.
**Teardown trial là một bước trong repatriation** (không tách rời sau demo); bill trial phải về 0.

**Dependency hiện hữu (KHÔNG tính vào danh mục đầu tư, không phải hạng mục mới)**:

- **Cloudinary** — media/upload (`internal/shared/infrastructure/media/cloudinary_service.go`). Đã dùng
  từ trước giai đoạn 020; **không nằm trong 4 hạng mục funded** và **không thêm mới**. Key tiêm qua
  env ở Cloud Run (từ GitHub Actions secrets, FR-052). Không chuyển provider trong phạm vi 020.
- **GCP Artifact Registry** — lưu image Cloud Run (native); dùng trong hạn mức free (≈0.5GB), không
  phải dịch vụ tính phí mới (FR-051).

## 2. Nguyên tắc bắt buộc

1. **Không phát sinh dịch vụ ngoài danh sách trên** (không LB/Armor/DB-managed khác).
2. **Runbook repatriation về VPS phải được viết song song** với runbook lên Cloud, không được
   để tới khi hết kinh phí mới viết.
3. **Funding clock**: hard deadline hết 3 tháng + checkpoint T-30/T-14/T-7/T-1 có xác nhận
   (verify backup, drill restore, rehearsal rollback DNS).
4. Repatriation phải hoàn tất **trước ngày renew** của từng hạng mục để không bị charge thêm
   chu kỳ mới.
5. VPS (3 service compose + Nginx gateway, không phải monolith) là đích quay về; tag
   `monolith-legacy` chỉ để tham chiếu khẩn cấp.

## 3. Yêu cầu repatriation (ràng buộc vòng đời, không phải quy định code)

Mọi hạ tầng managed có thời hạn đã biết **bắt buộc** đi kèm đường repatriation về self-hosted
đã được kiểm thử:

- Đường repatriation phải được drill thành công trên môi trường Cloud **trước T-30**.
- Dữ liệu phải migrate ngược đầy đủ có verify checksum (Supabase→Postgres, drain CloudAMQP,
  snapshot/warmup Redis).
- DNS phải trả về VPS và funded items phải tắt trước ngày renew.
- Không có ngoại lệ "để sau": runbook repatriation là deliverable cùng đợt với runbook lên Cloud.

## 4. Theo dõi

- Ngày bắt đầu tính 3 tháng: ghi vào plan 020 khi kích hoạt hạng mục đầu tiên.
- Mỗi checkpoint T-minus phải có người ký xác nhận + bằng chứng thực thi (log restore drill,
  ảnh billing, kết quả smoke sau rehearsal).
