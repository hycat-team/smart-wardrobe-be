# Phase 1 Contract: Community HTTP API

**Feature**: 022-community-social | **Base path**: `/api/v1` | **Date**: 2026-09-25

Envelope chuẩn: `{ "message": string, "data": T }` (lỗi theo `apperror`:
`{status,title,message,errors?}`). Tất cả JSON field **camelCase**; message **tiếng Việt**.

Phân trang: `PaginationResult<T> = { items: T[], metadata: { page, limit, totalItems, totalPages } }`.

## Quy ước xác thực

| Nhóm | Middleware |
|---|---|
| Đọc công khai | `OptionalHandle` (guest đọc được; nếu đăng nhập thì có `viewerUserID`) |
| Ghi | `Handle` + `RolesAuthorize(user)` |
| Admin | `Handle` + `RolesAuthorize(admin)` |

## 1. Post & Feed

### `GET /posts` — Bảng tin
- Auth: public (optional).
- Query: `type` (explore|following, default explore), `sort` (hot|latest, default hot),
  `postType` (outfit|media, optional), `username` (optional), `page`, `limit`.
- Guest gọi `type=following` → 401.
- Response: `200` `PaginationResult<PostRes>`.

### `GET /posts/{postPublicID}` — Chi tiết bài
- Auth: public (optional). Response: `200` `PostRes`. Ẩn/khóa → 404 với user thường.

### `POST /posts/upload-signature` — Chữ ký tải media
- Auth: required. Query: `resourceType` (image|video, default image).
- Response: `200` `{ signature, timestamp, apiKey, publicId, folder, resourceType }`.

### `POST /posts` — Tạo bài
- Auth: required. Body:
```json
{
  "postType": "outfit",
  "title": "optional",
  "content": "…",
  "outfitId": "uuid (bắt buộc nếu postType=outfit)",
  "media": [{ "mediaType": "image", "mediaUrl": "https://…", "publicId": "…", "sortOrder": 0 }]
}
```
- Quy tắc: `outfit` bắt buộc `outfitId` thuộc sở hữu; `media` không có `outfitId`; phải có
  `content` hoặc ≥1 media.
- Response: `201` `PostRes`.

### `PUT /posts/{postPublicID}` — Cập nhật bài (chỉ chủ bài)
- Body giống create, không có `postType` (không cho đổi loại). Có thể đổi `outfitId` sang outfit khác thuộc sở hữu.
- Response: `200` `PostRes`.

### `DELETE /posts/{postPublicID}` — Xóa bài (chỉ chủ bài)
- Response: `200` (data null). Trạng thái `deleted` (terminal, không khôi phục).

### `PUT /posts/{postPublicID}/like` — Like/unlike (idempotent)
- Body: `{ "isLiked": true|false }`. Response `200`.

### `GET /posts/{postPublicID}/likes` — Danh sách người thích
- Response `200` `PaginationResult<CommunityUserRes>`.

## 2. Comment

### `GET /posts/{postPublicID}/comments` — Bình luận gốc
- Response `200` `CommentRes[]` (mới nhất trước) — giữ nguyên dạng mảng như FE cũ.
  Mỗi phần tử có `replyCount`; nếu bị xóa còn con → `isDeleted=true` + `content=""`.

### `GET /posts/{postPublicID}/comments/{commentID}/replies` — Phản hồi
- Response `200` `CommentRes[]` (cũ nhất trước).

### `POST /posts/{postPublicID}/comments` — Thêm bình luận
- Body: `{ "content": "…", "parentCommentId": "uuid|null" }`.
- Nếu `parentCommentId` trỏ tới comment cấp 2 → tự động gắn lên comment gốc.
- Response `201` `CommentRes`.

### `PUT /posts/{postPublicID}/comments/{commentID}` — Sửa (chủ comment)
- Body `{ "content": "…" }`. Response `200` `CommentRes`.

### `DELETE /posts/{postPublicID}/comments/{commentID}` — Xóa (chủ comment)
- Response `200`. Xóa mềm (terminal).

## 3. Follow & Profile

### `PUT /users/{username}/follow` — Theo dõi (idempotent)
- Auth: required. Body `{ "isFollowing": true|false }`. Không cho tự theo dõi → 400.
- Response `200`.

### `GET /users/{username}` — Trang cá nhân công khai
- Auth: public (optional).
- Response `200`:
```json
{
  "user": { "userId": "uuid", "username": "…", "firstName": "…", "lastName": "…", "gender": 1, "avatarUrl": "…" },
  "stats": { "followerCount": 0, "followingCount": 0, "postCount": 0 },
  "isFollowing": false,
  "isMe": false
}
```

### `GET /users/{username}/posts` — Bài của người dùng
- Auth: public (optional). Query `page`,`limit`. Sắp `latest`. Response `PaginationResult<PostRes>`.

### `GET /users/{username}/follows` — Danh sách follow
- Auth: public (optional).
- Query: `type` (`following`|`followers`, **optional**), `q` (optional), `page`, `limit`.
- Không truyền `type` → trả **cả hai**, mỗi mục có `relation`.
- Response `PaginationResult<FollowUserRes>`:
```json
{ "user": { "userId": "uuid", "username": "…", "firstName": "…", "lastName": "…", "avatarUrl": "…" },
  "relation": "following|follower", "followedAt": "…" }
```

## 4. Search

### `GET /search` — Tìm user + post
- Auth: public (optional).
- Query: `q` (bắt buộc, ≥1 ký tự sau trim), `type` (all|users|posts, optional, default all),
  `postType` (optional), `page`, `limit`.
- `q` rỗng → trả cả hai nhóm rỗng, không lỗi.
- Response `200`:
```json
{
  "users": { "items": [CommunityUserRes], "metadata": {…} },
  "posts": { "items": [PostRes], "metadata": {…} }
}
```
- `CommunityUserRes`: `{ userId, username, firstName, lastName, avatarUrl, gender }` (xem mục DTO chính).
- Thứ tự: users theo follower giảm dần rồi mới nhất; posts theo hot. Post chỉ so khớp 200 ký tự đầu.
- **Giới hạn ứng viên user**: backend lấy tối đa `N` user khớp từ Account (config
  `Community.SearchUserCandidateCap`, mặc định 500) rồi enrich follower và phân trang; ngoài N có thể thiếu.

## 5. Admin moderation

| Method | Path | Mô tả |
|---|---|---|
| GET | `/admin/posts` | Danh sách (q, status, page, limit) |
| PATCH | `/admin/posts/{id}/hide` | Ẩn (hidden) |
| PATCH | `/admin/posts/{id}/restore` | Khôi phục → published (chỉ với nội dung do admin ẩn/xóa) |
| DELETE | `/admin/posts/{id}` | Xóa mềm bởi admin |
| GET | `/admin/comments` | Danh sách bình luận |
| PATCH | `/admin/comments/{id}/hide` | Ẩn |
| PATCH | `/admin/comments/{id}/restore` | Khôi phục |
| DELETE | `/admin/comments/{id}` | Xóa mềm bởi admin |

- Auth: `admin`. Mọi hành động ghi audit (`admin.audit.log`).
- Nội dung tác giả tự xóa (`deleted`, `moderated_by IS NULL`) không khôi phục được (4xx).
- Ẩn/xóa bình luận gốc **cascade** lên phản hồi con (ẩn/hiện theo cây).
- Tác giả vẫn thấy bài `hidden` của mình ở API công khai (qua field `status`); người khác nhận 404.
- Trạng thái user thay đổi (admin khóa/mở) phát event `identity.user.status.changed`; bài của user
  bị khóa chuyển `status=hidden` + `hidden_reason=author_disabled` và tự khôi phục khi mở lại.

## DTO chính

### CommunityUserRes (canonical user projection)
```json
{ "userId": "uuid", "username": "…", "firstName": "…", "lastName": "…",
  "avatarUrl": "…", "gender": 1 }
```
- `avatarUrl` dùng `omitempty` → **có thể vắng** khi user chưa có avatar.
- `gender` optional (chỉ xuất hiện khi biết).
- Dùng làm **object lồng** `user` trong `PostRes`/`CommentRes`/`FollowUserRes`, và dùng **trực tiếp** (phẳng) cho danh sách thuần user: likes, search users, profile user.

### PostRes
```json
{
  "id": "uuid", "publicId": "32char",
  "user": { "userId": "uuid", "username": "…", "firstName": "…", "lastName": "…", "avatarUrl": "…", "gender": 1 },
  "postType": "outfit|media", "status": "published|hidden|deleted",
  "title": "…|null", "content": "…",
  "outfit": { "id": "uuid", "name": "…", "coverImageUrl": "…" } ,
  "likeCount": 0, "commentCount": 0, "isLiked": false, "isFollowingAuthor": false,
  "sharePath": "/community/posts/{publicId}",
  "media": [{ "id": "uuid", "mediaType": "image|video", "mediaUrl": "…", "publicId": "…", "sortOrder": 0 }],
  "createdAt": "…", "updatedAt": "…"
}
```
> Thông tin tác giả đã chuyển vào object lồng `user` (trước đây phẳng ở root: `userId`, `username`,
> `firstName`, `lastName`, `avatarUrl`). Loại bỏ so với bản cũ: `contactInfo`, `totalPrice`,
> `globalHotnessScore`, `finalFeedScore`, `items`, `transferState`, `status` (transfer), `isDeleted`.

### CommentRes
```json
{ "id": "uuid",
  "user": { "userId": "uuid", "username": "…", "firstName": "…", "lastName": "…", "avatarUrl": "…" },
  "content": "…", "parentCommentId": "uuid|null",
  "replyCount": 0, "isDeleted": false, "createdAt": "…" }
```

### FollowUserRes
```json
{ "user": { "userId": "uuid", "username": "…", "firstName": "…", "lastName": "…", "avatarUrl": "…" },
  "relation": "following|follower", "followedAt": "…" }
```

### Danh sách người thích (`PostLikeUserListRes`)
```json
{ "items": [{ "userId": "uuid", "username": "…", "firstName": "…", "lastName": "…", "avatarUrl": "…", "gender": 1 }],
  "metadata": { "page": 1, "limit": 20, "totalItems": 0, "totalPages": 0 } }
```
> `PostLikeUserListRes = PaginationResult<CommunityUserRes>`; dùng trực tiếp `CommunityUserRes` (không bọc thêm `user`).

## Mã lỗi tiêu biểu

| Tình huống | HTTP | Message (vi) |
|---|---|---|
| Guest gọi `following` / thao tác ghi | 401 | Vui lòng đăng nhập |
| Không phải chủ bài/comment | 403 | Không có quyền truy cập |
| Outfit không thuộc sở hữu | 400 | Outfit không hợp lệ |
| Bài rỗng (không content/media) | 400 | Vui lòng nhập nội dung hoặc đính kèm media |
| Vượt giới hạn media/ký tự | 400 | (thông báo cụ thể theo trường) |
| Reply vào cấp 2 | — | không lỗi; tự gắn lên gốc |
| Tự follow chính mình | 400 | Không thể theo dõi chính mình |
| Username/ bài không tồn tại | 404 | Không tìm thấy dữ liệu |
| Vượt rate limit | 429 | Quá nhiều yêu cầu |
| Khôi phục bài tác giả đã xóa | 400 | Không thể khôi phục nội dung do người dùng đã xóa |

## Cross-service contract (identity)

Proto `contracts/identity/v1/identity.proto` thêm:
```proto
message SearchUserBriefsRequest {
  string keyword = 1;
  int32 limit = 2;
}
service IdentityService {
  rpc SearchUserBriefs(SearchUserBriefsRequest) returns (UserBriefList);
}
```
Go interface `IUserBriefContract` thêm:
```go
SearchUserBriefs(ctx context.Context, keyword string, limit int) ([]*dto.UserBriefContractDTO, error)
```
