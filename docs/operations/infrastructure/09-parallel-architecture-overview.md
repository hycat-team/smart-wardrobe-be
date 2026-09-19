# 09 — Kiến trúc 2 trục độc lập (VPS + box ngoài)

VPS và box ngoài (GĐ2: AWS) là **2 trục độc lập** — bật/tắt một bên không đụng bên còn lại.
DNS là điểm phối hợp duy nhất.

## 1. Trục 1 — VPS phục vụ gì (`NGINX_MODE` trên VPS, chỉ quyết chuyện VPS)

| Mode | `api.closy.hycat.online` | Monolith | Dùng khi |
|---|---|---|---|
| `services` | 3 service compose local | **dừng** | trạng thái ổn định (bất kể box ngoài có tồn tại hay không) |
| `services-redirect` | 3 service compose local + `api-v2`→301 về api.closy | dừng | grace 14–30 ngày sau khi tắt box ngoài, xong quay về `services` + xóa DNS api-v2 |
| `rollback` | monolith (`backend`) | chạy | khẩn cấp (`deploy-production.yml`) |

## 2. Trục 2 — Box ngoài bật hay tắt (độc lập 100% với trục 1)

- Bật (GĐ2, runbook `15-aws-compute.md`): provision box → `deploy-aws.yml` → smoke qua IP →
  trỏ DNS `api-v2.closy.hycat.online` sang IP box. VPS không đổi gì.
- Tắt: trỏ DNS api-v2 về VPS (grace `services-redirect`) → reunify data
  (`12-repatriate-vps.md`) → xóa box + hủy managed trước kỳ renew → revoke secrets.
- VPS không biết box ngoài bật hay tắt; box ngoài không biết VPS đang ở mode nào.

## 3. Hạ tầng mỗi bên (luật isolation, `01` §4)

| App cần | VPS (`services/`) | Box ngoài (GĐ2) |
|---|---|---|
| Postgres | container `postgres` local | Supabase pooler |
| Redis | container `redis` local | Redis Cloud |
| RabbitMQ | container `rabbitmq` local | CloudAMQP |
| Elasticsearch | container ngoài compose, tắt tay được (app fallback DB) | không chạy (`ENABLED=false`) |
| Email/Media/AI | Brevo/Cloudinary/keys hiện tại | copy y nguyên, share quota |
| Nginx upstream | tên service compose | tên service compose (hardcode) |

## 4. Dữ liệu khi chạy song song (divergence có chủ đích)

Hai bên **không share datastore lúc runtime**: ghi ở api.closy nằm ở Postgres VPS,
ghi ở api-v2 nằm ở Supabase. Không có sync live 2 chiều; reunify duy nhất bằng
dump/restore tay (`10-data-migration-vps-supabase.md`, `12-repatriate-vps.md`) trong
cửa sổ bảo trì. Song song càng lâu, divergence càng lớn — cân nhắc khi quyết định cắt.

## 5. Gate VPS sau khi vào `services`

```bash
# a) api.closy qua 3 service VPS
curl -sS -o /dev/null -w '%{http_code}\n' https://api.closy.hycat.online/api/v1/health

# b) backend đã dừng
docker ps --filter name=backend --format '{{.Names}}'   # trống

# c) Không đi qua Cloudflare
curl -sSI https://api.closy.hycat.online | grep -i cf-ray || echo "0 Cloudflare OK"
```

- [ ] `api.closy` 200 qua 3 service VPS (route + SSE + auth + upload).
- [ ] Backend dừng; VPS infra local-only (guard pass).
- [ ] Ghi mốc thời gian + checksum dữ liệu.
