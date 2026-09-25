# Phase 1 Data Model: Nhóm chức năng cộng đồng (Community Social)

**Feature**: 022-community-social | **Date**: 2026-09-25 | **Schema**: `community`

> Ghi chú: FK tới `identity.users` được phép xuyên schema (giữ chính sách user FK hiện hành).
> Không có FK xuyên module khác (đặc biệt KHÔNG FK từ `posts` sang `wardrobe.outfits`; liên kết
> outfit được enforce ở tầng ứng dụng + cascade qua sự kiện).

## Enums (domain/constants, lowercase)

| Enum | Giá trị | Ghi chú |
|---|---|---|
| `posttype` | `outfit`, `media` | loại bài đăng |
| `poststatus` | `published`, `hidden`, `deleted` | trạng thái hiển thị |
| `mediatype` | `image`, `video` | loại media |
| `feedtype` | `explore`, `following` | loại bảng tin (mặc định `explore`) |
| `feedsort` | `hot`, `latest` | sắp xếp (mặc định `hot`) |
| `followtype` | `following`, `followers` | lọc danh sách follow theo query `type` (`following` = người user đang theo dõi; `followers` = người đang theo dõi user). Trường `relation` trên mỗi mục trả `following`/`follower` |
| `hiddenreason` | `admin`, `author_disabled` | lý do bài bị ẩn (để restore đúng) |

## Entity: Post (`community.posts`)

| Field | Type | Constraints |
|---|---|---|
| `id` | uuid | PK, default `gen_random_uuid()` |
| `public_id` | varchar(32) | NOT NULL, UNIQUE — dùng trên URL (`postPublicID`) |
| `user_id` | uuid | NOT NULL, FK `identity.users(id)` |
| `post_type` | varchar(20) | NOT NULL, CHECK IN (`outfit`,`media`) |
| `title` | varchar(150) | NULL |
| `content` | text | NOT NULL DEFAULT `''` |
| `outfit_id` | uuid | NULL (không FK xuyên module) |
| `like_count` | int | NOT NULL DEFAULT 0 |
| `comment_count` | int | NOT NULL DEFAULT 0 |
| `status` | varchar(20) | NOT NULL DEFAULT `published`, CHECK IN (`published`,`hidden`,`deleted`) |
| `moderated_by` | uuid | NULL — admin id nếu bị ẩn/xóa bởi admin (phân biệt với tác giả tự xóa) |
| `hidden_reason` | varchar(30) | NULL, CHECK IN (`admin`,`author_disabled`) — lý do khi `status='hidden'` |
| `created_at` | timestamptz | NOT NULL DEFAULT `now()` |
| `updated_at` | timestamptz | NOT NULL DEFAULT `now()` |

**Constraints**:
- `chk_posts_outfit_ref`: `(post_type='outfit' AND outfit_id IS NOT NULL) OR (post_type='media' AND outfit_id IS NULL)`.
- `chk_posts_has_body`: enforce ở tầng ứng dụng — `content <> '' OR có ≥1 media` (FR-004).

**Indexes**:
- `idx_posts_published_created` ON `(created_at DESC, id DESC)` WHERE `status='published'`.
- `idx_posts_user_created` ON `(user_id, created_at DESC)` WHERE `status='published'`.
- `idx_posts_type_created` ON `(post_type, created_at DESC)` WHERE `status='published'`.
- `idx_posts_outfit` ON `(outfit_id)` WHERE `outfit_id IS NOT NULL AND status='published'`.
- `idx_posts_search` USING gin `(lower(left(content, 200)) gin_trgm_ops)` WHERE `status='published'`.

**State transitions** (`status`):
```
published --(author delete, moderated_by NULL)--> deleted  (terminal; không khôi phục, kể cả admin)
published --(admin hide, moderated_by=admin, hidden_reason=admin)--> hidden
hidden(admin)    --(admin restore)--------------> published
published/hidden --(admin delete, moderated_by=admin)--> deleted  (admin; khôi phục được)
deleted(moderated_by=admin) --(admin restore)----> published
published --(user disabled, hidden_reason=author_disabled)--> hidden
hidden(author_disabled) --(user active)----------> published
```
> Ưu tiên khi trùng: nếu bài đang `hidden` do admin (`hidden_reason=admin`) thì event user active
> KHÔNG đưa về `published`; chỉ xử lý `author_disabled`. Event user disabled chỉ áp cho bài `published`.

## Entity: PostMedia (`community.post_media`)

| Field | Type | Constraints |
|---|---|---|
| `id` | uuid | PK |
| `post_id` | uuid | NOT NULL, FK `community.posts(id)` ON DELETE CASCADE |
| `media_type` | varchar(10) | NOT NULL, CHECK IN (`image`,`video`) |
| `media_url` | varchar(500) | NOT NULL |
| `public_id` | varchar(255) | NULL |
| `sort_order` | smallint | NOT NULL DEFAULT 0 |

**Index**: `idx_post_media_post_sort` ON `(post_id, sort_order)`.

## Entity: PostLike (`community.post_likes`)

| Field | Type | Constraints |
|---|---|---|
| `post_id` | uuid | FK `community.posts(id)` ON DELETE CASCADE |
| `user_id` | uuid | FK `identity.users(id)` |
| `created_at` | timestamptz | NOT NULL DEFAULT `now()` |
| — | — | PK `(post_id, user_id)` — chống trùng (FR-007/FR-031) |

**Index**: `idx_post_likes_user` ON `(user_id, created_at DESC)`; `idx_post_likes_post` ON `(post_id, created_at DESC)`.

## Entity: PostComment (`community.post_comments`)

| Field | Type | Constraints |
|---|---|---|
| `id` | uuid | PK |
| `post_id` | uuid | NOT NULL, FK `community.posts(id)` ON DELETE CASCADE |
| `user_id` | uuid | NOT NULL, FK `identity.users(id)` |
| `parent_comment_id` | uuid | NULL, FK `community.post_comments(id)` ON DELETE CASCADE |
| `root_comment_id` | uuid | NULL — trỏ tới comment gốc (tiện truy vấn reply) |
| `content` | text | NOT NULL |
| `status` | varchar(20) | NOT NULL DEFAULT `published`, CHECK IN (`published`,`hidden`,`deleted`) |
| `moderated_by` | uuid | NULL — admin id nếu bị ẩn/xóa bởi admin |
| `created_at` | timestamptz | NOT NULL DEFAULT `now()` |
| `updated_at` | timestamptz | NOT NULL DEFAULT `now()` |

**Constraints**:
- `chk_comments_depth`: `parent_comment_id IS NULL OR (root_comment_id IS NOT NULL AND parent_comment_id <> id)`.
- Tầng ứng dụng: `parent_comment_id` LUÔN trỏ tới comment gốc; reply vào cấp 2 → gắn lên gốc
  (FR-009). Tối đa 2 cấp.

**Indexes**:
- `idx_comments_roots` ON `(post_id, created_at DESC)` WHERE `parent_comment_id IS NULL AND status='published'`.
- `idx_comments_replies` ON `(root_comment_id, created_at ASC)` WHERE `parent_comment_id IS NOT NULL AND status='published'`.
- `idx_comments_count` ON `(post_id)` WHERE `status='published'`.

**State transitions**:
```
published --(author delete)--> deleted (terminal; con giữ hiển thị dấu "đã xóa" nếu còn reply)
published --(admin hide)-----> hidden
hidden    --(admin restore)--> published
```
Comment bị xóa mềm; reply con vẫn tồn tại nhưng hiển thị dấu "bình luận đã bị xóa" ở cha nếu còn con;
nếu không còn con thì ẩn hẳn (FR-040).

## Entity: Follow (`community.follows`)

| Field | Type | Constraints |
|---|---|---|
| `follower_id` | uuid | FK `identity.users(id)` |
| `followee_id` | uuid | FK `identity.users(id)` |
| `created_at` | timestamptz | NOT NULL DEFAULT `now()` |
| — | — | PK `(follower_id, followee_id)` — chống trùng (FR-012/FR-031) |

**Constraints**: `chk_follows_no_self`: `follower_id <> followee_id`.

> **Quy ước nhãn (tránh drift)**: tên cột DB giữ `follower_id`/`followee_id` (nội bộ, ổn định);
> API/filter dùng `type=following|followers`; trường trả về dùng `relation=following|follower`.
> Enum `followtype` của API mang giá trị `following`/`followers`.

**Indexes**: `idx_follows_follower_created` ON `(follower_id, created_at DESC)`;
`idx_follows_followee_created` ON `(followee_id, created_at DESC)`.

> Follow là quan hệ đơn giản → xóa cứng khi unfollow (không dùng status). Toggle idempotent.

## Entity: UserFollowStats (`community.user_follow_stats`)

| Field | Type | Constraints |
|---|---|---|
| `user_id` | uuid | PK, FK `identity.users(id)` |
| `follower_count` | int | NOT NULL DEFAULT 0 |
| `following_count` | int | NOT NULL DEFAULT 0 |
| `updated_at` | timestamptz | NOT NULL DEFAULT `now()` |

Cập nhật cùng transaction khi follow/unfollow; worker reconcile định kỳ đối soát từ `follows` (R12).

## Validation rules (tổng hợp từ FR)

| Rule | Ngưỡng (default) | Nguồn |
|---|---|---|
| Số media/bài | ≤ 10 | FR-037 |
| Ảnh | ≤ 10MB; jpg/png/webp | FR-037 |
| Video | ≤ 100MB; mp4/webm; ≤ 60s | FR-037 |
| `title` | ≤ 150 ký tự | FR-037 |
| `content` | ≤ 5000 ký tự | FR-037 |
| Comment | ≤ 1000 ký tự | FR-037 |
| Search prefix | 200 ký tự đầu của `content` | FR-018 |
| Hot window / gravity | 30 ngày / 1.5 | FR-021 |
| Rate limit | post 10/h, comment 60/h, follow 200/h, like 600/h | FR-044 |

## Query/read models

- **PostFilter**: `ViewerUserID *uuid`, `Type feedtype`, `Sort feedsort`, `PostType *posttype`,
  `Username *string`, `Page`, `Limit`.
- **PostListResult**: `Items []*entities.Post`, `TotalCount int64`.
- **CommentFilter**: `PostID`, `ParentCommentID *uuid` (nil = root), `Page`, `Limit`.
- **FollowFilter**: `UserID`, `Type *followtype` (nil = cả hai), `Keyword *string`, `Page`, `Limit`.
- **FollowListResult**: `Items []*FollowRow` (kèm `relation`), `TotalCount int64`.
- **SearchUserRow**: `UserID`, `Username`, `FirstName`, `LastName`, `AvatarURL`, `Gender`.

## Data volume / scale assumptions

- Quy mô MVP: hàng chục nghìn post/comment, hàng trăm nghìn follow — offset pagination chấp nhận.
- Hot ranking giới hạn cửa sổ 30 ngày để bounded.
- Counter denormalized giảm COUNT khi render feed/profile.
