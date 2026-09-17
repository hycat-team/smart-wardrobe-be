# GCP trial teardown (một bước trong repatriation — FR-031/US6)

**Không tách rời sau demo.** Teardown chạy như một bước của repatriation khi
hệ thống đã về VPS ổn định.

## Thứ tự xoá

1. **Cloud Run**: xoá 6 units (3 HTTP + 3 worker) — scale về 0 trước rồi xoá.
2. **Artifacts**: xoá image Cloud Run trong Artifact Registry (giữ image
   `monolith-legacy` trên GHCR cho rollback).
3. **Alerts/monitors**: xoá alert policy + dashboard Cloud Monitoring của trial
   (giữ runbook để tái lập nếu cần).

## Verify bill về 0 (SC-010)

- [ ] `gcloud` không còn service/artifact trial nào (`run services list`, `artifacts docker images list`).
- [ ] Đối chiếu billing trial: **0 chi phí mới** sau teardown.
- [ ] Ghi bằng chứng (screenshot/export billing) vào checklist funding clock.

## Lưu ý

- Teardown SAU KHI repatriation xác nhận (không teardown khi managed còn phục vụ).
- Funded items (Supabase/CloudAMQP/Redis) tắt theo lịch riêng **trước ngày renew** (T107).
