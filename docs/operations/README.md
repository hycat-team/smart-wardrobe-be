# Runbook vận hành

Các file dưới đây, mỗi file đọc xuôi từ trên xuống, không nhảy qua lại:

| File | Dùng khi | Điểm vào |
|---|---|---|
| `deployment-pipeline.md` | Hiểu + vận hành Docker/CI/CD phạm vi VPS (dev mới đọc đầu tiên) | Mục 0 |
| `vps-runbook.md` | VPS monolith → 3 service (GĐ1) | Bước 0 |
| `grant-restore-runbook.md` | Tài khoản không có grant tính phí AI (bộ đếm `tier1_default_policy` tăng) | Bước 1 |
| `shared-reference.md` | Tra cứu chung (KHÔNG phải runbook, không có thứ tự bước) | đọc khi runbook trỏ tới |

`infrastructure/archive/` — tài liệu thời Cloud Run đã lỗi thời. Trục AWS box (GĐ2) đã
nghỉ hẳn: runbook + tooling nằm ở `../archive/aws-box/` (ADR 017), không đọc khi vận hành.

## Một trục duy nhất

Backend chỉ còn VPS. `NGINX_MODE` trên VPS quyết định những gì `api.closy.hycat.online`
phục vụ:

| Mode | `api.closy.hycat.online` | Monolith |
|---|---|---|
| `services` | 3 service VPS | **dừng** |
| `services-redirect` | 3 service + `api-v2`→301 | dừng (grace, xong quay về `services` + xoá DNS) |
| `rollback` | monolith | thủ công (ADR 012; không còn workflow tự động) |

`api-v2.closy.hycat.online` không còn trỏ tới box AWS: DNS của domain này chỉ còn ý nghĩa
trong giai đoạn grace redirect, và biến `API_V2_DOMAIN` vẫn giữ trong compose VPS vì lý do đó.
Khi hết grace, xoá DNS `api-v2` và trả mode về `services`.

## Quy ước registry & tag

- Image service → **GHCR** `ghcr.io/<owner>/<svc>:<git-sha>` (+ alias tên
  release tag); `:latest` chỉ trong CI; image private KHÔNG public.
- Image rollback `monolith-legacy` (**không** build/push tự động từ ADR 012): chỉ
  build thủ công từ tag frozen khi thật sự cần — cấm dời tag.

## Nguyên tắc

- Mọi runbook ghi rõ điều kiện tiên quyết, thứ tự bước, kiểm thử và rollback.
- Chuyển môi trường chỉ đổi config + DNS, **không sửa code**.
- VPS chỉ dùng infra local; không có môi trường thứ hai dùng managed infra.
