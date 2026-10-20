# Deferred — 031 lịch sử gói theo khoảng hiệu lực

Các mục dưới đây nằm ngoài phạm vi feature và **không** phải task chưa làm của 031.

## D-01 — Fixture campaign trong test DB dùng chung không tự dọn

- **Vấn đề**: chạy toàn bộ package
  `services/account/internal/modules/subscription/application/usecase/subscription`
  trên một database đã dùng cho lần chạy trước thì các test campaign fail ở precondition
  "registry isolation": registry còn các row `verify-026-*` (và campaign seed `launch-2026-10`).
  FR-040 chọn campaign có `starts_at` lớn nhất nên row lạ thắng grant và assertion kiểm tra sai campaign.
- **Vì sao ngoài phạm vi**: đây là lỗi cô lập fixture của harness campaign (feature 025/026), không liên
  quan tới bảng/khoảng hiệu lực của 031. Trên database mới tinh, toàn bộ package chạy xanh; gate của 031
  chỉ chạy các package bị tính năng chạm.
- **Ảnh hưởng tới task đã tick**: không. T027–T031 chạy xanh trên DB mới; kết quả này không bị mục này
  làm mất hiệu lực.
- **Gỡ bằng cách nào**: cho mỗi suite campaign dùng tiền tố campaign riêng và dọn theo tiền tố đó
  (`DELETE FROM subscription.campaigns WHERE code LIKE 'verify-026-...%'`) trước khi seed, hoặc tách
  database cho nhóm test campaign. Không cần thay đổi schema.
