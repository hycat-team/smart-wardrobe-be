# Hợp Đồng: Sự Kiện Tạo Tài Khoản (`identity.user.created`)

**Chủ sở hữu**: module `identity` (dịch vụ `account`) · **Bên nhận hiện tại**: cơ chế chiến dịch

---

## 1. Vì sao chủ đề này tồn tại

Trước tính năng này, hệ thống **không có** tín hiệu nào nói rằng một tài khoản vừa được tạo. Sự kiện
duy nhất phát ra lúc tạo tài khoản là sự kiện đổi gói, mà nó mang nghĩa *"tài khoản này có gói gì"*,
không phải *"tài khoản này vừa xuất hiện"*. Mọi tính năng sau này cần tín hiệu thứ hai đều phải chế
tạo một chủ đề riêng và sửa lại luồng đăng ký.

Chủ đề này lấp đúng lỗ hổng đó, và nó phát **vô điều kiện** (FR-084): người phát không biết có tính
năng nào đang lắng nghe, vì vậy thêm bên nhận thông báo không phải sửa lại luồng đăng ký (FR-090).

## 2. Nội dung thông điệp

```json
{
  "userId": "b1c2...  (uuid)",
  "provider": "local",
  "registeredAt": "2026-10-02T00:00:03+07:00",
  "createdByAdmin": false
}
```

| Trường | Bắt buộc | Nguồn | Có thể tin cậy? |
|---|---|---|---|
| `userId` | Có | Sinh ngay trước khi ghi bảng tài khoản | Có |
| `provider` | Có | Nguồn đăng ký | Có |
| `registeredAt` | Có | `users.created_at` | Có — đây là giá trị mà FR-031 so sánh với mốc ưu tiên |
| `createdByAdmin` | Có | Đường gọi đang thực thi | Có, **nhưng chỉ khi đọc từ thông điệp** — xem mục 5 |

**Cấm tuyệt đối** (FR-087): email, tên, họ, ngày sinh, địa chỉ, số điện thoại, mật khẩu, mã gói cước,
mã chính sách chi phí. Bên nhận thông báo cần thêm thông tin thì đọc cơ sở dữ liệu — cùng một dịch
vụ nên không tốn gọi mạng.

Nhãn JSON: `camelCase`, khớp quy ước chung của dự án cho dữ liệu thông điện.

## 3. Đường phát

| # | Đường tạo tài khoản | Tệp | `createdByAdmin` |
|---|---|---|---|
| 1 | Xác nhận mã đăng ký email | `identity/application/usecase/register_uc.go` | `false` |
| 2 | Đăng nhập Google tạo mới | `identity/application/usecase/google_auth_uc.go` | `false` |
| 3 | Quản trị viên tạo tài khoản | `identity/application/usecase/admin_uc.go` | `true` |

Cả ba phát **bên trong cùng transaction** ghi bảng tài khoản. Nghĩa là:

- Thông điệp chỉ tồn tại khi tài khoản thực sự được tạo. Không có thông điệp cho tài khoản bị
  rollback.
- Nếu ghi thông điệp thất bại, việc tạo tài khoản **thất bại theo** (FR-042). Đây là đánh đổi có chủ
  ý: ưu tiên không mất thông báo hơn là tạo ra một tài khoản mà không ai biết là chưa từng được thông
  báo.

### 3.1 Hành vi sửa lỗi tranh chấp đã có sẵn

Cả hai đường tự phục vụ đều có xử lý riêng cho tình huống hai người cùng tạo tài khoản cùng lúc:
`register_uc.go:189-195` và `google_auth_uc.go:246-252` kiểm tra lại rồi trả về lỗi nghiệp vụ. Thông
điệp **không** được phát trong các trường hợp đó, vì transaction đã rollback.

## 4. Hàng đỗi và vòng đời

| Thuộc tính | Giá trị |
|---|---|
| Tên hàng đỗi | `campaign_grant_user_created_queue` |
| Loại hàng đỗi | quorum, `x-delivery-limit: 5` |
| Số bên nhận thông báo | `Prefetch 20`, `Concurrency 1` |
| Hàng đỗi chết | `campaign_grant_user_created_queue.dlq` |

Tên hàng đỗi đặt theo **bên nhận thông báo**, không theo chiến dịch. Lý do ở R12: tên hàng đỗi phải
là hằng số tĩnh, và FR-090 chỉ đúng được khi hàng đỗi gắn với bên nhận chứ không gắn với chủ đề dữ
liệu.

### 4.1 Mở rộng trong tương lai

Khi có bên nhận thông báo thứ hai cho cùng chủ đề (email chào mừng, thưởng giới thiệu, đo tăng trưởng),
**mỗi bên một hàng đỗi riêng**, cùng gắn vào `identity.user.created`. Mô hình phát/thu, mỗi bên xử
lý độc lập và không chen nhau. Mẫu có sẵn trong dự án: `fashion.item.created` tách `.wardrobe` và
`.brand`.

Không cần sửa lại luồng đăng ký cho bên nhận thứ hai. Đó chính là điều FR-090 đòi.

## 5. Cạm bẫy: `createdByAdmin` không suy ra được từ dữ liệu

Bảng `identity.users` **không có** trường nào ghi ai đã tạo dòng. Hai đường tạo tài khoản dựng ra hai
hàng giống hệt nhau về cấu trúc.

⇒ `createdByAdmin` trong thông điệp lấy được **chỉ** từ đường gọi đang thực thi. Đây là lý do cột
`identity.users.created_by_admin` vẫn được thêm vào (mục 2 của `data-model.md`): cơ chế quét bù
đọc từ cơ sở dữ liệu, nên nó **không có thông điệp để đọc**.

**Hai nguồn phải khớp nhau.** Nếu ai đó đặt cờ ở một chỗ và phát thông điệp ở một chỗ khác, chúng
lệch nhau im lặng. Kiểm thử bắt buộc: cùng một tài khoản được tạo qua cả ba đường, đối chiếu giá trị
trong thông điệp với giá trị trong cột.

**Không** suy ra từ `email_verified = false`. Đó là hai sự thật khác nhau: quản trị viên hoàn toàn có
thể tạo tài khoản đã xác minh, và mọi tài khoản tự phục vụ đều đã xác minh. Suy ra sai là hỏng âm
thầm.

## 6. Cấm

| Cấm | Mã điều khoản |
|---|---|
| Thêm loại sự kiện đổi gói mới vào tập đóng dùng chung | FR-092 |
| Cho dịch vụ AI tiêu thụ thêm chủ đề này | FR-097 |
| Để module tạo tài khoản tra cứu gói cước để làm phong bị thông điện | FR-098 |
| Chứa dữ liệu nhận diện cá nhân trong thông điện | FR-087 |

## 7. Kiểm thử hợp đồng

| # | Kịch bản | Điều kiện đạt |
|---|---|---|
| 1 | Đăng ký bằng email | Đúng 1 thông điệp, `provider = local`, `createdByAdmin = false`, `registeredAt` khớp `users.created_at` |
| 2 | Đăng nhập Google tạo mới | Đúng 1 thông điệp, `provider = google`, `createdByAdmin = false` |
| 3 | Quản trị viên tạo tài khoản | Đúng 1 thông điệp, `createdByAdmin = true`, **và** cột `created_by_admin = true` |
| 4 | Liên kết Google vào tài khoản có sẵn | **0** thông điệp — tài khoản không phải mới |
| 5 | Hai người cùng đăng ký một email | Tài khoản thứ hai không được tạo ⇒ **0** thông điệp thứ hai |
| 6 | Ghi thông điệp thất bại | Yêu cầu đăng ký trả về lỗi, tài khoản không được tạo |
| 7 | Quét toàn bộ thông điệp phát ra | Không trường nào chứa email, tên, ho, ngày sinh, địa chỉ |
