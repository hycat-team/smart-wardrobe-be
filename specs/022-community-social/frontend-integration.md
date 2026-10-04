# Tài liệu tích hợp Frontend — Spec 022 Community Social

**Feature**: 022-community-social | **Date**: 2026-09-25 | **Cập nhật**: 2026-09-28 (v2)
**Đối tượng**: đội frontend `smart-wardrobe-fe`
**Mục đích**: mô tả toàn bộ thay đổi API so với code frontend hiện có, gồm **phần giữ nguyên**,
**phần mới**, và **phần chỉnh sửa**, để đội FE triển khai chỉ dựa trên tài liệu này (FR-046/SC-010).

> Base URL hiện tại của FE: `/api/v1` (axios `withCredentials`). Envelope `{ message, data }`.
> Mọi enum mới dùng **chữ thường**.

> **Nếu FE đã triển khai theo bản 26/09/2026:** đọc **[§0](#0-lịch-sử-cập-nhật--đọc-trước-khi-bắt-đầu)**
> trước khi làm tiếp — có 5 yêu cầu mới và 3 chỗ bản cũ nói sai.

---

## 0. Lịch sử cập nhật — đọc trước khi bắt đầu

So với bản đã commit `904d030d` (26/09/2026) mà FE dùng để triển khai, tài liệu này có
**5 yêu cầu mới** và **3 chỗ bản cũ nói sai sự thật**.

> Không có endpoint mới. Không có path/method nào đổi. Toàn bộ thay đổi nằm ở luồng
> **upload media** và validate bài đăng.

### 0.1 Yêu cầu MỚI — bắt buộc làm thêm

| # | Thay đổi | Mục | Việc FE phải làm |
|---|---|---|---|
| **N1** | `PostMediaReq` có thêm `bytes?` + `durationSeconds?` (bản cũ **không có** 2 field này) | [§3.2](#32-createpostreq--updatepostreq) | Gửi 2 field này khi tạo/sửa bài, lấy từ **response upload** Cloudinary (`bytes`, `duration`) |
| **N2** | `UploadSignatureResult` có thêm `allowedFormats` (bản cũ **không có**) | [§3.5](#35-upload-signature) | Đọc `allowedFormats` rồi gửi **nguyên văn** vào form upload |
| **N3** | Form upload **chỉ được gửi đúng 6 field** (bản cũ không nói) | [§3.5](#35-upload-signature) | Bỏ mọi field thừa FE đang gửi: `public_id`, `eager`, `transformation`, `tags`… |
| **N4** | `resourceType` sai giá trị → **400** (trước: backend âm thầm fallback về `image`) | [§3.5](#35-upload-signature) | Chỉ truyền `image`/`video`; bắt lỗi 400 thay vì coi như mặc định |
| **N5** | Validate **đuôi file** trong `mediaUrl` → **400** (bản cũ không hề kiểm tra) | [§3.2](#32-createpostreq--updatepostreq) | Chặn đuôi sai **trước khi** gọi `POST /posts`; danh sách đuôi ở §3.12 |

### 0.2 Sửa SAI trong bản cũ — FE có thể đã làm theo, cần sửa lại

| # | Bản cũ (26/09) nói | Thực tế | Mục |
|---|---|---|---|
| **S1** | "Giới hạn kích thước/thời lượng media **do Cloudinary + body limit enforce**" | **SAI.** Cloudinary không có tham số kích thước/thời lượng để ký vào signature. Backend **chỉ kiểm được khi FE gửi** `bytes`/`durationSeconds`. Nếu FE bỏ trống → **không có gì được kiểm tra** | [§3.12](#312-ràng-buộc--giới-hạn-từ-config-backend) |
| **S2** | Bảng giới hạn gộp 1 dòng: "Ảnh ≤ 10MB (jpg/png/webp)" | Đã **tách + mở rộng**: đuôi ảnh `jpg, jpeg, png, webp, avif, gif`; đuôi video `mp4, webm, mov, m4v, mkv, avi`; kích thước và thời lượng là các dòng riêng | [§3.12](#312-ràng-buộc--giới-hạn-từ-config-backend) |
| **S3** | `publicId` trong response signature: "luôn có key (có thể rỗng)" | Nay **vắng hẳn key** (đã thêm `omitempty`). Endpoint này không ký `public_id` → client **không được** gửi `public_id` lên Cloudinary | [§3.5](#35-upload-signature) |

### 0.3 Chỉ bổ sung giải thích — FE không cần đổi code

- [§3.2](#32-createpostreq--updatepostreq) — quy tắc validate đuôi file; `duration` là số **thập phân** (`3.041267`), gửi nguyên văn, không làm tròn
- [§3.5](#35-upload-signature) — bảng **3 nghĩa của `publicId`**; dùng `secure_url` (https) chứ không phải `url` (http)
- [§6](#6-checklist-migration-fe) — thêm **1 mục mới** (gửi `bytes`/`durationSeconds`); mục `Upload` cũ viết lại cho khớp; các mục còn lại giữ nguyên (23 → 24 mục)
- [§7](#7-cạm-bẫy) — 4 mục cạm bẫy mới (16–19); 15 mục cũ giữ nguyên

---

## 1. TL;DR — FE cần làm gì

| Nhóm | Việc |
|---|---|
| Giữ nguyên | Path + envelope + body của like/comment/list/detail; axios config (**upload KHÔNG còn giữ nguyên**) |
| Sửa kiểu dữ liệu | `postType`: `'OUTFIT'\|'SALE'` → `'outfit'\|'media'`; `mediaType`: `'IMAGE'\|'VIDEO'` → `'image'\|'video'` |
| Sửa payload tạo/sửa bài | Bỏ `items`, `contactInfo`; thêm `outfitId`; `media[]` thêm `bytes?`, `durationSeconds?` |
| Sửa response Post | Bỏ `items/totalPrice/globalHotnessScore/finalFeedScore/isDeleted`; thêm `outfit`, `isFollowingAuthor`, `status`; **thông tin tác giả gộp vào `post.user`** (breaking) |
| Sửa response Comment | Thêm `replyCount`, `isDeleted`; **thông tin tác giả gộp vào `comment.user`** (breaking) |
| Thêm mới | Follow, trang profile công khai, search `q`, feed tab explore/following + sort hot/latest |
| Sửa response Likes | `{ id, ... }` → `CommunityUserRes` phân trang (`userId`, thêm `gender?`) |
| Admin | Giữ moderation post/comment; bỏ tab `post-items`; response thêm `status`/`moderatedBy`/`hiddenReason`/`postId`, có `user` lồng |
| Bỏ | Mọi type/trường liên quan resale/transfer (`transferState`, `buyerUserId`, `soldAt`, `declinedAt`, `itemCondition`, `post-items`) |
| Giới hạn | `title`≤150, `content`≤5000, comment≤1000, ≤10 media, đuôi ảnh/video theo danh sách, ảnh≤10MB, video≤100MB/60s; `resourceType` sai → 400; rate limit bài/comment/follow/like (xem §3.12) |
| Upload | Gửi đúng 6 field lên form Cloudinary; thêm field → bị từ chối (xem §3.5) |

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
| `createPost` | `POST /posts` | body đổi (xem §3.2) |
| axios `baseURL`/credentials | — | Không đổi |

> **CẢNH BÁO — `getPostUploadSignature` KHÔNG còn thuộc nhóm giữ nguyên.** Path + method không đổi,
> nhưng **response thêm `allowedFormats`, `publicId` vắng hẳn, `resourceType` sai → 400, và
> form upload phải gửi đúng 6 field**. Bắt buộc đọc [§3.5](#35-upload-signature).

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

> **MỚI — có thay đổi so với bản 26/09:** xem [§0.1 N1, N5](#01-yêu-cầu-mới--bắt-buộc-làm-thêm).
> FE đã triển khai từ bản cũ thì **bỏ qua mục này sẽ thiếu 2 field bắt buộc**.

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
  bytes?: number;           // MỚI — kích thước file (byte), > 0
  durationSeconds?: number; // MỚI — thời lượng video (giây), > 0, chỉ dùng cho video
}
```
`bytes` và `durationSeconds` lấy từ **response upload** của Cloudinary (`bytes` và `duration`).
`duration` là số thập phân (VD `3.041267`) — gửi **nguyên văn**, không cần làm tròn.
Quy tắc validate: `postType='outfit'` ⇒ bắt buộc `outfitId` (thuộc sở hữu người đăng);
`postType='media'` ⇒ không gửi `outfitId`; phải có `content` khác rỗng **hoặc** ≥1 media.
Giới hạn (media/ký tự/kích thước/rate limit) chi tiết tại §3.12.

> **Validate `mediaUrl` (mới, backend có enforce):** phần mở rộng ở cuối path phải thuộc danh sách
> cho phép của `mediaType` → sai thì **400**. Bỏ qua query string/fragment, không phân biệt hoa
> thường. URL **không có** phần mở rộng vẫn được chấp nhận (cho các URL delivery dạng `f_auto`).
> - `image`: `jpg`, `jpeg`, `png`, `webp`, `avif`, `gif`
> - `video`: `mp4`, `webm`, `mov`, `m4v`, `mkv`, `avi`
>
> **Validate `bytes` / `durationSeconds` (mới):** gửi lại số liệu từ response Cloudinary
> (`bytes`, và `duration` cho video). Vượt ngưỡng → **400**. Bỏ trống = **không kiểm tra**
> (xem giới hạn đã biết tại §3.12).

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

> **ĐÂY LÀ MỤC THAY ĐỔI NHIỀU NHẤT** — 3 yêu cầu mới (N2, N3, N4) + 1 chỗ bản cũ sai (S3).
> Xem [§0.1](#01-yêu-cầu-mới--bắt-buộc-làm-thêm) và [§0.2](#02-sửa-sai-trong-bản-cũ--fe-có-thể-đã-làm-theo-cần-sửa-lại).
> Endpoint này **không** thuộc nhóm "giữ nguyên" ở §2 nữa.

```ts
// GET /posts/upload-signature?resourceType=image|video
interface UploadSignatureResult {
  signature: string; timestamp: number; apiKey: string;
  publicId?: string;                 // VẮNG với endpoint post — xem bảng bên dưới
  folder: string;
  resourceType: 'image' | 'video';   // MỚI — FE dùng để chọn endpoint upload
  allowedFormats: string;            // MỚI — "jpg,jpeg,png,webp,avif,gif" | "mp4,webm,mov,m4v,mkv,avi"
}
```
FE build URL upload Cloudinary: `https://api.cloudinary.com/v1_1/{cloud}/{resourceType}/upload`.

> **Form upload = các field đã ký + `file` + `api_key` + `signature`.** Endpoint này ký
> 3 field (`timestamp`, `folder`, `allowed_formats`) nên form gửi **đúng 6 field**:
>
> | Field | Giá trị | Bắt buộc |
> |---|---|---|
> | `file` | file ảnh/video | Có |
> | `api_key` | `apiKey` từ response | Có |
> | `timestamp` | `timestamp` từ response | Có |
> | `signature` | `signature` từ response | Có |
> | `folder` | `folder` từ response | Có |
> | `allowedFormats` | `allowedFormats` từ response, **nguyên văn** | Có |
>
> Không tự thêm field nào khác (`public_id`, `overwrite`, `eager`, `transformation`, `tags`…):
> Cloudinary **tự tính lại chữ ký từ đúng các field client gửi** → thêm, bớt hay đổi giá trị
> đều khiến **upload bị từ chối**. Riêng `allowedFormats` phải gửi nguyên văn — đây là cơ chế
> khiến giới hạn định dạng không lách được.
>
> `resourceType` sai giá trị (không phải `image`/`video`) → **400**, không fallback về `image`.

#### Bảng `publicId` — xuất hiện ở 3 nơi, 3 nghĩa khác nhau

| Field | Payload | Nghĩa | FE làm gì |
|---|---|---|---|
| `PostRes.publicId` | response `GET /posts` | public ID của **bài đăng** (32 ký tự) | dùng làm URL `/posts/{publicId}` |
| `PostMediaReq.publicId` | request `POST`/`PUT /posts` | `public_id` Cloudinary **sinh ra sau khi upload** | lấy từ response upload rồi gửi về BE |
| `UploadSignatureResult.publicId` | response `GET /posts/upload-signature` | **vắng** (BE không ký `public_id` cho endpoint này) | ❌ **không** gửi lên Cloudinary |

> **Luồng đầy đủ:** lấy signature → upload lên Cloudinary **không truyền `public_id`** (để
> Cloudinary tự sinh) → đọc `public_id`, `secure_url`, `bytes` (+ `duration` nếu video) từ
> **response upload** → gửi các giá trị đó trong `media[]` của `POST /posts`.
>
> `mediaUrl` phải lấy **`secure_url`** (https), **không** phải `url` (http) mà Cloudinary cũng trả.
> `PostMediaReq.publicId` là **optional**; backend hiện **lưu nhưng chưa dùng** (chưa có logic dọn
> media khi xoá bài) — nên gửi lên để sẵn sàng, nhưng không được phụ thuộc vào nó.

### 3.6 Feed params

`GET /posts` nhận thêm:
- `type`: `explore` (default) | `following`
- `sort`: `hot` (default) | `latest`
- `postType`: `outfit` | `media` (optional)
- `username`, `page`, `limit` (giữ)

Guest gọi `type=following` → 401. `following` chỉ chứa bài của người đang theo dõi (không gồm bài của chính mình).

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

> **MỚI — bảng này đã viết lại** so với bản 26/09: xem [S1, S2](#02-sửa-sai-trong-bản-cũ--fe-có-thể-đã-làm-theo-cần-sửa-lại).
> Đáng chú ý nhất: bản cũ ghi Cloudinary enforce kích thước/thời lượng, điều này **không đúng**.

FE nên validate client-side để tránh 400/429. Giá trị là mặc định trong `configs/config.yaml` (block `community`).

| Trường / hành động | Giới hạn | HTTP khi vi phạm |
|---|---|---|
| `title` | ≤ **150** ký tự | 400 |
| `content` (bài) | ≤ **5000** ký tự | 400 |
| `content` (bình luận, đã trim) | ≤ **1000** ký tự, không rỗng | 400 |
| `media` | ≤ **10** phần tử/bài | 400 |
| Đuôi file ảnh (`mediaType=image`) | `jpg`, `jpeg`, `png`, `webp`, `avif`, `gif` | 400 (URL sai đuôi) / từ chối upload |
| Đuôi file video (`mediaType=video`) | `mp4`, `webm`, `mov`, `m4v`, `mkv`, `avi` | 400 (URL sai đuôi) / từ chối upload |
| Ảnh — kích thước (`bytes`) | ≤ **10 MB** | 400 |
| Video — kích thước (`bytes`) | ≤ **100 MB** | 400 |
| Video — thời lượng (`durationSeconds`) | ≤ **60 giây** | 400 |
| Nội dung bài | phải có `content` khác rỗng **hoặc** ≥1 media | 400 |
| Tạo bài | ≤ **10** bài/giờ/user | 429 |
| Bình luận | ≤ **60** bình luận/giờ/user | 429 |
| Follow / Unfollow | ≤ **200** thao tác/giờ/user | 429 |
| Like / Unlike | ≤ **600** thao tác/giờ/user | 429 |
| Tìm kiếm bài | chỉ so khớp **200** ký tự đầu của `content` | — |
| Tìm kiếm user (server) | tối đa **500** ứng viên từ Account | — |
| Phân trang | `page≥1`, `limit` mặc định **20**, tối đa **100** | — |

- **Định dạng file** được enforce ở **hai tầng**: backend kiểm tra đuôi trong `mediaUrl` (400),
  và Cloudinary từ chối upload nhờ `allowed_formats` đã ký. Tầng Cloudinary mới là tầng không lách được.
  URL media **không có** đuôi vẫn được chấp nhận (cho URL delivery dạng `f_auto`).
- **Kích thước/thời lượng là dữ liệu client tự khai báo** qua `bytes` / `durationSeconds`.
  Cloudinary **không** cung cấp tham số giới hạn kích thước/thời lượng để ký vào signature
  (chỉ `allowed_formats` là ký được), nên backend **chỉ kiểm tra được khi FE gửi các field này**.
  Bỏ trống = không kiểm tra. → FE **phải** validate phía client trước khi upload để không mất UX,
  và luôn gửi `bytes`/`durationSeconds` khi tạo bài.
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
- [ ] Upload: truyền `resourceType`, xử lý video; dùng `resourceType` build URL Cloudinary;
  gửi **đúng 6 field** (`file`, `api_key`, `timestamp`, `signature`, `folder`, `allowedFormats`),
  **không** thêm `public_id`/`eager`/`tags`; dùng `secure_url` cho `mediaUrl` và gửi
  `public_id` từ response upload.
- [ ] Gửi kèm `bytes` (và `durationSeconds` nguyên văn, không làm tròn) khi tạo/sửa bài.
- [ ] Validate client-side theo giới hạn §3.12 (title ≤150, content ≤5000, comment ≤1000, ≤10 media,
  đuôi ảnh/video đúng danh sách, ảnh ≤10MB, video ≤100MB/60s) và xử lý 429.
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
16. **Chỉ gửi đúng 6 field lên form upload** — thiếu `allowedFormats` hoặc thừa bất kỳ field nào
    (kể cả `public_id`) đều làm lệch chữ ký và upload bị Cloudinary từ chối. Xem bảng 3 nghĩa
    `publicId` tại §3.5.
17. **`bytes`/`durationSeconds` là tự khai báo** — backend chỉ kiểm tra khi FE gửi. Bỏ trống = né check.
    Phía Cloudinary không có cơ chế chặn kích thước/thời lượng, nên validate client-side là bắt buộc
    để bảo vệ UX (và quota Cloudinary), không phải tuỳ chọn.
18. **`mediaUrl` sai đuôi → 400** — URL không có đuôi vẫn OK. Ảnh chỉ nhận
    `jpg/jpeg/png/webp/avif/gif` (`.svg`, `.heic`, `.bmp` bị từ chối); video chỉ nhận
    `mp4/webm/mov/m4v/mkv/avi` (ảnh `.mp4` bị từ chối).
19. **`mediaUrl` phải là `secure_url`** — Cloudinary trả cả `url` (http) và `secure_url` (https);
    chỉ dùng `secure_url`.
