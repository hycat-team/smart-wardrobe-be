# Admin Audit Log API Specs

Tài liệu đặc tả API tra cứu Nhật ký Quản trị (Audit Logs) — ghi lại mọi thao tác quan trọng của quản trị viên trên hệ thống. Xem thêm hằng số tại [constants.md](constants.md).

---

## Flow 1: Tra cứu Nhật ký Quản trị

### 1. Lấy danh sách nhật ký quản trị

- **Endpoint:** `GET /api/v1/admin/audit-logs`
- **Tác nhân (Actor):** Quản trị viên (Admin).
- **Mô tả:** Tra cứu lịch sử thao tác của các quản trị viên trong hệ thống. Hỗ trợ lọc theo hành động, loại đối tượng, khoảng thời gian và phân trang.
- **Query Parameters:**
    - `actorUserId` (string, optional) — UUID admin thực hiện.
    - `action` (string, optional) — Hành động. Tham chiếu [constants.md#1-hanh-dong-audit-log-auditaction](constants.md#1-hanh-dong-audit-log-auditaction).
    - `targetType` (string, optional) — Loại đối tượng. Tham chiếu [constants.md#2-loai-doi-tuong-audit-targettype](constants.md#2-loai-doi-tuong-audit-targettargettype).
    - `targetId` (string, optional) — UUID đối tượng.
    - `fromDate` (string, optional) — Từ ngày, định dạng `YYYY-MM-DD`.
    - `toDate` (string, optional) — Đến ngày, định dạng `YYYY-MM-DD`.
    - `page` (int, optional) — Số trang, mặc định `1`.
    - `limit` (int, optional) — Số bản ghi mỗi trang, mặc định `20`.
- **Response:** `200 OK` — Phân trang danh sách audit log.

**Cấu trúc Audit Log:**

| Field | JSON | Kiểu | Ý nghĩa |
|-------|------|------|---------|
| ID | `id` | uuid | ID bản ghi audit log |
| ActorUserID | `actorUserId` | uuid | ID admin thực hiện |
| Action | `action` | string | Hành động đã thực hiện |
| TargetType | `targetType` | string | Loại đối tượng bị ảnh hưởng |
| TargetID | `targetId` | uuid (nullable) | ID đối tượng bị ảnh hưởng |
| Payload | `payload` | jsonb (nullable) | Dữ liệu chi tiết thao tác |
| RequestIP | `requestIp` | string | IP của request |
| CreatedAt | `createdAt` | timestamptz | Thời điểm tạo |
