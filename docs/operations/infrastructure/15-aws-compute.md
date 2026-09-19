# 15 — Box ngoài AWS (provision + deploy + vận hành, GĐ2)

Box chạy 6 container (`APP_MODE=http`×3 + `worker`×3) + Nginx phục vụ
`api-v2.closy.hycat.online`, chỉ nói chuyện với managed
(Supabase/Redis Cloud/CloudAMQP). Không IAM, không token-helper.
Độc lập 100% với VPS (chi tiết `09`) — bật/tắt box không đụng VPS.

## 0. Quyết định đã chốt (verify trước khi làm)

- Compute: **Lightsail Small-2GB $12/tháng, region Singapore** (x86_64 — image amd64
  hiện tại dùng nguyên, không build lại). Ước tính ~$36/3 tháng + overage SG $0.12/GB.
- Credit $100 hết hạn 3/2027; Support đã Basic ($0).
- Fallback nếu Lightsail hết capacity: EC2 `t3.micro` cùng region (~$12/tháng khi chạy).
- Không dùng: Fargate+ALB+NAT (~$100+/tháng), App Runner (đóng khách mới 30/4/2026).

## 1. Điều kiện tiên quyết

- [ ] Tài khoản AWS + Lightsail khả dụng ở Singapore; đã bật billing alarm $20/40/60.
- [ ] Supabase / Redis Cloud / CloudAMQP đã tạo, đã có host/port/password (`00` §0.3–0.5).
- [ ] GitHub secrets `AWS_HOST/USER/SSH_PORT/SSH_KEY/KNOWN_HOSTS` + var `AWS_SMOKE_URL`
  (`00` §5); environment `aws-production` (nên bật Required reviewers).
- [ ] Image 3 service đã có trên GHCR (dispatch `deploy-aws.yml` tự build+push).

## 2. Provision box (làm 1 lần)

1. Lightsail Console → **Create instance** → Region **Singapore** → Platform **Linux** →
   Blueprint **Ubuntu 22.04** → Plan **$12 Small-2GB** → tên `closy-api-v2` → Create.
2. **Static IP**: Lightsail → Networking → **Create static IP** → attach vào instance
   (ghi lại IP → secret `AWS_HOST`). Không dùng public IP động (đổi mỗi lần stop/start).
3. **Firewall** (Lightsail → instance → Networking): mở `22` (giới hạn IP bạn),
   `80`, `443`. Đóng hết còn lại.
4. SSH vào box, cài Docker:
   ```bash
   curl -fsSL https://get.docker.com | sh
   sudo usermod -aG docker $USER   # đăng nhập lại
   docker --version && docker compose version
   ```

## 3. Chuẩn bị `/opt/closy-aws` (làm 1 lần)

```bash
sudo mkdir -p /opt/closy-aws && sudo chown $USER:$USER /opt/closy-aws
cd /opt/closy-aws
# copy .env.aws.example từ repo rồi điền:
# - DB_*/REDIS_*/RABBITMQ_*: từ dashboard managed (00 §0.3–0.5, §3–§4)
# - JWT/Cloudinary/Brevo/PayOS/AI keys: COPY NGUYÊN từ VPS .env.production (share quota)
# - ELASTICSEARCH_ENABLED=false (box không chạy ES)
# - INGRESS_*_URL: KHÔNG set (compose hardcode *-http)
chmod 600 .env.aws
```

## 4. TLS cert cho `api-v2` (làm 1 lần, trước DNS cũng được bằng DNS-01 — khuyến nghị HTTP-01 sau khi có IP)

Cách đơn giản: trỏ DNS `api-v2` → IP box trước (TTL thấp), rồi cấp cert trên box:
```bash
cd /opt/closy-aws
docker compose --env-file .env.aws -f deployments/docker-compose.aws.yml \
  --profile maintenance run --rm certbot certonly --standalone \
  -d api-v2.closy.hycat.online --agree-tos -m <email>
```
Renew sau này qua webroot (`certbot_www` đã mount; thêm cron/systemd timer gọi
`certbot renew` + reload nginx).

## 5. Deploy lần đầu + smoke qua IP (chưa cần DNS ổn định)

```bash
gh workflow run deploy-aws.yml -R "<OWNER/REPO>" --ref main -f services=all
```
Workflow: build+push GHCR → scp compose+templates → ghi `.env.deploy` (pin digest) →
`pull`/`up -d` → smoke `AWS_SMOKE_URL` với `--resolve` về IP box (không cần DNS).

Smoke tay (thay `<BOX_IP>`):
```bash
curl --fail -sS -o /dev/null -w '%{http_code}\n' \
  --resolve "api-v2.closy.hycat.online:443:<BOX_IP>" \
  https://api-v2.closy.hycat.online/api/v1/health
for p in auth me/ai system-catalog; do :; done  # route đủ 3 service: xem 09 §5
```

## 6. Cắt DNS `api-v2` sang box

1. Hạ TTL record `api-v2.closy.hycat.online` xuống 60 trước 24–48h.
2. Đổi value → IP box (Vercel DNS, `11`).
3. Smoke + quan sát 24–48h (log `docker compose logs -f`, error-rate, p95).
4. Rollback: trỏ DNS về IP VPS (grace `services-redirect` nếu đã bật redirect).

## 7. Vận hành thường ngày

```bash
cd /opt/closy-aws
docker compose --env-file .env.aws --env-file .env.deploy -f deployments/docker-compose.aws.yml ps
docker compose --env-file .env.aws --env-file .env.deploy -f deployments/docker-compose.aws.yml logs -f --tail=100 <svc>
# redeploy 1 service:
gh workflow run deploy-aws.yml -R "<OWNER/REPO>" --ref main -f services=commerce
```
> Không bao giờ `down -v` (box không có volume data, nhưng tránh nhầm sang VPS).
> Migration chạy lúc boot per container (worker + http cùng service share 1 bảng qua
> advisory lock — an toàn khi deploy rolling).

## 8. Dựng lại từ đầu (rebuild-from-scratch)

Box là stateless (không volume data): provision box mới → lặp lại §3–§5 → đổi DNS.
`.env.aws` backup ở đâu là tùy bạn (password manager — không commit).

## 9. Ứng phó sự cố

| Triệu chứng | Kiểm tra | Xử lý |
|---|---|---|
| `api-v2` 502/timeout | `ps` container, log app, managed conn (pooler/Redis/AMQP) | restart service; redeploy nếu do image mới |
| Cert hết hạn | `certbot certificates` | renew + recreate nginx |
| Box idle-stop/không SSH được | Lightsail console (stop/start), static IP còn attach | start lại; kiểm tra firewall |
| OOM (2GB) | `docker stats`, log oom-killer | giảm `mem_limit` tạm / nâng plan |
| Bill vượt | billing alarm, Lightsail metrics transfer | kiểm tra transfer overage $0.12/GB |

## 10. Tắt box (cắt AWS)

1. Trỏ DNS `api-v2` về VPS + `NGINX_MODE=services-redirect` (grace 14–30 ngày).
2. Reunify data Supabase → VPS (`12-repatriate-vps.md`).
3. Sau grace: xóa record DNS → stop/delete Lightsail instance (giữ snapshot 7 ngày nếu muốn).
4. Hủy managed trước kỳ renew; revoke secrets `AWS_*` khỏi GitHub (pattern `13`).
5. VPS về `NGINX_MODE=services`.

## 11. Verify

- [ ] 6 container healthy, `api-v2` 200 qua IP (SNI override) trước DNS.
- [ ] DNS cắt xong: `api-v2` 200 công khai, cert hợp lệ, renew tự động.
- [ ] VPS không đổi gì trong suốt quá trình (trừ khi chủ động cắt).
- [ ] Billing alarm hoạt động; transfer trong hạn mức.
