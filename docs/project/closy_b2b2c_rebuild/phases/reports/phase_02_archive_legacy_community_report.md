# Báo cáo Phase 02 - Archive legacy community/resale runtime

## File đã thay đổi

- `internal/api/routes/router.go`
- `internal/api/routes/provider.go`
- `internal/api/routes/admin/router.go`
- `internal/bootstrap/app.go`
- `internal/di/wire.go`
- `internal/di/wire_gen.go`
- `migrations/20260628072715_archive_legacy_community_resale_tables.sql`
- `docs/project/closy_b2b2c_rebuild/phases/reports/phase_02_archive_legacy_community_report.md`

## Migration đã thêm

- `migrations/20260628072715_archive_legacy_community_resale_tables.sql`

Migration này chỉ được tạo và viết nội dung, chưa chạy `migration-up`.

Up migration drop các bảng legacy theo thứ tự FK an toàn:

```text
likes
comments
transfer_requests
post_media
post_score_snapshots
post_items
posts
```

Down migration tạo lại các bảng/index legacy baseline để rollback schema.

## API đã thêm/thay đổi

- Không thêm API mới.
- Đã gỡ community/resale khỏi runtime router:
  - Không còn register `CommunityRouter.Init(api)` trong `internal/api/routes/router.go`.
  - Không còn `CommunityRouter` trong `AppRouter`.
  - Không còn `community.NewRouter` trong `RouterSet`.
- Đã gỡ admin moderation routes cho community/resale khỏi runtime:
  - `/api/v1/admin/posts`
  - `/api/v1/admin/comments`
  - `/api/v1/admin/post-items`
- Đã gỡ post hotness worker khỏi runtime workers.

## Test đã thêm/cập nhật

- Không thêm test code mới.
- Đã chạy `go test ./...`: pass.
- Đã chạy `make build`: pass.
- Đã chạy `make wire` để cập nhật DI generated file.
- Không chạy `make migration-up`.

## Ghi chú tương thích ngược

- Không xóa source code module `internal/modules/community`.
- Đã tạo migration drop các bảng legacy, nhưng chưa apply:
  - `posts`
  - `post_media`
  - `comments`
  - `likes`
  - `post_items`
  - `transfer_requests`
  - `post_score_snapshots`
- Core runtime không còn phụ thuộc compile vào community router/admin handler/post hotness worker.
- Dữ liệu cũ vẫn còn nguyên trong database hiện tại vì migration chưa chạy.

## Bước kiểm tra thủ công

- Audit dependency bằng `rg` cho `community`, `posts`, `post_items`, `comments`, `likes`, `transfer_requests`, `post_score_snapshots`, `transfers`.
- Xác nhận các route `/posts`, `/transfers` chỉ còn trong package community chưa được gọi runtime.
- Xác nhận `internal/api`, `internal/bootstrap`, `internal/di` không còn import `internal/modules/community` hoặc `internal/api/routes/community`.
- Chạy `make wire`.
- Chạy `go test ./...`.
- Chạy `make build`.
- Tạo migration bằng `make migration-create name=archive_legacy_community_resale_tables`.
- Kiểm tra thủ công nội dung SQL migration.
- Không chạy migration up theo yêu cầu.

## Giới hạn đã biết

- Migration drop bảng legacy đã có nhưng chưa được apply.
- Chưa xóa module `internal/modules/community`; module được giữ lại để tránh xóa code quá sớm.
- Swagger generated files hiện có thể vẫn còn mô tả endpoint community/resale vì annotation trong handler community chưa được cleanup. Runtime API đã gỡ route trước.
- Chưa chạy app smoke test thực tế bằng HTTP request.
