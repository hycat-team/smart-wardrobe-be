# Hợp Đồng: Giao Diện Tra Cứu Chiến Dịch Cho Quản Trị Viên

**Chủ sở hữu**: module `subscription` (dịch vụ `account`) · **Chỉ đọc**

---

## 1. Vì sao giao diện này tồn tại

FR-076 đưa việc khách hàng tự tra cứu ra ngoài phạm vi. Nhưng quyết định 1 của phiên làm rõ đã
chọn phương án **có** giao diện cho quản trị viên, vì quyết định cho phép vượt hạn mức một chút
(FR-031) kéo theo nghĩa vụ phải trả lời được câu hỏi của khách hàng: *"tôi có được tặng không"*.

Ba trạng thái phải phân biệt được, vì cách nói với khách hàng khác nhau:

| Trạng thái | Nghĩa | Cách nói với khách hàng |
|---|---|---|
| Đã được cấp | Có lượt cấp | "Bạn đã được tặng" |
| **Đủ điều kiện nhưng hết suất** | Thỏa mọi điều kiện, nhưng trần đã đầy hoặc tạo sau mốc ưu tiên | "Bạn đủ điều kiện nhưng chương trình đã hết suất" |
| Không đủ điều kiện | Không thỏa điều kiện nào, hoặc do quản trị viên tạo, hoặc đã tham gia chiến dịch trước | "Bạn không thuộc nhóm được tặng" |

**Điểm cốt lõi**: hai trường hợp giữa phải phân biệt được. Trả về danh sách rỗng cho cả hai là
không đáp ứng FR-068.

## 2. Đường dẫn

```
GET /admin/campaigns                        danh sách chiến dịch
GET /admin/campaigns/:code                  trạng thái một chiến dịch
GET /admin/campaigns/:code/claims           danh sách tài khoản đã cấp (phân trang)
GET /admin/campaigns/:code/eligibility/:userId   tra trạng thái một tài khoản
```

### 2.1 Bẫy định tuyến — bắt buộc đọc

Nhóm phải là **`/admin/campaigns`**, một nhóm riêng. **Không** đăng ký dưới nhóm
`/admin/subscriptions`, vì nhóm đó đang giữ `POST /:userId/cancel` và Gin **không** cho đặt một đoạn
tĩnh cạnh một tham số ở cùng cấp. Xung đột đó làm **hỏng ngay lúc khởi động**, cả dịch vụ Account
sẽ không boot được.

Tiền lệ đã bị dính và đã ghi lại tại `subscription/presentation/router/router.go:96-103`.

### 2.2 Phân quyền và giới hạn tần suất

| Lớp trung gian | Giá trị |
|---|---|
| `r.auth.Handle()` | Bắt buộc có JWT hợp lệ |
| `shared_middleware.RolesAuthorize(roleslug.Admin)` | FR-070 — tài khoản người dùng thường bị từ chối, kể cả khi biết đường dẫn |
| `AdminGrantRateLimitMiddleware` | FR-071 — tái dùng đúng giới hạn đang gắn lên đường cấp gói hàng loạt |

Không viết kiểm tra vai trò riêng, không mở thêm cấu hình giới hạn tần suất.

## 3. Mô tả dữ liệu

### 3.1 `GET /admin/campaigns` — danh sách

```json
{
  "items": [
    {
      "code": "launch-2026-10",
      "planSlug": "premium-monthly",
      "planName": "Premium Plan",
      "quota": 50,
      "reserve": 5,
      "grantedMainCount": 50,
      "grantedCompensationCount": 3,
      "grantedTotalCount": 53,
      "remainingMain": 0,
      "remainingCompensation": 2,
      "hardCap": 55,
      "startsAt": "2026-10-02T00:00:00+07:00",
      "endsAt": null,
      "watermarkAt": "2026-10-02T00:04:17+07:00",
      "status": "compensation",
      "degraded": false,
      "lastSweepAt": "2026-10-02T01:15:00+07:00"
    }
  ],
  "metadata": { "page": 1, "limit": 20, "totalItems": 1, "totalPages": 1 }
}
```

| Trường | Nguồn | Ghi chú |
|---|---|---|
| `quota`, `reserve`, `startsAt`, `endsAt`, `planSlug` | Cấu hình | |
| `grantedMainCount` | `COUNT(*)` lọc `grant_bucket = 'main'` | Lượt cấp từ **phần chính** |
| `grantedCompensationCount` | `COUNT(*)` lọc `grant_bucket = 'reserve'` | Lượt cấp từ **phần dự phòng** |
| `grantedTotalCount` | `grantedMainCount + grantedCompensationCount` | |
| `remainingMain` | `max(0, quota - grantedMainCount)` | |
| `remainingCompensation` | `max(0, reserve - grantedCompensationCount)` | |
| `hardCap` | `quota + reserve` | Trần cứng. FR-022 |
| `watermarkAt` | `registered_at` của bản ghi `fills_main_quota`; `null` nếu chưa chốt | FR-030 |
| `status` | Suy ra | Xem máy trạng thái mục 3.2 |
| `degraded`, `degradedReason` | Bộ nhớ worker quét bù | FR-059 |
| `lastSweepAt` | Bộ nhớ worker quét bù | |

Tất cả trường đếm đều **dẫn xuất lúc tra cứu**, không lưu. Một giá trị lưu sẽ phải đồng bộ bằng
tay và sẽ sai.

`grantedMainCount` và `grantedCompensationCount` là hai số **độc lập**, đọc từ cột `grant_bucket`
của mỗi lượt cấp. Tuyệt đối không suy ra số thứ nhất bằng `NOT fills_main_quota` — cột đó chỉ
đánh dấu đúng một dòng nên phủ định của nó lẫn cả phần dự phòng vào phần chính. Lý do đầy đủ ở
`research.md` R2b.

`grantedTotalCount` được thêm vì không có nó thì không kiểm tra được `grantedTotalCount <= hardCap`
từ phía client, mà đây chính là bất biến FR-022 cần soi.

### 3.2 Giá trị `status`

| Giá trị | Nghĩa |
|---|---|
| `not_started` | Chưa tới `startsAt` |
| `running` | Đang trong cửa sổ, chưa đủ hạn mức chính, chưa chốt mốc ưu tiên |
| `compensation` | Đã chốt mốc ưu tiên, còn suất dự phòng |
| `exhausted` | Đã đạt trần cứng — tự ngủ |
| `expired` | Đã qua `endsAt` — tự ngủ |

Chi tiết máy trạng thái: `data-model.md` mục 5.

### 3.3 `GET /admin/campaigns/:code/claims` — danh sách tài khoản đã cấp

Tham số truy vấn: `page`, `limit`, `from`, `to` (lọc theo khoảng thời điểm cấp, FR-067).

```json
{
  "items": [
    {
      "userId": "b1c2...",
      "username": "nguyenvana",
      "planSlug": "premium-monthly",
      "expiresAt": "2026-11-01T00:00:03+07:00",
      "grantedAt": "2026-10-02T00:00:05+07:00",
      "registeredAt": "2026-10-02T00:00:03+07:00",
      "source": "main"
    }
  ],
  "metadata": { "page": 2, "limit": 20, "totalItems": 53, "totalPages": 3 }
}
```

`source` cho biết lượt cấp đó lấy từ phần chính hay phần dự phòng — hữu ích khi khách hàng hỏi vì
sao bị trượt rồi lại được, hoặc ngược lại. Tập giá trị **chỉ gồm `"main"` và `"reserve"`**, đúng
bằng giá trị của cột `grant_bucket` trong cơ sở dữ liệu. Không dùng `"main_quota"` hay
`"compensation"`: đó là tên trạng thái, đặt vào trường này sẽ tạo thêm một bộ mã thứ ba cho cùng
một khái niệm.

`username` lấy từ `IUserBriefContract.GetByIDs`, **không** đọc thẳng bảng tài khoản. Phân tích theo
tài khoản dùng `sliceutils.IndexBy` (hiến phán §II).

### 3.4 `GET /admin/campaigns/:code/eligibility/:userId` — trạng thái một tài khoản

Đây là đường dẫn trả lời trực tiếp câu hỏi của khách hàng.

```json
{
  "campaignCode": "launch-2026-10",
  "userId": "b1c2...",
  "eligibility": "eligible_but_exhausted",
  "granted": false,
  "reason": "quota_main",
  "watermarkAt": "2026-10-02T00:04:17+07:00",
  "userRegisteredAt": "2026-10-02T00:05:02+07:00"
}
```

| Giá trị `eligibility` | Nghĩa | `granted` |
|---|---|---|
| `granted` | Đã được cấp trong chiến dịch này | `true` |
| `eligible_but_exhausted` | Đủ điều kiện, nhưng hết suất hoặc tạo sau mốc ưu tiên | `false` |
| `not_eligible` | Không thỏa điều kiện nào | `false` |
| `already_claimed_other_campaign` | Đã tham gia một chiến dịch trước | `false` |
| `created_by_admin` | Do quản trị viên tạo | `false` |

### 3.5 Tập mã `reason` — bộ chuẩn duy nhất

`reason` là mã ổn định để tra cứu. Tập giá trị dưới đây là **bộ chuẩn**: nó khớp 1-1 với cột
"Lý do bỏ qua" trong bảng quyết định điều kiện ở `data-model.md` mục 6, và `FR-061` trong
`spec.md` chỉ mô tả bằng văn xuôi chứ **không** khai báo mã riêng.

| Mã `reason` | Nghĩa | Sinh ra khi |
|---|---|---|
| `campaign_not_configured` | Chưa khai báo chiến dịch nào | Điều kiện 1 |
| `campaign_not_started` | Chưa tới `startsAt` | Điều kiện 2 |
| `campaign_sleeping` | Chiến dịch tạm ngủ: hết suất hoặc quá `endsAt` | Điều kiện 2 |
| `created_by_admin` | Tài khoản do quản trị viên tạo | Điều kiện 3 |
| `already_claimed` | Đã có lượt cấp cho tài khoản này | Điều kiện 4 |
| `already_on_plan` | Đang ở đúng gói quà tặng và còn hạn | Điều kiện 5 (FR-018) |
| `plan_not_settled` | Vừa bị hạ về gói miễn phí, gói cũ chưa dọn xong | Điều kiện 6 |
| `quota_main` | Hết suất phần chính | Điều kiện 8 |
| `quota_compensation` | Hết suất dự phòng | Điều kiện 8 |
| `quota_exhausted` | Đạt trần cứng `quota + reserve` — không còn suất nào ở phần nào | Nhánh cuối mục 6.2, `data-model.md` |
| `quota_anomaly` | Phần chính đầy nhưng mốc ưu tiên chưa chốt — dữ liệu bất thường | Điều kiện 8, xem `data-model.md` mục 6.2 |
| `not_eligible` | Không thỏa điều kiện nào khác | Điều kiện 1–7 |
| `duplicate_claim` | Ràng buộc `UNIQUE` chặn ghi trùng | Điều kiện 9 |
| `temporary_error` | Lỗi tạm thời, hàng đỗi sẽ thử lại — **không** phải từ chối vĩnh viễn | FR-062 |

Vì sao `temporary_error` phải có mã riêng: FR-062 đòi phân biệt lỗi tạm thời với lỗi không tự khỏi,
vì chúng dẫn tới hai hành vi khác nhau — một cái được thử lại, một cái thì bỏ. Nếu dùng chung một
mã, hàng đỗi không phân biệt được và sẽ vứt bỏ tài khoản đáng lẽ phải cấp.

Lưu ý cố ý: **không có** mã riêng cho "tạo sau mốc ưu tiên". Trường hợp đó rơi vào `quota_main` hoặc
`quota_compensation` tuỳ phần nào đã đầy, vì nguyên nhân sâu hơn là hết suất chứ không phải thiếu
điều kiện. Nếu sau này cần phân biệt thì tách mã mới, đừng đổi nghĩa mã cũ.

## 4. Lỗi

| Tình huống | Mã trả về | Thông điệp tiếng Việt |
|---|---|---|
| Mã chiến dịch không tồn tại | 404 | "Không tìm thấy chiến dịch với mã đã nhận." |
| Người dùng gọi vào | 403 | Thông báo chuẩn của `RolesAuthorize` |
| Tài khoản trong `:userId` không tồn tại | 404 | "Không tìm thấy tài khoản." |
| Thiếu tham số truy vấn bắt buộc | 400 | Thông báo chuẩn của `validation.BindQuery` |
| Vượt giới hạn tần suất | 429 | Thông báo chuẩn của `AdminGrantRateLimitMiddleware` |

FR-072: mã chiến dịch không tồn tại phải trả lỗi thân thiện nói rõ mã đó không có, **không** trả về
lỗi kỹ thuật và **không** trả về danh sách rỗng không giải thích.

## 5. Chỉ đọc — bảo đảm

| Bảo đảm | Kiểm chứng bằng |
|---|---|
| Không tiêu tốn suất (FR-069) | Gọi đường dẫn 50 lần, `grantedMainCount` không đổi |
| Không cấp gói (FR-069) | Số bản ghi lượt cấp không đổi sau khi gọi |
| Không gửi email (FR-069) | Không có bản ghi hàng đỗi chủ đề `notification.email` sinh ra |
| Không sửa dữ liệu (FR-069) | So sánh trạng thái toàn bộ bảng trước và sau |

## 6. Định dạng dữ liệu

| Quy tắc | Giá trị |
|---|---|
| Nhãn JSON cho khách hàng | `camelCase` |
| Nhãn JSON cho thông điệp nội bộ | `camelCase` |
| Tiền tệ / số lượng | Số nguyên thuần cho hạn mức và số đếm |
| Thời gian | RFC3339 kèm múi giờ, giờ Việt Nam |
| Thông báo lỗi | Tiếng Việt, khai báo hằng số ở đầu tệp handler |
| Phân trang | `shared_dto.PaginationQuery` + `shared_dto.PaginationResult` |
