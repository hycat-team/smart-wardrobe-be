# Frontend Integration Guide — Chiến Dịch Tặng Gói Cho Tài Khoản Mới

**Feature**: `025-signup-campaign-grant`
**Trạng thái**: Hoàn thiện (feature 025 đã triển khai, chiến dịch `launch-2026-10` đang chạy).
**Đối tượng đọc**: Lập trình viên frontend của **trang quản trị / bộ phận hỗ trợ**.
**Mục tiêu**: FE chỉ cần đọc tài liệu này để triển khai, KHÔNG cần đọc code backend hay Swagger.

> **Đây không phải API dành cho khách hàng.** Cả 4 endpoint đều nằm dưới `/api/v1/admin/campaigns`
> và yêu cầu vai trò `Admin`. Người dùng cuối không bao giờ gọi trực tiếp.
>
> Quy ước: mọi trường JSON dùng `camelCase`. Nội dung này mô tả hành vi API ở mức hợp đồng.

---

## 0. Tóm tắt nhanh (TL;DR)

- Chiến dịch tặng gói cho tài khoản đăng ký mới. Khi khách hàng hỏi *"tôi có được tặng gói
  không"*, 4 API dưới đây trả lời được, kèm lý do nếu là không.
- **4 endpoint mới**, tất cả dưới nhóm `/api/v1/admin/campaigns` (GET, chỉ đọc):
  danh sách chiến dịch · chi tiết một chiến dịch · danh sách lượt cấp · trạng thái một tài khoản.
- **Có 6 trạng thái `eligibility`** cho một tài khoản. Điểm cần nhấn: `eligible_pending` nghĩa là
  *"đủ điều kiện, đang chờ hệ thống xử lý"* — **KHÔNG** phải hết suất. Đừng dùng
  `eligible_but_exhausted` cho ca này, nói vậy là nói dối khách.
- **`reason` là mã, không phải câu.** 12 mã ổn định ở mục 2.2. FE tự đặt text hiển thị. Có một
  ca chủ định `reason` **rỗng**.
- **`status` của chiến dịch có 5 giá trị** và **tách khỏi** cờ `degraded`. `degraded = true`
  nghĩa là *đang chạy mà hỏng, không cấp được cho ai* — đây là cảnh báo nghiêm trọng nhất.
- **Mọi số đếm đều dẫn xuất lúc tra cứu**, không lưu. Đừng cache kết quả: ngân sách thay đổi theo
  phút và một giá trị đọc từ cache sẽ sai.

---

## 1. Tổng quan

### 1.1 Chiến dịch có thể ở trạng thái nào (`status`)

| `status` | Nghĩa | FE hiển thị |
|----------|-------|-------------|
| `not_started` | Chưa tới `startsAt` | "Sắp mở" + ngày bắt đầu |
| `running` | Đang mở, chưa đủ hạn mức chính, chưa chốt mốc ưu tiên | "Đang chạy" |
| `compensation` | Đã chốt mốc ưu tiên (hết phần chính), còn suất dự phòng | "Đang chạy — chỉ còn suất dự phòng" |
| `exhausted` | Đã đạt trần cứng `quota + reserve` — tự ngủ | "Đã hết suất" |
| `expired` | Đã qua `endsAt` — tự ngủ | "Đã kết thúc" |

### 1.2 Sức khoẻ cơ chế cấp (`degraded`) — TÁCH KHỎI `status`

Ba thứ này trông giống nhau từ bên ngoài nhưng **khác nhau về bản chất**, và FE phải tô đậm
khác nhau:

| Tình huống | `status` | `degraded` | Mức độ khẩn |
|------------|----------|------------|---------------|
| Chiến dịch đang chạy bình thường | `running` | `false` | — |
| Chiến dịch đã hết suất (thiết kế đúng) | `exhausted` | `false` | Thông báo thường |
| Chiến dịch **đang mở nhưng hỏng, không ai được cấp** | `running` | **`true`** | **Cảnh báo đỏ** |

`degraded` là cờ đáng tin nhất trong 4 trường này, và nó là cờ **đáng tin vì lý do kỹ thuật cụ
thể**: nó được đọc từ bảng dữ liệu chung, không phải từ bộ nhớ của worker. Hệ thống chạy nhiều
bản sao, mỗi bản một worker; nếu cờ nằm trong bộ nhớ thì một yêu cầu HTTP rơi vào bản sao B sẽ
không nhìn thấy lần quét hỏng do bản sao A chạy — và trang sẽ báo **lành trong khi thật ra hỏng**.
Vì vậy:

- `degraded: true` ⇒ **cảnh báo ngay**, đừng coi là chi tiết kỹ thuật để ẩn đi.
- `degradedReason` có thể **rỗng** nếu cơ chế vừa hồi phục (một lần quét thành công sẽ xoá lý do).
  Rỗng **không** phải lỗi.
- `lastSweepAt` là lần quét bù **thành công** gần nhất. Nếu nó cũ hơn bạn mong đợi (quét chạy
  mỗi 60 giây), cơ chế có thể đã kẹt mà `degraded` chưa kịp bật.

### 1.3 Hai trường dễ nhầm — `watermarkAt`

| Trường | Ý nghĩa |
|-------|---------|
| `watermarkAt` | `registeredAt` của lượt cấp thứ `quota` — mốc ưu tiên. Từ đây trở đi, tài khoản mới chỉ được cấp từ **phần dự phòng**. |
| `lastSweepAt` | Lần quét bù thành công gần nhất. |

Hai cái này **không liên quan**. Đừng dùng `watermarkAt` để đoán "lần cập nhật cuối".

---

## 2. Trạng thái một tài khoản

### 2.1 Bảng `eligibility` — 6 giá trị

Đây là bảng quan trọng nhất của tài liệu này.

| `eligibility` | Nghĩa | `granted` | FE nên nói với khách |
|---------------|-------|-----------|--------------------|
| `granted` | Đã được cấp trong chiến dịch này | `true` | "Bạn đã được tặng gói, hết hạn ngày `<expiresAt>`" |
| **`eligible_pending`** | **Đủ điều kiện, còn suất, nhưng CHƯA được cấp** — thông điệp còn trên đường hoặc quét bù chưa tới | `false` | **"Bạn đủ điều kiện, hệ thống đang xử lý — thường chưa tới 1 phút"** |
| `eligible_but_exhausted` | Đủ điều kiện nhưng hết suất, hoặc tạo sau mốc ưu tiên | `false` | "Chiến dịch đã hết suất" |
| `not_eligible` | Không thỏa điều kiện nào | `false` | Xem `reason` để biết vì sao |
| `already_claimed_other_campaign` | Đã tham gia một chiến dịch trước | `false` | "Bạn đã được tặng gói ở chiến dịch trước" |
| `created_by_admin` | Do quản trị viên tạo | `false` | "Tài khoản do quản trị viên tạo nên không áp dụng chiến dịch" |

> ⚠️ **Đừng gộp `eligible_pending` vào `eligible_but_exhausted`.** Kịch bản đó khiến nhân viên
> đọc ra *"chúng tôi hết suất tặng"* — sai theo cả hai nghĩa: chiến dịch còn suất, và lượt của
> khách sắp tới. Đây cũng là câu hỏi khách hàng gọi lên **nhiều nhất**, nên độ chính xác ở đây
> quan trọng hơn mọi chỗ khác.

### 2.2 Bảng mã `reason` — 12 mã

`reason` là **mã ổn định**, FE tự đặt text hiển thị. Danh sách này khớp **1-1** với những gì
backend thực sự phát ra; nếu bạn thấy mã ngoài bảng, hãy hiển thị thông báo mặc định chứ **đừng**
so khớp chuỗi.

| Mã `reason` | Nghĩa | Text FE nên hiển thị |
|-------------|-------|----------------------|
| *(rỗng)* | Đủ điều kiện, đang chờ — **không** phải lỗi | Chỉ dùng cho `eligible_pending` và `granted` |
| `campaign_not_configured` | Chưa khai báo chiến dịch nào | "Chưa có chiến dịch nào đang chạy" |
| `campaign_not_started` | Chưa tới giờ mở | "Chiến dịch chưa bắt đầu" |
| `campaign_sleeping` | Chiến dịch đã kết thúc | "Chiến dịch đã kết thúc" |
| `created_by_admin` | Tài khoản do quản trị viên tạo | "Tài khoản do quản trị viên tạo" |
| `already_claimed` | Đã có lượt cấp (giao lại thông điệp, hoặc đã tham gia chiến dịch trước) | "Bạn đã được tặng gói rồi" |
| `already_on_plan` | Đang ở đúng gói quà tặng và còn hạn — **cấp thêm sẽ chồng** | "Bạn đang có gói này rồi" |
| `plan_not_settled` | Vừa bị hạ về gói miễn phí, gói cũ chưa dọn xong | "Thử lại sau ít phút" |
| `outside_window` | Thời điểm tạo tài khoản nằm ngoài cửa sổ chiến dịch | "Tài khoản tạo ngoài thời gian chiến dịch" |
| `quota_main` | Hết suất phần chính | "Chiến dịch đã hết suất" |
| `quota_compensation` | Hết suất dự phòng — tức đã chạm trần cứng | "Chiến dịch đã hết suất" |
| `quota_anomaly` | Phần chính đầy nhưng mốc ưu tiên chưa chốt — **dữ liệu bất thường** | "Hệ thống đang kiểm tra, thử lại sau" |
| `plan_unavailable` | `planSlug` trỏ tới gói không tồn tại hoặc đã ngừng | "Chiến dịch đang được cấu hình lại" |

**Bốn mã KHÔNG tồn tại** và một mã từng được hứa rồi đã bị bỏ — đừng viết code theo chúng:

- ❌ `not_eligible` — nó là một giá trị của `eligibility`, **không** phải mã `reason`. Hai trường
  khác nhau, hai mức chi tiết khác nhau.
- ❌ `temporary_error` — thuộc tầng vận chuyển, không phải tầng kinh doanh. Thông điệp được thử
  lại không bao giờ ghi `reason`.
- ❌ `duplicate_claim` — điều kiện chống cấp trùng đã dùng `already_claimed`.
- ❌ `quota_exhausted` — phần dự phòng là **trần**, nên "hết dự phòng" chính là `quota_compensation`.

### 2.3 Khi nào `reason` rỗng

`reason` rỗng ở **đúng hai** ca: `granted` (đã cấp rồi, không có gì phải giải thích) và
`eligible_pending` (đang chờ, chưa có mã nào nói được "đang trên đường"). FE phải chịu được
`reason` rỗng và **không** coi đó là lỗi dữ liệu.

---

## 3. API

Cả 4 đều là `GET`, chỉ đọc, và không để lại tác dụng phụ. Bọc trong
`APIResponse { success, message, data }`.

| # | API | Dùng để |
|---|-----|---------|
| 1 | `GET /api/v1/admin/campaigns` | Danh sách chiến dịch + ngân sách |
| 2 | `GET /api/v1/admin/campaigns/{code}` | Chi tiết một chiến dịch |
| 3 | `GET /api/v1/admin/campaigns/{code}/claims` | Danh sách tài khoản đã được cấp |
| 4 | `GET /api/v1/admin/campaigns/{code}/eligibility/{userId}` | **Trả lời khách hàng** |

### 3.1 `GET /api/v1/admin/campaigns` — danh sách

Tham số: `page`, `limit` (int, tuỳ chọn).

```json
{
  "success": true,
  "message": "Thành công",
  "data": {
    "items": [
      {
        "code": "launch-2026-10",
        "planSlug": "premium-monthly",
        "planName": "Premium Tháng",
        "quota": 50,
        "reserve": 5,
        "grantedMainCount": 23,
        "grantedCompensationCount": 0,
        "grantedTotalCount": 23,
        "remainingMain": 27,
        "remainingCompensation": 5,
        "hardCap": 55,
        "startsAt": "2026-10-02T00:00:00+07:00",
        "endsAt": null,
        "watermarkAt": null,
        "status": "running",
        "degraded": false,
        "lastSweepAt": "2026-10-02T20:31:12+07:00"
      }
    ],
    "metadata": { "page": 1, "limit": 20, "totalItems": 1, "totalPages": 1 }
  }
}
```

**Ý nghĩa các trường ngân sách** — đây là chỗ dễ làm FE hiển thị sai:

| Trường | Công thức | FE dùng để |
|--------|-----------|------------|
| `quota` | (khai báo) | Hiển thị "hạn mức chính" |
| `reserve` | (khai báo) | Hiển thị "suất dự phòng" |
| `grantedMainCount` | đếm thật | Thanh tiến độ phần chính |
| `grantedCompensationCount` | đếm thật | Thanh tiến độ phần dự phòng |
| `grantedTotalCount` | `main + compensation` | Tổng đã cấp |
| `remainingMain` | `max(0, quota − grantedMainCount)` | Còn suất chính |
| `remainingCompensation` | `max(0, reserve − grantedCompensationCount)` | Còn suất dự phòng |
| `hardCap` | `quota + reserve` | Trần cứng — tổng **không bao giờ** vượt |

**Mọi trường đếm đều dẫn xuất lúc tra cứu, không lưu.** Không có bộ đếm nào cần đồng bộ thủ công,
và cũng nghĩa là giá trị đổi theo phút — **đừng cache** trên FE nếu bạn cần con số chính xác.

### 3.2 `GET /api/v1/admin/campaigns/{code}` — chi tiết

Cùng hình dạng với một phần tử của API 1.

```json
{
  "success": true,
  "message": "Thành công",
  "data": {
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
    "lastSweepAt": "2026-10-02T20:31:12+07:00"
  }
}
```

`endsAt: null` nghĩa là chiến dịch **không đặt ngày kết thúc**, không phải "chưa biết". `watermarkAt`
chỉ có giá trị khi đã chốt mốc ưu tiên (tức `status` đã sang `compensation` hoặc `exhausted`).

### 3.3 `GET /api/v1/admin/campaigns/{code}/claims` — danh sách lượt cấp

Tham số: `page`, `limit`, `from`, `to`.
`from`/`to` lọc theo **thời điểm cấp** (`grantedAt`), **không** phải thời điểm đăng ký.

```json
{
  "success": true,
  "message": "Thành công",
  "data": {
    "items": [
      {
        "userId": "b1c2d3e4-...",
        "username": "nguyenvana",
        "planSlug": "premium-monthly",
        "expiresAt": "2026-11-01T00:00:05+07:00",
        "grantedAt": "2026-10-02T00:00:05+07:00",
        "registeredAt": "2026-10-02T00:00:03+07:00",
        "source": "main"
      }
    ],
    "metadata": { "page": 2, "limit": 20, "totalItems": 53, "totalPages": 3 }
  }
}
```

**`source` chỉ có hai giá trị: `"main"` và `"reserve"`.** Không có `"compensation"` hay
`"main_quota"` — đó là tên trạng thái, không phải tên phần ngân sách. Field này trả lời được câu
hỏi *"sao lượt trước bị trượt mà lượt này lại được"* (tức trượt vì tạo sau mốc ưu tiên, được vì
rơi vào phần dự phòng).

> `expiresAt` là `grantedAt + số ngày của gói`, **không phải** `registeredAt + 30 ngày`. Với tài
> khoản đăng ký 00:00:03 nhưng chỉ được cấp lúc 00:00:05 như ví dụ trên, hạn là 00:00:05.

### 3.4 `GET /api/v1/admin/campaigns/{code}/eligibility/{userId}` — **API chính**

Đây là API trả lời trực tiếp câu hỏi của khách hàng.

```json
{
  "success": true,
  "message": "Thành công",
  "data": {
    "campaignCode": "launch-2026-10",
    "userId": "b1c2d3e4-...",
    "eligibility": "eligible_pending",
    "granted": false,
    "watermarkAt": null,
    "userRegisteredAt": "2026-10-02T00:31:40+07:00"
  }
}
```

```json
{
  "success": true,
  "message": "Thành công",
  "data": {
    "campaignCode": "launch-2026-10",
    "userId": "b1c2d3e4-...",
    "eligibility": "granted",
    "granted": true,
    "watermarkAt": "2026-10-02T00:04:17+07:00",
    "userRegisteredAt": "2026-10-02T00:00:03+07:00"
  }
}
```

Khi không được cấp, `reason` sẽ có mã (mục 2.2).

---

## 4. Mã lỗi

| Tình huống | HTTP | Thông báo tiếng Việt từ backend | FE làm gì |
|------------|------|-------------------------------|-----------|
| Mã chiến dịch không tồn tại | **404** | "Không tìm thấy chiến dịch với mã đã nhận: `<code>`." | Sửa ô mã, **không** hiển thị danh sách rỗng |
| `:userId` không tồn tại | **404** | "Không tìm thấy tài khoản." | Kiểm tra lại mã khách |
| `from`/`to` sai định dạng | **400** | "Tham số truy vấn `\"from\"` không hợp lệ: cần một thời điểm RFC3339 có múi giờ, ví dụ 2026-10-02T00:00:00+07:00." | Báo người dùng sai định dạng thời gian |
| Người dùng không có vai trò Admin | **403** | Thông báo chuẩn của `RolesAuthorize` | Ẩn màn hình, không phải lỗi nghiệp vụ |
| Vượt giới hạn tần suất | **429** | Thông báo chuẩn của rate-limit | Hiển thị "vui lòng thử lại sau", có thể retry |

Hai lỗi 404 là **của hai người gõ khác nhau**: một người tra cứu nhầm mã chiến dịch, một người
tra cứu nhầm tài khoản. Backend tách riêng và thông báo đã nói rõ — **đừng gộp về một thông báo
"không tìm thấy"**, vì người vận hành sẽ không biết cần sửa ô nào.

> Lưu ý về `from`/`to`: backend **tự** phân tích thời điểm theo RFC3339 và yêu cầu múi giờ rõ
> ràng, thay vì để bộ giải mã mặc định của framework nhận bất kỳ thứ gì. Chuỗi không có múi giờ
> sẽ bị **400**, không bị bỏ qua âm thầm.

---

## 5. Checklist frontend

- [ ] Cả 4 endpoint nằm dưới `/api/v1/admin/campaigns`; kiểm tra vai trò `Admin` trước khi gọi.
- [ ] Hiển thị **6** giá trị `eligibility`; `eligible_pending` **không** gộp vào
      `eligible_but_exhausted`.
- [ ] Chịu được `reason` **rỗng** ở `granted` và `eligible_pending`.
- [ ] `reason` là mã: tự đặt text hiển thị; **không** so khớp chuỗi; mã lạ → thông báo mặc định.
- [ ] Không dùng 4 mã không tồn tại: `not_eligible` (chỉ là `eligibility`), `temporary_error`,
      `duplicate_claim`, `quota_exhausted`.
- [ ] `source` chỉ có `"main"` / `"reserve"`.
- [ ] Phân biệt 3 tình huống: `degraded: true` ⇒ **cảnh báo đỏ**; `exhausted` ⇒ thông báo thường;
      `expired` ⇒ đã kết thúc.
- [ ] Không cache số đếm ngân sách — chúng đổi theo phút.
- [ ] Phân biệt `watermarkAt` (mốc ưu tiên) với `lastSweepAt` (lần quét bù cuối).
- [ ] `endsAt: null` nghĩa là không đặt ngày kết thúc.
- [ ] `from`/`to` gửi theo RFC3339 **có múi giờ**; xử lý 400 từ server.
- [ ] Phân biệt hai loại 404 và hiển thị đúng thông báo của từng loại.
- [ ] Xử lý 429 bằng thông báo "thử lại sau".

---

## 6. Changelog

| Ngày | Thay đổi |
|------|----------|
| 2026-10-02 | Tạo tài liệu. 4 API tra cứu chiến dịch. 6 giá trị `eligibility`. 12 mã `reason`. 5 trạng thái `status` + `degraded` tách riêng. Ghi rõ 4 mã `reason` không tồn tại và `eligible_pending` không được gộp vào `eligible_but_exhausted`. |