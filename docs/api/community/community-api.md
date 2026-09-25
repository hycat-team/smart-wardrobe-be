# Community API Specs

Tài liệu đặc tả các API cộng đồng (bài đăng, bảng tin, like, bình luận, theo dõi, trang cá nhân, tìm kiếm) thuộc module Community (service **commerce**). Tất cả các giá trị hằng số sử dụng trong request/response tham chiếu tại [constants.md](constants.md). API kiểm duyệt của quản trị viên xem tại [admin-community-api.md](admin-community-api.md).

> Tiền tố: `/api/v1`. Envelope: `{ message, data }` — `data` **vắng mặt** khi không có payload.
> Envelope phân trang: `{ items, metadata: { page, limit, totalItems, totalPages } }` (mặc định `page=1`, `limit=20`, tối đa 100).

---

## Flow 1: Đăng bài và xem bảng tin

### 1. Lấy bảng tin cộng đồng

- **Endpoint:** `GET /api/v1/posts`
- **Tác nhân (Actor):** Khách hàng (User) hoặc Khách vãng lai (Guest).
- **Đối tượng ảnh hưởng:** Đọc bảng `community_posts` (+ media, outfit).
- **Mô tả:** Trả về bảng tin cộng đồng có phân trang, hỗ trợ loại bảng tin, sắp xếp, lọc theo loại bài và theo tên người dùng. Chỉ trả bài `published` (và bài `hidden` của chính người xem).
- **Query Parameters:**
    - `type` (string, optional) — `explore` (mặc định) | `following`. Tham chiếu [constants.md#4-loại-bảng-tin-feedtype](constants.md#4-loại-bảng-tin-feedtype). Guest dùng `following` → `401`.
    - `sort` (string, optional) — `hot` (mặc định) | `latest`. Tham chiếu [constants.md#5-sắp-xếp-bảng-tin-feedsort](constants.md#5-sắp-xếp-bảng-tin-feedsort).
    - `postType` (string, optional) — `outfit` | `media`. Tham chiếu [constants.md#1-loai-bai-dang-posttype](constants.md#1-loại-bài-đăng-posttype).
    - `username` (string, optional) — Lọc theo tác giả.
    - `page`, `limit` (int, optional) — Phân trang.
- **Response:** `200 OK` — `PaginationResult<PostRes>`.

### 2. Xem chi tiết một bài đăng

- **Endpoint:** `GET /api/v1/posts/{postPublicID}`
- **Tác nhân (Actor):** User hoặc Guest.
- **Đối tượng ảnh hưởng:** Đọc một bản ghi `community_posts`.
- **Mô tả:** Trả về chi tiết bài đăng theo mã công khai. Bài `deleted` → `404`; bài `hidden` chỉ tác giả xem được (kèm `status='hidden'`), người khác nhận `404`.
- **Response:** `200 OK` — `PostRes`.

### 3. Lấy chữ ký tải media bài đăng

- **Endpoint:** `GET /api/v1/posts/upload-signature`
- **Tác nhân (Actor):** Người dùng đã đăng nhập.
- **Đối tượng ảnh hưởng:** Không thay đổi dữ liệu nghiệp vụ.
- **Mô tả:** Lấy Cloudinary signature để client tải ảnh/video trực tiếp lên Cloudinary.
- **Query Parameters:**
    - `resourceType` (string, optional) — `image` (mặc định) | `video`. Tham chiếu [constants.md#3-loai-media-mediatype](constants.md#3-loại-media-mediatype).
- **Response:** `200 OK` — `UploadSignatureResult`:

| Field | JSON | Kiểu | Ý nghĩa |
|-------|------|------|---------|
| Signature | `signature` | string | Chữ ký Cloudinary |
| Timestamp | `timestamp` | int64 | Thời điểm ký |
| ApiKey | `apiKey` | string | API key Cloudinary |
| PublicID | `publicId` | string | Mã media (có thể rỗng) |
| Folder | `folder` | string | Thư mục lưu media |
| ResourceType | `resourceType` | string | `image` hoặc `video` |

### 4. Tạo bài đăng

- **Endpoint:** `POST /api/v1/posts`
- **Tác nhân (Actor):** Người dùng đã đăng nhập.
- **Đối tượng ảnh hưởng:** Tạo bản ghi `community_posts` và các bản ghi media.
- **Mô tả:** Tạo bài `outfit` (bắt buộc `outfitId` thuộc sở hữu người đăng) hoặc bài `media`. Phải có `content` khác rỗng **hoặc** ít nhất 1 media. Áp dụng rate limit `post_rate_limit_per_hour`.
- **Request Body:**
    ```json
    {
        "postType": "outfit",
        "title": "Bộ đồ mùa thu",
        "content": "Phối đồ đi làm mùa thu",
        "outfitId": "3f2b6c1e-9a44-4c2e-8c11-0b2a4d5e6f70",
        "media": [
            { "mediaType": "image", "mediaUrl": "https://res.cloudinary.com/.../a.jpg", "publicId": "posts/a", "sortOrder": 0 }
        ]
    }
    ```
    - `postType` (required) — [constants.md#1](constants.md#1-loại-bài-đăng-posttype).
    - `title` (optional) — tối đa `max_title_length` (150).
    - `content` (optional) — tối đa `max_content_length` (5000).
    - `outfitId` (optional) — bắt buộc khi `postType='outfit'`; phải thuộc sở hữu người đăng.
    - `media` (optional) — tối đa `max_media_per_post` (10); mỗi media có `mediaType` [constants.md#3](constants.md#3-loại-media-mediatype).
- **Response:** `201 Created` — `PostRes`.

### 5. Cập nhật bài đăng

- **Endpoint:** `PUT /api/v1/posts/{postPublicID}`
- **Tác nhân (Actor):** Tác giả bài đăng.
- **Đối tượng ảnh hưởng:** Cập nhật `community_posts` và thay thế danh sách media.
- **Mô tả:** Cập nhật `title`, `content`, `outfitId` (với bài outfit) và media. **Không** cho đổi `postType`. Không phải chủ bài → `403`.
- **Request Body:** giống tạo bài nhưng **không có** `postType`.
- **Response:** `200 OK` — `PostRes`.

### 6. Xóa bài đăng

- **Endpoint:** `DELETE /api/v1/posts/{postPublicID}`
- **Tác nhân (Actor):** Tác giả bài đăng.
- **Đối tượng ảnh hưởng:** Xóa mềm `community_posts` (`status='deleted'`, không thể khôi phục bởi tác giả).
- **Response:** `200 OK` — (không có `data`).

---

## Flow 2: Like và bình luận

### 1. Thích / Bỏ thích bài đăng

- **Endpoint:** `PUT /api/v1/posts/{postPublicID}/like`
- **Tác nhân (Actor):** Người dùng đã đăng nhập.
- **Đối tượng ảnh hưởng:** Tạo/xóa bản ghi `community_post_likes`, cập nhật bộ đếm like.
- **Mô tả:** Idempotent; chỉ tác động khi trạng thái thay đổi. Áp dụng rate limit `like_rate_limit_per_hour`.
- **Request Body:**
    ```json
    { "isLiked": true }
    ```
- **Response:** `200 OK` — (không có `data`).

### 2. Lấy danh sách người đã thích bài

- **Endpoint:** `GET /api/v1/posts/{postPublicID}/likes`
- **Tác nhân (Actor):** User hoặc Guest.
- **Mô tả:** Trả về danh sách người dùng đã thích bài, có phân trang.
- **Query Parameters:** `page`, `limit` (optional).
- **Response:** `200 OK` — `PaginationResult<CommunityUserRes>`.

### 3. Lấy bình luận gốc của bài

- **Endpoint:** `GET /api/v1/posts/{postPublicID}/comments`
- **Tác nhân (Actor):** User hoặc Guest.
- **Mô tả:** Trả về **mảng** bình luận gốc (không phân trang), mới nhất trước, kèm `replyCount`. Bình luận gốc đã xóa nhưng còn phản hồi vẫn trả về với `isDeleted=true` và `content=""`.
- **Response:** `200 OK` — `CommentRes[]`.

### 4. Lấy phản hồi của một bình luận gốc

- **Endpoint:** `GET /api/v1/posts/{postPublicID}/comments/{commentID}/replies`
- **Tác nhân (Actor):** User hoặc Guest.
- **Mô tả:** Trả về **mảng** phản hồi (không phân trang), cũ nhất trước.
- **Response:** `200 OK` — `CommentRes[]`.

### 5. Thêm bình luận hoặc phản hồi

- **Endpoint:** `POST /api/v1/posts/{postPublicID}/comments`
- **Tác nhân (Actor):** Người dùng đã đăng nhập.
- **Đối tượng ảnh hưởng:** Tạo bản ghi `community_post_comments`, tăng `comment_count`.
- **Mô tả:** Thêm bình luận gốc hoặc phản hồi. Phản hồi vào cấp 2 tự động gắn lên bình luận gốc (tối đa 2 cấp). Áp dụng rate limit `comment_rate_limit_per_hour`; giới hạn `max_comment_length` (1000).
- **Request Body:**
    ```json
    { "content": "Bộ này đẹp quá!", "parentCommentId": null }
    ```
- **Response:** `201 Created` — `CommentRes`.

### 6. Cập nhật bình luận

- **Endpoint:** `PUT /api/v1/posts/{postPublicID}/comments/{commentID}`
- **Tác nhân (Actor):** Chủ bình luận.
- **Đối tượng ảnh hưởng:** Cập nhật `content` bình luận. Không phải chủ → `403`.
- **Request Body:** `{ "content": "Nội dung mới" }`.
- **Response:** `200 OK` — `CommentRes`.

### 7. Xóa bình luận

- **Endpoint:** `DELETE /api/v1/posts/{postPublicID}/comments/{commentID}`
- **Tác nhân (Actor):** Chủ bình luận.
- **Đối tượng ảnh hưởng:** Xóa mềm bình luận (`status='deleted'`), giảm `comment_count`.
- **Response:** `200 OK` — (không có `data`).

---

## Flow 3: Theo dõi người dùng và trang cá nhân công khai

### 1. Theo dõi / Bỏ theo dõi người dùng

- **Endpoint:** `PUT /api/v1/users/{username}/follow`
- **Tác nhân (Actor):** Người dùng đã đăng nhập.
- **Đối tượng ảnh hưởng:** Tạo/xóa bản ghi `community_follows`, cập nhật thống kê follower/following.
- **Mô tả:** Idempotent. Không cho tự theo dõi chính mình → `400`. Người dùng đích không tồn tại → `404`. Áp dụng rate limit `follow_rate_limit_per_hour`.
- **Request Body:**
    ```json
    { "isFollowing": true }
    ```
- **Response:** `200 OK` — (không có `data`).

### 2. Xem trang cá nhân công khai

- **Endpoint:** `GET /api/v1/users/{username}`
- **Tác nhân (Actor):** User hoặc Guest.
- **Đối tượng ảnh hưởng:** Đọc hồ sơ người dùng và thống kê follow/bài đăng.
- **Mô tả:** Trả về thông tin công khai kèm số lượng follower/following/bài đăng và trạng thái theo dõi của người xem.
- **Response:** `200 OK` — `PublicProfileRes`:
    ```json
    {
        "user": { "userId": "uuid", "username": "an", "firstName": "An", "lastName": "Nguyễn", "avatarUrl": "https://...", "gender": 1 },
        "stats": { "followerCount": 12, "followingCount": 3, "postCount": 5 },
        "isFollowing": false,
        "isMe": false
    }
    ```

### 3. Lấy bài đăng của người dùng

- **Endpoint:** `GET /api/v1/users/{username}/posts`
- **Tác nhân (Actor):** User hoặc Guest.
- **Mô tả:** Trả về bài `published` của người dùng, mới nhất trước, phân trang.
- **Query Parameters:** `page`, `limit` (optional).
- **Response:** `200 OK` — `PaginationResult<PostRes>`.

### 4. Lấy danh sách theo dõi của người dùng

- **Endpoint:** `GET /api/v1/users/{username}/follows`
- **Tác nhân (Actor):** User hoặc Guest.
- **Mô tả:** Trả về danh sách following/followers, mới nhất trước, phân trang.
- **Query Parameters:**
    - `type` (string, optional) — `following` | `followers`; bỏ trống trả cả hai. Tham chiếu [constants.md#6-loại-danh-sách-theo-dõi-followtype](constants.md#6-loại-danh-sách-theo-dõi-followtype).
    - `q` (string, optional) — Lọc theo username/họ tên.
    - `page`, `limit` (optional).
- **Response:** `200 OK` — `PaginationResult<FollowUserRes>`.

---

## Flow 4: Tìm kiếm người dùng và bài đăng

### 1. Tìm kiếm

- **Endpoint:** `GET /api/v1/search`
- **Tác nhân (Actor):** User hoặc Guest.
- **Đối tượng ảnh hưởng:** Đọc người dùng (Account) và bài đăng công khai.
- **Mô tả:** Tìm người dùng theo username/họ tên và bài đăng theo nội dung (so khớp `search_content_prefix_chars` (200) ký tự đầu). Khi `q` rỗng → cả hai nhóm rỗng (không lỗi). Trả về 2 nhóm phân trang **độc lập**.
- **Query Parameters:**
    - `q` (string, optional) — Từ khóa.
    - `type` (string, optional) — `all` (mặc định) | `users` | `posts`. Tham chiếu [constants.md#8-loại-kết-quả-tìm-kiếm](constants.md#8-loại-kết-quả-tìm-kiếm).
    - `postType` (string, optional) — `outfit` | `media` (lọc nhóm bài đăng).
    - `page`, `limit` (optional).
- **Response:** `200 OK` — `SearchRes`:
    ```json
    {
        "users": { "items": [CommunityUserRes], "metadata": { "page": 1, "limit": 20, "totalItems": 0, "totalPages": 0 } },
        "posts": { "items": [PostRes], "metadata": { "page": 1, "limit": 20, "totalItems": 0, "totalPages": 0 } }
    }
    ```
- **Ghi chú thứ tự & giới hạn:** Người dùng sắp theo số follower giảm dần rồi username tăng dần; chỉ lấy tối đa `search_user_candidate_cap` (500) ứng viên từ Account nên ngoài ngưỡng có thể thiếu. Bài đăng sắp theo `hot`.

---

## Cấu trúc DTO dùng chung

### CommunityUserRes

Thông tin người dùng đã chuẩn hóa, dùng làm object lồng `user` (trong `PostRes`/`CommentRes`/`FollowUserRes`) hoặc dùng trực tiếp cho danh sách thuần người dùng (likes, search, profile).

| Field | JSON | Kiểu | Ý nghĩa |
|-------|------|------|---------|
| UserID | `userId` | uuid | ID người dùng |
| Username | `username` | string | Tên đăng nhập |
| FirstName | `firstName` | string (omitempty) | Tên |
| LastName | `lastName` | string (omitempty) | Họ |
| AvatarURL | `avatarUrl` | string (omitempty) | Ảnh đại diện (có thể vắng) |
| Gender | `gender` | int (omitempty) | Giới tính (chỉ có khi biết) |

### PostRes

| Field | JSON | Kiểu | Ý nghĩa |
|-------|------|------|---------|
| ID | `id` | uuid | ID nội bộ |
| PublicID | `publicId` | string | Mã công khai 32 ký tự |
| User | `user` | CommunityUserRes | Tác giả |
| PostType | `postType` | string | [constants.md#1](constants.md#1-loại-bài-đăng-posttype) |
| Status | `status` | string | [constants.md#2](constants.md#2-trạng-thái-bài-đăng--bình-luận-poststatus) |
| Title | `title` | string (nullable) | Tiêu đề |
| Content | `content` | string | Nội dung |
| Outfit | `outfit` | OutfitBriefRes (omitempty) | Bộ phối đồ (bài `outfit`) |
| LikeCount | `likeCount` | int | Số like |
| CommentCount | `commentCount` | int | Số bình luận |
| IsLiked | `isLiked` | bool | Người xem đã thích chưa |
| IsFollowingAuthor | `isFollowingAuthor` | bool | Người xem đang theo dõi tác giả chưa |
| SharePath | `sharePath` | string | Đường dẫn chia sẻ `/community/posts/{publicId}` |
| Media | `media` | PostMediaRes[] (omitempty) | Danh sách media |
| CreatedAt | `createdAt` | timestamptz | Thời điểm tạo |
| UpdatedAt | `updatedAt` | timestamptz | Thời điểm cập nhật |

- `OutfitBriefRes`: `{ id, name, coverImageUrl? }`.
- `PostMediaRes`: `{ id, mediaType, mediaUrl, publicId?, sortOrder }`.
- `outfit`/`media` dùng `omitempty` → vắng key khi không có.

### CommentRes

| Field | JSON | Kiểu | Ý nghĩa |
|-------|------|------|---------|
| ID | `id` | uuid | ID bình luận |
| User | `user` | CommunityUserRes | Tác giả |
| Content | `content` | string | Nội dung (rỗng khi xóa còn phản hồi) |
| ParentCommentID | `parentCommentId` | uuid (omitempty) | Bình luận cha; vắng khi là bình luận gốc |
| ReplyCount | `replyCount` | int64 | Số phản hồi |
| IsDeleted | `isDeleted` | bool | Đã xóa chưa |
| CreatedAt | `createdAt` | timestamptz | Thời điểm tạo |

### FollowUserRes

| Field | JSON | Kiểu | Ý nghĩa |
|-------|------|------|---------|
| User | `user` | CommunityUserRes | Người dùng |
| Relation | `relation` | string | `following` hoặc `follower` |
| FollowedAt | `followedAt` | timestamptz | Thời điểm theo dõi |

### PublicProfileRes

`{ user: CommunityUserRes, stats: { followerCount, followingCount, postCount }, isFollowing: bool, isMe: bool }`.

### SearchRes

`{ users: PaginationResult<CommunityUserRes>, posts: PaginationResult<PostRes> }`.

---

## Mã lỗi tiêu biểu

| Tình huống | HTTP | Message (vi) |
|---|---|---|
| Body/query sai định dạng, thiếu trường | 400 | (thông báo theo trường) |
| Guest gọi `following` / thao tác ghi | 401 | Vui lòng đăng nhập |
| Không phải chủ bài/bình luận | 403 | Không có quyền truy cập |
| Outfit không hợp lệ hoặc không thuộc sở hữu | 400 | Outfit không hợp lệ |
| Bài rỗng (không content/media) | 400 | Vui lòng nhập nội dung hoặc đính kèm media |
| Vượt giới hạn media/ký tự | 400 | (thông báo theo trường) |
| Tự theo dõi chính mình | 400 | Không thể theo dõi chính mình |
| Không tìm thấy bài/bình luận/người dùng | 404 | Không tìm thấy dữ liệu |
| Vượt rate limit | 429 | Quá nhiều yêu cầu |

Cấu trúc lỗi trả về (không có envelope `data`):

```json
{
    "status": 400,
    "title": "Lỗi hệ thống",
    "message": "Thông báo tiếng Việt cho người dùng",
    "errors": [{ "field": "content", "message": "..." }]
}
```
