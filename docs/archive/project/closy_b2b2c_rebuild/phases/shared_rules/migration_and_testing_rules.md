# Migration and Testing Rules

## Migration safety

- Luôn inspect schema hiện tại trước khi viết migration.
- Không drop column/table ngay khi vừa tạo schema mới nếu code vẫn còn đọc field cũ.
- Ưu tiên migration 2 bước cho refactor lớn:
  - Step A: add new table/column + backfill + dual-read/compatibility.
  - Step B: switch code.
  - Step C: cleanup/drop sau khi verified.
- Không mất dữ liệu cũ.
- Với bảng lớn, backfill phải idempotent.
- Nếu dùng Goose embedded migration hiện tại, migration phải tương thích với cơ chế auto-run hiện có.

## Transaction safety

Các usecase sau bắt buộc chạy trong DB transaction:

```text
- Create brand customer + loyalty account + earn points
- Redeem benefit
- Adjust loyalty points
- Save outfit with outfit_items
- Create brand item with fashion_item
- Migrate/backfill fashion_items nếu chạy bằng script application-level
```

## Idempotency

Các request có rủi ro double submit cần idempotency:

```text
- Loyalty points grant/adjust
- Benefit redeem
- Offline loyalty acquisition by phone
```

MVP có thể dùng `idempotency_key` nullable trong `loyalty_point_transactions`.

Khuyến nghị constraint:

```sql
CREATE UNIQUE INDEX ... ON loyalty_point_transactions(brand_id, idempotency_key)
WHERE idempotency_key IS NOT NULL;
```

Nếu không dùng idempotency key, phải dùng `reference_type + reference_id` có unique logic tương đương.

## Test levels

Mỗi phase phải có ít nhất:

```text
- Unit test cho domain/usecase rule quan trọng
- Repository test hoặc integration test cho migration/query phức tạp
- Handler/API test cho endpoint mới hoặc endpoint thay đổi
- Manual verification checklist
```

## Compile gate

Sau mỗi phase:

```bash
go test ./...
go vet ./...    # nếu repo đang dùng
make build      # nếu Makefile hiện có hỗ trợ
```

Nếu repo không có command tương ứng, agent phải dùng command build/test hiện có trong project.

## Naming

Không tự ý đổi public API cũ nếu không cần.

Không tự ý đổi enum numeric hiện có sang string nếu DB/code đang dùng numeric, trừ khi phase yêu cầu và có migration rõ.

Field trong tài liệu là target behavior. Nếu schema hiện tại có tên khác nhưng cùng ý nghĩa, preserve tên hiện tại hoặc ghi rõ lý do đổi.
