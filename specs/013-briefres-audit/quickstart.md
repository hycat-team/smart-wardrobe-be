# Quickstart — Validation Guide Feature 013-briefres-audit

**Feature**: 013-briefres-audit | **Ngày**: 2026-08-22

> Hướng dẫn kiểm chứng kết quả audit (giai đoạn plan) và verify sau implement. Feature này là audit/analysis → giai đoạn trước mắt chỉ validate tính đầy đủ của báo cáo; sau khi refactor thì chạy test + đối chiếu API.

## 1. Validate giai đoạn AUDIT (trước khi implement)

### Tiền đề

- 9 file report trong specs/013-briefres-audit/reports/ (1/module)
- Spec: specs/013-briefres-audit/spec.md

### Checklist đối chiếu (thủ công)

1. Mỗi module trong internal/modules/ đều có 1 file report: admin, aiexecutor, brand, dashboard, fashion, identity, notification, subscription, wardrobe.
2. Mỗi report có các section: Tổng quan, Danh sách struct, Overdelivery, Underdelivery, Đề xuất BriefRes, Ưu tiên, API bị ảnh hưởng.
3. Struct overdelivery: có fields giữ vs bỏ cụ thể.
4. Struct underdelivery: có đề xuất BriefRes nested.
5. Mỗi đề xuất có mức ưu tiên (cao/trung bình/thấp) + lý do.
6. Admin/portal endpoints ghi nhận "không refactor".
7. Trùng khớp với
   esearch.md (12 BriefRes đề xuất).

### Lệnh kiểm tra tính thống nhất

`bash

# Liệt kê module có application/dto/ (so với reports/)

Get-ChildItem internal/modules -Directory | ForEach-Object { .Name }

# Đếm số report

Get-ChildItem specs/013-briefres-audit/reports/\*.md | Measure-Object
`

## 2. Validate sau IMPLEMENT (chạy khi refactor xong)

### Build & test

`bash
make check    # fmt + go.mod tidy + wire + swagger
make test     # toàn bộ unit tests
`

### Kiểm tra từng BriefRes mới được wire/import đúng

- go build ./internal/... không lỗi
- Mapper mới (MapBrandBrief, MapUserBrief, MapWardrobeItemBrief, ...) nằm trong application/mapper/ (không nằm trong usecase)

### Đối chiếu API Contract

- Với mỗi endpoint trong contracts/affected-api-contracts.md, gọi thử (hoặc kiểm tra swagger) đảm bảo response đúng struct đề xuất.
- Kiểm tra field đã bỏ không còn xuất hiện (vd: idempotencyKey không còn ở transactions user; phoneE164 không còn ở join-loyalty).

## 3. Tạo frontend-note.md (CUỐI giai đoạn implement + verify)

- Sau khi tất cả API thay đổi hoàn tất và make test/make check xanh:
    - Tạo frontend-note.md (tại specs/013-briefres-audit/ hoặc nơi dev quyết định).
    - Nội dung: danh sách API bị thay đổi, field cũ → field mới, hướng dẫn frontend sửa (tham chiếu contracts/affected-api-contracts.md).
- Gửi cho team frontend trước khi deploy.

## 4. Expected outcomes

| Giai đoạn     | Kết quả mong đợi                                                              |
| ------------- | ----------------------------------------------------------------------------- |
| Audit         | 9 reports đầy đủ, khớp research.md, không có [NEEDS CLARIFICATION] chưa xử lý |
| Implement     | make check + make test pass; swagger phản ánh struct mới                      |
| frontend-note | Liệt kê đầy đủ API thay đổi + field mapping + hướng dẫn sửa                   |
