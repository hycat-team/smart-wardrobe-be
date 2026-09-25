# Quickstart Validation: Nhóm chức năng cộng đồng (Community Social)

**Feature**: 022-community-social | **Date**: 2026-09-25

Hướng dẫn kiểm chứng end-to-end. Không chứa code triển khai (thuộc `tasks.md`).

## Prerequisites

- Docker/Postgres + Redis + RabbitMQ đang chạy (theo `docs/development/local-setup.md`).
- Đã chạy migration Commerce: `make migrate-up svc=commerce` (hoặc `migrate_on_start`).
- `ELASTICSEARCH_ENABLED=false` (mặc định) để kiểm chứng đường dự phòng DB.
- Có 2 user hợp lệ (A, B) và một outfit thuộc A.
- `make dev` (hoặc chạy Commerce service).

## Migration & build gates

```bash
make migrate-create svc=commerce name=create_community_schema
# điền DDL, rồi:
make migrate-up svc=commerce
make wire svc=commerce && make buf-gen
make lint && make test-race
```

Kỳ vọng: schema `community` xuất hiện cùng 6 bảng; `make lint`/`test-race` xanh.

## Kịch bản 1 — Đăng bài & feed

1. A lấy chữ ký: `GET /api/v1/posts/upload-signature?resourceType=image` → 200 có `signature`,`folder`.
2. A tạo outfit post: `POST /api/v1/posts` với `{postType:"outfit", content:"...", outfitId:"<A's outfit>"}`.
   - Kỳ vọng: 201, `postType=outfit`, `outfit.id` đúng, `likeCount=0`.
3. A tạo media post (video): `{postType:"media", content:"...", media:[{mediaType:"video",...}]}` → 201.
4. Tạo media post rỗng (`content:""`, không media) → 400.
5. `GET /api/v1/posts?type=explore` (không token) → 200, chứa 2 bài, mới nhất/hot theo `sort`.
6. `GET /api/v1/posts?type=following` (không token) → 401.

## Kịch bản 2 — Cascade outfit

1. A tạo outfit post gắn outfit X.
2. A xóa outfit X: `DELETE /api/v1/outfits/{X}`.
3. Chờ ≤ 1 phút; `GET /api/v1/posts/{postPublicId}` → 404 (bài đã `deleted`).
4. Kiểm tra worker/outbox chạy không lỗi; message vào DLQ nếu giả lập lỗi.

## Kịch bản 3 — Like & comment 2 cấp

1. B like bài của A: `PUT /posts/{id}/like {isLiked:true}` → 200; gọi lặp lại → không lỗi, `likeCount` không đổi.
2. Bình luận gốc: `POST /posts/{id}/comments {content:"c0"}` → 201.
3. B reply vào c0: `{content:"c1", parentCommentId:"<c0>"}` → 201, `parentCommentId=c0`.
4. B reply vào c1: `{content:"c2", parentCommentId:"<c1>"}` → 201, `parentCommentId=c0` (tự gắn lên gốc, không tạo cấp 3).
5. `GET /posts/{id}/comments` → gốc mới nhất trước; `replyCount` của c0 = 2.
6. A xóa c0 khi còn reply → `GET` trả c0 `isDeleted=true`; xóa c0 không còn reply → ẩn hẳn.

## Kịch bản 4 — Follow & profile

1. A follow B: `PUT /users/B/follow {isFollowing:true}` → 200; lặp lại → idempotent.
2. `GET /users/B` → `stats.followerCount ≥ 1`, `isFollowing=true` (khi xem bởi A).
3. A tự follow: `PUT /users/A/follow {isFollowing:true}` → 400.
4. `GET /users/B/follows` (không `type`) → danh sách có `relation` cho cả following/follower.
5. `GET /users/B/follows?type=followers&q=<từ khóa>` → lọc đúng (nhãn `type=following|followers`).
6. `GET /users/B/posts` → chỉ bài `published` của B, sắp `latest`.

## Kịch bản 4b — Ẩn bài theo trạng thái user (FR-035)

1. Admin khóa user A (đổi status) → Account publish `identity.user.status.changed`.
2. Chờ ≤ 1 phút; `GET /posts?username=A` → bài của A không còn hiển thị; bản ghi vẫn còn (`hidden_reason=author_disabled`).
3. Admin mở lại user A → bài `author_disabled` trở lại `published`.
4. Nếu bài đang `hidden` do admin (`hidden_reason=admin`), event mở user KHÔNG đưa về published.

## Kịch bản 5 — Search

1. `GET /search?q=<username B>` → `users.items` chứa B; `posts` theo hot.
2. `GET /search?q=<text ở 200 ký tự đầu của bài>` → `posts.items` chứa bài.
3. `GET /search?q=<text xuất hiện sau ký tự 200>` → không chứa bài đó.
4. `GET /search?q=%20` (rỗng sau trim) → cả hai mảng rỗng, 200.
5. Không truyền `type` → trả cả hai; `type=users` → `posts.items` rỗng.

## Kịch bản 6 — Admin moderation

1. Admin: `GET /admin/posts?status=published` → danh sách.
2. Admin ẩn 1 bài → người thường không thấy trong feed/search; admin vẫn thấy.
3. Admin restore → hiển thị lại.
4. Tác giả tự xóa bài → admin restore → 400.
5. Admin xóa comment → comment không còn hiển thị; restore → hiển thị lại.
6. Kiểm tra audit log được ghi (topic `admin.audit.log`).

## Kịch bản 7 — Hot ranking

1. Tạo bài mới ít tương tác (P_new) và bài cũ ~vài giờ trước nhiều like/comment (P_hot).
2. `GET /posts?sort=hot` → P_hot xếp trên P_new.
3. `GET /posts?sort=latest` → P_new xếp trên.
4. Bài ngoài cửa sổ 30 ngày không xuất hiện ở `hot` nhưng xuất hiện ở `latest`.

## Tiêu chí đạt

Đối chiếu `spec.md §Success Criteria` (SC-001…SC-011), đặc biệt SC-002 (cascade ≤ 1 phút),
SC-003 (0 trùng like/follow), SC-004 (0 comment cấp 3), SC-005 (search 200 ký tự),
SC-006 (hot ≠ latest), SC-010 (tài liệu FE đầy đủ).
