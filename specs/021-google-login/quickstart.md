# Quickstart — Kiểm chứng Đăng nhập Google

Hướng dẫn chạy và kiểm chứng end-to-end. Không chứa code triển khai; chi tiết ở
[data-model.md](./data-model.md) và [contracts/google-login-api.md](./contracts/google-login-api.md).

## 1. Chuẩn bị

**Cấu hình (env)** — thêm vào `.env` / secret của môi trường:

```text
GOOGLE_CLIENT_IDS=<web-client-id>,<ios-client-id>,<android-client-id>
GOOGLE_CLIENT_SECRET=<client-secret>        # chỉ cần cho luồng web Authorization Code
GOOGLE_REDIRECT_URI=https://<api-host>/api/v1/auth/google/callback
```

Đăng ký `GOOGLE_REDIRECT_URI` trong Google Cloud Console > Credentials > OAuth client.

**Build & migrate** (từ repo root):

```powershell
make wire
make migrate-up svc=account
make build svc=account
```

**Chạy test liên quan**:

```powershell
make test-race
# hoặc tập trung module:
go test ./internal/modules/identity/... -run Google
```

## 2. Kịch bản kiểm chứng (backend)

| # | Kịch bản | Cách thực hiện | Kết quả mong đợi |
|---|---|---|---|
| 1 | Tạo tài khoản mới bằng Google | `POST /auth/google` với ID token mới (sub chưa tồn tại, email_verified=true) | 200 + token; DB có user `provider=google`, `username`=tiền tố email, `email_verified=true`, `password_hash=NULL` |
| 2 | Đăng nhập lại | Lặp lại #1 | 200; đúng user cũ, không tạo thêm |
| 3 | Auto-link account local đã verify | Tạo account local (đăng ký OTP) rồi `POST /auth/google` cùng email | 200; user cũ được gán `provider=google`, `provider_id=sub`; mật khẩu cũ vẫn dùng được |
| 4 | Account admin tạo (chưa verify) | Admin tạo user, rồi `POST /auth/google` cùng email | 400 "Email đã được đăng ký. Vui lòng đăng nhập bằng mật khẩu." |
| 5 | Google email chưa xác thực | ID token `email_verified=false` | 400 từ chối, không tạo/link |
| 6 | Tài khoản bị vô hiệu hóa | Set status `inactive`/`suspended` rồi `POST /auth/google` | 403 account disabled |
| 7 | Đã link Google khác | Account `provider=google, provider_id=A`; login với sub B cùng email | 409/400 từ chối |
| 8 | Race first-login | Gửi 2 request `POST /auth/google` song song cùng sub mới | Cả hai 200; DB chỉ có 1 user (0 trùng) |
| 9 | Email khác hoa thường | Local `Duck@...`, Google `duCk@...` | Coi là cùng email (auto-link đúng) |
| 10 | Username trùng | Hai email khác nhau cùng tiền tố | Username thứ hai được sinh khác, vẫn hợp lệ |

### Luồng web

| # | Kịch bản | Cách thực hiện | Kết quả mong đợi |
|---|---|---|---|
| 11 | Happy path | Mở `GET /auth/google?redirectUrl=https://<fe-origin>/callback` trong trình duyệt | 302 Google → consent → callback set cookie → 302 về `.../callback`; FE gọi `/me` OK |
| 12 | `redirectUrl` host giả mạo | `redirectUrl=https://<fe-origin>.hehe/`, `https://<fe-origin>@evil.com`, `http://...` (ngoài localhost) | 400 JSON, không redirect |
| 13 | `redirectUrl` subdomain | `https://evil.<fe-origin>/` | 400 JSON |
| 14 | `redirectUrl` path hợp lệ | `https://<fe-origin>/a/b?x=1` | Được chấp nhận, quay về đúng URL |
| 15 | User huỷ consent | Bấm Cancel tại Google | 302 về `redirectUrl?error=access_denied` |
| 16 | `state` sai/hết hạn/dùng lại | Gọi callback với state lạ hoặc phát lại | 400 JSON, không cấp phiên |

### Vòng đời mật khẩu

| # | Kịch bản | Cách thực hiện | Kết quả mong đợi |
|---|---|---|---|
| 17 | Set password lần đầu | Account Google-only → luồng quên mật khẩu → đặt mật khẩu | Thành công; sau đó `POST /auth/login` bằng email+mật khẩu OK |
| 18 | Login mật khẩu khi chưa có mật khẩu | `POST /auth/login` cho account Google-only với mật khẩu bất kỳ | 400 "Sai tài khoản hoặc mật khẩu." |
| 19 | Đổi mật khẩu lần đầu | Account Google-only → `PUT /me/change-password` không cần `oldPassword` | Thành công, set được mật khẩu |

## 3. Tham chiếu artifact

- Hợp đồng API: [contracts/google-login-api.md](./contracts/google-login-api.md)
- Mô hình dữ liệu & migration: [data-model.md](./data-model.md)
- Nghiên cứu quyết định: [research.md](./research.md)
- Hướng dẫn frontend (web + mobile): [docs/api/identity/google-login-frontend-guide.md](../../docs/api/identity/google-login-frontend-guide.md)
