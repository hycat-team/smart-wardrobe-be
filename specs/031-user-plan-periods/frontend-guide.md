# Frontend Guide — Phân bổ gói theo kỳ (feature 031)

**Endpoint:** `GET /api/v1/admin/dashboard/subscriptions/distribution`
**Đối tượng:** System Admin Dashboard (FE).
**Loại thay đổi:** bổ sung (additive). Không đổi tên, kiểu hay ý nghĩa của bất kỳ
trường cũ nào.

## 1. Endpoint và tham số

- Method/path **không đổi**: `GET /api/v1/admin/dashboard/subscriptions/distribution`.
- Query parameters **không đổi** (từ 029):
    - `fromDate` (string, optional) — `YYYY-MM-DD`, ngày lịch Việt Nam.
    - `toDate` (string, optional) — `YYYY-MM-DD`, ngày lịch Việt Nam, bao gồm trọn ngày.
    - Gửi sai định dạng hoặc `fromDate > toDate` → HTTP 400 (giữ nguyên thông báo
      tiếng Việt từ BE).
- **Không gửi cả hai** = ảnh chụp gói đang có hiệu lực tại thời điểm gọi.
- **Chỉ gửi `fromDate`** = từ ngày đó đến hiện tại; các khoảng bắt đầu trong
  tương lai **không** được tính.
- **Chỉ gửi `toDate`** = từ đầu lịch sử có dữ liệu đến hết ngày đó.
- Giá trị rỗng/không gửi là **vô cực**, không phải mốc `0001-01-01`.
- Nếu kỳ mở rộng về phía trước mốc lịch sử, BE **chỉ tính phần từ mốc trở đi** và
  báo `partial` (hoặc `unavailable` nếu kỳ kết thúc trước mốc) — xem mục 3–4.

### Mã HTTP

| Status | Khi nào | FE xử lý |
|---|---|---|
| `200` | Request hợp lệ. Kỳ `unavailable` cũng trả `200` với `distribution: []`, `totalUsers: 0` — trạng thái nằm ở `dataCompleteness`. | Đọc `dataCompleteness` trước khi render; không coi mảng rỗng là lỗi mạng. |
| `400` | `fromDate`/`toDate` sai định dạng, hoặc `fromDate > toDate`. Message tiếng Việt từ BE. | Hiển thị `message`, giữ nguyên bộ lọc để người dùng sửa. |
| `401` | Thiếu/sai/hết hạn access token (middleware auth dùng chung). | Về màn đăng nhập. |
| `403` | Đã đăng nhập nhưng không phải System Admin (`RolesAuthorize(Admin)`). | Ẩn màn hình quản trị, không phải lỗi nghiệp vụ. |

## 2. Hai trường mới trong response

```json
{
  "message": "Lấy phân bổ subscription thành công",
  "data": {
    "distribution": [
      { "planCode": "free", "userCount": 10, "percentage": 41.67 },
      { "planCode": "premium-monthly", "userCount": 14, "percentage": 58.33 }
    ],
    "totalUsers": 24,
    "dataAvailableFrom": "2026-10-20",
    "dataCompleteness": "partial"
  }
}
```

| Field | Kiểu | Ý nghĩa |
|---|---|---|
| `dataAvailableFrom` | string `YYYY-MM-DD` (ngày VN) hoặc **vắng mặt** | Mốc bắt đầu lịch sử gói đầy đủ. Đây là **chuỗi ngày**, không phải RFC 3339/timestamp. Vắng khi BE không xác định được mốc. |
| `dataCompleteness` | string hoặc **vắng mặt** | `complete` \| `partial` \| `unavailable`. Mô tả kỳ đã yêu cầu so với mốc trên. |

Cả hai dùng `omitempty`: khi vắng, key không xuất hiện trong JSON. Client hiện
tại chỉ đọc `distribution`/`totalUsers` nên không bị ảnh hưởng (FR-015).

## 3. Hành vi trước → sau

| Tình huống | Trước 031 | Sau 031 |
|---|---|---|
| User dùng premium cả tháng 9, hạ free ngày 01/10; xem báo cáo tháng 9 | User **không** nằm trong nhóm nào (gói hiện tại là free, cửa sổ hiện tại không giao tháng 9) | User được tính cho `premium-monthly` trong tháng 9 |
| User đổi gói A → B giữa kỳ | Toàn kỳ bị quy về gói B | User xuất hiện ở **cả A và B** |
| User chỉ dùng free | Free xuất hiện mọi kỳ | Không đổi |
| Kỳ chỉ có `fromDate` | Chỉ lọc theo `expires_at`, có thể tính cả khoảng tương lai | Chỉ tính đến hiện tại, bỏ khoảng tương lai |
| Kỳ chỉ có `toDate` | Chỉ lọc `started_at < to`, thiếu điều kiện hết hạn | Tính các khoảng giao với `(-∞, to)` |
| Kỳ nằm hoàn toàn trước mốc lịch sử | Trả số 0 như số liệu thật | `dataCompleteness = "unavailable"`, `distribution: []`, `totalUsers: 0` |
| Kỳ cắt ngang mốc lịch sử | Không có tín hiệu | `dataCompleteness = "partial"`; chỉ tính phần từ mốc trở đi |
| Không gửi kỳ | Ảnh chụp hiện tại | Không đổi; `complete` (kèm `dataAvailableFrom` nếu có mốc) |

## 4. Cách render

### `complete`

- Hiển thị bình thường.
- Nếu có `dataAvailableFrom`, có thể hiển thị tooltip "Lịch sử đầy đủ từ
  `<dataAvailableFrom>`" — nhưng không bắt buộc.

### `partial`

- Vẫn vẽ biểu đồ với số liệu trả về, **kèm cảnh báo rõ ràng**, ví dụ:
  "Dữ liệu chỉ đầy đủ từ `<dataAvailableFrom>`; kỳ đã chọn bắt đầu trước mốc này."
- Không cộng thêm số ước lượng và không tự chia lại tỷ lệ phần trăm; BE đã tính
  trên đúng phần dữ liệu có thể chứng minh.

### `unavailable`

- **Không** vẽ biểu đồ 0% và **không** hiển thị "0 người dùng" như số liệu thật.
- Hiển thị trạng thái trống, ví dụ:
  "Không đủ dữ liệu cho giai đoạn này. Lịch sử gói bắt đầu từ
  `<dataAvailableFrom>`." (nếu trường có mặt)
- Nếu `dataAvailableFrom` vắng mặt (BE không xác định được mốc), chỉ nói
  "Không đủ dữ liệu cho giai đoạn này" và không bịa ngày.

### Mốc và ngày

- `dataAvailableFrom` là **ngày lịch VN**; không tự đổi sang UTC hay cộng/trừ
  múi giờ ở FE.
- `fromDate`/`toDate` đã gửi phải được giữ nguyên để hiển thị nhãn kỳ; response
  không echo lại kỳ.

## 5. `totalUsers` và `percentage` — đọc kỹ

- `totalUsers` là **tổng của các nhóm**, không phải số người duy nhất.
- Một người đổi gói giữa kỳ được tính ở cả hai gói, nên `totalUsers` có thể lớn
  hơn số user thực. Đây là chủ đích (clarification 2026-10-20).
- `percentage = userCount / totalUsers * 100`; khi có ít nhất một người, các
  nhóm cộng đúng 100%. Khi `unavailable`, `totalUsers = 0` và `distribution`
  rỗng — không chia cho 0, không vẽ biểu đồ.

## 6. Checklist tích hợp FE

- [ ] Giữ nguyên endpoint và hai query param.
- [ ] Đọc thêm `dataAvailableFrom` (string ngày VN) và `dataCompleteness`.
- [ ] Render `unavailable` bằng empty state, không phải chart 0%.
- [ ] Render `partial` kèm cảnh báo mốc dữ liệu.
- [ ] Không tự tính lại `percentage`/`totalUsers` từ danh sách user.
- [ ] Client cũ: bỏ qua hai field mới vẫn chạy bình thường.
- [ ] Với dữ liệu seed/dev trước ngày deploy, kỳ quá khứ sẽ trả `unavailable`
      cho đến khi có lịch sử tích luỹ — đây là hành vi đúng, không phải lỗi.
