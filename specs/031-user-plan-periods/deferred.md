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

## Ghi nhận rà soát Phase 6 (2026-10-20)

- **Không có phát hiện ngoài phạm vi mới** trong lượt polish (T032–T036, T038–T040). D-01 vẫn là
  mục duy nhất; lượt này chỉ chạy test scoped theo tên (`TestIntegrationUserPlanPeriods`,
  `TestIntegrationPlanPeriod`, `TestPlanPeriodRecorder`, ...) nên lỗi cô lập fixture campaign của
  toàn package `usecase/subscription` không tái hiện.
- **Quan sát môi trường, không phải debt mã nguồn**: database tạm `verify_plan031` của lượt này chỉ
  được migrate account (031); các schema `aiexecutor`, `dashboard`, `community` không có mặt dù
  quickstart §1 có lệnh migrate commerce/ai. Phạm vi test scoped và đo hiệu năng của 031 không cần
  các schema đó. Khi chạy `make test-db-integration` toàn bộ (T037), cần DB được dựng đủ theo
  quickstart.
- **Đã sửa trong phạm vi, không đưa vào deferred**: inventory `docs/system-design/architecture.md`
  lệch so với DB thật trước 031 (thiếu `identity.user_activity_events` và 3 bảng campaign của 025,
  thiếu 1 FK `subscription.campaign_claims → identity.users`) — đã cập nhật đúng số thật trong T032.
- Không sửa `subscription.user_subscriptions` và không thêm job/endpoint ngoài phạm vi 031.
