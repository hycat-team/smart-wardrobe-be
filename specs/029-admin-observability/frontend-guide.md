# Frontend Integration Guide — Khả năng quan sát cho admin (029)

**Feature**: `029-admin-observability`
**Trạng thái**: Đã đồng bộ có đếm với mã nguồn và **đối chiếu live 2026-10-16** trên stack Docker dev — xem biên bản §8 (ví dụ JSON đã đối chiếu hình dạng/trường; số liệu trong ví dụ là minh hoạ).
**Đối tượng đọc**: Lập trình viên frontend repo `smart-wardrobe-fe` (trang quản trị admin).
**Mục tiêu**: FE triển khai được 2 trang quản trị mới + sửa dashboard **chỉ bằng tài liệu này** — KHÔNG cần đọc mã nguồn backend hay Swagger.

> Quy ước: mọi trường JSON là `camelCase`; thời gian là ISO 8601; mọi "ngày" trong tài liệu là **ngày lịch Việt Nam (+07:00)**, không phải UTC.
> Nguồn hợp đồng phía backend: `specs/029-admin-observability/contracts/admin-api-delta.md`, `specs/029-admin-observability/data-model.md`, `docs/api/identity/admin-activity-api.md`, `docs/api/subscription/admin-billing-api.md`, `docs/api/dashboard/dashboard-admin-api.md`.
> Repo frontend KHÔNG nằm trong workspace này; mọi việc sửa FE (mục 5) được bàn giao tại đây, không nằm trong phạm vi code của 029.

---

## 0. Tóm tắt nhanh (TL;DR)

Đọc hết mục này là nắm 90% thay đổi; chi tiết ở các mục sau.

- **1 endpoint MỚI**: `GET /api/v1/admin/activity-events` — nhật ký hoạt động người dùng (`login`/`refresh`) phân trang, lọc `userId` / `activityType` / kỳ ngày, kèm danh tính (`userEmail`, `userFirstName`, `userLastName`). Dùng cho trang **"Hoạt động người dùng"** (mục 2).
- **Trang "Giao dịch payOS"** (mục 3):
  - `GET /transactions` — `status` từ **số → chuỗi 11 giá trị** (khứ hồi vào filter, thêm `failed_legacy`); kỳ `fromDate`/`toDate` **sửa lỗi múi giờ** (trước lệch 7h, nuốt lỗi im lặng → nay **400** tiếng Việt).
  - `GET /transactions/{id}` — từ **entity thô tên trường Go** → **DTO camelCase** đúng bộ trường như item trong danh sách.
  - **4 hành động đối soát không đổi** về đường dẫn/nghiệp vụ; phản hồi **không có khối `data`** (null).
- **Dashboard admin** (mục 4) — 7 endpoint nhận kỳ `fromDate`/`toDate` (4 endpoint vừa mở tham số, 3 endpoint đổi múi giờ sang ngày lịch VN + 400 khi đảo chiều):
  - `payos-reconciliation`: phạm vi **cả `direct_purchase` + `wallet_topup`**, tổng hợp **theo kỳ** (mặc định hôm nay), `totalSuccessValue` là **tổng tiền giao dịch thành công thật** (không còn mượn doanh thu gói).
  - `ai/error-breakdown`: tử số, mẫu số, phân nhóm **cùng kỳ**; `errorRate` **≤ 100%**.
  - `kpi-cards.payOSSuccessCount` **đổi giá trị** theo phạm vi hai loại (shape không đổi); KPI vẫn là **ảnh chụp**.
- **FE phải chủ động gửi kỳ** cho preset ("Hôm nay", "Tháng này") — backend **không** cung cấp preset; preset là việc FE tính và gửi (mục 4.8).
- **Bỏ tính lại cục bộ** `errorRate` và bỏ fallback im lặng khi range sai (mục 5, vấn đề 5 & 7).
- **13 lỗi UX dashboard FE** đã khảo sát + phân loại trách nhiệm ở mục 5; checklist triển khai ở mục 7.

---

## 1. Quy ước chung (đọc trước khi code)

### 1.1 Xác thực

- Mọi endpoint trong tài liệu này yêu cầu **Bearer JWT của tài khoản vai trò `admin`**:
  `Authorization: Bearer <access_token>` (hệ thống cũng nhận cookie `access_token`).
- `401` — thiếu / sai / hết hạn token (phiên đăng xuất): đưa về màn đăng nhập, **không** hiển thị nội dung `message` thô.
- `403` — đã đăng nhập nhưng không phải admin: ẩn màn hình quản trị, không phải lỗi nghiệp vụ.

### 1.2 Vỏ phản hồi (response envelope)

Thành công (HTTP 2xx):

```json
{ "message": "…mô tả tiếng Việt…", "data": { } }
```

- `data` **vắng mặt hoàn toàn** khi giá trị null (ví dụ phản hồi của 4 hành động đối soát — mục 3.5). Đừng code `data!.something`; hãy kiểm tra tồn tại.
- FE nên hiển thị thông báo thành công từ `message` (hoặc tự đặt text theo `message` ổn định).

Lỗi (HTTP 4xx/5xx):

```json
{
  "status": 400,
  "title": "Thao tác không thành công",
  "message": "Định dạng ngày bắt đầu (fromDate) không hợp lệ. Vui lòng sử dụng YYYY-MM-DD.",
  "errors": [ { "field": "fromDate", "message": "…" } ]
}
```

- Hiển thị `message`; `errors[]` (nếu có) dùng để tô đỏ từng ô nhập.
- `debugMessage` / `stackTrace` chỉ xuất hiện ở môi trường không-phải-production — **đừng** phụ thuộc vào chúng.

### 1.3 Kỳ ngày — quy tắc chuẩn (áp dụng cho MỌI endpoint có `fromDate`/`toDate`)

- `fromDate` / `toDate` định dạng `YYYY-MM-DD`, **ngày lịch Việt Nam (+07:00)**, **bao gồm trọn cả hai ngày**.
- Nội bộ BE biểu diễn `[from, to)` với `from` = 00:00 VN ngày đầu, `to` = 00:00 VN **ngày kế tiếp ngày cuối**; mọi truy vấn dùng `>= from AND < to`.
- **Gửi một đầu thì đầu còn lại để mở**: chỉ `fromDate` = từ ngày đó trở đi (không giới hạn trên); chỉ `toDate` = đến hết ngày đó (không giới hạn dưới).
- **400 chỉ khi**: định dạng sai, hoặc gửi **cả hai** mà `fromDate > toDate`. Thông báo tiếng Việt chính xác:
  - `"Định dạng ngày bắt đầu (fromDate) không hợp lệ. Vui lòng sử dụng YYYY-MM-DD."`
  - `"Định dạng ngày kết thúc (toDate) không hợp lệ. Vui lòng sử dụng YYYY-MM-DD."`
  - `"Ngày bắt đầu (fromDate) không được sau ngày kết thúc (toDate)."`
- Kỳ không có dữ liệu → trả số 0 / danh sách rỗng, **không lỗi**.
- **Trước 029** một số endpoint hiểu ngày là nửa đêm **UTC** (lệch 7h: `toDate` bỏ sót gần hết ngày cuối) và **bỏ qua im lặng** ngày sai; nay tất cả đã theo quy tắc trên.

Gợi ý đối chiếu nhanh (ví dụ `fromDate=2026-10-15&toDate=2026-10-15`):

```text
VN 2026-10-15 00:00:00  ─────────────────────────►  VN 2026-10-16 00:00:00
│  giao dịch 00:00 15/10 ... 23:59:59.999 15/10 nằm trong kỳ  │  (00:00 16/10 thuộc kỳ sau)
```

FE **luôn gửi đủ cả hai đầu** nếu muốn giới hạn chặt (gửi một đầu = truy vấn mở một phía, kết quả đổi theo lúc gọi).

### 1.4 Phân trang

- Danh sách phân trang bọc trong `data.items` + `data.metadata`:
  `{ "page": 1, "limit": 20, "totalItems": 125, "totalPages": 7 }`.
- Mặc định `page=1`, `limit=20`; **trần 100 bản ghi/trang** (giá trị > 100 bị chuẩn hoá về 100 trong metadata). Khuyến nghị FE gửi `limit ≤ 100`.
- Khi không có bản ghi: `items: []` và `totalPages: 0` — không phải lỗi.
- Danh sách giao dịch sắp xếp `createdAt` giảm dần; nhật ký hoạt động sắp xếp `occurredAt` giảm dần (tie-break `id`). Không có tham số sort.

### 1.5 Múi giờ hiển thị

- Thời gian trả về là ISO 8601 kèm offset. Khi hiển thị, luôn format theo `Asia/Ho_Chi_Minh`.
- **Không** parse chuỗi `yyyy-MM-dd` bằng `new Date('2026-10-15')` (JS hiểu là UTC — xem vấn đề 10, mục 5).

---

## 2. (a) NHÓM MỚI — Nhật ký hoạt động người dùng (`activity-events`)

### 2.1 Hợp đồng endpoint

**`GET /api/v1/admin/activity-events`** — `[MỚI]`, chỉ đọc (không có endpoint ghi).

| Param | Kiểu | Bắt buộc | Ghi chú |
|---|---|---|---|
| `userId` | uuid | không | sai định dạng → **400** `"Mã người dùng (userId) không hợp lệ. Vui lòng sử dụng định dạng UUID."` |
| `activityType` | string | không | chỉ `login` \| `refresh`; giá trị khác → **400** `"Loại hoạt động (activityType) không hợp lệ. Chỉ chấp nhận: login, refresh."` |
| `fromDate` | `YYYY-MM-DD` | không | ngày lịch VN, bao gồm trọn ngày (mục 1.3) |
| `toDate` | `YYYY-MM-DD` | không | ngày lịch VN, bao gồm trọn ngày; `fromDate > toDate` → 400 |
| `page` | int | không | mặc định 1 |
| `limit` | int | không | mặc định 20, trần 100 |

Trường của mỗi item (`ActivityEventRes`):

| JSON | Kiểu | Ý nghĩa |
|---|---|---|
| `id` | string (uuid) | Mã sự kiện |
| `userId` | string (uuid) | Mã người dùng |
| `activityType` | string | `login` hoặc `refresh` |
| `occurredAt` | string (ISO 8601) | Thời điểm phát sinh — **khóa sắp xếp** (giảm dần) |
| `createdAt` | string (ISO 8601) | Thời điểm ghi nhận |
| `userEmail` | string | Email người dùng — **luôn có** |
| `userFirstName` | string | **có thể vắng mặt** khi tài khoản chưa đặt tên |
| `userLastName` | string | **có thể vắng mặt** khi tài khoản chưa đặt tên |

Hành vi đặc biệt:

- Sắp xếp `occurredAt` giảm dần, tie-break `id` — ổn định cho phân trang.
- Sự kiện của tài khoản đã **vô hiệu hoá (soft delete)** vẫn hiển thị kèm danh tính; tài khoản bị xoá cứng thì sự kiện cũng bị xoá (ràng buộc dữ liệu, hành vi hiện có).
- Bộ lọc không khớp gì → `200` với `items: []`.

### 2.2 Ví dụ request / response

**Request**

```http
GET /api/v1/admin/activity-events?userId=1c2d3e4f-5a6b-4c7d-8e9f-0a1b2c3d4e5f&activityType=login&fromDate=2026-10-15&toDate=2026-10-15&page=1&limit=20
Authorization: Bearer <admin_access_token>
```

**Response 200**

```json
{
  "message": "Lấy danh sách hoạt động người dùng thành công",
  "data": {
    "items": [
      {
        "id": "9f0b6a3e-4c5d-4e6f-8a9b-0c1d2e3f4a5b",
        "userId": "1c2d3e4f-5a6b-4c7d-8e9f-0a1b2c3d4e5f",
        "activityType": "login",
        "occurredAt": "2026-10-15T08:31:02+07:00",
        "createdAt": "2026-10-15T08:31:02+07:00",
        "userEmail": "an.nguyen@example.com",
        "userFirstName": "An",
        "userLastName": "Nguyễn"
      },
      {
        "id": "3a4b5c6d-7e8f-4a0b-9c1d-2e3f4a5b6c7d",
        "userId": "5e6f7a8b-9c0d-4e1f-8a2b-3c4d5e6f7a8b",
        "activityType": "refresh",
        "occurredAt": "2026-10-15T08:46:10+07:00",
        "createdAt": "2026-10-15T08:46:10+07:00",
        "userEmail": "binh.tran@example.com"
      }
    ],
    "metadata": { "page": 1, "limit": 20, "totalItems": 2, "totalPages": 1 }
  }
}
```

<!-- đối chiếu live 2026-10-16 -->

> Item thứ hai minh hoạ `userFirstName`/`userLastName` **vắng mặt** khi người dùng chưa đặt tên. Đây không phải lỗi dữ liệu.

**Response 400 (lọc sai)**

```json
{
  "status": 400,
  "title": "Thao tác không thành công",
  "message": "Loại hoạt động (activityType) không hợp lệ. Chỉ chấp nhận: login, refresh."
}
```

<!-- đối chiếu live 2026-10-16 -->

### 2.3 Mã HTTP

| Mã | Khi nào | FE làm gì |
|---|---|---|
| `200` | Luôn khi bộ lọc hợp lệ (kể cả không có kết quả) | Hiển thị bảng/rỗng |
| `400` | `userId` sai UUID; `activityType` ngoài `login`/`refresh`; `fromDate`/`toDate` sai định dạng hoặc đảo chiều | Hiển thị `message`; giữ nguyên bộ lọc để người dùng sửa |
| `401` | Thiếu/sai/hết hạn token | Về màn đăng nhập |
| `403` | Không phải admin | Ẩn màn hình |
| `500` | Lỗi hệ thống | Thông báo chung + cho phép thử lại |

### 2.4 Hướng dẫn tích hợp trang **"Hoạt động người dùng"**

1. **Bộ lọc** (đồng bộ vào URL query để chia sẻ / back-forward được):
   - `userId`: ô nhập UUID. Validate định dạng UUID phía FE trước khi gọi (tránh 400 không cần thiết); rỗng thì **bỏ hẳn** param.
   - `activityType`: dropdown `Tất cả | login | refresh`. Chọn "Tất cả" ⇒ **không gửi** `activityType` (đừng gửi chuỗi `"Tất cả"` hay `"all"` — xem vấn đề 8, mục 5).
   - Kỳ ngày: tuỳ chọn "Hôm nay | 7 ngày | 30 ngày | Tuỳ chọn"; gửi đủ `fromDate`/`toDate` (mục 1.3). Nhãn kỳ hiển thị phải khớp giá trị đã gửi.
   - `page`/`limit`: phân trang server-side, giữ `limit=20` (hoặc cho chọn 20/50/100).
2. **Bảng kết quả** — cột gợi ý:
   - **Thời điểm**: `occurredAt` format `dd/MM/yyyy HH:mm` theo `Asia/Ho_Chi_Minh`.
   - **Người dùng**: tên hiển thị = `[userFirstName, userLastName].filter(Boolean).join(' ')`; **rỗng thì fallback `userEmail`**; `userEmail` luôn có nên không bao giờ trống ô. Có thể hiển thị phụ `userId` (rút gọn) để tra cứu.
   - **Loại hoạt động**: badge theo `activityType` (`login` → "Đăng nhập", `refresh` → "Làm mới phiên"). Mã lạ (ngoài 2 giá trị — chỉ xảy ra khi dữ liệu tương lai) → hiển thị nguyên mã, đừng crash.
   - Tuỳ chọn: `id` (rút gọn, nút copy) cho mục đích điều tra.
3. **Không cần** nút tạo/sửa/xoá — endpoint chỉ đọc (BE không đăng ký method ghi nào).
4. **Trạng thái rỗng**: hiển thị "Không có hoạt động nào khớp bộ lọc" + nút xoá bộ lọc (đừng hiển thị lỗi).
5. **Refetch**: gọi lại khi bộ lọc/`page`/`limit` đổi (debounce ô `userId` ~300ms). Không có realtime; không cần polling.

---

## 3. (b) NHÓM CHANGED — Giao dịch payOS (admin)

### 3.1 Before → after

| | Trước 029 | Sau 029 |
|---|---|---|
| `GET /transactions` — `items[].status` | số (`1`, `4`…) | **chuỗi 11 giá trị** (`pending`, `success`, `failed_legacy`, … — mục 3.2) |
| `GET /transactions` — `fromDate`/`toDate` | parse nửa đêm **UTC**, `toDate` dùng `<=` ⇒ **bỏ sót gần hết ngày cuối (lệch 7h)**; ngày sai **bị bỏ qua im lặng** (trả trang không lọc) | **ngày lịch VN, bao gồm trọn ngày**, cận nửa mở; ngày sai định dạng hoặc `fromDate > toDate` → **400** tiếng Việt |
| `GET /transactions` — filter `status` | không nhận `failed_legacy` | nhận đủ **11 giá trị** (khứ hồi vào/ra: giá trị nào API in ra thì dùng lại được làm filter) |
| `GET /transactions/{id}` | **entity thô**, tên trường Go (`ID`, `UserID`, `OrderCode`, `Status` dạng số, …) | **`DepositTransactionRes` camelCase** — đúng bộ trường như item trong danh sách (`status` chuỗi) |
| 4 hành động `POST …/resolve-*`, `retry-investigation` | — | **KHÔNG đổi** đường dẫn/nghiệp vụ; phản hồi **không có `data`** |
| Phân trang, các filter còn lại | — | **KHÔNG đổi** |

> Lưu ý: DTO user-facing của người dùng cuối (`PaymentLinkDTO.paymentStatus`) **vẫn là số** — ngoài phạm vi admin, đừng nhầm.

### 3.2 Bảng 11 trạng thái (`status`)

`status` trong **mọi phản hồi admin** là **chuỗi**; mọi giá trị in ra dùng lại được làm filter `status`.

| # | Chuỗi | Nghĩa nghiệp vụ gợi ý cho UI |
|---|---|---|
| 1 | `pending` | Chờ thanh toán (link đã tạo) |
| 2 | `success` | Thành công |
| 3 | `failed_legacy` | Thất bại (dữ liệu cũ — trước đây là số 2) |
| 4 | `creating` | Đang tạo link |
| 5 | `reconciliation_required` | Cần đối soát |
| 6 | `reconciling` | Đang đối soát |
| 7 | `creation_failed` | Tạo link thất bại |
| 8 | `cancelled` | Đã huỷ |
| 9 | `expired` | Hết hạn |
| 10 | `investigation_required` | Cần điều tra |
| 11 | `refunded` | Đã hoàn tiền |

- FE nên có **một map** `status → { label tiếng Việt, màu badge }` dùng chung cho list + detail; map đủ 11 giá trị.
- **`unknown`**: giá trị ngoài 11 giá trị (chỉ có thể phát sinh khi dòng DB bị sửa tay/out-of-contract) được trả là `"unknown"`; `unknown` **không phải đầu vào hợp lệ** của filter. Hiển thị "Không xác định" thay vì crash.
- Filter `status` **không** trả 400 khi giá trị lạ: BE **bỏ qua** giá trị không parse được (coi như không lọc). FE vì vậy **chỉ gửi giá trị trong dropdown**, đừng dựa vào 400 để bắt lỗi.

### 3.3 `GET /api/v1/admin/subscriptions/transactions` — `[CHANGED]`

Params: `status` (11 giá trị, không bắt buộc), `fromDate`, `toDate`, `page`, `limit`.

**Request**

```http
GET /api/v1/admin/subscriptions/transactions?status=reconciliation_required&fromDate=2026-10-15&toDate=2026-10-15&page=1&limit=20
Authorization: Bearer <admin_access_token>
```

**Response 200**

```json
{
  "message": "Lấy danh sách giao dịch thành công",
  "data": {
    "items": [
      {
        "id": "9f0b6a3e-4c5d-4e6f-8a9b-0c1d2e3f4a5b",
        "userId": "1c2d3e4f-5a6b-4c7d-8e9f-0a1b2c3d4e5f",
        "orderCode": 1792027862456,
        "amount": 250000,
        "currency": "VND",
        "status": "reconciliation_required",
        "transactionType": "direct_purchase",
        "targetPlanId": "7a1b2c3d-4e5f-4a6b-8c7d-9e0f1a2b3c4d",
        "paymentUrl": "https://pay.payos.vn/web/abc123",
        "reference": "REF-20261015-0001",
        "paymentLinkId": "pay_link_abc123",
        "failureReason": "gateway_timeout",
        "cancelledAt": "2026-10-15T10:30:00+07:00",
        "nextReconciliationAt": "2026-10-15T11:00:00+07:00",
        "createdAt": "2026-10-15T08:00:00+07:00",
        "updatedAt": "2026-10-15T09:00:00+07:00"
      },
      {
        "id": "1a2b3c4d-5e6f-4a7b-8c9d-0e1f2a3b4c5d",
        "userId": "5e6f7a8b-9c0d-4e1f-8a2b-3c4d5e6f7a8b",
        "orderCode": 1792027900123,
        "amount": 100000,
        "currency": "VND",
        "status": "pending",
        "transactionType": "wallet_topup",
        "createdAt": "2026-10-15T09:12:44+07:00",
        "updatedAt": "2026-10-15T09:12:44+07:00"
      }
    ],
    "metadata": { "page": 1, "limit": 20, "totalItems": 2, "totalPages": 1 }
  }
}
```

<!-- đối chiếu live 2026-10-16 -->

| JSON | Kiểu | Ghi chú |
|---|---|---|
| `id` | uuid | Dùng cho `GET /transactions/{id}` |
| `userId` | uuid | Người dùng |
| `orderCode` | int64 | Mã đơn payOS; dùng cho 4 hành động (path). Sinh theo `unixMillis*1000 + 0..999` nên **luôn nằm dưới giới hạn số an toàn của JS** — hiển thị dạng text để an toàn hiển thị |
| `amount` | number | VND |
| `currency` | string | `"VND"` |
| `status` | string | 11 giá trị (mục 3.2) |
| `transactionType` | string | `direct_purchase` (mua gói) hoặc `wallet_topup` (nạp ví) |
| `targetPlanId` | uuid? | **vắng mặt** khi không gắn gói (nạp ví) |
| `paymentUrl` | string? | Link thanh toán (vắng khi chưa/không có) |
| `reference` | string? | Mã tham chiếu nhà cung cấp |
| `paymentLinkId` | string? | Mã link payOS |
| `failureReason` | string? | Lý do thất bại (nếu có) |
| `cancelledAt` | ISO? | Thời điểm huỷ |
| `nextReconciliationAt` | ISO? | Mốc đối soát kế tiếp |
| `createdAt`, `updatedAt` | ISO | Thời gian tạo/cập nhật |

Các trường có `?` **vắng mặt** khi NULL — FE dùng optional (`field?: string | null`) và không hiển thị `null`/`undefined` thô.

### 3.4 `GET /api/v1/admin/subscriptions/transactions/{id}` — `[CHANGED]`

- Path: `id` là **UUID** của giao dịch (`item.id` từ danh sách).
- Response `data` là **một object cùng bộ trường** như item trong danh sách (mục 3.3) — kể cả `status` dạng chuỗi. **Không còn** trường kiểu Go (`ID`, `UserID`, `OrderCode`, `Status` số).

**Response 200 (ví dụ rút gọn)**

```json
{
  "message": "Lấy chi tiết giao dịch thành công",
  "data": {
    "id": "9f0b6a3e-4c5d-4e6f-8a9b-0c1d2e3f4a5b",
    "userId": "1c2d3e4f-5a6b-4c7d-8e9f-0a1b2c3d4e5f",
    "orderCode": 1792027862456,
    "amount": 250000,
    "currency": "VND",
    "status": "success",
    "transactionType": "direct_purchase",
    "targetPlanId": "7a1b2c3d-4e5f-4a6b-8c7d-9e0f1a2b3c4d",
    "reference": "REF-20261015-0001",
    "paymentLinkId": "pay_link_abc123",
    "createdAt": "2026-10-15T08:00:00+07:00",
    "updatedAt": "2026-10-15T08:05:00+07:00"
  }
}
```

<!-- đã đối chiếu live 2026-10-16: id không tồn tại trả HTTP 404 (fixup 5cbf38ac) -->

> **Giao dịch không tồn tại → HTTP 404** (`message`: "Không tìm thấy giao dịch."). Đã xác minh trên stack dev ngày 2026-10-16.

### 3.5 4 hành động đối soát — `[KHÔNG ĐỔI đường dẫn/nghiệp vụ]`

Path param `{orderCode}` là `orderCode` (số) của giao dịch. **Luôn gửi JSON body** (tối thiểu `{}`; body rỗng hoàn toàn có thể bị từ chối 400 khi parse).

| # | Hành động | Method + path | Body | Ghi chú nghiệp vụ |
|---|---|---|---|---|
| 1 | Xác nhận **đã thanh toán** | `POST /api/v1/admin/subscriptions/transactions/{orderCode}/resolve-paid` | `{ "reason": "...", "reference": "...", "paymentLinkId": "..." }` | **Bắt buộc** `reference` + `paymentLinkId` (thiếu → 400) |
| 2 | Xác nhận **đã huỷ** | `POST …/{orderCode}/resolve-cancelled` | `{ "reason": "..." }` | Từ chối giao dịch đã `success` (400) |
| 3 | Đánh dấu **tạo link thất bại** | `POST …/{orderCode}/resolve-creation-failed` | `{ "reason": "..." }` | Chỉ áp dụng khi trạng thái hiện tại là `creating`, `reconciliation_required` hoặc `investigation_required` (khác → 400) |
| 4 | **Gửi lại yêu cầu đối soát** | `POST …/{orderCode}/retry-investigation` | `{ "reason": "..." }` | Đưa giao dịch về `reconciliation_required`; từ chối giao dịch đã `success` (400) |

**Request (ví dụ 1)**

```json
{
  "reason": "Khách hàng đã chuyển khoản thành công",
  "reference": "123456789",
  "paymentLinkId": "pay_xxx"
}
```

**Response 200 của cả 4 hành động — KHÔNG có khối `data`:**

```json
{ "message": "Xác nhận giao dịch đã thanh toán thành công" }
```

<!-- đối chiếu live 2026-10-16 -->

- Phản hồi **không chứa trạng thái giao dịch** — đừng đọc `data.status`. Sau khi hành động thành công: **gọi lại `GET /transactions/{id}`** (hoặc list) để lấy trạng thái mới (xem 3.7).
- Lỗi nghiệp vụ (sai trạng thái, thiếu bằng chứng) → **400** với `"Không thể áp dụng quyết định xử lý thủ công vì trạng thái hoặc bằng chứng thanh toán không hợp lệ."`
- Gợi ý bật/tắt nút theo `status` hiện tại (đọc từ detail/list):
  - `resolve-paid`, `resolve-cancelled`, `retry-investigation`: hiện khi `status !== "success"` (đã `success` → BE từ chối).
  - `resolve-creation-failed`: chỉ khi `status ∈ { creating, reconciliation_required, investigation_required }`.
  - Vẫn luôn xử lý 400 từ BE (dữ liệu có thể đổi giữa lúc mở trang và lúc bấm nút).

### 3.6 Mã HTTP

| Endpoint | 200 | 400 | 401 | 403 | 404 | 500 |
|---|---|---|---|---|---|---|
| `GET /transactions` | danh sách (kể cả rỗng) | ngày sai/đảo chiều (`message` tiếng Việt) | thiếu/sai token | không phải admin | — | lỗi hệ thống |
| `GET /transactions/{id}` | DTO camelCase | — | ✔ | ✔ | 404 khi id không tồn tại | ✔ |
| `POST …/resolve-paid` | `{ message }`, không `data` | thiếu bằng chứng / trạng thái không hợp lệ | ✔ | ✔ | — | ✔ |
| `POST …/resolve-cancelled` | `{ message }`, không `data` | trạng thái không hợp lệ | ✔ | ✔ | — | ✔ |
| `POST …/resolve-creation-failed` | `{ message }`, không `data` | trạng thái không hợp lệ | ✔ | ✔ | — | ✔ |
| `POST …/retry-investigation` | `{ message }`, không `data` | trạng thái không hợp lệ | ✔ | ✔ | — | ✔ |

### 3.7 Hướng dẫn tích hợp trang **"Giao dịch payOS"**

**Danh sách:**

1. Bộ lọc: `status` (dropdown đủ **11 giá trị** + "Tất cả" — "Tất cả" ⇒ không gửi param), kỳ ngày (mục 1.3), `page`/`limit` server-side.
2. Cột gợi ý: `orderCode` (text), `transactionType` (map `direct_purchase` → "Mua gói", `wallet_topup` → "Nạp ví"), `amount` (định dạng VND), `status` (badge từ map 11 giá trị), `createdAt` (giờ VN), người dùng (`userId` rút gọn), nút "Chi tiết".
3. Sắp xếp: `createdAt` giảm dần (BE cố định) — đừng thêm param sort.
4. Refetch khi filter/page đổi; giữ filter trong URL.

**Chi tiết:**

1. Lấy `id` từ item danh sách → `GET /transactions/{id}`.
2. Hiển thị **cùng shape** với item — không cần hai model khác nhau; có thể tái dùng đúng interface TypeScript.
3. Trường vắng mặt hiển thị "—"; `paymentUrl` (nếu có) mở tab mới; `nextReconciliationAt`/`cancelledAt` format giờ VN.
4. Điều hướng "quay lại" phải **khôi phục filter danh sách**.

**4 hành động + refresh sau hành động:**

```text
Bấm hành động
  → mở dialog xác nhận (nhập `reason`; resolve-paid thêm `reference` + `paymentLinkId` bắt buộc)
  → POST action
      ├─ 200 (không có data) → toast `message` → GET /transactions/{id} để lấy trạng thái mới
      │                        → (tuỳ chọn) refetch danh sách nếu user quay lại list
      └─ 400                 → hiển thị `message`, giữ dialog để sửa
```

- **Không** tự đoán trạng thái mới sau hành động (không có `data` trả về).
- Nếu refetch detail thấy trạng thái chưa đổi ngay, chờ ~1s và thử lại một lần (hành động ghi bất đồng bộ phía BE trong một số luồng) — không spam.
- Nút hành động dựa trên `status` mới nhất đang hiển thị.

---

## 4. (c) NHÓM CHANGED — Dashboard admin

### 4.1 Before → after tổng quan

| Endpoint | Trước 029 | Sau 029 |
|---|---|---|
| `GET /admin/dashboard/subscriptions/revenue-breakdown` | nhận `fromDate`/`toDate` nhưng hiểu **nửa đêm UTC**, tự cộng thêm 1 ngày cho `toDate` (lệch 7h) | **ngày lịch VN**, cận nửa mở, bao gồm trọn ngày; **400** khi sai định dạng/đảo chiều |
| `GET /admin/dashboard/subscriptions/distribution` | **không nhận kỳ** — luôn là hiện trạng | **nhận kỳ** `[MỚI]`: đếm user có gói hiệu lực **giao** với kỳ; không gửi kỳ = hiện trạng |
| `GET /admin/dashboard/subscriptions/renewal-stats` | **không nhận kỳ** — toàn thời gian | **nhận kỳ** `[MỚI]`: attempt **tạo trong kỳ** (`created_at`); không gửi kỳ = toàn thời gian |
| `GET /admin/dashboard/ai/error-breakdown` | **không nhận kỳ** — `totalErrors`/`breakdown` **toàn thời gian** nhưng `totalRequests` 30 ngày ⇒ `errorRate` có thể **> 100%** | **nhận kỳ** `[MỚI]`: tử số + mẫu số + phân nhóm **cùng kỳ** (mặc định 30 ngày); `errorRate` ≤ 100%, 0 khi mẫu 0 |
| `GET /admin/dashboard/payos-reconciliation` | đọc **ảnh chụp** mới nhất; chỉ `direct_purchase`; `totalSuccessValue` **mượn** doanh thu gói | `[MỚI]` nhận kỳ; **cả `direct_purchase` + `wallet_topup`**; tổng hợp **theo kỳ** từ bảng giao dịch; `totalSuccessValue` = **SUM tiền giao dịch thành công thật** |
| `GET /admin/dashboard/ai-margin` | nhận kỳ nhưng hiểu nửa đêm **UTC**; đảo chiều bị bỏ qua | kỳ **VN**, bao gồm trọn ngày; **400** khi sai/đảo |
| `GET /admin/dashboard/ai/usage-by-operation` | nhận kỳ nửa đêm **UTC**; đảo chiều bị bỏ qua | kỳ **VN**, bao gồm trọn ngày; **400** khi sai/đảo |
| `GET /admin/dashboard/overview` | `payosReconcile` theo ảnh chụp cũ | `payosReconcile` mang **ngữ nghĩa mới** (hai loại, theo kỳ, mặc định hôm nay VN); `kpiCards.payOSSuccessCount` đổi giá trị |
| `GET /admin/dashboard/kpi-cards` | `payOSSuccessCount` chỉ `direct_purchase` | `payOSSuccessCount` đếm **cả `wallet_topup`** — shape không đổi, **giá trị đổi chủ đích** |

> `kpi-cards` và `overview` **không nhận tham số kỳ**; gửi kèm `fromDate=...` vào hai endpoint này bị **bỏ qua** (không 400) — nhưng đừng gửi.

### 4.2 Kỳ và mặc định từng endpoint (khi FE KHÔNG gửi gì)

| Endpoint | Nhận kỳ? | Không gửi kỳ = mặc định | Gửi một đầu = |
|---|---|---|---|
| `subscriptions/revenue-breakdown` | có (VN) | **6 tháng gần nhất** (cùng ngày, lùi 6 tháng) → hôm nay | đầu thiếu được **điền mặc định tương ứng** |
| `subscriptions/distribution` | có (mới) | **hiện trạng** (gói còn hiệu lực) | đầu thiếu để **mở** |
| `subscriptions/renewal-stats` | có (mới) | **toàn thời gian** | đầu thiếu để **mở** |
| `ai/error-breakdown` | có (mới) | **30 ngày gần nhất** | đầu thiếu để **mở** |
| `payos-reconciliation` | có (mới) | **hôm nay VN** | đầu thiếu để **mở** |
| `ai-margin` | có (VN) | 30 ngày gần nhất | đầu thiếu được **điền mặc định tương ứng** |
| `ai/usage-by-operation` | có (VN) | 30 ngày gần nhất | đầu thiếu được **điền mặc định tương ứng** |
| `overview`, `kpi-cards` | **không** | KPI = ảnh chụp mới nhất; `aiMargin` = 30 ngày; `payosReconcile` = hôm nay | — |

**Khuyến nghị mạnh cho FE: luôn gửi đủ `fromDate` + `toDate`** — vì "gửi một đầu" cho ra truy vấn mở, kết quả khác nhau giữa hai lần gọi cùng một preset.

### 4.3 `payos-reconciliation` — ngữ nghĩa mới

| Field | Trước | Sau |
|---|---|---|
| `totalTransactions` | chỉ `direct_purchase`, từ snapshot | **COUNT mọi giao dịch payOS trong kỳ** (`direct_purchase` + `wallet_topup`) |
| `successCount` | theo `direct_purchase` | `status = success` trên **cả hai loại** |
| `failedCount` | `total − success` (gồm cả pending/voided — gây hiểu nhầm) | **thất bại thật**: `failed_legacy` + `creation_failed` + `investigation_required` |
| `pendingCount` | — | `pending` + `creating` + `reconciliation_required` + `reconciling` |
| `voidedCount` | — | `cancelled` + `expired` + `refunded` |
| `totalSuccessValue` | mượn `subscriptionRevenueVnd` (doanh thu gói) | **tổng tiền `SUM(amount)` của giao dịch `success`** trên cả hai loại, cùng kỳ; 0 khi không có |

- **Nguồn dữ liệu đổi** từ ảnh chụp (worker 15 phút/lần) sang **truy vấn trực tiếp bảng giao dịch theo kỳ** ⇒ số liệu phản ánh ngay giao dịch mới phát sinh (không còn trễ tới 15 phút).
- Shape không đổi: `{ totalTransactions, successCount, failedCount, pendingCount, voidedCount, totalSuccessValue }` (không có trường kỳ trong response — FE giữ kỳ đã gửi để hiển thị nhãn).

### 4.4 `ai/error-breakdown` — cùng kỳ

| Field | Trước | Sau |
|---|---|---|
| `totalErrors` | toàn thời gian | trong kỳ; và **= Σ `breakdown[].errorCount`** |
| `breakdown[]` | toàn thời gian | trong **cùng kỳ** |
| `totalRequests` | 30 ngày (khác kỳ với tử số) | **cùng kỳ** |
| `errorRate` | `totalErrors/totalRequests × 100` với hai kỳ khác nhau ⇒ **có thể > 100%** | cùng kỳ; **0 khi `totalRequests = 0`**; không vượt 100% khi E ≤ R |

**FE hiển thị thẳng `errorRate` của BE** — không nhân/chia lại từ hai panel khác kỳ (xem vấn đề 5, mục 5).

### 4.5 `overview` và `kpi-cards`

- `overview` = `{ kpiCards, aiMargin, payosReconcile }`:
  - `kpiCards` — **ảnh chụp** (mục 4.6), kèm `updatedAt`.
  - `aiMargin` — 30 ngày gần nhất (hành vi cũ, **không** nhận tham số kỳ).
  - `payosReconcile` — ngữ nghĩa mới (mục 4.3), **mặc định hôm nay VN**.
- `kpi-cards` — snapshot mới nhất; `payOSSuccessCount` **đổi giá trị** theo phạm vi hai loại (đếm cả nạp ví). `subscriptionRevenueVnd` **không đổi** (chỉ `direct_purchase`).
- Nếu chưa có snapshot nào, KPI fallback truy vấn trực tiếp và `updatedAt` = thời điểm gọi.

### 4.6 Hợp đồng + ví dụ JSON từng endpoint

Ghi chú chung:
- Mọi endpoint dưới đây: `401` thiếu/sai token; `403` không phải admin; `500` lỗi hệ thống. 7 endpoint nhận kỳ thêm `400` (sai định dạng / `fromDate > toDate`).
- Thời gian trong ví dụ là ISO 8601; `updatedAt`/`fromDate`/`toDate` (kiểu thời gian) phản ánh offset thực tế của server — luôn format theo VN khi hiển thị.

#### 4.6.1 `GET /api/v1/admin/dashboard/kpi-cards` — `[CHANGED giá trị payOSSuccessCount]`

```json
{
  "message": "Lấy danh sách thẻ KPI Admin thành công",
  "data": {
    "totalUsers": 1500,
    "activeUsersDau": 720,
    "totalBrands": 45,
    "pendingReviewBrands": 3,
    "subscriptionRevenueVnd": 125000000.0,
    "netAiMarginVnd": 87500000.0,
    "payOSSuccessCount": 98,
    "updatedAt": "2026-10-16T09:45:00+07:00"
  }
}
```

<!-- đối chiếu live 2026-10-16 -->

- `activeUsersDau` = số user phân biệt có sự kiện `login`/`refresh` **trong ngày VN** — nhưng lấy từ **snapshot** (có thể cũ tới ~15 phút; xem vấn đề 6, mục 5).
- `payOSSuccessCount` = giao dịch `success` **cả hai loại** trong ngày snapshot.
- `subscriptionRevenueVnd` = doanh thu gói (`direct_purchase`, `success`) trong ngày snapshot.

#### 4.6.2 `GET /api/v1/admin/dashboard/ai-margin` — `[CHANGED kỳ VN + 400]`

Request: `?fromDate=2026-10-01&toDate=2026-10-15`

```json
{
  "message": "Lấy thông tin biên lợi nhuận AI thành công",
  "data": {
    "fromDate": "2026-10-01T00:00:00+07:00",
    "toDate": "2026-10-15T00:00:00+07:00",
    "totalSubscriptionRevVnd": 125000000.0,
    "totalAiActualCostVnd": 37500000.0,
    "netAiMarginVnd": 87500000.0,
    "marginPercentage": 70.0,
    "totalAiPaidTokens": 12500000,
    "aiRequestsCount": 43000,
    "aiErrorCount": 215
  }
}
```

<!-- đối chiếu live 2026-10-16 -->

> **Lưu ý `fromDate`/`toDate` trong response**: là **00:00 ngày VN của ngày đầu và ngày cuối kỳ** (ví dụ trên: kỳ 01/10→15/10 ⇒ `toDate = 00:00 15/10`). Đã đối chiếu live 2026-10-16 (fixup 9edb4b28). Hiển thị: lấy phần ngày VN của giá trị, không convert theo timezone browser.

400 (sai định dạng/đảo chiều): giống mục 1.3.

#### 4.6.3 `GET /api/v1/admin/dashboard/payos-reconciliation` — `[CHANGED]`

Request: `?fromDate=2026-10-01&toDate=2026-10-15` (không gửi = hôm nay VN)

```json
{
  "message": "Lấy thông tin đối soát PayOS thành công",
  "data": {
    "totalTransactions": 320,
    "successCount": 298,
    "failedCount": 6,
    "pendingCount": 11,
    "voidedCount": 5,
    "totalSuccessValue": 187500000.0
  }
}
```

<!-- đối chiếu live 2026-10-16 -->

- Với dữ liệu trong hợp đồng 11 trạng thái: `successCount + failedCount + pendingCount + voidedCount` = `totalTransactions` (các nhóm phân hoạch đủ). Dòng dữ liệu out-of-contract (bị sửa tay) có thể làm đẳng thức này lệch — hiển thị cảnh báo dữ liệu bất thường thay vì coi là lỗi FE.
- Response **không** echo kỳ; FE giữ giá trị đã gửi để hiển thị "Kỳ: 01/10 – 15/10".

#### 4.6.4 `GET /api/v1/admin/dashboard/subscriptions/distribution` — `[CHANGED nhận kỳ]`

Request: `?fromDate=2026-10-01&toDate=2026-10-15` (không gửi = hiện trạng)

```json
{
  "message": "Lấy phân bổ subscription thành công",
  "data": {
    "distribution": [
      { "planCode": "free", "userCount": 1200, "percentage": 80.0 },
      { "planCode": "premium-monthly", "userCount": 300, "percentage": 20.0 }
    ],
    "totalUsers": 1500
  }
}
```

<!-- đối chiếu live 2026-10-16 -->

- Có kỳ: đếm user có cửa sổ gói **hiện tại** (`started_at` → `expires_at`) giao với kỳ; gói FREE (`expires_at` NULL) **hiện diện mọi kỳ**.
- **Giới hạn chấp nhận**: không tái dựng lịch sử đổi gói — user chỉ tính theo **gói hiện tại**. Đừng kết luận "số user gói X trong quá khứ".

#### 4.6.5 `GET /api/v1/admin/dashboard/subscriptions/revenue-breakdown` — `[CHANGED kỳ VN + 400]`

Request: `?fromDate=2026-10-01&toDate=2026-10-15` (không gửi = 6 tháng gần nhất)

```json
{
  "message": "Lấy doanh thu subscription thành công",
  "data": {
    "fromDate": "2026-10-01",
    "toDate": "2026-10-15",
    "breakdown": [
      { "planCode": "premium-monthly", "month": "2026-10", "transactionCount": 42, "revenueVnd": 10500000.0 }
    ],
    "totalRevenueVnd": 10500000.0
  }
}
```

<!-- đối chiếu live 2026-10-16 -->

- `fromDate`/`toDate` ở đây là **chuỗi `YYYY-MM-DD`** (bao gồm hai đầu — khác `ai-margin` trả mốc thời gian).
- `month` = `YYYY-MM`; nhóm theo **tháng lịch** (không có tham số granularity).

#### 4.6.6 `GET /api/v1/admin/dashboard/ai/usage-by-operation` — `[CHANGED kỳ VN + 400]`

Request: `?fromDate=2026-10-01&toDate=2026-10-15`

```json
{
  "message": "Lấy thống kê sử dụng AI thành công",
  "data": {
    "fromDate": "2026-10-01",
    "toDate": "2026-10-15",
    "operations": [
      {
        "operation": "outfit-recommend",
        "logicalRoute": "normal",
        "requestCount": 12000,
        "totalCostVnd": 850000.0,
        "totalPromptTokens": 5400000,
        "totalOutputTokens": 1800000
      }
    ]
  }
}
```

<!-- đối chiếu live 2026-10-16 -->

#### 4.6.7 `GET /api/v1/admin/dashboard/ai/error-breakdown` — `[CHANGED cùng kỳ]`

Request: `?fromDate=2026-10-01&toDate=2026-10-15` (không gửi = 30 ngày gần nhất)

```json
{
  "message": "Lấy thống kê lỗi AI thành công",
  "data": {
    "totalErrors": 215,
    "totalRequests": 43000,
    "errorRate": 0.5,
    "breakdown": [
      { "errorCode": "rate_limit_exceeded", "provider": "openai", "model": "gpt-4", "errorCount": 120 },
      { "errorCode": "timeout", "provider": "anthropic", "model": "claude-3", "errorCount": 95 }
    ]
  }
}
```

<!-- đối chiếu live 2026-10-16 -->

- Kiểm tra chéo để tin số liệu: `Σ breakdown[].errorCount = totalErrors`; `errorRate = totalErrors / totalRequests × 100`; khi `totalRequests = 0` thì `errorRate = 0`.
- Response **không** echo kỳ.

#### 4.6.8 `GET /api/v1/admin/dashboard/subscriptions/renewal-stats` — `[CHANGED nhận kỳ]`

Request: `?fromDate=2026-10-01&toDate=2026-10-15` (không gửi = toàn thời gian)

```json
{
  "message": "Lấy thống kê gia hạn subscription thành công",
  "data": {
    "totalAttempts": 85,
    "successCount": 70,
    "failedCount": 5,
    "successRate": 93.33,
    "avgRetryCount": 1.24
  }
}
```

<!-- đối chiếu live 2026-10-16 -->

- Có kỳ: chỉ tính attempt **tạo trong kỳ** (theo `created_at`), vẫn loại trạng thái `processing`.
- Attempt tạo **trước** kỳ nhưng hoàn tất trong kỳ **không** được tính; attempt tạo trong kỳ nhưng chưa xong cũng bị loại khỏi tổng (vì `processing`). Ghi rõ điều này khi chú thích panel.
- `successRate = successCount / (successCount + failedCount) × 100` — các kết quả phi-thanh-toán (`skipped`, `downgraded`, `renewed`) không tính là thất bại.

#### 4.6.9 `GET /api/v1/admin/dashboard/overview` — `[CHANGED payosReconcile]`

```json
{
  "message": "Lấy thông tin tổng quan hệ thống thành công",
  "data": {
    "kpiCards": {
      "totalUsers": 1500,
      "activeUsersDau": 720,
      "totalBrands": 45,
      "pendingReviewBrands": 3,
      "subscriptionRevenueVnd": 125000000.0,
      "netAiMarginVnd": 87500000.0,
      "payOSSuccessCount": 98,
      "updatedAt": "2026-10-16T09:45:00+07:00"
    },
    "aiMargin": {
      "fromDate": "2026-09-16T00:00:00+07:00",
      "toDate": "2026-10-16T09:45:00+07:00",
      "totalSubscriptionRevVnd": 125000000.0,
      "totalAiActualCostVnd": 37500000.0,
      "netAiMarginVnd": 87500000.0,
      "marginPercentage": 70.0,
      "totalAiPaidTokens": 12500000,
      "aiRequestsCount": 43000,
      "aiErrorCount": 215
    },
    "payosReconcile": {
      "totalTransactions": 46,
      "successCount": 41,
      "failedCount": 1,
      "pendingCount": 3,
      "voidedCount": 1,
      "totalSuccessValue": 24500000.0
    }
  }
}
```

<!-- đối chiếu live 2026-10-16 -->

- `payosReconcile` ở đây = **hôm nay VN** (mặc định), ngữ nghĩa hai loại giao dịch (mục 4.3).
- `aiMargin` = 30 ngày gần nhất (hành vi hiện hành, không đổi 029) — **nhãn phải ghi rõ 30 ngày**, không dùng chung nhãn với kỳ đang chọn.
- `kpiCards` = ảnh chụp; luôn hiển thị `updatedAt`.

#### 4.6.10 `GET /api/v1/admin/dashboard/drilldown/ai-events` — `[KHÔNG ĐỔI]`

Params: `status` (string, tuỳ chọn — lọc thô theo trạng thái, ví dụ `FAILED`), `page`, `limit`.

```json
{
  "message": "Lấy chi tiết sự kiện AI thành công",
  "data": {
    "items": [
      {
        "requestId": "req_01HZX…",
        "provider": "openai",
        "model": "gpt-4",
        "statusCode": 429,
        "errorMessage": "rate limit exceeded",
        "errorCode": "rate_limit_exceeded",
        "createdAt": "2026-10-15T22:04:11+07:00"
      }
    ],
    "metadata": { "page": 1, "limit": 20, "totalItems": 1, "totalPages": 1 }
  }
}
```

<!-- đối chiếu live 2026-10-16 -->

- "Tất cả" trong dropdown trạng thái ⇒ **không gửi** `status` (xem vấn đề 8, mục 5).
- Response có **7 trường** (gồm `errorCode` — tài liệu cũ có thể thiếu trường này).

### 4.7 Mã HTTP — nhóm dashboard

| Endpoint | 200 | 400 | 401 | 403 | 500 |
|---|---|---|---|---|---|
| `subscriptions/revenue-breakdown` | ✔ | ngày sai/đảo chiều | ✔ | ✔ | ✔ |
| `subscriptions/distribution` | ✔ | ngày sai/đảo chiều | ✔ | ✔ | ✔ |
| `subscriptions/renewal-stats` | ✔ | ngày sai/đảo chiều | ✔ | ✔ | ✔ |
| `ai/error-breakdown` | ✔ | ngày sai/đảo chiều | ✔ | ✔ | ✔ |
| `payos-reconciliation` | ✔ | ngày sai/đảo chiều | ✔ | ✔ | ✔ |
| `ai-margin` | ✔ | ngày sai/đảo chiều | ✔ | ✔ | ✔ |
| `ai/usage-by-operation` | ✔ | ngày sai/đảo chiều | ✔ | ✔ | ✔ |
| `overview` | ✔ | **không có 400 theo kỳ** (không nhận kỳ) | ✔ | ✔ | ✔ |
| `kpi-cards` | ✔ | **không có 400 theo kỳ** (tham số ngày bị bỏ qua) | ✔ | ✔ | ✔ |
| `drilldown/ai-events` | ✔ | `page`/`limit` sai kiểu | ✔ | ✔ | ✔ |

### 4.8 Hướng dẫn tích hợp sửa dashboard

**Bước 1 — FE tự tính preset và gửi kỳ tường minh** (BE không có preset):

| Preset | `fromDate` | `toDate` |
|---|---|---|
| Hôm nay | hôm nay VN | hôm nay VN |
| Tháng này | ngày **01** của tháng hiện tại (VN) | hôm nay VN |
| 30 ngày gần nhất | hôm nay VN − 29 ngày (đủ 30 ngày lịch, kể cả hôm nay) — hoặc − 30 ngày nếu muốn trùng đúng mặc định BE | hôm nay VN |
| Tuỳ chọn | người dùng chọn | người dùng chọn |

```ts
// Sinh chuỗi YYYY-MM-DD theo LỊCH VN — KHÔNG dùng toISOString() (UTC).
const vnDate = (d: Date) =>
  new Intl.DateTimeFormat('en-CA', { timeZone: 'Asia/Ho_Chi_Minh' }).format(d);

const todayVN = () => vnDate(new Date());

const startOfMonthVN = () => todayVN().slice(0, 8) + '01'; // YYYY-MM-01

const daysAgoVN = (n: number) => vnDate(new Date(Date.now() - n * 86_400_000));

const presetToday = { fromDate: todayVN(), toDate: todayVN() };
const presetThisMonth = { fromDate: startOfMonthVN(), toDate: todayVN() };
const presetLast30 = { fromDate: daysAgoVN(29), toDate: todayVN() };
```

- Gửi **đủ hai đầu** cho mọi endpoint (mục 4.2).
- Nhãn hiển thị ("Kỳ: 01/10/2026 – 15/10/2026") phải lấy từ **chính giá trị đã gửi**, không hardcode.
- "Tháng này" theo spec: từ 00:00 ngày mùng 1 tháng hiện tại đến hiện tại, giờ VN.

**Bước 2 — gắn nhãn "ảnh chụp" vs "theo kỳ" cho từng panel:**

| Panel | Nguồn | Nhãn đề xuất |
|---|---|---|
| KPI cards (`kpi-cards`, `overview.kpiCards`) | Ảnh chụp ~15 phút/lần | "Ảnh chụp — cập nhật lúc `updatedAt`" + tooltip "số liệu tổng hợp định kỳ, không phải realtime" |
| `overview.aiMargin` | 30 ngày gần nhất (cố định) | "30 ngày gần nhất" |
| `payos-reconciliation` | Theo kỳ (mặc định hôm nay) | "Kỳ đã chọn" (hoặc "Hôm nay" khi không gửi) |
| `distribution` (có kỳ) | Giao cửa sổ gói với kỳ | "Gói hiện tại của user giao với kỳ" + chú thích giới hạn lịch sử đổi gói |
| `distribution` (không kỳ) | Hiện trạng | "Hiện tại" |
| `renewal-stats` (không kỳ) | Toàn thời gian | "Toàn thời gian" |
| `revenue-breakdown`, `ai/error-breakdown`, `ai/usage-by-operation`, `ai-margin` | Theo kỳ (có mặc định) | "Kỳ đã chọn" |

**Bước 3 — xử lý lỗi đúng:**

- Bỏ mọi `normalizeDashboardFilters` kiểu "range sai → im lặng dùng 30 ngày". Gửi nguyên giá trị người dùng chọn; nếu BE trả **400**, hiển thị `message` và **giữ nguyên input** để sửa (vấn đề 7, mục 5).
- Với mọi panel theo kỳ, cùng một `{fromDate, toDate}` phải được gửi cho **tất cả** panel (trừ hai panel snapshot KPI) — nếu panel nào không đổi số khi đổi kỳ, đó là bug FE (quên gửi tham số).

**Bước 4 — bỏ tính lại cục bộ:**

- `errorRate`: hiển thị thẳng từ BE.
- `totalSuccessValue`: hiển thị thẳng từ BE (không cộng `subscriptionRevenueVnd`).
- Tổng doanh thu: dùng `totalRevenueVnd` của `revenue-breakdown` (kỳ đã chọn), không dùng KPI snapshot cho nhãn "trong kỳ".

---

## 5. (d) Đánh giá UX dashboard FE — 13 vấn đề

Phạm vi: các phát hiện từ khảo sát repo `smart-wardrobe-fe` ngày 2026-10-15. Repo FE **ngoài** workspace BE — mục này là bàn giao, BE 029 không sửa code FE.

Phân loại: **BE-029** = backend đã sửa trong 029; **FE** = frontend cần làm; **Deferred** = chưa làm trong 029 (cần quyết định/bổ sung sau).

### 5.1 Bảng tổng hợp

| # | Vấn đề | Phân loại chính |
|---|---|---|
| 1 | Không tách khu vực daily/monthly | BE-029 + FE |
| 2 | Mặc định 30 ngày cuốn, thiếu preset "Hôm nay"/"Tháng này" | FE |
| 3 | Bộ lọc ngày chỉ áp dụng một phần panel | BE-029 + FE |
| 4 | Nhãn "Doanh thu" trùng 3 nơi, 3 kỳ khác nhau | BE-029 + FE |
| 5 | "Tỷ lệ lỗi" trùng và có thể > 100% | BE-029 + FE |
| 6 | "DAU hôm nay" lấy từ snapshot có thể cũ | FE |
| 7 | Range sai bị thay thế im lặng bằng 30 ngày | BE-029 + FE |
| 8 | Filter bảng AI events lỗi giá trị mặc định (`'Tất cả'` vs `'all'`) | FE |
| 9 | `console.log` sót | FE |
| 10 | Lệch múi giờ khi parse `new Date('yyyy-MM-dd')` | FE |
| 11 | Granularity hardcode `'day'` + label "Tính theo : day" | FE + Deferred (phần BE) |
| 12 | Trộn ngôn ngữ hiển thị | FE |
| 13 | Cột "Loại" hiển thị model + dữ liệu chết `totalBrands`/`pendingReviewBrands` | FE + Deferred (phần quyết định sản phẩm) |

### 5.2 Chi tiết từng vấn đề

**1. Không tách khu vực daily/monthly trên dashboard**

- **Mô tả**: các panel số liệu "hôm nay", "toàn thời gian", "6 tháng cuốn" và snapshot nằm lẫn trong một lưới; admin không biết số nào thuộc kỳ nào.
- **Nguyên nhân gốc**: (a) layout FE một lưới chung, không nhãn kỳ; (b) một phần panel bỏ qua bộ lọc ngày (xem #3) nên các số trông "vô chủ".
- **Phân loại**: **BE-029** (mọi panel số liệu đã nhận/chuẩn hoá kỳ để FE có thể gắn nhãn theo kỳ) + **FE** (tách khu vực "Ảnh chụp hiện tại" vs "Theo kỳ đã chọn", gắn nhãn từng panel).

**2. Mặc định 30 ngày cuốn, thiếu preset "Hôm nay"/"Tháng này"**

- **Mô tả**: bộ lọc khởi tạo `DASHBOARD_DEFAULT_LOOKBACK_DAYS = 30` (`features/admin/utils/dashboard.ts`); không có nút "Hôm nay"/"Tháng này"; không chọn được "từ đầu tháng → nay".
- **Nguyên nhân gốc**: FE hardcode lookback; BE **chủ đích** không cung cấp preset (spec: FE tự tính khoảng và gửi — mục 4.8).
- **Phân loại**: **FE** (thêm preset; "Tháng này" = mùng 1 VN → nay).

**3. Bộ lọc ngày chỉ áp dụng một phần panel (KPI, PayOS, distribution, revenue, renewal, error-breakdown bỏ qua)**

- **Mô tả**: đổi khoảng ngày nhưng nhiều panel không đổi số.
- **Nguyên nhân gốc**: trước 029, `payos-reconciliation`/`distribution`/`renewal-stats`/`ai/error-breakdown` **không nhận** kỳ; `revenue-breakdown` có nhận nhưng FE không gửi nên rơi vào mặc định 6 tháng. KPI cards là **snapshot theo thiết kế** (không có kỳ).
- **Phân loại**: **BE-029** (4 endpoint vừa mở tham số kỳ; revenue chuẩn hoá ngày VN) + **FE** (gửi `fromDate`/`toDate` cho mọi panel theo kỳ; KPI ghi rõ "ảnh chụp", không gửi kỳ).

**4. Nhãn "Doanh thu" trùng 3 nơi với 3 kỳ khác nhau**

- **Mô tả**: KPI `subscriptionRevenueVnd` (snapshot ngày), AI margin `totalSubscriptionRevVnd` (theo kỳ), và "Tổng doanh thu trong kỳ" thực chất là `totalRevenueVnd` của revenue-breakdown mặc định **6 tháng** khi không gửi kỳ.
- **Nguyên nhân gốc**: ba nguồn số liệu khác nhau dùng chung một nhãn; revenue-breakdown không được gửi kỳ nên luôn rơi mặc định 6 tháng nhưng vẫn gắn nhãn "trong kỳ".
- **Phân loại**: **BE-029** (kỳ + mặc định tường minh, 400) + **FE** (đặt nhãn riêng theo nguồn: "Doanh thu hôm nay (ảnh chụp)", "Doanh thu kỳ đã chọn"; luôn gửi kỳ cho revenue-breakdown).

**5. "Tỷ lệ lỗi" trùng và trước đây có thể > 100%**

- **Mô tả**: panel "Tỷ lệ lỗi" có lúc hiển thị > 100%.
- **Nguyên nhân gốc**: BE cũ trộn kỳ (tử số toàn thời gian / mẫu số 30 ngày). FE cũng có thể nhân/chia lại từ hai số khác nguồn.
- **Phân loại**: **BE-029** (tử/mẫu/phân nhóm cùng kỳ; `errorRate` ≤ 100%, 0 khi mẫu 0) + **FE** (hiển thị thẳng `errorRate` của BE; **bỏ mọi tính lại cục bộ**).

**6. "DAU hôm nay" lấy từ snapshot có thể cũ**

- **Mô tả**: `overview.kpiCards.activeUsersDau` (và `kpi-cards`) lấy từ ảnh chụp worker (~15 phút/lần) nhưng nhãn "hôm nay" khiến admin tưởng realtime.
- **Nguyên nhân gốc**: KPI là snapshot theo thiết kế (FR-017 giữ nguyên); FE không hiển thị `updatedAt`.
- **Phân loại**: **FE** (hiển thị mốc `updatedAt`, chú thích "số liệu ảnh chụp định kỳ"). Không yêu cầu đổi BE.

**7. Range sai bị thay thế im lặng bằng 30 ngày**

- **Mô tả**: `normalizeDashboardFilters` biến khoảng ngày sai/không hợp lệ thành mặc định 30 ngày; admin tưởng đang xem kỳ mình chọn.
- **Nguyên nhân gốc**: FE fallback thủ công; trước 029 BE cũng nuốt lỗi (các endpoint kỳ parse nửa đêm UTC, đảo chiều bị bỏ qua, ngày sai bị bỏ qua).
- **Phân loại**: **BE-029** (mọi endpoint nhận kỳ trả **400** tiếng Việt khi sai định dạng/đảo chiều) + **FE** (bỏ fallback im lặng; hiển thị lỗi 400 từ BE).

**8. Filter bảng AI events lỗi giá trị mặc định (`'Tất cả'` vs `'all'`)**

- **Mô tả**: `AIEventsTable.tsx` — dropdown trạng thái mặc định gửi/so sánh giá trị không khớp enum hợp đồng; bảng lọc sai hoặc trả rỗng sai.
- **Nguyên nhân gốc**: FE dùng sentinel hiển thị (`'Tất cả'`, `'all'`) làm giá trị gửi lên; endpoint `drilldown/ai-events` nhận `status` **tuỳ chọn** (không có enum đóng).
- **Phân loại**: **FE** (option "Tất cả" ⇒ bỏ hẳn param `status`; chỉ gửi giá trị hợp lệ như `FAILED`).

**9. `console.log` sót**

- **Mô tả**: `DashboardInsights.tsx:165` còn `console.log` gỡ lỗi trong luồng render.
- **Nguyên nhân gốc**: log debug chưa được dọn.
- **Phân loại**: **FE** (xoá; thêm quy tắc lint chặn `console.log`).

**10. Lệch múi giờ khi parse `new Date('yyyy-MM-dd')` (UTC)**

- **Mô tả**: `DashboardTrendChart.tsx` — nhãn/trục ngày lệch 1 ngày (hoặc lệch giờ) so với ngày VN.
- **Nguyên nhân gốc**: JS hiểu chuỗi `yyyy-MM-dd` là **nửa đêm UTC**; khi format/đối chiếu với giờ local VN (+07:00) sinh lệch.
- **Phân loại**: **FE** (parse tường minh: dựng `Date` với `+07:00` hoặc parse thủ công; format bằng `Intl` với `timeZone: 'Asia/Ho_Chi_Minh'`). Với chuỗi ISO có offset từ BE, `new Date(iso)` là đúng.

**11. Granularity hardcode `'day'` + label "Tính theo : day"**

- **Mô tả**: `DashboardTrendChart.tsx:131` luôn hiển thị literal "Tính theo : day" và luôn tính theo ngày, kể cả khi preset là tháng.
- **Nguyên nhân gốc**: FE hardcode; BE `revenue-breakdown` chỉ nhóm theo **tháng**, các endpoint số liệu khác trả số vô hướng — **không** có tham số granularity.
- **Phân loại**: **FE** (sửa/xoá label; suy ra granularity từ preset) + **Deferred** (muốn doanh thu nhóm theo **ngày/tuần** cần bổ sung tham số granularity phía BE — ngoài phạm vi 029).

**12. Trộn ngôn ngữ hiển thị**

- **Mô tả**: các nhãn tiếng Anh lẫn tiếng Việt: "AI Health", "Paid tokens", "FAILED", …
- **Nguyên nhân gốc**: copy UI + dữ liệu enum (`FAILED`) hiển thị thô.
- **Phân loại**: **FE** (đồng bộ ngôn ngữ hiển thị; đặt text tiếng Việt cho mọi nhãn và map giá trị enum sang text thân thiện).

**13. Cột "Loại" hiển thị model + dữ liệu chết `totalBrands`/`pendingReviewBrands`**

- **Mô tả**: `AIEventsTable.tsx` — cột "Loại" đang hiển thị `model` (sai ngữ nghĩa); hai trường `totalBrands`/`pendingReviewBrands` có trong `kpi-cards` nhưng KPI card tương ứng bị comment nên không hiển thị (dữ liệu chết).
- **Nguyên nhân gốc**: drift giữa định nghĩa cột FE và bộ trường DTO (cột gán nhầm `model` — nhiều khả năng đúng ý là `errorCode`/operation hoặc `provider`); quyết định hiển thị số liệu brand chưa có.
- **Phân loại**: **FE** (sửa mapping cột "Loại" theo DTO thật — chọn đúng trường cần hiển thị, ví dụ `errorCode`; commit lại KPI card nếu muốn hiển thị) + **Deferred** (quyết định có hiển thị `totalBrands`/`pendingReviewBrands` hay không — cần ý kiến sản phẩm, không thuộc 029).

### 5.3 Danh sách deferred (phía FE)

1. Tham số **granularity** phía BE cho doanh thu (nhóm theo ngày/tuần) — cần thiết kế contract mới ngoài 029.
2. Quyết định hiển thị `totalBrands`/`pendingReviewBrands` trên dashboard — dữ liệu đã có trong `kpi-cards`, chờ yêu cầu sản phẩm.
3. Toàn bộ 13 fix UX phía FE nêu trên — ngoài phạm vi repo BE; theo dõi ở backlog `smart-wardrobe-fe`.

---

## 6. Bảng mã HTTP tổng hợp (mọi endpoint trong tài liệu)

`401` (thiếu/sai token) và `403` (không phải admin) áp dụng cho **mọi** endpoint; `500` áp dụng khi lỗi hệ thống. Bảng dưới chỉ liệt kê thêm các mã đặc thù.

| # | Method + path | Mã đặc thù | Điều kiện |
|---|---|---|---|
| 1 | `GET /api/v1/admin/activity-events` | 400 | `userId` sai UUID; `activityType` ngoài `login`/`refresh`; ngày sai định dạng/đảo chiều |
| 2 | `GET /api/v1/admin/subscriptions/transactions` | 400 | ngày sai định dạng/đảo chiều (`status` lạ bị **bỏ qua**, không lỗi) |
| 3 | `GET /api/v1/admin/subscriptions/transactions/{id}` | 404 | không tìm thấy giao dịch |
| 4 | `POST …/resolve-paid` | 400 | thiếu `reference`/`paymentLinkId`; trạng thái/bằng chứng không hợp lệ |
| 5 | `POST …/resolve-cancelled` | 400 | trạng thái không hợp lệ (đã `success`) |
| 6 | `POST …/resolve-creation-failed` | 400 | trạng thái hiện tại không thuộc `creating`/`reconciliation_required`/`investigation_required` |
| 7 | `POST …/retry-investigation` | 400 | trạng thái không hợp lệ (đã `success`) |
| 8 | `GET /api/v1/admin/dashboard/kpi-cards` | — (không có 400 theo kỳ) | tham số ngày bị bỏ qua |
| 9 | `GET /api/v1/admin/dashboard/overview` | — (không có 400 theo kỳ) | — |
| 10 | `GET /api/v1/admin/dashboard/ai-margin` | 400 | ngày sai định dạng/đảo chiều |
| 11 | `GET /api/v1/admin/dashboard/payos-reconciliation` | 400 | ngày sai định dạng/đảo chiều |
| 12 | `GET /api/v1/admin/dashboard/subscriptions/distribution` | 400 | ngày sai định dạng/đảo chiều |
| 13 | `GET /api/v1/admin/dashboard/subscriptions/revenue-breakdown` | 400 | ngày sai định dạng/đảo chiều |
| 14 | `GET /api/v1/admin/dashboard/subscriptions/renewal-stats` | 400 | ngày sai định dạng/đảo chiều |
| 15 | `GET /api/v1/admin/dashboard/ai/error-breakdown` | 400 | ngày sai định dạng/đảo chiều |
| 16 | `GET /api/v1/admin/dashboard/ai/usage-by-operation` | 400 | ngày sai định dạng/đảo chiều |
| 17 | `GET /api/v1/admin/dashboard/drilldown/ai-events` | 400 | `page`/`limit` sai kiểu |

---

## 7. Checklist triển khai frontend

- [ ] **Chung**
  - [ ] Gửi `Authorization: Bearer <admin token>` cho mọi call; xử lý 401 → đăng nhập lại, 403 → ẩn màn hình.
  - [ ] Đọc lỗi qua `message` (không phụ thuộc `debugMessage`/`stackTrace`); hiển thị `errors[]` lên từng ô khi có.
  - [ ] Không parse `yyyy-MM-dd` bằng `new Date(str)`; format mọi thời gian theo `Asia/Ho_Chi_Minh`.
  - [ ] Xử lý `data` vắng mặt ở phản hồi của 4 hành động (200 chỉ có `message`).

- [ ] **Nhóm (a) — trang "Hoạt động người dùng"**
  - [ ] Gọi `GET /api/v1/admin/activity-events` với bộ lọc `userId`/`activityType`/`fromDate`/`toDate`/`page`/`limit`.
  - [ ] "Tất cả" activityType ⇒ bỏ param; validate UUID trước khi gửi.
  - [ ] Hiển thị tên = `firstName + lastName`, fallback `userEmail`; chịu được `userFirstName`/`userLastName` vắng mặt.
  - [ ] Phân trang server-side theo `metadata` (`totalPages = 0` khi rỗng — không phải lỗi).
  - [ ] Hiển thị `message` khi 400 thay vì nuốt lỗi.

- [ ] **Nhóm (b) — trang "Giao dịch payOS"**
  - [ ] Map đủ **11 giá trị `status`** → label + màu; có nhánh `unknown`.
  - [ ] Filter `status` chỉ gửi giá trị hợp lệ; kỳ ngày đủ hai đầu (mục 1.3); xử lý 400.
  - [ ] Detail dùng **cùng interface dữ liệu** với item danh sách.
  - [ ] 4 nút hành động: dialog xác nhận + `reason`; `resolve-paid` bắt buộc `reference` + `paymentLinkId`; bật/tắt nút theo `status` (mục 3.5).
  - [ ] Sau hành động 200 (không có `data`): toast `message` + **refetch detail**; không tự đoán trạng thái.
  - [ ] Xử lý 404 "Không tìm thấy giao dịch." khi mở detail với id không tồn tại (đã verify live).

- [ ] **Nhóm (c) — dashboard**
  - [ ] Thêm preset "Hôm nay", "Tháng này" (tính theo VN, mục 4.8); bỏ mặc định 30 ngày cuốn cứng.
  - [ ] Gửi `fromDate`/`toDate` **đủ hai đầu** cho cả 7 endpoint theo kỳ; cùng một kỳ cho mọi panel.
  - [ ] Bỏ `normalizeDashboardFilters` im lặng; hiển thị 400 từ BE và giữ nguyên input.
  - [ ] Tách khu vực "Ảnh chụp hiện tại" (KPI + `overview.kpiCards`) vs "Theo kỳ đã chọn"; hiển thị `updatedAt` cho KPI.
  - [ ] Gắn nhãn từng panel theo bảng mục 4.8; ghi rõ `distribution` (giới hạn lịch sử đổi gói), `renewal-stats` (attempt tạo trong kỳ, loại `processing`), `payos-reconciliation` (hai loại giao dịch).
  - [ ] Hiển thị thẳng `errorRate`, `totalSuccessValue`, `totalRevenueVnd` từ BE — bỏ mọi tính lại cục bộ.
  - [ ] `ai-margin` trả `fromDate`/`toDate` là **00:00 ngày VN** của ngày đầu/cuối kỳ; `revenue-breakdown`/`usage-by-operation` trả `YYYY-MM-DD` bao gồm hai đầu — hiển thị phần ngày VN, không convert theo timezone browser.
  - [ ] Bảng AI events: "Tất cả" ⇒ bỏ param `status`; sửa cột "Loại" đúng trường DTO (có `errorCode`).
  - [ ] Xoá `console.log` sót; thống nhất ngôn ngữ hiển thị (tiếng Việt).

---

## 8. Biên bản đồng bộ T046 (đối chiếu có đếm, 2026-10-16)

Phiên verify live chạy trên stack Docker dev (`localhost:8080`), 2 tài khoản verify mới trong DB dev
(admin029/user029), dữ liệu giao dịch/subscription/renewal chèn trực tiếp vào DB dev (order_code 29001–29004).
Kết quả đếm đối chiếu guide ↔ mã nguồn:

| Hạng mục | Số đếm | Kết quả |
|---|---|---|
| Giá trị enum `status` (Go ↔ guide, 2 chiều) | 11/11 | khớp; `unknown` là giá trị phòng thủ ngoài enum, có ghi chú |
| Trường JSON của các DTO nhóm (a)/(b)/(c) xuất hiện trong guide | 71/71 | khớp |
| Route guide liệt kê (dạng OpenAPI `{param}`) ↔ router gin | 17/17 | khớp |
| Mã HTTP guide khẳng định (400/401/403/404/500) | 4 nhóm | 400/401/403/404 đã gọi live; 500 là nhánh middleware lỗi hệ thống |

Đã verify live các kịch bản 1–5 của `quickstart.md` (chi tiết: log phiên orchestrate 029):
activity-events (lọc/400/403), transactions (status chuỗi 11 giá trị, `failed_legacy` khứ hồi,
biên 23:30 VN, 400 đảo chiều, **404 id không tồn tại**), payOS theo kỳ (value 150000 khớp SQL
`SUM(FILTER status=success)` cả 2 loại; tách khỏi revenue direct-only 100000), distribution giao
cửa sổ, renewal theo `created_at`, error-breakdown cùng kỳ (rate 0 khi không dữ liệu), 400 nhất
quán 3 endpoint mẫu, echo ngày VN của `ai-margin`/`usage-by-operation`, cap `limit` 100.

Hai lệch phát hiện khi verify đã được sửa trong 029 (kèm test + negative control):
1. `GET /transactions/{id}` id không tồn tại: 200-rỗng → **404** (`fixup! 5cbf38ac`).
2. `ai-margin` echo `toDate` là cận mở (00:00 ngày kế tiếp) → **00:00 ngày cuối kỳ** (`fixup! cd920cfe`).

Ví dụ JSON trong guide đã được đối chiếu hình dạng/trường với response live; **số liệu trong ví dụ
là minh hoạ** (không khớp dữ liệu dev). Hai điểm đã ghi nhận nhưng KHÔNG sửa trong 029 (pre-existing,
xem `deferred.md`): pagination `limit` của list giao dịch (handler nhận tới 200 trong khi metadata
chuẩn hoá trần 100 — FE nên gửi ≤ 100), và message 401 hiện là chuỗi mã nội bộ (FE không hiển thị thô).
