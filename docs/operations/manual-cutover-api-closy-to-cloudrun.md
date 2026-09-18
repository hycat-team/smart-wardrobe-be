# Runbook cutover thủ công `api.closy` → Cloud Run (FR-041)

**Chỉ thực thi khi owner quyết định** `api-v2` đủ ổn định. Không tự động hoá.

## 1. Gate đo được (cố định trong spec)

- Smoke test **100% pass**.
- Error-rate **≤ 1%**, p95 **≤ 500ms**, SSE **không rớt**, ổn định **7 ngày liên tiếp**.
- Runbook này chỉ chi tiết hoá, không nới ngưỡng.

## 2. Điều kiện tiên quyết

- [ ] `api-v2` ổn định đủ 7 ngày (bằng chứng từ dashboard `infrastructure/14-monitoring.md`).
- [ ] Dữ liệu đã đồng bộ (dump VPS → Supabase tại thời điểm cutover + checksum) —
  `infrastructure/10-data-migration-vps-supabase.md`.
- [ ] Rollback 2 mức sẵn sàng (§4).

## 3. Thứ tự bước

1. Đổi `api.closy` → Cloud Run (đổi Nginx server block hoặc DNS), giữ `api-v2` nguyên.
   Với Nginx compose: chỉnh `NGINX_MODE` phù hợp rồi
   ```bash
   docker compose --env-file .env.production -f deployments/docker-compose.prod.yml \
     up -d --force-recreate nginx
   ```
2. Xác minh dữ liệu + kiểm thử đầy đủ (host × route + SSE + auth + upload):
   ```bash
   curl -sS -o /dev/null -w '%{http_code}\n' https://api.closy.hycat.online/api/v1/health
   curl -N -m 35 https://api.closy.hycat.online/<sse-path>
   ```
3. Giám sát 24–48h; nếu gate vỡ → rollback ngay (§4).

## 4. Rollback 2 mức

1. **Cloud Run revision-level** (tức thời, per-service): shift traffic về revision trước
   của service lỗi:
   ```bash
   gcloud run services update-traffic smart-wardrobe-<svc> \
     --region asia-southeast1 --to-revisions <PREV_REVISION>=100
   ```
2. **Ingress-level** (toàn hệ): đổi Nginx `api.closy` về monolith (`http://backend:8080`,
   image `monolith-legacy`) — đặt `NGINX_MODE=rollback` rồi recreate nginx.

## 5. Drill (bắt buộc trước)

Drill đầy đủ trên môi trường Cloud **trước**: đổi → xác minh → rollback về monolith;
đo thời gian; ghi lại bước lệch (T117). Chỉ đổi Nginx/DNS + đồng bộ dữ liệu — **không sửa
mã nghiệp vụ**.
