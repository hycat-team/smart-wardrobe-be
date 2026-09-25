# Tài liệu tích hợp Frontend — Spec 022 Community Social

**Feature**: 022-community-social | **Date**: 2026-09-25
**Đối tượng**: đội frontend `smart-wardrobe-fe`
**Mục đích**: mô tả toàn bộ thay đổi API so với code frontend hiện có, gồm **phần giữ nguyên**,
**phần mới**, và **phần chỉnh sửa**, để đội FE triển khai chỉ dựa trên tài liệu này (FR-046/SC-010).

> Base URL hiện tại của FE: `/api/v1` (axios `withCredentials`). Envelope `{ message, data }`.
> Mọi enum mới dùng **chữ thường**.

---

## 1. TL;DR — FE cần làm gì

| Nhóm | Việc |
|---|---|
| Giữ nguyên | Path + envelope + body của like/comment/list/detail/upload; axios config |
| Sửa kiểu dữ liệu | `postType`: `'OUTFIT'\|'SALE'` → `'outfit'\|'media'`; `mediaType`: `'IMAGE'\|'VIDEO'` → `'image'\|'video'` |
| Sửa payload tạo/sửa bài | Bỏ `items`, `contactInfo`; thêm `outfitId`; `media[]` giữ shape |
| Sửa response Post | Bỏ `items/totalPrice/globalHotnessScore/finalFeedScore/isDeleted`; thêm `outfit`, `isFollowingAuthor`, `status`; **thông tin tác giả gộp vào `post.user`** (breaking) |
| Sửa response Comment | Thêm `replyCount`, `isDeleted`; **thông tin tác giả gộp vào `comment.user`** (breaking) |
| Thêm mới | Follow, trang profile công khai, search `q`, feed tab explore/following + sort hot/latest |
| Sửa response Likes | `{ id, ... }` → `CommunityUserRes` phân trang (`userId`, thêm `gender?`) |
| Admin | Giữ moderation post/comment; bỏ tab `post-items`; response thêm `status`/`moderatedBy`/`hiddenReason`/`postId`, có `user` lồng |
| Bỏ | Mọi type/trường liên quan resale/transfer (`transferState`, `buyerUserId`, `soldAt`, `declinedAt`, `itemCondition`, `post-items`) |
| Giới hạn | `title`≤150, `content`≤5000, comment≤1000, ≤10 media, ảnh≤10MB, video≤100MB/60s; rate limit bài/comment/follow/like (xem §3.12) |

---

## 2. Phần GIỮ NGUYÊN (không cần đổi code gọi API)

Các endpoint sau giữ **nguyên path + method + envelope + body**, FE không phải sửa lời gọi:

| API hàm FE | Method + Path | Ghi chú |
|---|---|---|
| `getCommunityPosts` | `GET /posts` | **Chỉ thêm** params `type`, `sort` (xem §3) |
| `getPostDetails` | `GET /posts/{postPublicID}` | response Post đổi (xem §3.3) |
| `getPostComments` | `GET /posts/{postPublicID}/comments` | vẫn là mảng |
| `getCommentReplies` | `GET /posts/{postPublicID}/comments/{commentID}/replies` | vẫn là mảng |
| `likePost` | `PUT /posts/{postPublicID}/like` | body `{ isLiked }` |
| `addComment` | `POST /posts/{postPublicID}/comments` | body `{ content, parentCommentId? }` |
| `updateComment` | `PUT /posts/{postPublicID}/comments/{commentID}` | body `{ content }` |
| `deleteComment` | `DELETE /posts/{postPublicID}/comments/{commentID}` | — |
| `deletePost` | `DELETE /posts/{postPublicID}` | — |
| `getPostUploadSignature` | `GET /posts/upload-signature` | **Chỉ thêm** query `resourceType` |
| `createPost` | `POST /posts` | body đổi (xem §3.2) |
| axios `baseURL`/credentials | — | Không đổi |

Giữ nguyên shape phân trang: `{ items, metadata: { page, limit, totalItems, totalPages } }`.
`page` mặc định 1, `limit` mặc định 20 (tối đa 100).

> Envelope dùng `omitempty`: khi response không có payload, key `data` **vắng mặt** (không phải `data: null`).
> Ví dụ like/unlike, follow/unfollow, delete → `{ "message": "…" }`.

---

## 3. Phần CHỈNH SỬA (thay đổi so với code FE hiện tại)

### 3.1 Enum lowercase (breaking)

| Type FE hiện tại | Sau 022 |
|---|---|
| `postType: 'OUTFIT' \| 'SALE'` | `postType: 'outfit' \| 'media'` |
| `mediaType: string` (gửi `"IMAGE"`) | `mediaType: 'image' \| 'video'` |

Cập nhật mọi so sánh `post.postType === 'OUTFIT'` → `=== 'outfit'`; `'SALE'` → `'media'`.

### 3.2 `CreatePostReq` / `UpdatePostReq`

**Trước:**
```ts
interface CreatePostReq {
  postType: 'OUTFIT' | 'SALE';
  title: string;
  content: string;
  contactInfo?: string;
  items?: PostItemInputReq[];   // resale — BỎ
  media?: PostMediaReq[];
}
```
**Sau:**
```ts
interface CreatePostReq {
  postType: 'outfit' | 'media';
  title?: string;
  content: string;
  outfitId?: string;            // bắt buộc khi postType='outfit'
  media?: PostMediaReq[];
}
// UpdatePostReq: giống trên nhưng KHÔNG có postType (không cho đổi loại)
interface PostMediaReq {
  mediaType: 'image' | 'video';
  mediaUrl: string;
  publicId?: string;
  sortOrder: number;
}
```
Quy tắc validate: `postType='outfit'` ⇒ bắt buộc `outfitId` (thuộc sở hữu người đăng);
`postType='media'` ⇒ không gửi `outfitId`; phải có `content` khác rỗng **hoặc** ≥1 media.
Giới hạn (media/ký tự/kích thước/rate limit) chi tiết tại §3.12.

### 3.3 `PostRes` — **BREAKING: thông tin tác giả chuyển vào `user`**

**Bỏ** các field phẳng ở root: `userId`, `username`, `firstName`, `lastName`, `avatarUrl`
(đã gộp vào object lồng `user`); bỏ `items`, `PostItemRes`, `totalPrice`, `contactInfo`,
`globalHotnessScore`, `finalFeedScore`, `isDeleted` (và mọi field transfer).
**Thêm**: `user` (object), `status`, `outfit`, `isFollowingAuthor`.

```ts
interface CommunityUserRes {
  userId: string;
  username: string;
  firstName?: string;
  lastName?: string;
  avatarUrl?: string;   // omitempty: có thể VẮNG nếu user chưa có avatar
  gender?: number;      // omitempty: VẮNG khi chưa biết; khi có: 1=Nam, 2=Nữ, 3=Khác
}

interface OutfitBriefRes { id: string; name: string; coverImageUrl?: string; }

interface PostRes {
  id: string;
  publicId: string;
  user: CommunityUserRes;           // THAY cho userId/username/firstName/lastName/avatarUrl phẳng
  postType: 'outfit' | 'media';
  status: 'published' | 'hidden' | 'deleted';  // MỚI — bài của chính mình bị ẩn trả 'hidden'
  title?: string | null;
  content: string;
  outfit?: OutfitBriefRes | null;   // có khi postType='outfit'
  likeCount: number;
  commentCount: number;
  isLiked: boolean;
  isFollowingAuthor: boolean;
  sharePath: string;
  media?: PostMediaRes[];
  createdAt: string;
  updatedAt: string;
}
```

> Di trú FE: `post.username` → `post.user.username`; `post.avatarUrl` → `post.user.avatarUrl`;
> `post.userId` → `post.user.userId`; ownership check `profile.username === post.username`
> → `profile.username === post.user.username`.
> Render ảnh: outfit → `post.outfit.coverImageUrl`; media → `post.media[0].mediaUrl`.
> `outfit`/`media` dùng `omitempty` → **vắng key** (không phải `null`) khi không có. `title` luôn có key, giá trị `null` khi trống.
> `status` luôn có: bình thường `'published'`; bài do admin ẩn mà bạn là tác giả → `'hidden'` (người khác nhận 404). `'deleted'` không xuất hiện ở API công khai.
> `gender`: `0=Không xác định` bị lược bỏ (key vắng); chỉ nhận `1` (Nam), `2` (Nữ), `3` (Khác). Tham chiếu hằng số Identity.
> Phòng thủ: nếu backend không lấy được thông tin tác giả (Account tạm lỗi), `user` có thể là `null` — FE nên fallback an toàn (`post.user?.username`).

### 3.4 `CommentRes` — **BREAKING: thông tin tác giả chuyển vào `user`**

Thêm `replyCount`, `isDeleted`; gộp thông tin tác giả vào `user`:
```ts
interface CommentRes {
  id: string;
  user: CommunityUserRes;       // THAY cho userId/username/firstName/lastName/avatarUrl phẳng
  content: string;              // rỗng khi isDeleted=true (còn reply)
  parentCommentId?: string | null;
  replyCount: number;
  isDeleted: boolean;
  createdAt: string;
}
```
> Di trú FE: `comment.username` → `comment.user.username`; `comment.userId` → `comment.user.userId`;
> `comment.avatarUrl` → `comment.user.avatarUrl`.
> `parentCommentId` dùng `omitempty` → **vắng key** khi là bình luận gốc (không phải `null`).
> Quy tắc hiển thị: bình luận gốc **mới nhất trước**; phản hồi trong mỗi gốc **cũ nhất trước**.
> Bình luận bị xóa còn phản hồi → hiển thị "Bình luận đã bị xóa"; không còn phản hồi → không trả về.
> Reply vào phản hồi cấp 2 tự động gắn lên gốc (không cần FE xử lý gì thêm).

### 3.5 Upload signature

```ts
// GET /posts/upload-signature?resourceType=image|video
interface UploadSignatureResult {
  signature: string; timestamp: number; apiKey: string;
  publicId: string;                  // luôn có key (có thể rỗng)
  folder: string;
  resourceType: 'image' | 'video';   // MỚI — FE dùng để chọn endpoint upload
}
```
FE build URL upload Cloudinary: `https://api.cloudinary.com/v1_1/{cloud}/{resourceType}/upload`.

### 3.6 Feed params

`GET /posts` nhận thêm:
- `type`: `explore` (default) | `following`
- `sort`: `hot` (default) | `latest`
- `postType`: `outfit` | `media` (optional)
- `username`, `page`, `limit` (giữ)

Guest gọi `type=following` → 401. `following` chỉ chứa bài của người đang theo dõi (+ bài của chính mình).

### 3.7 Share path

FE hiện có ngầm định `/community/posts/{publicId}` (fallback) trong khi route chi tiết là
`/posts/[postPublicId]`. Backend trả `sharePath = "/community/posts/{publicId}"` — FE nên thống
nhất một route, không phụ thuộc fallback.

### 3.8 Admin

- Giữ: `GET /admin/posts`, `DELETE /admin/posts/{id}`, `PATCH /admin/posts/{id}/restore`,
  `GET /posts/{postPublicID}/comments`, `DELETE /admin/comments/{id}`, `PATCH /admin/comments/{id}/restore`.
- Thêm: `GET /admin/comments`, `PATCH /admin/posts/{id}/hide`, `PATCH /admin/comments/{id}/hide`.
- **Bỏ hẳn** tab/API `post-items` (`GET /admin/post-items`, `DELETE /admin/post-items/{id}`,
  `PATCH /admin/post-items/{id}/hide`) và các type resale trong `admin/types`.
- `{id}` của admin là **UUID nội bộ** (khác `publicId` dùng ở API công khai).
- Ẩn/xóa bình luận gốc **cascade** lên phản hồi con. Khôi phục nội dung do tác giả tự xóa → **400** (chỉ khôi phục được nội dung do admin ẩn/xóa).
- Trong `AdminPostRes`, `isLiked`/`isFollowingAuthor` luôn `false` (không enrich viewer).

Query params:
- `GET /admin/posts`: `q` (từ khóa), `status` (vd `published|hidden|deleted`), `page`, `limit`.
- `GET /admin/comments`: `q`, `page`, `limit`.

Response types (admin post/comment nay **có `user`** lồng, giống public):
```ts
// GET /admin/posts -> PaginationResult<AdminPostRes>
interface AdminPostRes extends PostRes {   // status đã có sẵn trong PostRes
  moderatedBy?: string;        // UUID admin (vắng nếu user tự xóa)
  hiddenReason?: string;       // vd author_disabled (vắng nếu không có)
}

// GET /admin/comments -> PaginationResult<AdminCommentRes>
interface AdminCommentRes extends CommentRes {
  postId: string;              // UUID bài đăng chứa bình luận
  status: string;              // published | hidden | deleted
}
```
> `moderatedBy`/`hiddenReason` dùng `omitempty` → **vắng key** khi không có.

### 3.9 Danh sách người thích (`GET /posts/{publicId}/likes`) — **BREAKING**

Trước đây trả `PostLikeUserRes` (`{ id, username, firstName, lastName, avatarUrl }`).
Nay trả **phân trang** `PaginationResult<CommunityUserRes>`:
```ts
interface PostLikeUserListRes {
  items: CommunityUserRes[];   // { userId, username, firstName?, lastName?, avatarUrl?, gender? }
  metadata: { page: number; limit: number; totalItems: number; totalPages: number };
}
```
> Di trú FE: bỏ `likeUser.id` → dùng `likeUser.userId`; thêm `gender?`. **Đây là list phân trang** (trước đây nếu FE coi là mảng phẳng thì phải đọc `.items`).

### 3.10 Response body & mã trạng thái

Mọi endpoint trả envelope `{ message, data? }`. `data` **vắng** khi không có payload.

| Endpoint | Status | `data` |
|---|---|---|
| `GET /posts` | 200 | `PaginationResult<PostRes>` |
| `GET /posts/{publicId}` | 200 | `PostRes` |
| `POST /posts` | **201** | `PostRes` |
| `PUT /posts/{publicId}` | 200 | `PostRes` |
| `DELETE /posts/{publicId}` | 200 | (vắng) |
| `GET /posts/upload-signature` | 200 | `UploadSignatureResult` |
| `PUT /posts/{publicId}/like` | 200 | (vắng) |
| `GET /posts/{publicId}/likes` | 200 | `PaginationResult<CommunityUserRes>` |
| `GET /posts/{publicId}/comments` | 200 | `CommentRes[]` (**mảng phẳng, không phân trang**) |
| `GET /posts/{publicId}/comments/{id}/replies` | 200 | `CommentRes[]` (mảng phẳng) |
| `POST /posts/{publicId}/comments` | **201** | `CommentRes` |
| `PUT /posts/{publicId}/comments/{id}` | 200 | `CommentRes` |
| `DELETE /posts/{publicId}/comments/{id}` | 200 | (vắng) |
| `PUT /users/{username}/follow` | 200 | (vắng) |
| `GET /users/{username}` | 200 | `PublicProfileRes` |
| `GET /users/{username}/posts` | 200 | `PaginationResult<PostRes>` |
| `GET /users/{username}/follows` | 200 | `PaginationResult<FollowUserRes>` |
| `GET /search` | 200 | `SearchRes` |
| `GET /admin/posts` | 200 | `PaginationResult<AdminPostRes>` |
| `GET /admin/comments` | 200 | `PaginationResult<AdminCommentRes>` |
| Admin hide/restore/delete | 200 | (vắng) |

### 3.11 Định dạng lỗi

Lỗi trả JSON **không** có envelope `data`:
```ts
interface ApiError {
  status: number;            // trùng HTTP status
  title?: string;            // vd "Lỗi hệ thống"
  message?: string;          // thông báo tiếng Việt hiển thị cho user
  errors?: { field: string; message: string }[];  // lỗi validate (400)
  debugMessage?: string;     // chỉ ở môi trường non-production
}
```
| Tình huống | HTTP |
|---|---|
| Body/query sai định dạng, thiếu trường | 400 |
| Chưa đăng nhập (guest gọi `following`/thao tác ghi) | 401 |
| Không phải chủ bài/comment | 403 |
| Không tìm thấy post/comment/user | 404 |
| Vượt rate limit | 429 |

### 3.12 Ràng buộc & giới hạn (từ config backend)

FE nên validate client-side để tránh 400/429. Giá trị là mặc định trong `configs/config.yaml` (block `community`).

| Trường / hành động | Giới hạn | HTTP khi vượt |
|---|---|---|
| `title` | ≤ **150** ký tự | 400 |
| `content` (bài) | ≤ **5000** ký tự | 400 |
| `content` (bình luận, đã trim) | ≤ **1000** ký tự, không rỗng | 400 |
| `media` | ≤ **10** phần tử/bài | 400 |
| Ảnh (`mediaType=image`) | ≤ **10 MB** (jpg/png/webp) | 400/từ chối upload |
| Video (`mediaType=video`) | ≤ **100 MB**, ≤ **60 giây** (mp4/webm) | 400/từ chối upload |
| Nội dung bài | phải có `content` khác rỗng **hoặc** ≥1 media | 400 |
| Tạo bài | ≤ **10** bài/giờ/user | 429 |
| Bình luận | ≤ **60** bình luận/giờ/user | 429 |
| Follow / Unfollow | ≤ **200** thao tác/giờ/user | 429 |
| Like / Unlike | ≤ **600** thao tác/giờ/user | 429 |
| Tìm kiếm bài | chỉ so khớp **200** ký tự đầu của `content` | — |
| Tìm kiếm user (server) | tối đa **500** ứng viên từ Account | — |
| Phân trang | `page≥1`, `limit` mặc định **20**, tối đa **100** | — |

- Giới hạn kích thước/thời lượng media do Cloudinary + body limit enforce; FE nên chặn trước khi upload để UX tốt hơn.
- Khi 429, hiển thị `message` và cho phép thử lại sau; không retry dồn dập.

---

## 4. Phần MỚI (FE cần thêm)

### 4.1 Follow
```ts
// PUT /users/{username}/follow
interface FollowReq { isFollowing: boolean; }
```
Idempotent; tự follow chính mình → 400.

### 4.2 Trang cá nhân công khai
```ts
// GET /users/{username}
interface PublicProfileRes {
  user: CommunityUserRes;   // { userId, username, firstName?, lastName?, avatarUrl?, gender? }
  stats: { followerCount: number; followingCount: number; postCount: number };
  isFollowing: boolean;
  isMe: boolean;
}

// GET /users/{username}/posts  -> PaginationResult<PostRes> (sort latest)

// GET /users/{username}/follows?type=following|followers&q=&page=&limit=
interface FollowUserRes {
  user: CommunityUserRes;               // THAY cho userId/username/firstName/lastName/avatarUrl phẳng
  relation: 'following' | 'follower';   // khi không truyền type => trả cả hai
  followedAt: string;
}
```
`type` **luôn tùy chọn** (`following` = người user đang theo dõi; `followers` = người theo dõi user);
bỏ trống trả cả hai, dùng `relation` để phân biệt. `type` sai giá trị → **400** (không fallback).
Danh sách follow sắp theo `followedAt` **giảm dần** (mới nhất trước); `q` lọc theo username/họ tên.
> Di trú FE: `follow.username` → `follow.user.username`; `follow.avatarUrl` → `follow.user.avatarUrl`.
> Profile user nay có thêm `userId` và `avatarUrl` nằm trong `user`.
> `GET /users/{username}/posts` của **chính mình** bao gồm cả bài `hidden` (bài `deleted` không trả về); của người khác chỉ `published`.

### 4.3 Search
```ts
// GET /search?q=&type=all|users|posts&postType=&page=&limit=
interface SearchRes {
  users: PaginationResult<CommunityUserRes>;
  posts: PaginationResult<PostRes>;
}
// CommunityUserRes: { userId, username, firstName?, lastName?, avatarUrl?, gender? }
```
- `q` tối thiểu 1 ký tự sau trim; rỗng → cả hai mảng rỗng (200).
- Tìm user theo username **và** họ tên; post chỉ so khớp **200 ký tự đầu** của `content`.
- **Bài đăng tìm được giới hạn trong `hot_window_days` = 30 ngày gần nhất và chỉ bài `published`** → bài cũ hơn sẽ không xuất hiện.
- Thứ tự: `users` theo số follower giảm dần rồi username tăng dần; `posts` theo `hot`.
- Không truyền `type` (hoặc `all`) → trả cả hai; `users` → posts rỗng; `posts` → users rỗng. `type` sai → coi như `all`.
- `postType` sai giá trị → bỏ qua (không lỗi).
- `users.items` là **phẳng** `CommunityUserRes` (không bọc thêm `user`).
- Trang `/search` hiện đang lọc wardobe client-side — cần thay bằng gọi `/search` và render 2 nhóm.

### 4.4 Feed tab (explore/following)
- Thêm UI tab + sort; gọi `GET /posts?type=...&sort=...`.
- `CommunityClient` hiện gọi `useInfiniteCommunity()` không tham số → truyền filter.
- `type`/`sort` sai giá trị → backend fallback về `explore`/`hot` (không lỗi); `postType` sai → bỏ qua.

### 4.5 Hành vi bổ sung cần lưu ý

- **Feed/Profile**: chỉ trả bài `published`; riêng tác giả còn thấy bài `hidden` của mình (kèm `status='hidden'`). Không trả bài `deleted`.
- **Chi tiết bài** (`GET /posts/{publicId}`): bài `deleted` → 404; bài `hidden` → 404 với người khác, tác giả nhận 200.
- **Bình luận**: endpoint gốc/replies trả **mảng** (không phân trang). Bình luận gốc đã xóa nhưng còn phản hồi → vẫn trả với `isDeleted=true`, `content=""`; xóa mà không còn phản hồi → bị lược bỏ. `replyCount` chỉ có ở gốc, phản hồi luôn `0`.
- **Replies**: nếu `commentID` không thuộc bài → trả `[]` (200, không 404).
- **`isFollowingAuthor`**: luôn `false` khi người xem chính là tác giả.
- **Danh sách like**: nếu backend không lấy được hồ sơ một user, mục đó bị lược khỏi `items` (có thể ít hơn `metadata.totalItems`).
- **`outfit`**: enrichment best-effort — nếu lấy outfit lỗi, key `outfit` có thể vắng dù `postType='outfit'`; FE nên fallback an toàn.
- **Like/Follow**: idempotent — gửi trùng trạng thái hiện tại trả 200 nhưng không đổi dữ liệu (bộ đếm giữ nguyên).

---

## 5. Bảng endpoint đầy đủ

| Method | Path | Auth | Query / Body | Response `data` |
|---|---|---|---|---|
| GET | `/posts` | optional | `type`, `sort`, `postType`, `username`, `page`, `limit` | `PaginationResult<PostRes>` |
| GET | `/posts/{publicId}` | optional | — | `PostRes` |
| GET | `/posts/upload-signature` | user | `resourceType=image\|video` | `UploadSignatureResult` |
| POST | `/posts` | user | `CreatePostReq` | `PostRes` (201) |
| PUT | `/posts/{publicId}` | user (chủ) | `UpdatePostReq` | `PostRes` |
| DELETE | `/posts/{publicId}` | user (chủ) | — | (vắng) |
| PUT | `/posts/{publicId}/like` | user | `{ isLiked }` | (vắng) |
| GET | `/posts/{publicId}/likes` | optional | `page`, `limit` | `PaginationResult<CommunityUserRes>` |
| GET | `/posts/{publicId}/comments` | optional | — | `CommentRes[]` (không phân trang) |
| GET | `/posts/{publicId}/comments/{id}/replies` | optional | — | `CommentRes[]` (không phân trang) |
| POST | `/posts/{publicId}/comments` | user | `{ content, parentCommentId? }` | `CommentRes` (201) |
| PUT | `/posts/{publicId}/comments/{id}` | user (chủ) | `{ content }` | `CommentRes` |
| DELETE | `/posts/{publicId}/comments/{id}` | user (chủ) | — | (vắng) |
| PUT | `/users/{username}/follow` | user | `{ isFollowing }` | (vắng) |
| GET | `/users/{username}` | optional | — | `PublicProfileRes` |
| GET | `/users/{username}/posts` | optional | `page`, `limit` | `PaginationResult<PostRes>` |
| GET | `/users/{username}/follows` | optional | `type`, `q`, `page`, `limit` | `PaginationResult<FollowUserRes>` |
| GET | `/search` | optional | `q`, `type`, `postType`, `page`, `limit` | `SearchRes` |
| GET | `/admin/posts` | admin | `q`, `status`, `page`, `limit` | `PaginationResult<AdminPostRes>` |
| PATCH | `/admin/posts/{id}/hide` | admin | — | (vắng) |
| PATCH | `/admin/posts/{id}/restore` | admin | — | (vắng) |
| DELETE | `/admin/posts/{id}` | admin | — | (vắng) |
| GET | `/admin/comments` | admin | `q`, `page`, `limit` | `PaginationResult<AdminCommentRes>` |
| PATCH | `/admin/comments/{id}/hide` | admin | — | (vắng) |
| PATCH | `/admin/comments/{id}/restore` | admin | — | (vắng) |
| DELETE | `/admin/comments/{id}` | admin | — | (vắng) |

> `{id}` bình luận và admin là **UUID nội bộ**; `{publicId}` là mã công khai 32 ký tự của bài đăng.

---

## 6. Checklist migration FE

- [ ] Đổi union `postType` → `'outfit' | 'media'`, `mediaType` → `'image' | 'video'` toàn repo.
- [ ] Cập nhật `CreatePostReq`/`UpdatePostReq`: bỏ `items`/`contactInfo`, thêm `outfitId`, cho phép media rỗng + content.
- [ ] Chọn outfit trong composer khi `postType='outfit'` (cần API danh sách outfit hiện có).
- [ ] Cập nhật `PostRes`: bỏ field resale, thêm `outfit`, `isFollowingAuthor`, `status`, và **gộp thông tin tác giả vào `post.user`** (bỏ `userId/username/firstName/lastName/avatarUrl` phẳng).
- [ ] Cập nhật render ảnh `PostCard`: dùng `outfit.coverImageUrl` / `media[0].mediaUrl`, hỗ trợ video.
- [ ] Cập nhật `CommentRes`: `replyCount`, `isDeleted`, **`comment.user`** (bỏ field phẳng); áp thứ tự gốc/reply.
- [ ] Đổi mọi chỗ đọc user phẳng → nested: `post.user.*`, `comment.user.*`, `follow.user.*`; ownership check dùng `*.user.userId`/`*.user.username`; truy cập phòng thủ khi `user` có thể `null`.
- [ ] Map `gender` số (1=Nam, 2=Nữ, 3=Khác) sang nhãn; xử lý key vắng.
- [ ] `FollowUserRes` → `{ user: CommunityUserRes, relation, followedAt }`.
- [ ] `PublicProfileRes.user` → `CommunityUserRes` (có thêm `userId`).
- [ ] Search users dùng `CommunityUserRes` (phẳng trong `users.items`, không bọc `user`).
- [ ] Likes (`GET /posts/{id}/likes`): đổi `likeUser.id` → `likeUser.userId`, đọc qua `.items` (phân trang).
- [ ] Admin: thêm type `AdminPostRes`/`AdminCommentRes` (extends + `status`/`moderatedBy`/`hiddenReason`/`postId`), dùng `user` lồng.
- [ ] Xử lý lỗi theo §3.11 (`message`, `errors[]`, 401/403/404/429); lưu ý `data` vắng khi không có payload.
- [ ] Upload: truyền `resourceType`, xử lý video; dùng `resourceType` build URL Cloudinary.
- [ ] Validate client-side theo giới hạn §3.12 (title ≤150, content ≤5000, comment ≤1000, ≤10 media, ảnh ≤10MB, video ≤100MB/60s) và xử lý 429.
- [ ] Feed: thêm tab explore/following + sort hot/latest.
- [ ] Thêm follow button, trang `/users/[username]`, danh sách follow (có `relation`).
- [ ] Thay search page bằng gọi `/search`, render 2 nhóm users/posts.
- [ ] Xóa code/types resale (`PostItemRes`, `transferState`, `buyerUserId`, `soldAt`, `declinedAt`, `itemCondition`, `post-items`, marketplace/mock liên quan).
- [ ] Cập nhật admin moderation: bỏ tab `post-items`, thêm comments + hide; admin post/comment nay có `user`.
- [ ] Kiểm tra luồng guest (đọc được) và 401 khi gọi `following`/thao tác ghi.
- [ ] Đồng bộ `sharePath` với route chi tiết.

---

## 7. Cạm bẫy

1. **Enum lowercase là breaking** — grep toàn bộ `'OUTFIT'`, `'SALE'`, `'IMAGE'`, `'VIDEO'`.
2. **`postType` vẫn là string union trên FE**, nhưng admin type hiện dùng numeric (`postType?: number`) — cần đồng bộ về string nếu admin render.
3. **`items` biến mất** — mọi chỗ đọc `post.items[0].item` phải sửa.
4. **Comment bị xóa** có thể trả về phần tử với `isDeleted=true` và `content=""` — UI phải xử lý placeholder.
5. **Search user + post độc lập** — 2 metadata phân trang riêng, không dùng chung `page`.
6. **Follow list `relation`** — khi không truyền `type`, item có thể là following hoặc follower.
7. **`isFollowingAuthor`** giúp ẩn/hiện nút follow ngay trên feed mà không cần gọi thêm.
8. **Nested `user` (breaking)** — `post.user`, `comment.user`, `follow.user` là object; danh sách likes/search dùng `CommunityUserRes` **phẳng** (không bọc `user`). `avatarUrl` có `omitempty` → có thể vắng.
9. **Likes là phân trang** — đọc `res.items`, không coi là mảng phẳng; field `id` cũ đổi thành `userId`.
10. **Admin post/comment cũng có `user` lồng** và thêm `status`/`moderatedBy`/`hiddenReason` (post) hoặc `postId`/`status` (comment).
11. **Envelope `data` bị lược bỏ khi rỗng** (`omitempty`) — đừng giả định luôn có `data`. Lỗi trả object `{ status, title?, message?, errors? }` (không có `data`).
12. **`status` trên PostRes (mới)** — bình thường `'published'`; bài của chính bạn bị admin ẩn → `'hidden'` (vẫn hiển thị trong feed/detail của bạn); người khác nhận 404. Có thể dùng để gắn badge "đang bị ẩn".
13. **Search bài chỉ 30 ngày** (`hot_window_days`) và chỉ bài `published` — đừng kỳ vọng tìm được bài cũ.
14. **`user` có thể `null`** khi backend không lấy được hồ sơ tác giả — luôn truy cập phòng thủ (`post.user?.username`).
15. **`gender` là số** (`1`=Nam, `2`=Nữ, `3`=Khác; `0` bị lược bỏ) — cần map sang nhãn hiển thị.
