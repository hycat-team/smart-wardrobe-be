# Báo cáo Audit Response DTOs — Module Admin

## Tổng quan

Module admin hiện chỉ có **1 nhóm chức năng** là Audit Logs (nhật ký quản trị), phục vụ quản trị viên tra cứu lịch sử thao tác. Toàn bộ DTO response nằm tại `internal/modules/admin/application/dto/audit_dto.go`, handler tại `internal/modules/admin/presentation/handler/admin_audit_handler.go`.

- Số struct response: **1** (`AdminAuditLogDTO`) + 1 alias list (`AdminAuditLogListRes`).
- Endpoint khảo sát: `GET /api/v1/admin/audit-logs` (yêu cầu role Admin).
- Không phát hiện overdelivery có ý nghĩa (các trường đều cần thiết cho ngữ cảnh quản trị).

## Danh sách struct

### 1. `AdminAuditLogDTO` — `internal/modules/admin/application/dto/audit_dto.go:20`

| # | Field | Kiểu | JSON |
|---|-------|------|------|
| 1 | `ID` | `uuid.UUID` | `id` |
| 2 | `ActorUserID` | `uuid.UUID` | `actorUserId` |
| 3 | `Action` | `string` | `action` |
| 4 | `TargetType` | `string` | `targetType` |
| 5 | `TargetID` | `*uuid.UUID` | `targetId,omitempty` |
| 6 | `Payload` | `map[string]any` | `payload` |
| 7 | `RequestIP` | `*string` | `requestIp,omitempty` |
| 8 | `CreatedAt` | `time.Time` | `createdAt` |

- **Tổng field: 8** (bằng ngưỡng, không vượt quá).
- Được dùng làm item của `AdminAuditLogListRes = shared_dto.PaginationResult[*AdminAuditLogDTO]`.
- Nguồn dữ liệu: mapper `MapAdminAuditLogToDTO` (application/mapper/audit_log.go) — ánh xạ 1:1 toàn bộ field từ entity `AdminAuditLog`, không dư field nào.

### 2. `AdminAuditLogListRes` — alias (không phải struct thuần)

- `shared_dto.PaginationResult[*AdminAuditLogDTO]`: gồm `Items` + `Metadata` (phân trang chuẩn dùng chung), không phát sinh vấn đề.

## Ghi nhận

Theo chính sách đã chốt: **toàn bộ admin endpoints chỉ ghi nhận "không đề xuất refactor"** — giữ full fields cho ngữ cảnh quản trị.

- `AdminAuditLogDTO`: **KHÔNG ĐỀ XUẤT REFACTOR** (ghi nhận).
  - Lý do: struct gồm 8 field, đúng ngưỡng không bị xem là overdelivery; hơn nữa đây là context admin — payload/requestIp/targetId đều hữu ích cho việc điều tra, đối soát.
  - Không thuộc phạm vi `BriefRes` vì chính sách admin giữ nguyên.

## Đề xuất (nếu có)

Không có đề xuất refactor cho module admin theo chính sách.

## Uu tiên

Không áp dụng (module admin loại khỏi phạm vi refactor).

## API bị ảnh hưởng

- `GET /api/v1/admin/audit-logs` — response `AdminAuditLogDTO` / `AdminAuditLogListRes`: **không bị ảnh hưởng** (giữ nguyên).