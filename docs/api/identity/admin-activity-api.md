# Admin Activity API Specs

Tài liệu này mô tả API nhật ký hoạt động người dùng thuộc module Identity - tra cứu bởi quản trị viên.

Hợp đồng chi tiết: `specs/029-admin-observability/contracts/admin-api-delta.md` §1.

---

## Flow 6: Admin tra cứu nhật ký hoạt động người dùng

### 1. Lấy danh sách hoạt động người dùng
*   **Endpoint:** `GET /api/v1/admin/activity-events`
*   **Tác nhân (Actor):** Quản trị viên (Admin).
*   **Đối tượng ảnh hưởng:** Đọc sự kiện hoạt động (`user_activity_events`). API **chỉ đọc** — không tạo, sửa hay xoá sự kiện; bảng giữ nguyên tính chất append-only.
*   **Mô tả:** Lấy danh sách phân trang các sự kiện hoạt động (đăng nhập, làm mới phiên) kèm danh tính người dùng (email, tên hiển thị). Hỗ trợ lọc theo người dùng, loại hoạt động và khoảng ngày. Loại hoạt động tham chiếu [constants.md](constants.md#4-loại-hoạt-động-người-dùng-useractivitytype).
*   **Query params:**

| Param | Kiểu | Bắt buộc | Ghi chú |
|---|---|---|---|
| `userId` | uuid | không | sai định dạng → 400 |
| `activityType` | string | không | chỉ `login` \| `refresh`; giá trị khác → 400 |
| `fromDate` | `YYYY-MM-DD` | không | ngày lịch Việt Nam, bao gồm trọn ngày |
| `toDate` | `YYYY-MM-DD` | không | bao gồm trọn ngày; `fromDate > toDate` → 400 |
| `page` | int | không | mặc định 1 |
| `limit` | int | không | mặc định 20, trần 100 |

*   **Quy tắc kỳ ngày:** gửi một đầu thì đầu còn lại để mở (chỉ `fromDate` = từ ngày đó trở đi; chỉ `toDate` = đến hết ngày đó).
*   **Response:**
    *   `200 OK`: Trả về `PaginationResult` gồm `items` và `metadata` (`page`, `limit`, `totalItems`, `totalPages`).

```json
{
  "message": "Lấy danh sách hoạt động người dùng thành công",
  "data": {
    "items": [
      {
        "id": "9f0b6a3e-...",
        "userId": "1c2d3e4f-...",
        "activityType": "login",
        "occurredAt": "2026-10-15T08:31:02+07:00",
        "createdAt": "2026-10-15T08:31:02+07:00",
        "userEmail": "user@example.com",
        "userFirstName": "An",
        "userLastName": "Nguyễn",
        "deviceName": "iPhone 15",
        "deviceIp": "203.0.113.7",
        "userAgent": "Closy/1.0 (iOS 18)"
      }
    ],
    "metadata": { "page": 1, "limit": 20, "totalItems": 1, "totalPages": 1 }
  }
}
```

*   **Ghi chú trường:**
    *   Sắp xếp `occurredAt` giảm dần (tie-break `id`).
    *   `userFirstName`/`userLastName` có thể vắng mặt khi người dùng chưa đặt tên (NULL); `userEmail` luôn có.
    *   Sự kiện của tài khoản đã bị vô hiệu hoá (soft delete) vẫn hiển thị kèm danh tính.
    *   **Thay đổi 029/D-07 — `deviceName`, `deviceIp`, `userAgent`:** trước đây response chỉ có danh tính và thời điểm, admin phải ghép với bảng session để biết đăng nhập từ đâu. Nay 3 trường này được ghi cùng sự kiện (lấy từ session tương ứng: input khi login, token/input khi refresh). Chúng **vắng mặt** khi: (a) sự kiện được ghi trước migration này (không có nguồn để backfill), hoặc (b) client không gửi thông tin thiết bị. Không có giá trị mặc định thay thế — vắng mặt nghĩa là "không ghi nhận". Dữ liệu được lưu đầy đủ và hiển thị nguyên văn cho admin; không có job purge riêng, vòng đời bằng vòng đời bản ghi activity.
*   **Mã lỗi:**
    *   `400 Bad Request`: filter/kỳ ngày sai định dạng hoặc `fromDate > toDate` (thông báo tiếng Việt).
    *   `401 Unauthorized`: thiếu/sai token.
    *   `403 Forbidden`: không có quyền admin.
