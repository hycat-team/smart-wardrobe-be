# Admin Community API Specs

Tài liệu đặc tả các API kiểm duyệt nội dung cộng đồng dành cho quản trị viên (Admin). Hằng số trạng thái/lý do ẩn tham chiếu tại [constants.md](constants.md). Luồng người dùng thường xem tại [community-api.md](community-api.md).

> Tiền tố: `/api/v1/admin`. Yêu cầu quyền **admin**. Mọi hành động ghi được ghi audit log.
> Envelope: `{ message, data }` — `data` **vắng mặt** khi không có payload.

---

## Flow 1: Kiểm duyệt bài đăng

### 1. Lấy danh sách bài đăng (quản trị)

- **Endpoint:** `GET /api/v1/admin/posts`
- **Tác nhân (Actor):** Quản trị viên (Admin).
- **Đối tượng ảnh hưởng:** Đọc bảng `community_posts` (mọi trạng thái).
- **Query Parameters:**
    - `q` (string, optional) — Từ khóa tìm kiếm.
    - `status` (string, optional) — `published` | `hidden` | `deleted`. Tham chiếu [constants.md#2](constants.md#2-trạng-thái-bài-đăng--bình-luận-poststatus).
    - `page`, `limit` (int, optional) — Phân trang.
- **Response:** `200 OK` — `PaginationResult<AdminPostRes>`.

### 2. Ẩn bài đăng

- **Endpoint:** `PATCH /api/v1/admin/posts/{postID}/hide`
- **Tác nhân (Actor):** Admin.
- **Đối tượng ảnh hưởng:** Chuyển `status='hidden'`, `hidden_reason='admin'`, ghi nhận `moderated_by`.
- **Mô tả:** Ẩn bài vi phạm; có thể khôi phục sau.
- **Response:** `200 OK` — (không có `data`).

### 3. Khôi phục bài đăng

- **Endpoint:** `PATCH /api/v1/admin/posts/{postID}/restore`
- **Tác nhân (Actor):** Admin.
- **Đối tượng ảnh hưởng:** Chuyển bài về `status='published'` nếu trước đó do admin ẩn/xóa.
- **Mô tả:** Không khôi phục được bài do tác giả tự xóa → `400`.
- **Response:** `200 OK` — (không có `data`).

### 4. Xóa bài đăng

- **Endpoint:** `DELETE /api/v1/admin/posts/{postID}`
- **Tác nhân (Actor):** Admin.
- **Đối tượng ảnh hưởng:** Xóa mềm `status='deleted'` (admin có thể khôi phục).
- **Response:** `200 OK` — (không có `data`).

---

## Flow 2: Kiểm duyệt bình luận

### 1. Lấy danh sách bình luận (quản trị)

- **Endpoint:** `GET /api/v1/admin/comments`
- **Tác nhân (Actor):** Admin.
- **Đối tượng ảnh hưởng:** Đọc bảng `community_post_comments` (mọi trạng thái).
- **Query Parameters:**
    - `q` (string, optional) — Từ khóa tìm kiếm.
    - `page`, `limit` (int, optional) — Phân trang.
- **Response:** `200 OK` — `PaginationResult<AdminCommentRes>`.

### 2. Ẩn bình luận

- **Endpoint:** `PATCH /api/v1/admin/comments/{commentID}/hide`
- **Tác nhân (Actor):** Admin.
- **Đối tượng ảnh hưởng:** Chuyển bình luận `status='hidden'`.
- **Mô tả:** Ẩn bình luận gốc sẽ **cascade** ẩn/hiện theo cây phản hồi con.
- **Response:** `200 OK` — (không có `data`).

### 3. Khôi phục bình luận

- **Endpoint:** `PATCH /api/v1/admin/comments/{commentID}/restore`
- **Tác nhân (Actor):** Admin.
- **Đối tượng ảnh hưởng:** Khôi phục bình luận do admin ẩn/xóa.
- **Mô tả:** Không khôi phục được bình luận do tác giả tự xóa → `400`.
- **Response:** `200 OK` — (không có `data`).

### 4. Xóa bình luận

- **Endpoint:** `DELETE /api/v1/admin/comments/{commentID}`
- **Tác nhân (Actor):** Admin.
- **Đối tượng ảnh hưởng:** Xóa mềm bình luận (cascade lên phản hồi con).
- **Response:** `200 OK` — (không có `data`).

---

## Cấu trúc DTO quản trị

### AdminPostRes

Kế thừa toàn bộ [PostRes](community-api.md#postres) (bao gồm `user` lồng và `status`), bổ sung:

| Field | JSON | Kiểu | Ý nghĩa |
|-------|------|------|---------|
| ModeratedBy | `moderatedBy` | uuid (omitempty) | ID admin xử lý (vắng nếu tác giả tự xóa) |
| HiddenReason | `hiddenReason` | string (omitempty) | Lý do ẩn. Tham chiếu [constants.md#7](constants.md#7-lý-do-ẩn-bài-hiddenreason) |

### AdminCommentRes

Kế thừa toàn bộ [CommentRes](community-api.md#commentres) (bao gồm `user` lồng), bổ sung:

| Field | JSON | Kiểu | Ý nghĩa |
|-------|------|------|---------|
| PostID | `postId` | uuid | ID bài đăng chứa bình luận |
| Status | `status` | string | [constants.md#2](constants.md#2-trạng-thái-bài-đăng--bình-luận-poststatus) |

---

## Ghi chú nghiệp vụ

- **Khóa/mở tài khoản tác giả:** Khi trạng thái người dùng thay đổi (admin khóa/mở), hệ thống phát sự kiện `identity.user.status.changed`; bài của người dùng bị khóa chuyển `status='hidden'` với `hidden_reason='author_disabled'` và tự khôi phục khi mở lại.
- **Cascade outfit:** Khi outfit bị xóa, bài đăng outfit liên quan được xử lý qua sự kiện `wardrobe.outfit.deleted`.
- **Audit:** Mọi thao tác ẩn/khôi phục/xóa đều ghi nhật ký quản trị (`admin.audit.log`).
