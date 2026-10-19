# Quickstart: Kiểm chứng bàn giao quyền sở hữu DDL (030)

**Spec**: [spec.md](spec.md) | **Data model**: [data-model.md](data-model.md) | **Contract**: [contracts/ownership-contract.md](contracts/ownership-contract.md)

Tài liệu này chỉ dẫn cách *chạy kiểm chứng*, không chứa code implement. Mọi lệnh đều chỉ-đọc hoặc trỏ vào database tạm `verify_d06_*`. **Cấm `down -v`, cấm xóa volume, cấm trỏ vào database production** (SC-006).

> Quy ước tên: `verify_d06_*` (gạch dưới) = tên database tạm; `verify-d06` (gạch ngang) = tên project Compose cách ly (nếu dùng stack riêng thay vì database tạm). Hai tên khác nhau, cùng quy tắc: chỉ tạo/xóa đối tượng tạm.

## Điều kiện tiên quyết

- Stack dev đang chạy (`docker ps` thấy `postgres-closy-dev` healthy).
- Mọi môi trường đã cùng 1 version Account mới nhất (điểm cắt, xem spec Assumptions).
- Lệnh ngừng bắn FR-011 có hiệu lực: không tạo migration mới chạm `aiexecutor.*`/`dashboard.*` trong lúc làm.

## V1 — Database trắng cách ly (chứng minh replay đứng được)

```powershell
# 1. Tạo database tạm (không đụng volume, không đụng DB thật)
docker exec -e PGPASSWORD=123456 postgres-closy-dev psql -U admin -d smart_wardrobe_db -c "CREATE DATABASE verify_d06_fresh;"
# 2. Chạy migration 3 service vào DB tạm (đúng thứ tự creator trước)
make migrate-up svc=account db=verify_d06_fresh
make migrate-up svc=ai db=verify_d06_fresh
make migrate-up svc=commerce db=verify_d06_fresh
# 3. Kỳ vọng: đủ 7 bảng aiexecutor + 3 bảng dashboard, constraint tên mới,
#    goose_db_version_ai/commerce có dòng baseline, goose_db_version đủ 35 dòng.
# 4. Dọn: chỉ xóa DB tạm
docker exec -e PGPASSWORD=123456 postgres-closy-dev psql -U admin -d smart_wardrobe_db -c "DROP DATABASE verify_d06_fresh;"
```

> Tên target `make migrate-up` ở trên là minh họa; lệnh thực tế theo Makefile tại thời điểm chạy — nguyên tắc bất biến là **target DB luôn là `verify_d06_*`**.

## V1b — Boot đồng thời trên DB trắng, lock serialize (chứng minh production trắng boot ổn định)

1. Tạo `verify_d06_race` như V1-bước 1 (trống hoàn toàn, chưa migrate gì).
2. Khởi động đồng thời 3 binary mới (account/commerce/ai) cùng trỏ vào `verify_d06_race` — mô phỏng production dựng trắng.
3. Kỳ vọng: lock phiên advisory serialize toàn bộ DDL — log không có lỗi bảng đã tồn tại, không deadlock, không xung đột schema; cả 3 service boot xanh; schema cuối đồng nhất với V1 (SC-002).
4. Dọn: `DROP DATABASE verify_d06_race` (chỉ DB tạm).

## V2 — Hội tụ từ bản sao production (chứng minh không mất dữ liệu)

```powershell
# 1. Restore backup mới nhất vào DB tạm verify_d06_rehearsal (không phải prod)
# 2. Boot binary mới trỏ vào DB tạm: Account Up ~0, AI/Commerce hội tụ 1 lần rồi no-op
# 3. So sánh schema V1 vs V2: bảng/cột/index/constraint đồng nhất 100% (SC-002)
# 4. Đếm dòng bảng nghiệp vụ trước/sau: bằng nhau tuyệt đối (SC-005)
# 5. DROP DATABASE verify_d06_rehearsal sau khi xong
```

## V3 — Gate đỏ/xanh (chứng minh FR-009)

1. Tạo branch thử, thêm 1 migration mẫu vi phạm mỗi luật (Account chạm `aiexecutor`, Commerce chạm `subscription`, AI chạm `public`) → `make migration-check` phải đỏ cả 3.
2. Xóa mẫu → xanh. Xóa branch thử.

## V4 — Smoke nghiệp vụ (chứng minh SC-005)

Trên DB rehearsal sau hội tụ: đăng ký → mua gói → dùng AI (grant ghi đúng) → xem dashboard admin. Sau đó chạy lại V2-bước 4 để khẳng định dữ liệu nguyên vẹn.

## V5 — Rehearsal trên DB dev hiện tại, cổng go/no-go giờ G (chứng minh SC-007)

Schema dev y chang production nên V5 xanh thì prod 99% xanh. DB dev là database sống (không phải tạm) nên tuân thủ nghiêm:

1. Backup DB dev trước (`pg_dump` đầy đủ, verify file backup đọc được) — đường về nếu abort.
2. Deploy 3 binary mới vào stack dev (trỏ đúng DB dev hiện tại): Account `Up` ~0, AI/Commerce hội tụ 1 lần rồi no-op.
3. Kiểm tra: schema đồng nhất với V1 (SC-002); đếm dòng nghiệp vụ trước/sau bằng nhau; 3 service boot xanh; smoke V4 xanh trên dev.
4. Kỳ vọng: pass 100% mới được đặt giờ G production. Nếu đỏ: abort, restore dev từ backup bước 1, không sửa tiếp trên DB sống.
5. Không dọn gì thêm (DB dev giữ nguyên để team dùng tiếp).

## Dọn dẹp và kiểm tra cuối

- `docker volume ls` — không volume nào của `closy-dev`/`closy-prod` bị xóa hay tạo mới ngoài `verify-d06` (nếu dùng stack cách ly thì chỉ xóa đúng volume `verify-d06_*`).
- Tìm kiếm repo: tên constraint cũ còn 0 kết quả ngoài dòng lịch sử tài liệu (SC-003); tài liệu/kịch bản: 0 lệnh xóa volume, 0 target production (SC-006).
