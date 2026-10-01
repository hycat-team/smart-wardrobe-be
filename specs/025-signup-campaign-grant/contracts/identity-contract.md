# Hợp Đồng: `CampaignAccountLister` — liệt kê tài khoản theo cửa sổ thời gian

**Chủ sở hữu**: module `identity` (dịch vụ `account`) · **Bên dùng**: cơ chế chiến dịch trong module
`subscription`

---

## 1. Vấn đề cần giải

Cơ chế quét bù (FR-079) phải tìm các tài khoản đủ điều kiện mà lượt cấp chưa được ghi. Nhưng module
`subscription` **không được** đọc bảng `identity.users` — hiến phán §II cấm import `domain/entities` và
`domain/repositories` của module khác, và đó là vi phạm mức `fail` của quy tắc H4.

`IUserBriefContract` hiện có 4 phương thức: `GetByID`, `GetByIDs`, `GetByUsernamesOrEmails`,
`SearchUserBriefs`. **Không có** phương thức nào liệt kê theo thời điểm tạo.

Phương án đã loại: quét bảng hàng đỗi theo chủ đề (buộc cơ chế chiến dịch phụ thuộc lược đồ hạ tầng
khác và phải phân tích JSON thô); ghi dòng ứng viên lúc tạo tài khoản (phá vỡ chính quyết định hướng
publisher của FR-085). Chi tiết ở R5.

## 2. KHÔNG mở rộng `IUserBriefContract` — khai báo interface hẹp riêng

Đây là điểm then chốt của hợp đồng này, và là chỗ thiết kế ban đầu đã sai.

`IUserBriefContract` **được phục vụ qua gRPC**. Nó có adapter client
(`contracts/identity/v1/client/user_brief_client.go:24`, có khẳng định biên dịch
`var _ IUserBriefContract = (*UserBriefClient)(nil)`) và adapter server
(`services/account/internal/modules/identity/presentation/grpc/server.go:31`). Thêm một phương thức vào đó bắt buộc phải sửa `identity.proto` và cả hai adapter —
tức là phải đổi hợp đồng **xuyên dịch vụ**, thứ mà FR-058 của tính năng này cấm và tiền lệ 024 đã
tránh.

Vì vậy khai báo một interface hẹp **riêng**, đặt trong cùng tệp hợp đồng, theo đúng tiền lệ
`AdminGrantLookup` mà 024 đã dựng:

```go
// CampaignAccountLister là khoảng hẹp mà cơ chế quét bù của chiến dịch cần từ
// module identity (025).
//
// Tách khỏi `IUserBriefContract` có chủ đích. Hợp đồng đó đi qua gRPC, và mở rộng
// nó sẽ buộc phải đổi `identity.proto` cho một nhu cầu chỉ có bên trong dịch vụ
// này. Đây là cùng cách tách mà 024 T136 đã làm cho `AdminGrantLookup`.
//
// Cùng lý do, hợp đồng này trả về DTO chứ không trả về `entities.User`: hiến phán
// §II cấm domain entity đi qua ranh giới hợp đồng, và DTO `UserBriefContractDTO`
// đã mang đủ hai trường mà mục 3 bổ sung.
type CampaignAccountLister interface {
    // ListCreatedInWindow trả tối đa `filter.Limit` tài khoản có
    // `created_at` trong [from, to), theo thứ tự thời điểm tạo tăng dần.
    //
    // Phân trang bằng khoá tổ hợp: lô sau bắt đầu ở bản ghi có
    // `created_at > filter.CursorTime`, hoặc `created_at = filter.CursorTime` nhưng
    // `id > filter.CursorID`. Khoá tổ hợp là bắt buộc: nhiều tài khoản có thể có
    // cùng thời điểm tạo, và phân trang chỉ theo thời điểm sẽ bỏ sót hoặc lặp
    // vô hạn — xem kịch bản 2 ở mục 6.
    //
    // Chỉ phát hành tài khoản KHÔNG bị xoá mềm. Tài khoản đã xoá mềm không được
    // cấp quyền lợi mới.
    ListCreatedInWindow(ctx context.Context, filter CampaignAccountFilter) (*CampaignAccountPage, error)
}

// NewCampaignAccountLister thu hẹp hợp đồng cụ thể về khoảng hẹp của chiến dịch.
func NewCampaignAccountLister(c *UserBriefContract) CampaignAccountLister { return c }
```

`*UserBriefContract` thoả `CampaignAccountLister` **mà không** phải thoả ràng buộc của gRPC: đó là
toàn bộ điểm của việc tách. Mọi nơi dùng `IUserBriefContract` hiện có **không đổi một dòng nào**.

### 2.1 Bộ lọc

Đặt trong `identity/domain/repositories/readmodels.go` theo quy tắc tầng repository của
`design-pattern.md`. Tên đổi từ `GetCreatedInWindowFilter` thành `CampaignAccountFilter` để không
trông như bộ lọc chung của hợp đồng gRPC:

```go
type CampaignAccountFilter struct {
    From       time.Time
    To         time.Time
    CursorTime time.Time   // zero = lô đầu
    CursorID   uuid.UUID   // zero = lô đầu
    Limit      int
}
```

### 2.2 Kết quả

Đặt cùng chỗ, và trả về **DTO** — không phải `entities.User`:

```go
type CampaignAccountPage struct {
    Items      []*dto.UserBriefContractDTO
    TotalCount int64
}
```

`TotalCount` là tổng số tài khoản trong cửa sổ, **không** phải số bản ghi của lô. Nó phục vụ cơ chế
quét bù để biết còn bao nhiêu lô nữa; giao diện tra cứu FR-065 phân trang trên bảng
`campaign_claims`, không liên quan tới hợp đồng này.

Không cần khai báo DTO mới cho phần tử: `UserBriefContractDTO` đã có `ID`, và mục 3 bổ sung
`CreatedAt` cùng `CreatedByAdmin` — đúng ba trường mà cơ chế quét bù cần.

## 3. Hai trường bổ sung trên `UserBriefContractDTO`

Đặt trong `services/account/dto/identity/contract_dto.go`.

```go
// 025: the campaign module owns the rule "an account created by an
// administrator is not a campaign target", but it does not own
// identity.users. Carrying the flag on the contract is what lets it enforce
// that rule from the catch-up sweep, which reads the database and never sees
// the creation event.
//
// It is deliberately NOT derivable: the table records nothing about who created
// the row, and an administrator may create an already-verified account.
CreatedByAdmin bool `json:"createdByAdmin"`

// 025: the campaign priority rule compares this against the frozen watermark
// (FR-031). Carried so the campaign module never reads identity.users.
CreatedAt time.Time `json:"createdAt"`
```

**Vì sao mở rộng DTO sẵn có thay vì khai báo DTO mới**: `design-pattern.md` cấm khai báo lại một
cấu trúc đã có ở module cung cấp — đó là đường gây trôi lệch âm thầm. Tiền lệ trực tiếp: trường
`RoleSlug` đã được thêm vào chính cấu trúc này cho cùng một lý do (FR-056 của 024, ghi chú ngay
trong tệp).

Hai trường này **không** đi qua gRPC: adapter `v1` chuyển đổi sang protobuf riêng nên bỏ qua chúng.
Đó không phải hạn chế — bên dùng duy nhất là cơ chế quét bù chạy trong cùng dịch vụ.

## 4. Chỉ đọc, không ghi

Phương thức mới **chỉ đọc**. Không thêm phương thức ghi nào, không thêm phương thức cập nhật nào. Lý do:
cơ chế chiến dịch không có bất kỳ lý do kinh doanh nào để sửa tài khoản. Mọi thay đổi trạng thái tài
khoản đều đi qua luồng hiện có.

## 5. Điều kiện hiệu lực

| Điều kiện | Nguồn |
|---|---|
| Không import `identity/domain/entities` từ module khác | Hiến phán §II |
| Không trả về `entities.User` qua hợp đồng — chỉ DTO | Hiến phán §II ("using DTOs, never domain entities") |
| **Không** sửa `IUserBriefContract`, `identity.proto`, hay adapter gRPC | FR-058; tiền lệ 024 `AdminGrantLookup` |
| Trả về `[]` rỗng, không phải `nil`, khi không có kết quả | Quy ước hợp đồng |
| `Limit` được chuẩn hoá về giá trị mặc định khi bằng 0 hoặc vượt trần | `shared_dto.PaginationQuery.Normalize` |
| Bảo đảm phân trang không bỏ sót khi nhiều tài khoản trùng thời điểm tạo | Khoá tổ hợp `(created_at, id)` |
| Không trả về tài khoản đã xoá mềm | `is_deleted = false` |

## 6. Kiểm thử hợp đồng

| # | Kịch bản | Điều kiện đạt |
|---|---|---|
| 1 | 10 tài khoản, lấy lô 3 | Đúng 3 bản ghi, `CreatedAt` tăng dần, `TotalCount = 10` |
| 2 | 3 tài khoản **cùng** thời điểm tạo, `limit=1`, duyệt 3 lô | Duyệt đủ 3 tài khoản, không lặp, không bỏ sót |
| 3 | Cửa sổ không có tài khoản nào | `Items` là `[]` rỗng, `TotalCount = 0`, **không phải** `nil` |
| 4 | Trong cửa sổ có tài khoản bị xoá mềm | Không xuất hiện trong kết quả |
| 5 | Tài khoản tạo bởi quản trị viên | `CreatedByAdmin = true` trong DTO, khớp với cột trong cơ sở dữ liệu |
| 6 | Ba đường tạo tài khoản | Cờ trong DTO khớp cờ trong thông điệp `identity.user.created` |
| 7 | Biên dịch toàn bộ `services/commerce` | Xanh — chứng minh hợp đồng gRPC **không** bị đụng tới |

Kịch bản 2 là bẫy thật của khoá tổ hợp: `created_at` có độ chính xác hạng mili-giây nên nhiều tài
khoản đăng ký trong cùng một lần kiểm thử sẽ trùng thời điểm, và phân trang chỉ theo thời điểm sẽ
bỏ sót hoặc lặp. Kịch bản 7 là bằng chứng cho quyết định tách interface ở mục 2 — nếu ai đó sau này
vô tình thêm phương thức vào `IUserBriefContract`, kịch bản này đỏ.

Kịch bản 6 là kiểm thử chống trôi lệch quan trọng nhất về mặt nghiệp vụ: hai nguồn suy ra cùng một
sự thật từ hai đường khác nhau.
