# GCP trial teardown (một bước trong repatriation — FR-031/US6)

> **ĐÃ LỖI THỜI (GĐ2).** Không có GCP trial nào để teardown (Cloud Run chưa từng tồn tại).
> Giữ lại tham khảo thứ tự dọn dẹp. Tắt box ngoài AWS + managed theo
> `../../aws-runbook.md` (Bước 9 chu kỳ / Bước 10 teardown cuối).

**Không tách rời sau demo.** Teardown chạy như một bước của repatriation khi hệ thống
đã về VPS ổn định.

> **Khi nào dùng**: bước cuối của `../../vps-runbook.md` (repatriate), sau khi hạ tầng mới (VPS) đã
> chạy ổn định ≥ 7 ngày.
> **Điều kiện**: repatriation đã xác nhận; `api.closy`/`api-v2` đã trỏ về hạ tầng mới.
> **Không dùng khi**: Cloud Run (managed) còn phục vụ traffic — teardown khi đó sẽ gây downtime.

## 1. Điều kiện tiên quyết

- [ ] Repatriation đã xác nhận (managed KHÔNG còn phục vụ).
- [ ] `api.closy` + `api-v2` đã trỏ về hạ tầng mới ổn định.

## 2. Thứ tự xoá

1. **Cloud Run**: xoá 6 units (3 HTTP + 3 worker) — scale về 0 trước rồi xoá:
   ```bash
   gcloud run services list --region asia-southeast1
   for svc in account commerce ai; do
     gcloud run services delete "closy-$svc" --region asia-southeast1 --quiet
     gcloud run services delete "closy-$svc-worker" --region asia-southeast1 --quiet
   done
   ```
2. **Artifacts**: xoá image Cloud Run trong Artifact Registry (giữ image `monolith-legacy`
   trên GHCR cho rollback):
   ```bash
   gcloud artifacts docker images list \
     asia-southeast1-docker.pkg.dev/<PROJECT>/closy --include-tags
   # xoá theo từng digest hoặc xoá repo:
   gcloud artifacts repositories delete closy --location=asia-southeast1 --quiet
   ```
3. **Alerts/monitors**: xoá alert policy + dashboard Cloud Monitoring của trial
   (giữ runbook để tái lập nếu cần).

## 3. Verify bill về 0 (SC-010)

```bash
gcloud run services list --region asia-southeast1          # rỗng
gcloud artifacts repositories list --location=asia-southeast1
```
- [ ] `gcloud` không còn service/artifact trial nào.
- [ ] Đối chiếu billing trial: **0 chi phí mới** sau teardown.
- [ ] Ghi bằng chứng (screenshot/export billing) vào checklist funding clock.

## 4. Lưu ý

- Teardown SAU KHI repatriation xác nhận (không teardown khi managed còn phục vụ).
- Funded items (Supabase/CloudAMQP/Redis) tắt theo lịch riêng **trước ngày renew** (T107).
