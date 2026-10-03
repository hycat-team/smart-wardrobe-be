# Hợp Đồng: Khác Biệt Cộng Thêm cho API Tra Cứu của 025

**Chủ sở hữu**: module `subscription` (dịch vụ `account`) · **Chỉ đọc**
**Hợp đồng gốc**: `specs/025-signup-campaign-grant/contracts/campaign-query-api.md` (không sửa)
**Đặc tả**: [../spec.md](../spec.md) FR-050, FR-051

Tài liệu này ghi **đúng những gì đổi** so với hợp đồng đã lên production. Các đường dẫn, hình dạng
response, mã lỗi, 6 giá trị `eligibility` và **12 mã `reason`** giữ nguyên.

---

## 1. Bốn endpoint đọc — không đổi

| # | Endpoint | Trạng thái |
|---|---|---|
| 1 | `GET /api/v1/admin/campaigns` | không đổi (chỉ thêm trường — mục 2) |
| 2 | `GET /api/v1/admin/campaigns/{code}` | không đổi (chỉ thêm trường — mục 2) |
| 3 | `GET /api/v1/admin/campaigns/{code}/claims` | không đổi |
| 4 | `GET /api/v1/admin/campaigns/{code}/eligibility/{userId}` | không đổi |

Nguồn danh sách chiến dịch **đổi từ cấu hình sang CSDL**, nhưng điều đó không nhìn thấy được từ hợp
đồng: một mã không có trong sổ vẫn trả `404` như trước (025 coi mã không khai báo là "không tồn tại"
— FR-072).

---

## 2. Trường thêm vào `CampaignSummaryRes` (1 và 2)

| Trường | Kiểu | Trước 026 | Sau 026 |
|---|---|---|---|
| `closedAt` | `string \| null` | không có | **mới** — thời điểm đóng cưỡng bức, `null` nếu chưa đóng |
| `budgetLocked` | `boolean` | không có | **mới** — `true` khi đã có ≥ 1 lượt cấp; FE khoá ô `quota`/`reserve`/`planSlug` |
| `version` | `number` | không có | **mới** — phiên bản để gửi lại khi sửa/đóng (If-Match) |

Các trường cũ (`code`, `planSlug`, `planName`, `quota`, `reserve`, `grantedMainCount`,
`grantedCompensationCount`, `grantedTotalCount`, `remainingMain`, `remainingCompensation`, `hardCap`,
`startsAt`, `endsAt`, `watermarkAt`, `status`, `degraded`, `degradedReason`, `lastSweepAt`) **giữ
nguyên tên, kiểu và nghĩa**.

### 2.1 Ví dụ sau 026

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

---

## 3. Giá trị `status` — thêm một giá trị

| `status` | Trước 026 | Sau 026 |
|---|---|---|
| `not_started`, `running`, `compensation`, `exhausted`, `expired` | có | không đổi |
| `closed` | không có | **mới** — đã bị đóng cưỡng bức; ưu tiên cao nhất |

`closed` đứng trước mọi trạng thái khác: một chiến dịch đã đóng hiển thị `closed` bất kể `now` và
ngân sách. FE MUST hiển thị mặc định cho giá trị `status` lạ thay vì so khớp chuỗi (hợp đồng gốc đã
khuyến nghị điều này cho `reason`).

---

## 4. Tập mã `reason` — **không đổi, đúng 12 mã**

| Mã | Có đổi không |
|---|---|
| `campaign_not_configured`, `campaign_not_started`, `campaign_sleeping`, `created_by_admin`, `already_claimed`, `already_on_plan`, `plan_not_settled`, `outside_window`, `quota_main`, `quota_compensation`, `quota_anomaly`, `plan_unavailable` | **không** |

Chiến dịch đã đóng làm điều kiện 2 trả `campaign_sleeping` — cùng mã 025 đã dùng cho "quá `ends_at`"
(và cùng nghĩa "chiến dịch đã kết thúc"). **Không** thêm mã cho "đã đóng".

---

## 5. `eligibility` — không đổi (6 giá trị)

Tra một tài khoản trong một chiến dịch đã đóng trả `eligibility = "not_eligible"` kèm
`reason = "campaign_sleeping"`. Không có giá trị `eligibility` mới.

---

## 6. Lỗi — không đổi

| Tình huống | HTTP |
|---|---|
| Mã chiến dịch không tồn tại | 404 |
| `:userId` không tồn tại | 404 |
| `from`/`to` sai định dạng | 400 |
| Người dùng thường | 403 |
| Vượt giới hạn tần suất | 429 |

---

## 7. Check-list frontend (bổ sung vào check-list của 025)

- [ ] Đọc thêm `closedAt`, `budgetLocked`, `version` trên danh sách và chi tiết chiến dịch.
- [ ] Khoá ô sửa `quota`/`reserve`/`planSlug` khi `budgetLocked = true`, kèm chú thích "chiến dịch đã
      cấp — hãy đóng và mở mã mới".
- [ ] Hiển thị 6 giá trị `status`; chịu được giá trị lạ.
- [ ] Gửi `version` khi sửa/đóng; xử lý `412` bằng thông báo "dữ liệu đã thay đổi, tải lại".
- [ ] **Không** thêm mã `reason` mới; đóng chiến dịch ⇒ `campaign_sleeping`.
- [ ] Mọi trường cũ vẫn đọc như trước — không có trường nào bị xoá hay đổi kiểu.
