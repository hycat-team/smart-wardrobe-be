# Hợp đồng: giao diện lập trình cấp gói hàng loạt cho quản trị viên

**Ngày**: 2026-09-29 | **Dịch vụ**: Account (`services/account`) | **Yêu cầu**: FR-045 đến FR-085
**Một và chỉ một** giao diện lập trình cho thao tác cấp gói (FR-060). Cấp cho một tài khoản chính là danh sách một phần tử, dùng chung giao diện, chung đáp lời, chung quy tắc kiểm tra.

---

## Vì sao đường dẫn này

Giao diện cũ `POST /api/v1/admin/subscriptions/:userId/grant` **bị gỡ**.

Giao diện mới không thể đặt ở `/api/v1/admin/subscriptions/grants`: nhóm `adminSubs` đang có `POST /:userId/cancel` và `POST /:userId/grant`, mà Gin không cho phép một đoạn tĩnh (`grants`) cùng cấp với một đoạn đại lượng (`:userId`) — ứng dụng sẽ **panic lúc khởi động**. Vì vậy dùng tiền tố riêng. Chi tiết ở [research.md](../research.md) R5.

---

## `POST /api/v1/admin/subscription-grants`

Cấp gói cho một danh sách tài khoản.

**Phân quyền**: `auth.Handle()` + `RolesAuthorize(roleslug.Admin)` (FR-080)
**Giới hạn tần suất**: hai mức, xem bên dưới (FR-078)
**Ghi nhận ký**: một bản ghi cho mỗi tài khoản được cấp thành công (FR-082)
**Tính đồng bộ**: kết quả trả ngay trong chính yêu cầu (FR-075)

### Tiêu đề yêu cầu

| Tiêu đề | Bắt buộc | Mô tả |
|---|---|---|
| `Authorization` | ✅ | Đã có sẵn từ `auth.Handle()` |
| `Idempotency-Key` | ✅ | Chuỗi bất biến do quản trị viên sinh ra, dùng để chống gửi lại (FR-068). Xem [research.md](../research.md) R11 |
| `Content-Type` | ✅ | `application/json` |

### Thân yêu cầu

```json
{
  "items": [
    { "userIdentifier": "khachhang@example.com", "planSlug": "premium-monthly" },
    { "userIdentifier": "nguyenvana",              "planSlug": "premium-monthly" },
    { "userIdentifier": "ad11ad11-ad11-ad11-ad11-ad11ad11ad12", "planSlug": "premium-monthly" }
  ]
}
```

| Trường | Kiểu | Bắt buộc | Ràng buộc |
|---|---|---|---|
| `items` | mảng | ✅ | **1 đến 500 phần tử** (FR-062) |
| `items[].userIdentifier` | `string` | ✅ | Không rỗng, không toàn khoảng trắng |
| `items[].planSlug` | `string` | ✅ | Mã gói dạng chữ |

Quy tắc nhận diện (FR-045, FR-046): hệ thống **tự** nhận biết `userIdentifier` là UUID, tên đăng nhập hay email. Quản trị viên **không** khai báo loại. So khớp tên đăng nhập và email không phân biệt hoa thường (FR-047).

### Thân đáp lời — `200 OK`

```json
{
  "message": "Đã xử lý yêu cầu cấp gói.",
  "data": {
    "total": 3,
    "succeeded": 2,
    "skipped": 0,
    "failed": 1,
    "results": [
      {
        "index": 0,
        "userIdentifier": "khachhang@example.com",
        "resolvedUserId": "3f7c1a20-…",
        "planSlug": "premium-monthly",
        "status": "success",
        "planName": "Gói Premium Tháng",
        "expiresAt": "2026-10-29T08:15:00Z"
      },
      {
        "index": 1,
        "userIdentifier": "khachhang2@example.com",
        "planSlug": "premium-monthly",
        "status": "error",
        "reason": "user_not_found",
        "message": "Không tìm thấy tài khoản khớp với mã định danh đã cung cấp."
      }
    ]
  }
}
```

| Trường | Mô tả |
|---|---|
| `total` | Số phần tử đã gửi |
| `succeeded` / `skipped` / `failed` | Tổng theo trạng thái |
| `results` | **Đúng** số phần tử đã gửi, cùng thứ tự (FR-065) |
| `results[].index` | Vị trí trong danh sách gốc |
| `results[].userIdentifier` | **Đúng chuỗi đã gửi**, không phải giá trị đã phân giải |
| `results[].resolvedUserId` | Chỉ có khi nhận diện thành công |
| `results[].status` | `success` \| `skipped` \| `error` |
| `results[].planName` | Tên gói tiếng Việt |
| `results[].expiresAt` | Thời điểm hết hạn **mới** sau khi cộng dồn (FR-059) |
| `results[].reason` | Mã lý do, chỉ khi không thành công |
| `results[].message` | Thông báo tiếng Việt, chỉ khi không thành công (FR-085) |

Mọi thông báo lỗi trả về cho quản trị viên viết bằng tiếng Việt và nêu rõ nguyên nhân cụ thể (FR-085). Các hằng số thông báo đặt ở đầu tệp handler, không viết rải rác.

### Ba trạng thái của một phần tử

| Trạng thái | Nghĩa | Có phần tử nào bị thay đổi không |
|---|---|---|
| `success` | Cấp gói thành công | Có |
| `skipped` | Bỏ qua có chủ đích, không phải lỗi | Không |
| `error` | Lỗi phần tử này | **Không** (FR-058) |

Một phần tử lỗi **không** làm hỏng các phần tử còn lại (FR-064). Mỗi phần tử là một giao dịch độc lập.

### Mã lý do

| `reason` | Nghĩa | FR |
|---|---|---|
| `identifier_blank` | Chuỗi rỗng hoặc toàn khoảng trắng | Edge case |
| `user_not_found` | Không khớp tài khoản nào | FR-057 |
| `identifier_ambiguous` | Khớp **nhiều** tài khoản — tuyệt đối không tự chọn | FR-048 |
| `plan_not_found` | Mã gói không có trong danh mục | FR-054 |
| `plan_inactive` | Gói đã bị vô hiệu hoá | FR-055 |
| `target_is_admin` | Đích là tài khoản quản trị viên | FR-056 |
| `user_soft_deleted` | Tài khoản đã bị xoá mềm | FR-049 |
| `duplicate_in_request` | Trùng tài khoản trong cùng danh sách | FR-066 |
| `duplicate_request` | Yêu cầu đã được xử lý với khoá này | FR-068 |
| `concurrent_modification` | Thao tác khác vừa đổi gói tài khoản này | FR-069 |
| `identifier_too_long` | Định danh dài hơn cột cho phép | FR-062 |
| `plan_slug_too_long` | Mã gọi dài hơn cột cho phép | FR-062 |
| `grant_failed` | Ghi bản ghi gói **thất bại** – nguyên nhân hạ tầng | FR-059, T129 |
| `batch_timeout` | Hết ngân sách thời gian của yêu cầu, các mục còn lại không được xử lý | FR-076, T143 |

`identifier_too_long` và `plan_slug_too_long` là **lỗi định dạng**, bị từ chối trước khi truy vấn cơ sở dữ liệu: một giá trị dài hơn cột không thể dùng index, nên để nó lọt vào sẽ biến tra cứu của cả yêu cầu thành quét bảng.

`grant_failed` là lý do duy nhất trong bảng này chỉ ra **lỗi hệ thống** chứ không phải yêu cầu sai. Trước T129 nó bị gộp vào `concurrent_modification`, nên một sự cố hạ tầng được báo cho quản trị viên như thể tài khoản vừa bị người khác thay đổi – khiến họ đi tìm một thao tác không tồn tại.

`batch_timeout` không phải lỗi của riêng một mục mà là hết thời gian cho **cả yêu cầu**. Các mục đã cấp vẫn giữ nguyên (FR-076); quản trị viên được yêu cầu gửi lại với **cùng** `Idempotency-Key`, và FR-068 biến những dòng đó thành no-op.


`identifier_ambiguous` là lý do cần một cột `LOWER(username)` và `LOWER(email)` duy nhất — hai cột đó đã có, nên trường hợp này về lý thuyết không xảy ra. Yêu cầu vẫn giữ để hành vi không phụ thuộc vào ràng buộc cơ sở dữ liệu.

### Mã lý do của luồng khôi phục

`POST /admin/subscription-grants/restore` dùng bộ mã **riêng**, không dùng chung bảng trên:

| `reason` | Nghĩa |
|---|---|
| `plan_not_in_catalog` | Gói mà bản ghi gói trỏ tới không còn trong danh mục, không có gì để dựng lại quyền |
| `subscription_expired` | Gói đã hết hạn – cố ý bỏ qua, vì phát sự kiện cho tài khoản hết hạn sẽ tạo ra một quyền “đang hoạt động” có ngày kết thúc nằm ở quá khứ |
| `publish_failed` | Không phát được sự kiện cho tài khoản này |
| `already_restored` | Một lần chạy trước đã khôi phục tài khoản này; khoá sự kiện suy ra từ `version` của gói mà khôi phục không đổi, nên chạy lại là an toàn |


---

## Phản hồi lỗi — từ chối **toàn bộ** yêu cầu

Ba trường hợp dưới đây từ chối cả yêu cầu, **không** cấp gói cho bất kỳ phần tử nào.

| Tình huống | Mã | Nội dung tiếng Việt | FR |
|---|---|---|---|
| `items` rỗng | `400` | Danh sách cấp gói không được để trống. | FR-063 |
| `items` vượt 500 | `400` | Danh sách cấp gói tối đa 500 tài khoản mỗi lần. | FR-062 |
| Vượt giới hạn tần suất | `429` | Bạn đang thao tác quá nhanh. Vui lòng thử lại sau **{n}** giây. | FR-079 |

Từ chối toàn bộ là hành vi **đúng** ở ba trường hợp này: danh sách rỗng hoặc quá lớn thì không thể xử lý từng phần tử, còn vượt tần suất thì quản trị viên phải chờ chứ không phải nhận kết quả thiếu.

Lỗi khác — sai định dạng JSON, thiếu trường bắt buộc, `Idempotency-Key` thiếu — trả `400` với thông báo tiếng Việt.

---

## Giới hạn tần suất (FR-078)

Hai mức độc lập, đếm **theo quản trị viên**, cửa sổ 60 giây.

| Mức | Áp dụng khi | Hạn ngạch |
|---|---|---|
| Chung | Mọi yêu cầu | 60 yêu cầu / 60 giây |
| Hàng loạt | `items` có **hơn 1** phần tử | 6 yêu cầu / 60 giây |

Cơ chế: `IRateLimiter.Allow(ctx, key, limit, windowSeconds)` — tham số truyền vào nên hỗ trợ hạn ngạch tuỳ biến. Hai khoá Redis: `rl:admin-grant:{adminId}:all` và `rl:admin-grant:{adminId}:multi`. Mẫu có sẵn để theo: `LoginRateLimitMiddleware` dùng khoá theo danh tính thay vì theo IP.

Cần trần cứng vì một tài khoản quản trị viên bị lộ thông tin đăng nhập không được phát tán hàng nghìn yêu cầu trong một phút.

Giới hạn này **bổ sung**, không thay thế giới hạn chung theo IP đang áp dụng ở tầng engine.

---

## Ngưỡng thời gian (FR-076)

| Quy mô | Hạn ngạch |
|---|---|
| 1 phần tử | Phản hồi trong **5 giây** |
| 500 phần tử | Hoàn tất trong **60 giây** |

Ngưỡng này chỉ đo **ghi nhận kết quả cấp gói**, không gồm thời gian mức quyền tính phí AI có hiệu lực — phần đó là bất đồng bộ và đo bằng tiêu chí riêng (SC-014: trong 60 giây).

**Cảnh báo triển khai (đã chốt, đã xử lý ở T001)**: `server.write_timeout_seconds` **đã được nâng lên `120`** ở cả `configs/config.yaml` và `configs/config.production.yaml` (dòng 14 của mỗi tệp), nên hạn chót ghi đáp lời lớn hơn ngưỡng 60 giây của 500 phần tử và đáp lời không bị cắt giữa chừng dù mọi phần tử đã cấp xong. *(Trước T001 giá trị là `60` — trùng đúng ngưỡng, và đó là lý do task này tồn tại. Nếu sau này có đổi cấu hình, phải giữ `120`.)* Xem [research.md](../research.md) R9.

---

## Hành vi sau khi cấp thành công

| Việc | Mô tả | FR |
|---|---|---|
| Ghi nhận ký kiểm toán | Quản trị viên, tài khoản, mã gói, thời điểm, địa chỉ mạng | FR-081 |
| Ghi bản ghi lịch sử | `actor_admin_id` được điền lần đầu tiên trong toàn bộ kho mã | FR-012 |
| Phát tín hiệu | `subscription.plan.changed` phiên bản 2, trong cùng giao dịch | FR-015 |
| Xoá bộ nhớ đệm tổng quan gói | Tài khoản thấy gói mới ngay khi đăng nhập lại | FR-073 |
| Gửi thông báo | Cùng một mẫu `admin_grant` bất kể danh sách một hay nhiều phần tử | FR-083 |
| Thất bại gửi thông báo | **Không** làm thất bại thao tác cấp quyền; thao tác vẫn thành công và vẫn ghi nhận ký | FR-084 |

Không có thao tác tra cứu lần hai trong bất kỳ trường hợp nào (SC-016).

---

## `POST /api/v1/admin/subscription-grants/restore`

Khôi phục mức quyền tính phí AI cho tài khoản đang có gói nhưng thiếu bản ghi (FR-088).

**Phân quyền**: `auth.Handle()` + `RolesAuthorize(roleslug.Admin)` (FR-089)
**Ghi nhận ký**: bắt buộc (FR-089)

### Thân yêu cầu

```json
{ "dryRun": true }
```

| Trường | Mặc định | Mô tả |
|---|---|---|
| `dryRun` | `true` | Chỉ báo cáo, không phát tín hiệu. Đặt `false` để thực sự phát lại tín hiệu |

Mặc định là `true` vì đây là thao tác ghi hàng loạt lên dữ liệu người dùng — cần thấy kết quả thử trước khi áp dụng.

### Thân đáp lời

```json
{
  "message": "Đã khôi phục mức quyền tính phí AI.",
  "data": {
    "dryRun": true,
    "scanned": 6,
    "restored": 6,
    "failed": 0,
    "results": [
      { "userId": "…", "planSlug": "free",        "status": "success", "reason": "" },
      { "userId": "…", "planSlug": "premium-monthly", "status": "error",
        "reason": "plan_not_in_catalog", "message": "…" }
    ]
  }
}
```

Quét **mọi** tài khoản thiếu bản ghi mức quyền tính phí, **không** phân biệt gói miễn phí hay gói trả phí (FR-088). Chạy lại nhiều lần được mà không tạo tác dụng phụ — nhờ chống trùng theo `sourceEventId`, lần thứ hai trở đi bỏ qua im lặng.

Không được yêu cầu can thiệp trực tiếp vào cơ sở dữ liệu.

---

## Ghi chú về thay đổi phá vỡ

| Đường dẫn cũ | Trạng thái | Lý do |
|---|---|---|
| `POST /api/v1/admin/subscriptions/{userId}/grant` | **Gỡ** | FR-060 chỉ cho phép một giao diện; FR-012 nói thao tác này hiện **không hoạt động** vì không sinh bản ghi lịch sử |
| `POST /api/v1/admin/subscriptions/{userId}/cancel` | **Giữ nguyên** | Không thuộc phạm vi FR-060 (thao tác cấp gói). Xoá đi sẽ mất chức năng thu hồi gói đang chạy |
| `POST /api/v1/admin/ai/grants` (dịch vụ AI) | **Gỡ** | FR-035 cho phép gỡ nếu không tạo được bản ghi đầy đủ. Nó luôn vi phạm `policy_snapshot NOT NULL`. Xem [research.md](../research.md) R4 |
| `GET /api/v1/admin/ai/grants` (dịch vụ AI) | **Giữ nguyên** | Cần cho quan sát |

Sau khi đổi route phải chạy `make generate svc=account` để sinh lại Swagger và tuyến nginx.
