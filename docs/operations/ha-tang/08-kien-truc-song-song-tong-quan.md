# 08 — Kiến trúc song song tổng quan (monolith + Cloud Run)

## Trạng thái song song

| Host | Đích | Nguồn ghi |
|------|------|-----------|
| `api.closy...` | Monolith frozen (`monolith-legacy`) trên VPS | **Chính** (người dùng thật) |
| `api-v2.closy...` | VPS Nginx → 3 Cloud Run | Test nội bộ (dữ liệu dump 1 lần) |

## Nguyên tắc (US3, SC-003/SC-015)

- Hai hệ chạy **độc lập**: `api-v2` lỗi không ảnh hưởng `api.closy`; tắt/bật lại `api-v2` an toàn.
- Dữ liệu `api-v2` (Supabase) lệch dần so với VPS → **dump lại VPS→Supabase tại
  thời điểm merge/cutover**, xác nhận không mất ghi trong cửa sổ chuyển.
- Drill runbook trên môi trường Cloud bằng **schema tạm rồi xoá**, không đụng schema chính (FR-027).
- Cutover `api.closy` **không** nằm trong phase này — là thao tác thủ công có
  runbook riêng (FR-041), timing do owner quyết định khi gate đo được đạt.

## Gate song song (T082)

- [ ] `api-v2` chạy ổn qua VPS Nginx (host × route + SSE + auth + upload).
- [ ] `api.closy` **0 thay đổi routing**; monolith phục vụ bình thường suốt quá trình.
- [ ] Ghi mốc thời gian + checksum dữ liệu.
