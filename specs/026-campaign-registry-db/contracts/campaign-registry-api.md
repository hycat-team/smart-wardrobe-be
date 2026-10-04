# Hợp Đồng: API Quản Trị Vòng Đời Chiến Dịch

**Chủ sở hữu**: module `subscription` (dịch vụ `account`) · **Quyền**: chỉ `Admin`
**Liên quan**: `contracts/campaign-query-api-delta.md` (thay đổi cộng thêm cho API đọc của 025)
**Đặc tả**: [../spec.md](../spec.md) FR-052 đến FR-055, FR-060 đến FR-063

---

## 1. Vì sao có nhóm endpoint này

025 cung cấp 4 API **chỉ đọc** để trả lời khách hàng. Việc mở/sửa/đóng chiến dịch vẫn phải sửa YAML
rồi deploy. 026 bổ sung ba endpoint **ghi** và một endpoint **đọc nhật ký**, để vòng đời chiến dịch
đi qua API có kiểm toán, **không cần deploy**.

---

## 2. Đường dẫn

Tất cả nằm trong nhóm riêng `/api/v1/admin/campaigns` (bẫy định tuyến Gin của 025 R11 — **không**
đặt dưới `/admin/subscriptions`, vì nơi đó có `POST /:userId/cancel` và sẽ panic lúc khởi động).

```
POST  /api/v1/admin/campaigns                       tạo chiến dịch
PATCH /api/v1/admin/campaigns/{code}                sửa chiến dịch
POST  /api/v1/admin/campaigns/{code}/close          đóng cưỡng bức
GET   /api/v1/admin/campaigns/{code}/audit          nhật ký kiểm toán của chiến dịch
```

### 2.1 Phân quyền và giới hạn tần suất

| Lớp trung gian | Giá trị |
|---|---|
| `r.auth.Handle()` | JWT hợp lệ bắt buộc |
| `shared_middleware.RolesAuthorize(roleslug.Admin)` | FR-053; người dùng thường bị `403` kể cả khi biết đường dẫn |
| `AdminGrantRateLimitMiddleware` (`r.adminGrantRateLimit()`) | FR-053; tái dùng đúng giới hạn đang gắn lên đường cấp gói hàng loạt |

Không viết kiểm tra vai trò riêng, không mở cấu hình giới hạn tần suất mới.

### 2.2 Định dạng chung

| Quy tắc | Giá trị |
|---|---|
| Nhãn JSON | `camelCase` |
| Thời gian | RFC3339 **có múi giờ** (ví dụ `2026-11-01T00:00:00+07:00`); thiếu múi giờ bị `400` |
| Response | Bọc trong `APIResponse { success, message, data }` |
| `version` | số nguyên, dùng cho If-Match (FR-055) |

---

## 3. `POST /api/v1/admin/campaigns` — tạo

### 3.1 Request

```json
{
  "code": "tet-2027",
  "planSlug": "premium-monthly",
  "quota": 100,
  "reserve": 10,
  "startsAt": "2027-01-20T00:00:00+07:00",
  "endsAt": "2027-02-05T00:00:00+07:00"
}
```

| Trường | Kiểu | Bắt buộc | Quy tắc |
|---|---|---|---|
| `code` | chuỗi | Có | `^[a-z0-9][a-z0-9-]{2,63}$` |
| `planSlug` | chuỗi | Có | không rỗng; tồn tại/active kiểm **lúc chạy** |
| `quota` | số nguyên | Có | `>= 1` |
| `reserve` | số nguyên | Không | mặc định 0; `>= 0` |
| `startsAt` | chuỗi RFC3339 | Có | có múi giờ |
| `endsAt` | chuỗi RFC3339 hoặc `null` | Không | nếu có thì phải **sau** `startsAt` |

### 3.2 Response `201`

Thân là một `CampaignSummaryRes` (xem delta), với `grantedMainCount = 0`,
`grantedCompensationCount = 0`, `budgetLocked = false`, `version = 1`, `status` suy ra từ thời gian.

### 3.3 Lỗi

| Tình huống | HTTP | Thông báo tiếng Việt |
|---|---|---|
| `code` sai định dạng / rỗng | 400 | "Mã chiến dịch không hợp lệ." |
| `starts_at`/`endsAt` không phải RFC3339 **có múi giờ** | 400 | "Tham số thời gian không hợp lệ: cần RFC3339 có múi giờ." |
| Thiếu trường bắt buộc | 400 | Thông báo chuẩn của `validation.Bind*` |
| `quota < 1` / `reserve < 0` / `endsAt <= startsAt` / `planSlug` rỗng | 422 | "Dữ liệu chiến dịch không hợp lệ: `<trường>`." |
| `code` đã tồn tại | 409 | "Mã chiến dịch đã tồn tại: `<code>`." |
| Không phải Admin | 403 | Thông báo chuẩn của `RolesAuthorize` |
| Vượt giới hạn tần suất | 429 | Thông báo chuẩn của rate-limit |

**Ma trận này chỉ hiện thực** bảng ánh xạ ở `spec.md` FR-054 và phải khớp từng dòng; khi lệch thì
FR-054 thắng. `400` = hỏng hình dạng; `422` = đúng hình dạng nhưng sai nghĩa. `409` chỉ dành cho trùng
mã lúc tạo.

---

## 4. `PATCH /api/v1/admin/campaigns/{code}` — sửa

### 4.1 Request

```json
{
  "version": 1,
  "quota": 120,
  "startsAt": "2027-01-20T00:00:00+07:00",
  "endsAt": null
}
```

| Trường | Kiểu | Bắt buộc | Quy tắc |
|---|---|---|---|
| `version` | số nguyên | Có | phiên bản FE đang giữ; lệch → `412` |
| `planSlug` | chuỗi | Không | đổi được **chỉ khi** `budgetLocked = false` |
| `quota` | số nguyên | Không | `>= 1`; đổi được **chỉ khi** `budgetLocked = false` |
| `reserve` | số nguyên | Không | `>= 0`; đổi được **chỉ khi** `budgetLocked = false` |
| `startsAt` | chuỗi RFC3339 | Không | thay được bất kể đã cấp |
| `endsAt` | chuỗi RFC3339 hoặc `null` | Không | `null` = xoá ngày kết thúc; phải sau `startsAt` |

Trường vắng mặt = không đổi. `code` **không** có trong thân (bất biến, FR-024).

### 4.2 Quy tắc bất biến ngân sách (FR-020)

Nếu chiến dịch đã có ít nhất một lượt cấp (`budgetLocked = true`) và thân có đổi `planSlug`,
`quota` hoặc `reserve`, từ chối **`422`**:

> "Chiến dịch đã cấp gói — không thể đổi hạn mức hoặc gói. Hãy đóng chiến dịch và mở mã mới."

Trigger CSDL (data-model §2.2) là chốt chặn không thể vòng qua, kể cả khi có SQL gõ tay.

### 4.3 Phiên bản (FR-055)

**Phân biệt `404` và `412` — bắt buộc theo đúng thứ tự này.** `UPDATE ... WHERE code = ? AND
version = ?` trả **0 dòng** cho **cả hai** ca: mã không tồn tại, và phiên bản lệch. Nếu chỉ nhìn số
dòng, mã lạ sẽ trả `412` — sai, vì người gõ sai ô mã phải nhận "không tìm thấy". Vì vậy:

1. `GetByCode(code)` **trong cùng transaction** → không có thì trả `404` ngay.
2. Rồi mới `UPDATE ... WHERE code = ? AND version = ?` → 0 dòng thì `412`.

Cùng quy tắc này áp dụng cho `close`. Riêng `close` còn một bước nữa **đứng trước cả hai**: nếu chiến
dịch đã đóng thì trả `200` idempotent, không so phiên bản (xem 5.3).

`version` tăng 1 mỗi lần sửa/đóng thành công.

### 4.4 Response `200`

Thân là `CampaignSummaryRes` đã cập nhật.

| Tình huống | HTTP |
|---|---|
| Thiếu trường bắt buộc / thời gian sai định dạng | 400 |
| `quota`/`reserve`/`endsAt` sai nghĩa; đổi ngân sách khi `budgetLocked` | 422 |
| Không tìm thấy `code` | 404 |
| Phiên bản lệch | 412 |
| Không phải Admin | 403 |
| Vượt giới hạn tần suất | 429 |
| JWT thiếu hoặc hỏng | 401 |
| Lỗi hệ thống (CSDL, ghi thất bại) | 500 |

`401` và `500` không nằm trong bảng ánh xạ ở `spec.md` FR-054 vì chúng do lớp trung gian và lớp lỗi
tập trung phát ra chứ không do ba endpoint này phân loại; chúng vẫn là những mã client **có thể** nhận
về, nên hợp đồng phải liệt kê.

---

## 5. `POST /api/v1/admin/campaigns/{code}/close` — đóng cưỡng bức

### 5.1 Request

```json
{ "version": 3, "reason": "Hết ngân sách khuyến mãi tháng 11" }
```

| Trường | Kiểu | Bắt buộc | Quy tắc |
|---|---|---|---|
| `version` | số nguyên | Có | lệch → `412` (trừ khi đã đóng — xem 5.3) |
| `reason` | chuỗi | Có | không rỗng; lưu vào nhật ký kiểm toán |

### 5.2 Hiệu ứng

- Ghi `closed_at = now()`; `status` trả về `closed`.
- Từ đó chiến dịch **không cấp gói** cho bất kỳ tài khoản nào (FR-031), kể cả tài khoản tạo trước thời
  điểm đóng nhưng xử lý sau (nhờ FR-045 đọc lại dưới khoá chia sẻ).
- Các tài khoản đã được cấp **không đổi** gói/thời hạn (FR-033).
- Ghi `closed_at` không đụng `ends_at`; đóng một chiến dịch **chưa tới** `starts_at` vẫn hợp lệ.

### 5.3 Idempotent (FR-034)

Nếu chiến dịch **đã đóng**, trả `200` thành công (no-op), **không** ghi thêm bản ghi kiểm toán.
Kiểm tra idempotent chạy **trước** kiểm tra phiên bản, nên gọi lại với `version` cũ vẫn thành công.

### 5.4 Lỗi

| Tình huống | HTTP |
|---|---|
| Thiếu `reason` hoặc `version` | 400 |
| Không tìm thấy `code` | 404 |
| `reason` rỗng | 422 |
| Phiên bản lệch (chiến dịch **chưa** đóng) | 412 |
| Không phải Admin | 403 |
| Vượt giới hạn tần suất | 429 |
| JWT thiếu hoặc hỏng | 401 |
| Lỗi hệ thống (CSDL, ghi thất bại) | 500 |

`401`/`500` do lớp trung gian và lớp lỗi tập trung phát ra — xem ghi chú ở §4.4.

**Đóng là một chiều qua API (FR-034).** Không có endpoint mở lại. Muốn chạy lại → tạo mã mới.

---

## 6. `GET /api/v1/admin/campaigns/{code}/audit` — nhật ký kiểm toán (FR-063)

Đọc nhật ký của **đúng một** chiến dịch. Chỉ đọc, phân trang.

### 6.1 Truy vấn

`page`, `limit` (int, tuỳ chọn).

### 6.2 Response `200`

```json
{
  "success": true,
  "message": "Đã xử lý yêu cầu tra cứu nhật ký chiến dịch.",
  "data": {
    "items": [
      {
        "id": "9f1c...",
        "actorUserId": "5a2b...",
        "action": "campaign.update",
        "targetType": "campaign",
        "targetId": null,
        "payload": {
          "campaignCode": "launch-2026-10",
          "before": { "quota": 50, "reserve": 5 },
          "after":  { "quota": 60, "reserve": 5 }
        },
        "requestIp": "203.0.113.7",
        "createdAt": "2026-10-03T10:15:00+07:00"
      }
    ],
    "metadata": { "page": 1, "limit": 20, "totalItems": 1, "totalPages": 1 }
  }
}
```

### 6.3 Chi tiết lưu trữ

| Khía cạnh | Giá trị |
|---|---|
| Bảng | `shared.admin_audit_logs` (không tạo bảng mới) |
| `targetType` | `"campaign"` (hằng `adminaudit.Campaign`) |
| `targetId` | `null` — vì bộ lọc `targetId` sẵn có nhận UUID, mã chiến dịch là chuỗi |
| Mã chiến dịch | `payload.campaignCode` (chuỗi) |
| `Hành động` | `campaign.create`, `campaign.update`, `campaign.close` |
| Cũ/mới | `payload.before`, `payload.after` (từng trường đổi; **vắng mặt** ở `create`) |
| Lý do đóng | `payload.reason` (**vắng mặt** trừ ở `close`) |
| Thời điểm | `createdAt`; người thực hiện | `actorUserId` |

**Endpoint này dùng DTO trả lời riêng `CampaignAuditLogRes`** (đặt trong
`services/account/dto/subscription/campaign_audit_dto.go`), **không** tái dùng
`dto/admin.AdminAuditLogDTO`: DTO đó khai `targetId` với `json:"targetId,omitempty"`, nên con trỏ `nil`
**bị bỏ khỏi JSON** thay vì phát ra `null`. `CampaignAuditLogRes` khai `TargetID *uuid.UUID` với
`json:"targetId"` — **không** `omitempty` — để bất biến "mọi bản ghi chiến dịch có `targetId` là null"
**quan sát được** và kiểm thử được. Đây là endpoint **mới**, nên không phát sinh nghĩa vụ tương thích
với hợp đồng nhật ký chung của 025.

### 6.4 Đọc xuyên module (FR-063)

Đọc qua **hợp đồng hẹp** `ICampaignAuditLookup` trong `services/account/contract/admin/`, cài trong
module `admin` (phương thức mới `ListByCampaignCode` lọc
`target_type = 'campaign' AND payload->>'campaignCode' = ?`). Handler của `subscription` tra
`GetByCode` **trước** và trả `404` bằng `ErrCampaignNotFound` nếu mã không tồn tại, vì module `admin`
không được biết chiến dịch nào tồn tại và sẽ chỉ trả một trang rỗng. **Không** mở rộng
`IAuditLogContract` (được phục vụ qua gRPC — thêm phương thức buộc sửa proto, thay đổi hợp đồng xuyên
dịch vụ).

### 6.5 Lỗi

| Tình huống | HTTP | Thông báo tiếng Việt |
|---|---|---|
| Không tìm thấy `code` | 404 | "Không tìm thấy chiến dịch với mã đã nhận: `<code>`." |
| Không phải Admin | 403 | Thông báo chuẩn của `RolesAuthorize` |
| Vượt giới hạn tần suất | 429 | Thông báo chuẩn của rate-limit |
| `page`/`limit` sai định dạng | 400 | Thông báo chuẩn của `validation.BindQuery` |
| JWT thiếu hoặc hỏng | 401 | Thông báo chuẩn của lớp xác thực |
| Lỗi hệ thống khi đọc nhật ký | 500 | Thông báo chuẩn của lớp lỗi tập trung |

Lỗi đọc nhật ký **không** được biến thành một trang rỗng — phải là lỗi, vì trang rỗng là câu trả lời sai
cho câu hỏi "chiến dịch này có thay đổi gì không".

---

## 7. Nhật ký kiểm toán — bảo đảm giao dịch

`IAuditLogContract.Record` phát qua **outbox** (`publisher.Publish(ctx, TopicAuditLog, input)`), nên
dòng outbox nằm **trong cùng transaction** với thao tác ghi chiến dịch. Bảng `admin_audit_logs` được
ghi bất đồng bộ bởi consumer sẵn có. Hệ quả:

- Không có thao tác ghi thành công mà không có dòng outbox kiểm toán.
- Nếu `Record` lỗi, transaction rollback và thao tác chiến dịch thất bại (không có chiến dịch nào
  thay đổi mà không có vết).

---

## 8. Check-list hợp đồng

- [ ] Mọi endpoint dưới `/api/v1/admin/campaigns`; kiểm vai trò `Admin`.
- [ ] `PATCH`/`close` gửi `version` và xử lý `412`.
- [ ] `budgetLocked = true` ⇒ vô hiệu hoá ô `quota`/`reserve`/`planSlug`.
- [ ] `close` idempotent; sau đóng `status = "closed"`; xử lý `reason` rỗng (`422`).
- [ ] `code` bất biến; không có ô đổi mã.
- [ ] Thời gian gửi RFC3339 có múi giờ; `endsAt: null` để xoá ngày kết thúc.
- [ ] Nhật ký trả `targetId: null` và mã trong `payload.campaignCode`.
