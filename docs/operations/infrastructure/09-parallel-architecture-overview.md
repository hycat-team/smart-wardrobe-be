# 09 — Kiến trúc song song (monolith + Cloud Run)

## 1. Trạng thái song song

| Host | Đích | Nguồn ghi |
|---|---|---|
| `api.closy...` | Monolith frozen (`monolith-legacy`) trên VPS | **Chính** (người dùng thật) |
| `api-v2.closy...` | VPS Nginx → 3 Cloud Run | Test nội bộ (dữ liệu dump 1 lần) |

## 2. Dựng giai đoạn song song (theo thứ tự)

1. Deploy Cloud Run: `00` §6 (hoặc `05-compute-cloudrun.md`).
2. Cấu hình VPS ingress mode `parallel`: `06-vps-nginx-ingress-cloudrun.md`.
3. Thêm DNS `api-v2`: `11-dns-vercel-ci-cd.md`.
4. Dump dữ liệu VPS → Supabase (một lần, phục vụ test `api-v2`):
   `10-data-migration-vps-supabase.md`.
5. Chạy gate ở §4 trước khi quyết định cutover.

## 3. Nguyên tắc (US3, SC-003/SC-015)

- Hai hệ chạy **độc lập**: `api-v2` lỗi không ảnh hưởng `api.closy`; tắt/bật lại `api-v2` an toàn.
- Dữ liệu `api-v2` (Supabase) lệch dần so với VPS → **dump lại VPS→Supabase tại thời điểm
  merge/cutover**, xác nhận không mất ghi trong cửa sổ chuyển.
- Drill trên môi trường Cloud dùng **schema tạm rồi xoá**, không đụng schema chính (FR-027).
- Cutover `api.closy` **không** nằm trong phase này — là thao tác thủ công có runbook riêng
  (FR-041), timing do owner quyết định khi gate đo được đạt.

## 5. Biến thể parallel-v2 (api.closy→VPS microservices, api-v2→Cloud Run verify)

- Routing: `api.closy` → 3 service compose trên VPS (template `parallel-v2/vps-3service`);
  `api-v2` → Cloud Run (template `parallel-v2/api-v2`, cần token-helper).
- Backend cũ (monolith) **dừng hẳn**; verify `docker ps` trống backend.
- DB: 2 DB độc lập (VPS local live, Supabase verify); đồng bộ dump VPS→Supabase
  (`10-data-migration-vps-supabase.md`) trước đợt verify `api-v2`.
- Gate: như §4 nhưng `api.closy` đo trên VPS (không còn monolith để so sánh giữ nguyên).

## 4. Gate song song (T082) — kiểm tra từng mục

```bash
# a) api.closy giữ nguyên
curl -sS -o /dev/null -w '%{http_code}\n' https://api.closy.hycat.online/api/v1/health

# b) api-v2 route đủ 3 service
for p in account commerce ai; do
  curl -sS -o /dev/null -w "$p %{http_code}\n" "https://api-v2.closy.hycat.online/api/v1/<path-$p>"
done

# c) Không đi qua Cloudflare
curl -sSI https://api-v2.closy.hycat.online | grep -i cf-ray || echo "0 Cloudflare OK"
```

- [ ] `api-v2` chạy ổn qua VPS Nginx (host × route + SSE + auth + upload) (SC-014).
- [ ] `api.closy` **0 thay đổi routing**; monolith phục vụ bình thường suốt quá trình (SC-015).
- [ ] Kết nối Cloud Run thiếu ID token bị từ chối 403 (SC-013).
- [ ] Ghi mốc thời gian + checksum dữ liệu.
