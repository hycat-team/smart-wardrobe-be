# 13 — Ngừng VPS cũ và thu hồi credential (FR-016)

Chạy **sau khi** cutover/repatriation đã xác nhận ổn định (gate T082/T108 pass).
Mục tiêu: không để credential cũ còn hiệu lực ngoài hệ thống mới, và ngừng chi phí VPS cũ.

Thứ tự chung: **rotate managed trước, VPS sau**; mỗi bước redeploy service tương ứng
rồi mới thu hồi (tránh downtime).

## 1. Rotate/revoke từng loại credential

### 1.1 Closy JWT (HS256 legacy)
```bash
NEW_SECRET=$(openssl rand -base64 48)
gh secret set JWT_SECRET -R "<OWNER/REPO>" --body "$NEW_SECRET"
gh workflow run deploy-aws.yml -R "<OWNER/REPO>" --ref main -f services=all
gh workflow run deploy-vps-services.yml -R "<OWNER/REPO>" --ref main -f services=all
# (nếu còn VPS) cập nhật .env.production + up -d --force-recreate account commerce ai
```
- [ ] Token phát bằng secret cũ không verify được.

### 1.2 SSH key box ngoài (AWS)
```bash
# Xoá keypair khỏi AWS (Lightsail/EC2 console) + revoke secrets GitHub:
gh secret remove AWS_SSH_KEY -R "<OWNER/REPO>"
gh secret remove AWS_KNOWN_HOSTS -R "<OWNER/REPO>"
# (giữ AWS_HOST/USER/PORT nếu còn box khác; xoá nốt khi tắt hẳn)
```
- [ ] Không còn SSH được vào box bằng key cũ; secret đã xoá khỏi GitHub.

### 1.3 Supabase / CloudAMQP / Redis Cloud
- Rotate credential trong dashboard từng dịch vụ → cập nhật GitHub secrets (`DB_PASSWORD`,
  `RABBITMQ_PASSWORD`, `REDIS_PASSWORD`) → redeploy.
- [ ] Kết nối bằng credential cũ bị từ chối.

### 1.4 Cloudinary / Brevo
- Rotate API key/SMTP key → cập nhật secrets `CLOUDINARY_*`, `EMAIL_SMTP_KEY` → redeploy.
- [ ] Upload/email bằng key cũ fail.

### 1.5 `SWAGGER_ACCESS_CODE`
- Rotate nếu từng lộ; docs cũ phải dùng code mới.

## 2. Ngừng VPS cũ

1. Xác nhận `api.closy` (+ `api-v2` nếu còn) phục vụ từ hạ tầng mới **≥ 7 ngày**,
   error-rate/p95 ổn định.
2. Snapshot/dump cuối VPS (DB + Redis RDB + AMQP definitions) và lưu trữ:
   ```bash
   cd /opt/closy
   docker compose --env-file .env.production -f deployments/docker-compose.prod.yml \
     exec -T postgres pg_dump -U "$DB_USER" -d "$DB_NAME" -Fc > backup/final_$(date +%F).dump
   ```
3. Dừng stack:
   ```bash
   docker compose --env-file .env.production -f deployments/docker-compose.prod.yml down
   ```
   (hoặc stop profile `rollback` nếu đang chạy monolith-legacy).
4. Giữ DNS TTL thấp trong grace period; sau đó trỏ hẳn về hạ tầng mới.
5. Sau 14–30 ngày không incident: xoá container/volume cũ; huỷ box ngoài
   (VPS luôn giữ làm đích — không huỷ VPS).

## 3. Checklist ký xác nhận

- [ ] Mọi credential cũ đã rotate/revoke (bảng trên có bằng chứng lệnh).
- [ ] Không còn process/container cũ chạy (`docker ps`, `systemctl`).
- [ ] Billing VPS/managed không còn charge mục đã ngừng.
- [ ] Runbook rollback (về monolith-legacy / 3-service compose) vẫn khả dụng.
