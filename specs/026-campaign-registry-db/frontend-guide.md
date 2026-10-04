# Frontend Integration Guide — Sổ Đăng Ký Chiến Dịch (mở / sửa / đóng qua API)

**Feature**: `026-campaign-registry-db`
**Trạng thái**: Hoàn thiện (feature 026 đã triển khai; chiến dịch `launch-2026-10` đã được
chuyển từ YAML sang bảng `subscription.campaigns`).
**Đối tượng đọc**: Lập trình viên frontend của **trang quản trị** (màn hình vận hành chiến dịch).
**Mục tiêu**: FE chỉ cần đọc tài liệu này để triển khai, KHÔNG cần đọc code backend hay Swagger.

> **Đây không phải API dành cho khách hàng.** Cả 8 endpoint nằm dưới `/api/v1/admin/campaigns`
> và yêu cầu vai trò `Admin`. Người dùng cuối không bao giờ gọi trực tiếp, và 026 **không** thêm
> bất kỳ endpoint nào cho khách hàng.
>
> Quy ước: mọi trường JSON dùng `camelCase`; thời gian là RFC3339 **có múi giờ**
> (`2027-01-20T00:00:00+07:00`). Nội dung này mô tả hành vi API ở mức hợp đồng.

---

## 0. Tóm tắt nhanh (TL;DR)

- **4 endpoint MỚI**: 3 endpoint **ghi** (`POST` tạo, `PATCH` sửa, `POST .../close` đóng) và
  1 endpoint **đọc nhật ký kiểm toán** (`GET .../audit`). Trước 026, mở/sửa/đóng chiến dịch phải
  sửa YAML rồi deploy; giờ là một request.
- **4 endpoint đọc cũ KHÔNG ĐỔI** đường dẫn, không đổi tên/kiểu trường nào; 026 chỉ **thêm** 3
  trường: `closedAt`, `budgetLocked`, `version` (mục 5).
- ⚠️ **Hạn chế quan trọng nhất của tài liệu này**: **3 endpoint ghi KHÔNG trả số liệu ngân sách
  thật.** Xem mục 2 — đọc mục đó **trước khi** viết màn hình.
- `status` giờ có **6** giá trị (thêm `closed`). Chịu được giá trị lạ.
- `budgetLocked = true` ⇒ **khoá** ô `quota` / `reserve` / `planSlug`, kèm chú thích
  *"chiến dịch đã cấp — hãy đóng và mở mã mới"*.
- `version` phải **gửi lại** khi `PATCH` / `close`; lệch ⇒ `412` ⇒ **tải lại** rồi thao tác lại.
- Tập `reason` **giữ nguyên đúng 12 mã** của 025. Đóng chiến dịch không sinh mã mới.
- Nhật ký kiểm toán có `targetId` **luôn là `null`** (không bị bỏ khỏi JSON); mã chiến dịch nằm ở
  `payload.campaignCode`.

---

## 1. Bản đồ endpoint

Tất cả nằm trong nhóm riêng `/api/v1/admin/campaigns`. Trạng thái = **MỚI** (026 thêm) hoặc
**ĐÃ CÓ** (từ 025, không đổi).

| # | Method | Path | Trạng thái | Dùng để |
|---|--------|------|-----------|---------|
| 1 | `GET` | `/api/v1/admin/campaigns` | ĐÃ CÓ | Danh sách chiến dịch + ngân sách |
| 2 | `GET` | `/api/v1/admin/campaigns/{code}` | ĐÃ CÓ | Chi tiết một chiến dịch |
| 3 | `GET` | `/api/v1/admin/campaigns/{code}/claims` | ĐÃ CÓ | Danh sách tài khoản đã được cấp |
| 4 | `GET` | `/api/v1/admin/campaigns/{code}/eligibility/{userId}` | ĐÃ CÓ | Trạng thái một tài khoản trong chiến dịch |
| 5 | `POST` | `/api/v1/admin/campaigns` | **MỚI** | **Mở** chiến dịch mới |
| 6 | `PATCH` | `/api/v1/admin/campaigns/{code}` | **MỚI** | **Sửa** chiến dịch đang tồn tại |
| 7 | `POST` | `/api/v1/admin/campaigns/{code}/close` | **MỚI** | **Đóng cưỡng bức** (một chiều) |
| 8 | `GET` | `/api/v1/admin/campaigns/{code}/audit` | **MỚI** | Nhật ký ai / lúc nào / đổi gì |

Cả 8 đều bọc trong `APIResponse { success, message, data }` và đều trả:

| Mã | Khi nào | FE làm gì |
|----|---------|-----------|
| `403` | Không có vai trò `Admin` | Ẩn màn hình; đây không phải lỗi nghiệp vụ để hiển thị |
| `429` | Vượt giới hạn tần suất theo quản trị viên | Thông báo "vui lòng thử lại sau", cho phép retry |

Ngoài ra mỗi endpoint **ghi** còn trả `401` nếu không xác định được người gọi trong token
("Vui lòng đăng nhập để thực hiện thao tác quản lý chiến dịch.") — xử lý như một phiên đăng
nhập hết hạn.

---

## 2. ⚠️ Hạn chế của response 3 endpoint ghi — ĐỌC TRƯỚC

**Đường ghi cố tình KHÔNG đọc sổ lượt cấp** (`campaign_claims`). Ba use case ghi nhận
`CampaignGrantCounts{}` rỗng, nên các trường sau trong response của `POST` (tạo),
`PATCH` (sửa) và `POST .../close` (đóng) là **số 0 hoặc số dẫn xuất từ hàng đã ghi**, KHÔNG phải
số liệu thật:

| Trường | Giá trị trong response ghi | Vì sao |
|--------|---------------------------|--------|
| `grantedMainCount` | **`0`** | Sổ lượt cấp không được đọc |
| `grantedCompensationCount` | **`0`** | như trên |
| `grantedTotalCount` | **`0`** | là tổng của hai số trên |
| `budgetLocked` | **`false`** | dẫn xuất từ `grantedTotalCount > 0` |
| `remainingMain` | **`quota`** | dẫn xuất từ ngân sách, không phải từ số đã cấp |
| `remainingCompensation` | **`reserve`** | như trên |
| `hardCap` | **`quota + reserve`** | như trên |
| `watermarkAt` | **`null`** | mốc ưu tiên nằm trong sổ lượt cấp |

Ngoài ra 4 trường sau cũng rỗng trên response ghi, vì chúng đến từ những bảng mà đường ghi không
cần đọc: `planName` (`""`), `degraded` (`false`), `degradedReason` (rỗng), `lastSweepAt` (`null`).

### Hệ quả bắt buộc cho FE

1. **Sau MỌI lần ghi, phải `GET` lại chiến dịch** (`GET /api/v1/admin/campaigns/{code}`) trước khi
   hiển thị số ngân sách. Đừng vẽ thanh tiến độ hay "còn N suất" từ response ghi — con số đó sẽ
   sai và sai theo hướng **nguy hiểm**: nó báo còn nguyên hạn mức cho một chiến dịch đã cấp hết.
2. **Đừng dùng `budgetLocked` từ response ghi** để quyết định khoá ô sửa. Nó **luôn** là `false`
   ở đó. Phải đọc nó từ `GET`.
3. Những gì response ghi **có** giá trị và dùng được ngay: `code`, `planSlug`, `quota`, `reserve`,
   `startsAt`, `endsAt`, **`closedAt`**, **`version`**, và `status` (dẫn xuất lại từ đồng hồ).
4. Sau `close`, `status` trong response là `closed` và `closedAt` đã có giá trị — nhưng FE vẫn nên
   `GET` lại để lấy số liệu ngân sách thật và để danh sách ở mục 1 được đồng bộ.

> Lý do kỹ thuật: đường ghi không giữ repository của sổ lượt cấp. Cho nó giữ chỉ để lấy một
> boolean là biến đường ghi thành người đọc thứ hai của sổ lượt cấp — đúng thứ 025 R7 cảnh báo.
> Đây là **giới hạn đã biết và cố ý**, không phải lỗi.

---

## 3. Ba endpoint ghi và endpoint nhật ký (chi tiết)

### 3.1 `POST /api/v1/admin/campaigns` — **MỚI** — mở chiến dịch

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

| Trường thân | Kiểu | Bắt buộc | Quy tắc |
|---|---|---|---|
| `code` | chuỗi | Có | `^[a-z0-9][a-z0-9-]{2,63}$` — 3–64 ký tự, chữ thường / số / gạch ngang, bắt đầu bằng chữ thường hoặc số. **Sai luật là `400`** |
| `planSlug` | chuỗi | Có | rỗng ⇒ `422`. Không kiểm tra gói tồn tại lúc tạo — sẽ kiểm lúc chạy |
| `quota` | số nguyên | Có | `>= 1`; `0` là `422` (không phải `400`) |
| `reserve` | số nguyên | Không | mặc định `0`; `>= 0` |
| `startsAt` | chuỗi RFC3339 | Có | **phải có múi giờ** |
| `endsAt` | chuỗi RFC3339 \| `null` | Không | nếu có thì phải **sau** `startsAt`; `null` = không đặt ngày kết thúc |

Trả `201` với một `CampaignSummaryRes` (mục 5.1). Số liệu ngân sách là số 0/dẫn xuất — xem mục 2.
`version` trả về là `1`.

Mã lỗi: `400` (mã sai định dạng, thiếu trường bắt buộc, thời gian không RFC3339 có múi giờ),
`401`, `403`, **`409`** (mã đã tồn tại), `422`, `429`, `500`.

### 3.2 `PATCH /api/v1/admin/campaigns/{code}` — **MỚI** — sửa chiến dịch

```json
{ "version": 1, "quota": 120, "startsAt": "2027-01-20T00:00:00+07:00", "endsAt": null }
```

| Trường thân | Kiểu | Bắt buộc | Quy tắc |
|---|---|---|---|
| `version` | số nguyên | Có | phiên bản FE đang giữ; lệch ⇒ `412` |
| `planSlug` | chuỗi | Không | đổi được **chỉ khi** `budgetLocked = false` |
| `quota` | số nguyên | Không | `>= 1`; đổi được **chỉ khi** `budgetLocked = false` |
| `reserve` | số nguyên | Không | `>= 0`; đổi được **chỉ khi** `budgetLocked = false` |
| `startsAt` | chuỗi RFC3339 | Không | sửa được **bất kể** đã cấp hay chưa |
| `endsAt` | chuỗi RFC3339 \| `null` | Không | `null` = **xoá** ngày kết thúc; phải sau `startsAt` |

Ba điều FE cần biết chính xác:

- **Trường vắng mặt = không đổi.** Đừng gửi lại toàn bộ dữ liệu form: một trường bạn gửi bằng
  `null`/`0`/`""` sẽ được hiểu là *muốn đặt về giá trị đó* và thường bị `422`.
- **`code` KHÔNG có trong thân.** Mã là bất biến sau khi tạo. Gửi `code` trong thân bị **bỏ qua
  âm thầm** — không có ô nào để đổi mã, và đừng hiển thị ô nhập mã ở màn hình sửa.
- **Không có trường nào đổi ⇒ `200` no-op.** Không ghi, không tăng `version`, không sinh dòng nhật
  ký. FE vẫn nên cập nhật `version` từ response (nó không đổi).

Trả `200` với `CampaignSummaryRes` đã cập nhật (số liệu ngân sách vẫn là số 0/dẫn xuất — mục 2).
Mã lỗi: `400`, `401`, `403`, `404` (mã không tồn tại), `412` (phiên bản lệch), `422`, `429`, `500`.

### 3.3 `POST /api/v1/admin/campaigns/{code}/close` — **MỚI** — đóng cưỡng bức

```json
{ "version": 3, "reason": "Hết ngân sách khuyến mãi tháng 11" }
```

| Trường thân | Kiểu | Bắt buộc | Quy tắc |
|---|---|---|---|
| `version` | số nguyên | Có | lệch ⇒ `412` (trừ khi chiến dịch đã đóng) |
| `reason` | chuỗi | Có | rỗng ⇒ `422`; được lưu vào nhật ký kiểm toán |

Hệ quả phía server (không phải phía FE nhưng FE cần biết để giải thích cho người vận hành):

- Chiến dịch **ngừng cấp gói ngay**, kể cả tài khoản đã đăng ký nhưng thông điệp tới trễ.
- Tài khoản **đã được cấp** giữ nguyên gói và hạn — đóng không lấy lại quyền lợi.
- **Đóng là một chiều.** Không có endpoint mở lại. Muốn chạy lại → tạo **mã mới**.
- **Gọi lại trên chiến dịch đã đóng trả `200` idempotent**, không ghi thêm dòng nhật ký. FE có thể
  cho phép bấm lại nút "Đóng" mà không cần xử lý riêng.

Trả `200` với `CampaignSummaryRes` (số liệu ngân sách là số 0/dẫn xuất — mục 2). `status` là
`closed`, `closedAt` có giá trị. **Nên `GET` lại** để hiện số liệu thật và đồng bộ danh sách.

Mã lỗi: `400`, `401`, `403`, `404`, `412`, `422`, `429`, `500`.

### 3.4 `GET /api/v1/admin/campaigns/{code}/audit` — **MỚI** — nhật ký kiểm toán

Tham số: `page`, `limit` (int, tuỳ chọn). Chỉ đọc, có phân trang.

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

Đây là **DTO riêng** của endpoint này, không phải DTO nhật ký chung:

| Trường | Kiểu | Ghi chú |
|--------|------|---------|
| `id` | chuỗi (UUID) | |
| `actorUserId` | chuỗi (UUID) | ai đã thao tác |
| `action` | chuỗi | `campaign.create` / `campaign.update` / `campaign.close` (mục 5.5) |
| `targetType` | chuỗi | luôn là `campaign` |
| `targetId` | **`null` — luôn có mặt** | ⚠️ **không** bị bỏ khỏi JSON (xem dưới) |
| `payload` | object | tài liệu tự do, xem bên dưới |
| `requestIp` | chuỗi, có thể vắng | vắng khi không có IP |
| `createdAt` | chuỗi RFC3339 | thời điểm ghi nhật ký |

⚠️ **`targetId` luôn là `null` và luôn có mặt trong JSON.** Đừng viết `if (item.targetId)` để ẩn
cột — nó luôn là `null` ở đây vì mã chiến dịch là chuỗi chứ không phải UUID. Mã chiến dịch nằm ở
`payload.campaignCode`. Endpoint này được tách DTO riêng chính để bất biến "mọi dòng chiến dịch có
`targetId` là `null`" nhìn thấy được.

Nội dung `payload` theo từng `action`:

| `action` | `payload` chứa |
|---|---|
| `campaign.create` | `campaignCode`, `planSlug`, `quota`, `reserve`, `startsAt`, `endsAt` — **không** có `before`/`after` (trước đó không tồn tại) |
| `campaign.update` | `campaignCode`, `before`, `after`. Mỗi vế **luôn** có `version`, cộng thêm **chỉ những trường đã thay đổi**: `planSlug`, `quota`, `reserve`, `startsAt`, `endsAt`. Thời gian ở dạng UTC RFC3339; `endsAt` đã xoá hiển thị là `""` (chuỗi rỗng), không phải vắng mặt |
| `campaign.close` | `campaignCode`, `reason`, và `before`/`after` — **luôn có cả hai**, mỗi vế gồm `closedAt` (`before` là `""`) và `version` (`before` là phiên bản cũ, `after` là phiên bản mới) |

> **Đừng dựng bảng trước → sau bằng cách lặp toàn bộ khoá của `payload`.** `before` và `after` là
> **tài liệu tự do** với tập khoá khác nhau theo `action` (mục 5.5). Vẽ `—` cho khoá không có ở
> vế kia, và coi `version` là cột bắt buộc ở cả hai vế.

> **Nhật ký được ghi BẤT ĐỒNG BỘ.** Một thao tác vừa thành công có thể chưa xuất hiện trong danh
> sách ngay lập tức. Đừng hiển thị "chưa có thay đổi nào" như một kết luận về hệ thống; hãy tải
> lại sau một nhịp ngắn.

Mã lỗi: `400`, `403`, **`404`** (mã không tồn tại — nhật ký rỗng **không phải** câu trả lời cho một
mã lạ), `429`, `500`.

---

## 4. Bốn API đọc — bảng TRƯỚC → SAU

| Endpoint | Trước 026 | Sau 026 |
|---|---|---|
| `GET /api/v1/admin/campaigns` | Danh sách chiến dịch, 17 trường mỗi phần tử | **không đổi đường dẫn, không đổi tên/kiểu trường nào**; thêm `closedAt`, `budgetLocked`, `version` |
| `GET /api/v1/admin/campaigns/{code}` | Chi tiết một chiến dịch, 17 trường | **không đổi**; thêm `closedAt`, `budgetLocked`, `version` |
| `GET /api/v1/admin/campaigns/{code}/claims` | Danh sách tài khoản được cấp | **không đổi** — không thêm trường nào |
| `GET /api/v1/admin/campaigns/{code}/eligibility/{userId}` | Kết luận cho một tài khoản | **không đổi** — không thêm trường nào; vẫn 6 giá trị `eligibility` và 12 mã `reason` |

Nguồn danh sách chiến dịch **đổi từ tệp cấu hình sang cơ sở dữ liệu**, nhưng điều đó không nhìn
thấy được từ hợp đồng: một mã không có trong sổ vẫn trả `404` y như trước.

### 4.1 Ba trường mới

| Trường | Kiểu | Ở đâu | Nghĩa |
|--------|------|-------|-------|
| `closedAt` | `string` \| `null` | API 1 và 2 | thời điểm quản trị viên đóng cưỡng bức; `null` = chưa từng bị đóng |
| `budgetLocked` | `boolean` | API 1 và 2 | `true` khi chiến dịch đã cấp cho **ít nhất một** tài khoản |
| `version` | `number` | API 1 và 2 | phiên bản để gửi lại khi `PATCH` / `close` |

### 4.2 Ví dụ sau 026 (`GET /api/v1/admin/campaigns/{code}`)

```json
{
  "code": "launch-2026-10",
  "planSlug": "premium-monthly",
  "planName": "Premium Tháng",
  "quota": 50,
  "reserve": 5,
  "grantedMainCount": 50,
  "grantedCompensationCount": 5,
  "grantedTotalCount": 55,
  "remainingMain": 0,
  "remainingCompensation": 0,
  "hardCap": 55,
  "startsAt": "2026-10-02T00:00:00+07:00",
  "endsAt": null,
  "watermarkAt": "2026-10-02T00:04:17+07:00",
  "status": "exhausted",
  "degraded": false,
  "degradedReason": "",
  "lastSweepAt": "2026-10-02T20:31:12+07:00",
  "closedAt": null,
  "budgetLocked": true,
  "version": 1
}
```

Ba trường cũ khác FE hay nhầm, giữ nguyên nghĩa:

| Trường | Ý nghĩa |
|-------|---------|
| `endsAt` | `null` = **không đặt** ngày kết thúc, không phải "chưa biết" |
| `closedAt` | `null` = chưa từng bị đóng cưỡng bức |
| `endsAt` **và** `closedAt` | không thay cho nhau: `endsAt` đã qua = chiến dịch **tự** kết thúc; `closedAt` có giá trị = **con người** dừng nó. Một chiến dịch có thể có cả hai. |

### 4.3 Quy tắc khoá ô sửa khi `budgetLocked = true`

Khi `budgetLocked = true`, **vô hiệu hoá** ba ô `quota`, `reserve`, `planSlug` và hiển thị chú
thích:

> **Chiến dịch đã cấp — hãy đóng và mở mã mới.**

Đây không phải giới hạn của UI: server trả `422` cho mọi thay đổi ba trường đó, và cơ sở dữ liệu
cũng chặn. Ô `startsAt` / `endsAt` **vẫn sửa được** (đó là tính năng của 026, US4).

---

## 5. Các tập giá trị

### 5.1 `status` của chiến dịch — **6** giá trị

| `status` | Nghĩa | FE hiển thị |
|---|---|---|
| `not_started` | Chưa tới `startsAt` | "Sắp mở" + ngày bắt đầu |
| `running` | Đang mở, hạn mức chính chưa đầy, chưa chốt mốc ưu tiên | "Đang chạy" |
| `compensation` | Đã chốt mốc ưu tiên, còn suất dự phòng | "Đang chạy — chỉ còn suất dự phòng" |
| `exhausted` | Đạt trần cứng `quota + reserve` — tự ngủ | "Đã hết suất" |
| `expired` | Đã qua `endsAt` — tự ngủ | "Đã kết thúc" |
| **`closed`** | **Quản trị viên đóng cưỡng bức (MỚI ở 026)** | **"Đã đóng"** + thời điểm `closedAt` |

`closed` là trạng thái **ưu tiên cao nhất**: chiến dịch đã đóng luôn hiển thị `closed`, bất kể đồng
hồ và ngân sách. Nó đứng trước cả `expired`.

> ⚠️ **Chịu được giá trị lạ.** Hãy có một nhánh `default` khi map `status` → nhãn hiển thị. Backend
> có thể thêm giá trị trong các phiên bản sau và trang không được hiện trống hay vỡ.

`status` **tách khỏi** cờ `degraded`. `degraded: true` nghĩa là *chiến dịch đang mở nhưng hỏng,
không ai được cấp* — đây là cảnh báo nghiêm trọng nhất, tô đập khác mọi thứ khác.

### 5.2 `eligibility` của một tài khoản — **6** giá trị (không đổi)

| `eligibility` | Nghĩa | `granted` | FE nên nói với khách |
|---|---|---|---|
| `granted` | Đã được cấp trong chiến dịch này | `true` | "Bạn đã được tặng gói, hạn đến `<expiresAt>`" |
| **`eligible_pending`** | **Đủ điều kiện, còn suất, nhưng CHƯA được cấp** | `false` | **"Bạn đủ điều kiện, hệ thống đang xử lý"** |
| `eligible_but_exhausted` | Đủ điều kiện nhưng hết suất, hoặc tạo sau mốc ưu tiên | `false` | "Chiến dịch đã hết suất" |
| `not_eligible` | Không thỏa điều kiện nào | `false` | Xem `reason` |
| `already_claimed_other_campaign` | Đã tham gia chiến dịch trước | `false` | "Bạn đã được tặng gói ở chiến dịch trước" |
| `created_by_admin` | Tài khoản do quản trị viên tạo | `false` | "Tài khoản do quản trị viên tạo nên không áp dụng chiến dịch" |

Tra một tài khoản trong **chiến dịch đã đóng** trả `eligibility = "not_eligible"` kèm
`reason = "campaign_sleeping"` — không có giá trị `eligibility` mới.

### 5.3 `reason` — **12** mã, KHÔNG đổi

026 **không thêm** mã `reason` nào. Chiến dịch đã đóng làm điều kiện tra cứu trả
`campaign_sleeping` — cùng mã 025 đã dùng cho "quá `endsAt`".

| Mã `reason` | Nghĩa | Text FE nên hiển thị |
|---|---|---|
| *(rỗng)* | Đủ điều kiện, đang chờ — **không** phải lỗi | chỉ dùng cho `eligible_pending` và `granted` |
| `campaign_not_configured` | Chưa khai báo chiến dịch nào | "Chưa có chiến dịch nào đang chạy" |
| `campaign_not_started` | Chưa tới giờ mở | "Chiến dịch chưa bắt đầu" |
| `campaign_sleeping` | Chiến dịch đã kết thúc **hoặc đã bị đóng** | "Chiến dịch đã kết thúc" |
| `created_by_admin` | Tài khoản do quản trị viên tạo | "Tài khoản do quản trị viên tạo" |
| `already_claimed` | Đã có lượt cấp | "Bạn đã được tặng gói rồi" |
| `already_on_plan` | Đang ở đúng gói quà tặng và còn hạn — cấp thêm sẽ chồng | "Bạn đang có gói này rồi" |
| `plan_not_settled` | Gói cũ chưa dọn xong | "Thử lại sau ít phút" |
| `outside_window` | Tạo tài khoản ngoài cửa sổ chiến dịch | "Tài khoản tạo ngoài thời gian chiến dịch" |
| `quota_main` | Hết suất phần chính | "Chiến dịch đã hết suất" |
| `quota_compensation` | Hết suất dự phòng — tức chạm trần cứng | "Chiến dịch đã hết suất" |
| `quota_anomaly` | Phần chính đầy nhưng mốc ưu tiên chưa chốt — dữ liệu bất thường | "Hệ thống đang kiểm tra, thử lại sau" |
| `plan_unavailable` | `planSlug` trỏ tới gói không tồn tại hoặc đã ngừng | "Chiến dịch đang được cấu hình lại" |

`reason` là **mã ổn định** — FE tự đặt text hiển thị, **đừng so khớp chuỗi** để quyết định nghiệp
vụ. Mã lạ → hiển thị thông báo mặc định.

### 5.4 `source` của một lượt cấp — **2** giá trị (không đổi)

Chỉ có `"main"` và `"reserve"`. Không có `"compensation"` hay `"main_quota"` — đó là tên trạng thái,
không phải tên phần ngân sách.

### 5.5 `action` của nhật ký kiểm toán chiến dịch — **3** giá trị (MỚI ở 026)

| `action` | Khi nào | `payload` |
|---|---|---|
| `campaign.create` | Mở chiến dịch | `campaignCode`, `planSlug`, `quota`, `reserve`, `startsAt`, `endsAt` (không có `before`/`after`) |
| `campaign.update` | Sửa chiến dịch | `campaignCode`, `before`, `after` — mỗi vế có `version` **luôn**, cộng chỉ những trường đã đổi (`planSlug` / `quota` / `reserve` / `startsAt` / `endsAt`) |
| `campaign.close` | Đóng cưỡng bức | `campaignCode`, `reason`, `before`, `after` — mỗi vế gồm `closedAt` và `version` |

Đây là **3 giá trị riêng**, không trộn vào nhóm hành động của gói cước: một thao tác chiến dịch
không cấp gói cho ai, nên người vận hành lọc "ai đã trao Premium" theo `subscription.grant` sẽ
không thấy nó. `targetType` của cả ba là `campaign`.

### 5.6 `targetType` — **1** giá trị trên endpoint nhật ký chiến dịch

Luôn là `campaign`.

---

## 6. Mã lỗi chi tiết

### 6.1 Endpoint ghi

| Tình huống | HTTP | Thông báo tiếng Việt từ backend | FE làm gì |
|---|---|---|---|
| Sai định dạng `code`, hoặc thời gian không phải RFC3339 **có múi giờ** | `400` | "Mã chiến dịch không hợp lệ. Mã phải dài 3-64 ký tự, chỉ gồm chữ thường, số và dấu gạch ngang, và bắt đầu bằng chữ thường hoặc số." / "Tham số thời gian không hợp lệ: `<trường>` phải là một thời điểm RFC3339 có múi giờ, ví dụ 2026-10-02T00:00:00+07:00." | Báo lỗi ngay cạnh ô nhập, không phải thông báo chung |
| Thiếu trường bắt buộc | `400` | Thông báo chuẩn của bộ kiểm tra thân | Báo lỗi ngay cạnh ô nhập |
| `quota < 1`, `reserve < 0`, `endsAt` không sau `startsAt`, `planSlug` rỗng, `reason` đóng rỗng | `422` | "Trường `<tên trường>` không hợp lệ: `<lý do>`." | Báo lỗi ngay cạnh ô nhập |
| Sửa `planSlug`/`quota`/`reserve` khi `budgetLocked = true` | `422` | "Chiến dịch `<code>` đã cấp gói cho ít nhất một tài khoản, nên hạn mức (quota), lượng dự phòng (reserve) và mã gói (planSlug) không thể thay đổi nữa. Vui lòng đóng chiến dịch này và mở một mã chiến dịch mới." | Đã khoá ô ở UI (mục 4.3); nếu vẫn dính, hiện đúng thông báo này |
| Mã đã tồn tại (chỉ khi tạo) | `409` | "Đã tồn tại chiến dịch với mã `<code>`. Vui lòng chọn một mã khác." | Báo lỗi ngay cạnh ô `code` |
| Phiên bản lệch | `412` | "Chiến dịch `<code>` đã được thay đổi bởi thao tác khác. Vui lòng tải lại thông tin chiến dịch rồi thực hiện lại thay đổi." | **Tải lại** chiến dịch, thông báo "dữ liệu đã thay đổi" |
| Mã không tồn tại | `404` | "Không tìm thấy chiến dịch với mã đã nhận: `<code>`." | Sửa ô mã, **không** hiển thị danh sách rỗng |
| Chưa xác định được người gọi | `401` | "Vui lòng đăng nhập để thực hiện thao tác quản lý chiến dịch." | Điều hướng đăng nhập lại |
| Không phải `Admin` | `403` | Thông báo chuẩn | Ẩn màn hình |
| Vượt giới hạn tần suất | `429` | Thông báo chuẩn | "Vui lòng thử lại sau", cho phép retry |
| Lỗi hạ tầng | `500` | "Không thể lưu thông tin chiến dịch. Vui lòng thử lại sau." | "Thử lại", **không** mời sửa dữ liệu đã đúng |

### 6.2 Endpoint đọc nhật ký

| Tình huống | HTTP | Thông báo | FE làm gì |
|---|---|---|---|
| Mã không tồn tại | `404` | "Không tìm thấy chiến dịch với mã đã nhận: `<code>`." | Sửa ô mã; **không** hiển thị "chưa có thay đổi nào" |
| Sai tham số phân trang | `400` | Thông báo chuẩn | Báo lỗi tham số |
| Lỗi hạ tầng | `500` | "Không thể tra cứu nhật ký kiểm toán của chiến dịch. Vui lòng thử lại sau." | "Thử lại" |

### 6.3 Bốn API đọc (không đổi so với 025)

| Tình huống | HTTP | Thông báo | FE làm gì |
|---|---|---|---|
| Thiếu/sai tham số đường dẫn hoặc truy vấn | `400` | Thông báo chuẩn của bộ kiểm tra tham số | Báo lỗi tham số |
| `from`/`to` sai định dạng ở API danh sách lượt cấp | `400` | "Tham số truy vấn `"from"` không hợp lệ: cần một thời điểm RFC3339 có múi giờ, ví dụ 2026-10-02T00:00:00+07:00." | Báo sai định dạng thời gian |
| Mã chiến dịch không tồn tại | `404` | "Không tìm thấy chiến dịch với mã đã nhận: `<code>`." | Sửa ô mã, **không** hiển thị danh sách rỗng |
| `userId` không tồn tại ở API tra một tài khoản | `404` | "Không tìm thấy tài khoản." | Kiểm tra lại mã khách |
| Không phải `Admin` | `403` | Thông báo chuẩn | Ẩn màn hình |
| Vượt giới hạn tần suất | `429` | Thông báo chuẩn | "Vui lòng thử lại sau" |
| Lỗi hạ tầng | `500` | Thông báo chuẩn, ví dụ "Không thể tra cứu danh sách chiến dịch. Vui lòng thử lại sau." | "Thử lại" |

> **Hai loại `404` là của hai người gõ khác nhau**: một người tra cứu nhầm mã chiến dịch, một người
> tra cứu nhầm mã tài khoản. **Đừng gộp về một thông báo "không tìm thấy"** — người vận hành sẽ
> không biết cần sửa ô nào.

---

## 7. Sáu màn hình quản trị

| # | Màn hình | Dùng endpoint nào | Ghi chú hiển thị |
|---|----------|-------------------|-----------------|
| 1 | **Danh sách chiến dịch** | `GET /api/v1/admin/campaigns` | Cột `status` (6 giá trị) + cảnh báo đỏ khi `degraded: true`. Thêm cột/huy hiệu `budgetLocked`. Nút "Mở chiến dịch" → màn hình 5 |
| 2 | **Chi tiết chiến dịch** | `GET /api/v1/admin/campaigns/{code}` | Thanh tiến độ ngân sách lấy từ **API đọc**. Hiện `closedAt`, `version`, cờ `budgetLocked`. Nút "Sửa" → màn hình 3; "Đóng" → màn hình 4; "Nhật ký" → màn hình 6 |
| 3 | **Sửa chiến dịch** | `PATCH /api/v1/admin/campaigns/{code}` | `budgetLocked = true` ⇒ **khoá** `quota` / `reserve` / `planSlug` kèm chú thích (mục 4.3). Gửi `version` từ lần đọc gần nhất. **Chỉ gửi trường người dùng thật sự sửa.** `endsAt` rỗng ⇒ gửi `null` để xoá ngày kết thúc. Xử lý `412` bằng cách tải lại. Sau khi lưu: **`GET` lại** rồi mới vẽ số liệu |
| 4 | **Đóng chiến dịch** | `POST /api/v1/admin/campaigns/{code}/close` | Bắt buộc nhập `reason` (lưu vào nhật ký). Gửi `version`. Nhắc "thao tác một chiều, không mở lại được". Sau khi đóng: **`GET` lại** để thấy `status: closed` và `closedAt` |
| 5 | **Mở chiến dịch mới** | `POST /api/v1/admin/campaigns` | Validate `code` ngay trên form theo `^[a-z0-9][a-z0-9-]{2,63}$` và nhắc định dạng thời gian. Xử lý `409` bằng thông báo "mã đã tồn tại". Sau khi tạo: điều hướng sang màn hình 2 và **`GET` lại** |
| 6 | **Nhật ký kiểm toán** | `GET /api/v1/admin/campaigns/{code}/audit` | Bảng: thời điểm, ai (`actorUserId`), hành động (3 giá trị mục 5.5), IP, và phần **trước → sau** từ `payload.before` / `payload.after`. Với `campaign.close`, hiện `payload.reason`. Nhắc nhật ký ghi bất đồng bộ (mục 3.4) |

---

## 8. Checklist frontend

- [ ] Cả 8 endpoint nằm dưới `/api/v1/admin/campaigns`; kiểm vai trò `Admin` trước khi gọi.
- [ ] ⚠️ **Sau mọi lần `POST` / `PATCH` / `close`, gọi lại `GET /{code}` trước khi hiển thị số liệu
      ngân sách.** Không dùng `grantedMainCount`, `grantedCompensationCount`,
      `grantedTotalCount`, `budgetLocked`, `remainingMain`, `remainingCompensation`, `hardCap`,
      `watermarkAt`, `planName`, `degraded`, `degradedReason`, `lastSweepAt` từ response ghi (mục 2).
- [ ] `budgetLocked = true` ⇒ khoá ô `quota` / `reserve` / `planSlug` kèm chú thích
      *"chiến dịch đã cấp — hãy đóng và mở mã mới"*.
- [ ] Gửi `version` trong thân `PATCH` và `close`; xử lý `412` bằng cách tải lại rồi thao tác lại.
- [ ] `PATCH` chỉ gửi trường **thật sự** thay đổi; `endsAt: null` = xoá ngày kết thúc.
- [ ] Không có ô đổi `code` ở màn hình sửa (mã bất biến).
- [ ] Hiển thị **6** giá trị `status`, ưu tiên `closed`; có nhánh mặc định cho giá trị lạ.
- [ ] Phân biệt `endsAt` (tự kết thúc) với `closedAt` (con người đóng).
- [ ] `degraded: true` ⇒ cảnh báo đỏ, tô đập khác mọi trạng thái khác.
- [ ] Không thêm mã `reason` mới; đóng chiến dịch ⇒ `campaign_sleeping`.
- [ ] `reason` là mã: tự đặt text hiển thị; chịu được `reason` rỗng; mã lạ → thông báo mặc định.
- [ ] `source` chỉ có `"main"` / `"reserve"`.
- [ ] Nhật ký: coi `targetId` là `null` **luôn có mặt**; mã chiến dịch ở `payload.campaignCode`;
      vẽ `before` → `after`; hiện `reason` cho `campaign.close`.
- [ ] Nhắc người dùng nhật ký ghi bất đồng bộ; không kết luận "hệ thống hỏng" khi trang trống.
- [ ] Thời gian gửi theo RFC3339 **có múi giờ**; xử lý `400` từ server.
- [ ] Xử lý `429` bằng "thử lại sau"; `500` thì mời thử lại chứ **không** mời sửa dữ liệu.
- [ ] Phân biệt hai loại `404` (mã chiến dịch lạ) với danh sách rỗng.

---

## 9. Changelog

| Ngày | Thay đổi |
|---|---|
| 2026-10-04 | Tạo tài liệu. 4 endpoint **MỚI**: `POST /admin/campaigns`, `PATCH /admin/campaigns/{code}`, `POST /admin/campaigns/{code}/close`, `GET /admin/campaigns/{code}/audit`. Ba trường mới `closedAt` / `budgetLocked` / `version`. `status` có **6** giá trị (thêm `closed`, ưu tiên cao nhất). Tập `reason` giữ nguyên **12** mã — đóng chiến dịch không sinh mã mới. Ghi rõ hạn chế: **3 endpoint ghi không trả số liệu ngân sách thật**, phải `GET` lại sau mỗi lần ghi. Ghi rõ `budgetLocked` ⇒ khoá ô sửa, `version` + `412` ⇒ tải lại, nhật ký có `targetId` luôn là `null`, 6 màn hình quản trị. |