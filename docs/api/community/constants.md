# Hằng số Nghiệp vụ Cộng đồng (Community Constants)

Các hằng số dùng trong API module Community (bài đăng, like, bình luận, theo dõi, tìm kiếm, kiểm duyệt).
Đường dẫn package gốc: `services/commerce/internal/modules/community/domain/constants/`.

---

## 1. Loại bài đăng (PostType)

*   **Đường dẫn package:** `.../domain/constants/posttype`
*   **Các giá trị hợp lệ:**
    *   `outfit`: Bài đăng gắn với một bộ phối đồ do người dùng sở hữu (`outfitId` bắt buộc).
    *   `media`: Bài đăng ảnh/video tự do (không gắn outfit).

## 2. Trạng thái bài đăng & bình luận (PostStatus)

*   **Đường dẫn package:** `.../domain/constants/poststatus`
*   **Các giá trị hợp lệ:**
    *   `published`: Đang hiển thị công khai.
    *   `hidden`: Bị ẩn (admin ẩn hoặc tác giả bị khóa); tác giả vẫn xem được bài của mình, người khác nhận 404.
    *   `deleted`: Đã xóa mềm; không xuất hiện ở API công khai.

> Trường `status` có mặt trong `PostRes` (công khai) và `AdminPostRes`/`AdminCommentRes` (quản trị).

## 3. Loại media (MediaType)

*   **Đường dẫn package:** `.../domain/constants/mediatype`
*   **Các giá trị hợp lệ:**
    *   `image`: Ảnh (jpg/png/webp).
    *   `video`: Video (mp4/webm, tối đa 60 giây).

## 4. Loại bảng tin (FeedType)

*   **Đường dẫn package:** `.../domain/constants/feedtype`
*   **Các giá trị hợp lệ:**
    *   `explore`: Bảng tin khám phá (mặc định).
    *   `following`: Chỉ bài của người đang theo dõi (+ bài của chính mình); yêu cầu đăng nhập.

## 5. Sắp xếp bảng tin (FeedSort)

*   **Đường dẫn package:** `.../domain/constants/feedsort`
*   **Các giá trị hợp lệ:**
    *   `hot`: Xếp hạng theo độ nóng tương tác (mặc định).
    *   `latest`: Mới nhất trước.

## 6. Loại danh sách theo dõi (FollowType)

*   **Đường dẫn package:** `.../domain/constants/followtype`
*   **Query param `type` trên `GET /users/{username}/follows`:**
    *   `following`: Những người mà user đang theo dõi.
    *   `followers`: Những người đang theo dõi user.
    *   Bỏ trống: trả **cả hai**, phân biệt bằng trường `relation`.
*   **Giá trị `relation` trả về cho mỗi mục:**
    *   `following`: Mục thuộc nhóm đang theo dõi.
    *   `follower`: Mục thuộc nhóm người theo dõi.

## 7. Lý do ẩn bài (HiddenReason)

*   **Đường dẫn package:** `.../domain/constants/hiddenreason`
*   **Các giá trị hợp lệ:**
    *   `admin`: Do quản trị viên ẩn/xóa (khôi phục được).
    *   `author_disabled`: Do tài khoản tác giả bị khóa (tự khôi phục khi mở lại).

## 8. Loại kết quả tìm kiếm

*   **Nguồn:** giá trị nội bộ của use case tìm kiếm (`all` | `users` | `posts`).
*   **Query param `type` trên `GET /search`:**
    *   `all` (mặc định): Trả cả người dùng và bài đăng.
    *   `users`: Chỉ trả người dùng (nhóm `posts` rỗng).
    *   `posts`: Chỉ trả bài đăng (nhóm `users` rỗng).

---

## 9. Giới hạn cấu hình (config `community`)

Các ngưỡng dưới đây điều khiển validate và rate limit; giá trị là mặc định trong `configs/config.yaml`.

| Khóa config | Mặc định | Ý nghĩa |
|---|---|---|
| `max_media_per_post` | 10 | Số media tối đa mỗi bài. |
| `max_image_bytes` | 10485760 (10 MB) | Kích thước ảnh tối đa. |
| `max_video_bytes` | 104857600 (100 MB) | Kích thước video tối đa. |
| `max_video_duration_seconds` | 60 | Thời lượng video tối đa (giây). |
| `max_title_length` | 150 | Độ dài tiêu đề tối đa (ký tự). |
| `max_content_length` | 5000 | Độ dài nội dung bài tối đa (ký tự). |
| `max_comment_length` | 1000 | Độ dài bình luận tối đa (ký tự). |
| `post_rate_limit_per_hour` | 10 | Số bài tối đa/giờ/user. |
| `comment_rate_limit_per_hour` | 60 | Số bình luận tối đa/giờ/user. |
| `follow_rate_limit_per_hour` | 200 | Số thao tác follow tối đa/giờ/user. |
| `like_rate_limit_per_hour` | 600 | Số thao tác like tối đa/giờ/user. |
| `search_content_prefix_chars` | 200 | Số ký tự đầu của nội dung dùng để so khớp tìm kiếm bài. |
| `search_user_candidate_cap` | 500 | Số ứng viên người dùng tối đa lấy từ Account khi tìm kiếm. |
