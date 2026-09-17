# Runbook cutover thủ công `api.closy` → Cloud Run (FR-041)

**Chỉ thực thi khi owner quyết định** `api-v2` đủ ổn định. Không tự động hoá.

## Gate đo được (cố định trong spec)

- Smoke test **100% pass**.
- Error-rate **≤ 1%**, p95 **≤ 500ms**, SSE **không rớt**, ổn định **7 ngày liên tiếp**.
- Runbook này chỉ chi tiết hoá, không nới ngưỡng.

## Điều kiện tiên quyết

- `api-v2` ổn định đủ 7 ngày (bằng chứng metrics).
- Dữ liệu đã đồng bộ (dump VPS → Supabase tại thời điểm cutover + checksum).
- Rollback sẵn sàng (2 mức dưới).

## Thứ tự bước

1. Đổi Nginx `api.closy` → Cloud Run (hoặc DNS), giữ `api-v2` nguyên.
2. Xác minh dữ liệu + kiểm thử đầy đủ (host × route + SSE + auth + upload).
3. Giám sát 24–48h; nếu gate vỡ → rollback ngay.

## Rollback 2 mức

1. **Cloud Run revision-level** (tức thời, per-service): shift traffic về revision
   trước của service lỗi.
2. **Ingress-level** (toàn hệ): đổi Nginx `api.closy` về monolith
   (`http://backend:8080`, image `monolith-legacy`).

## Drill

Drill đầy đủ trên môi trường Cloud **trước** (đổi → xác minh → rollback về
monolith); đo thời gian; ghi lại bước lệch (T117). Chỉ đổi Nginx/DNS + đồng bộ
dữ liệu — **không sửa mã nghiệp vụ**.
