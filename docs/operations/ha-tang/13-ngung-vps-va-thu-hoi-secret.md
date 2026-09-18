# 13 — Ngừng VPS cũ và thu hồi credential (FR-016)

Chạy **sau khi** cutover/repatriation đã xác nhận ổn định (gate T082/T108 pass).
Mục tiêu: không để credential cũ (JWT key, SA key, DB/AMQP/Redis, Cloudinary,
Brevo) còn hiệu lực ngoài hệ thống mới, và ngừng chi phí VPS cũ.

## 1. Thu hồi credential cũ

| Loại | Hành động | Xác nhận |
|------|-----------|----------|
| Closy JWT (HS256 legacy) | Xoá `JWT_SECRET` cũ khỏi VPS/managed; rotate sang key mới + redeploy từng service | Không còn token cũ verify được |
| Token-helper SA key | Revoke key trong GCP IAM; xoá file key trên VPS (`0600`); rotate + redeploy helper | `gcloud iam service-accounts keys list` không còn key cũ |
| Supabase/CloudAMQP/Redis Cloud | Rotate credential trong dashboard từng managed service; cập nhật GitHub secrets + redeploy | Kết nối bằng credential cũ bị từ chối |
| Cloudinary/Brevo | Rotate API key; cập nhật secrets + redeploy | Upload/email bằng key cũ fail |
| `SWAGGER_ACCESS_CODE` | Rotate nếu từng lộ | Docs cũ yêu cầu code mới |

Thứ tự: rotate managed trước, VPS sau; mỗi bước redeploy service đó rồi mới thu hồi (tránh downtime).

## 2. Ngừng VPS cũ

1. Xác nhận `api.closy` + `api-v2` (nếu còn) đều phục vụ từ hạ tầng mới ≥ 7 ngày, error-rate/p95 ổn định.
2. Snapshot/dump cuối VPS (DB + Redis RDB + AMQP định nghĩa) và lưu trữ.
3. `docker compose -f deployments/docker-compose.prod.yml down` trên VPS cũ (hoặc stop monolith-legacy profile `rollback`).
4. Giữ DNS TTL thấp trong grace period; sau đó trỏ hẳn về hạ tầng mới.
5. Sau 14–30 ngày không incident: xoá container/volume cũ, huỷ VPS (nếu repatriation ngược lại thì giữ VPS làm đích).

## 3. Checklist ký xác nhận

- [ ] Mọi credential cũ đã rotate/revoke (bảng trên có bằng chứng lệnh).
- [ ] Không còn process/container cũ chạy (`docker ps`, `systemctl`).
- [ ] Billing VPS/managed không còn charge mục đã ngừng.
- [ ] Runbook rollback (về monolith-legacy / 3-service compose) vẫn khả dụng.
