# Runbook vận hành

3 file, mỗi file đọc xuôi từ trên xuống, không nhảy qua lại:

| File | Dùng khi | Điểm vào |
|---|---|---|
| `vps-runbook.md` | VPS monolith → 3 service (GĐ1) | Bước 0 |
| `aws-runbook.md` | Box ngoài AWS + managed (GĐ2): đăng ký → deploy → chu kỳ xóa/tạo → teardown $0 | Bước 0 |
| `shared-reference.md` | Tra cứu chung cả 2 trục (KHÔNG phải runbook, không có thứ tự bước) | đọc khi runbook trỏ tới |

`infrastructure/archive/` — tài liệu thời Cloud Run đã lỗi thời (Cloud Run chưa từng
tồn tại, không đọc khi vận hành).

## Hai trục độc lập

| Trục VPS (`NGINX_MODE`) | `api.closy.hycat.online` | Monolith |
|---|---|---|
| `services` | 3 service VPS | **dừng** |
| `services-redirect` | 3 service + `api-v2`→301 | dừng (grace sau khi tắt box) |
| `rollback` | monolith | chạy (khẩn cấp, `deploy-production.yml`) |

| Trục box ngoài | `api-v2.closy.hycat.online` |
|---|---|
| tắt | chưa có DNS (GĐ1) |
| bật | box AWS → managed (chu kỳ phù du) |
| cắt/teardown | về VPS → xóa DNS → bill \$0 |

**Luồng tổng quát**: `vps-runbook` (VPS → `services`) → `aws-runbook` (box + DNS
`api-v2`) → chu kỳ xóa/tạo → teardown cuối giữ VPS.

## Quy ước registry & tag

- Image service (VPS + AWS) → **GHCR** `ghcr.io/<owner>/<svc>:<git-sha>` (+ alias tên
  release tag); `:latest` chỉ trong CI; image private KHÔNG public.
- Image rollback `monolith-legacy` → **GHCR** (build 1 lần từ tag frozen — cấm dời tag).

## Nguyên tắc

- Mọi runbook ghi rõ điều kiện tiên quyết, thứ tự bước, kiểm thử và rollback.
- Chuyển môi trường chỉ đổi config + DNS, **không sửa code**.
- VPS chỉ dùng infra local; box ngoài chỉ dùng managed (luật isolation).
