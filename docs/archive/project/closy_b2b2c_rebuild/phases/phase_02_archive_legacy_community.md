# Phase 02 - Archive Legacy Community and Resale Scope

## Mục tiêu

Loại khỏi runtime MVP các nghiệp vụ community/resale cũ vì Closy chuyển sang B2B2C loyalty/co-creation. Phase này phải làm an toàn, tránh drop dữ liệu trước khi code không còn tham chiếu.

## Bảng legacy cần xử lý

Theo schema/report cũ, các bảng community/resale không còn thuộc MVP:

```text
posts
post_media
comments
likes
post_items
transfer_requests
post_score_snapshots
```

Nếu repo có thêm bảng community khác, agent phải liệt kê và xin duyệt trước khi drop.

## Không làm trong phase này

```text
- Không xóa module nếu module khác vẫn import.
- Không drop table khi chưa có migration backup/rollback note.
- Không tạo campaign thay thế.
- Không tạo social feed mới.
```

## Cách triển khai an toàn

### Step 1: Compile dependency audit

Search toàn repo:

```text
posts
comments
likes
transfer_requests
community
resale
post_items
```

Ghi lại:

```text
- entity files
- repositories
- usecases
- handlers/routes
- background jobs
- swagger docs
- tests
```

### Step 2: Remove routes from runtime

Nếu có route community/resale, disable route registration trước.

Không cần xóa code ngay nếu xóa làm vỡ compile. Có thể archive bằng cách:

```text
- remove route registration
- remove from swagger tags nếu cần
- mark package deprecated bằng comment
```

### Step 3: Remove cross-module references

Nếu wardrobe/outfit đang tham chiếu post/community, tách ra.

Ví dụ:

```text
outfit should not require post
wardrobe item should not require transfer request
```

### Step 4: Migration drop tables

Chỉ tạo drop migration khi:

```text
- code không còn query tables đó
- tests pass
- đã có backup policy hoặc migration note
```

Migration nên dùng:

```sql
DROP TABLE IF EXISTS post_score_snapshots;
DROP TABLE IF EXISTS post_items;
DROP TABLE IF EXISTS transfer_requests;
DROP TABLE IF EXISTS likes;
DROP TABLE IF EXISTS comments;
DROP TABLE IF EXISTS post_media;
DROP TABLE IF EXISTS posts;
```

Thứ tự phải tôn trọng FK thực tế. Nếu FK khác, chỉnh thứ tự theo schema hiện tại.

## Tests

- `go test ./...` pass sau khi remove routes/imports.
- App start không register community/resale endpoints.
- Existing core APIs auth/wardrobe/subscription vẫn pass.
- Migration up/down nếu repo yêu cầu down migration.

## Acceptance checklist

- [ ] Không còn runtime route community/resale trong MVP.
- [ ] Không còn compile dependency từ core module tới community/resale.
- [ ] Drop migration có thứ tự FK đúng.
- [ ] Không động tới brand campaign trong phase này.

## Lỗi cần tránh

- Xóa folder làm import cycle/compile fail.
- Drop table trước khi xóa code query.
- Thay community bằng campaign trong cùng phase.
