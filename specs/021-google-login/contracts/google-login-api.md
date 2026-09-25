# Contract — Google Login API (identity)

Base: `/api/v1/auth`. Nguyên tắc chung theo [docs/api/README.md](../../../docs/api/README.md): JSON camelCase,
message tiếng Việt, lỗi qua `apperror`. Phiên đăng nhập dùng cùng cơ chế access/refresh token hiện có.
Mọi endpoint dưới `/auth` (kể cả Google) chịu **rate-limit shield** hiện có (`claim_rate_limit`, key theo
IP + định danh, `window_seconds`) — xem FR-022.

Tài liệu hướng dẫn tích hợp frontend (web + mobile): [docs/api/identity/google-login-frontend-guide.md](../../../docs/api/identity/google-login-frontend-guide.md).

---

## 1. `POST /api/v1/auth/google` — đăng nhập bằng Google ID token

Dùng cho **mobile/app** (Google Sign-In SDK) và tuỳ chọn cho **web** (Google Identity Services trả ID token).

**Request body**

```json
{
  "idToken": "eyJhbGciOiJSUzI1NiIs...",
  "deviceName": "iPhone 15",
  "deviceIp": "1.2.3.4",
  "userAgent": "Closy/1.0"
}
```

| Trường | Kiểu | Bắt buộc | Ghi chú |
|---|---|---|---|
| `idToken` | string | ✅ | ID token Google (credential) |
| `deviceName` / `deviceIp` / `userAgent` | string | ❌ | Thông tin phiên; nếu trống, backend lấy IP/User-Agent từ request |

**Success 200**

```json
{
  "statusCode": 200,
  "message": "Đăng nhập thành công",
  "data": { "accessToken": "...", "refreshToken": "..." }
}
```

Kèm `Set-Cookie: accessToken=...; HttpOnly; Secure; SameSite=Strict` và `refreshToken` tương tự
(cookie phục vụ web; mobile dùng token trong body).

**Lỗi**

| HTTP | Tình huống | Message (VI) |
|---|---|---|
| 400 | ID token sai/hết hạn/không đúng audience | Token Google không hợp lệ. |
| 400 | Google email chưa xác thực | Email Google chưa được xác thực. |
| 400 | Email trùng tài khoản chưa verify (admin tạo) | Email đã được đăng ký. Vui lòng đăng nhập bằng mật khẩu. |
| 403 | Tài khoản bị vô hiệu hóa/tạm ngưng/blacklist | Tài khoản đã bị khoá hoặc vô hiệu hoá... |
| 409 | Email đã liên kết danh tính Google khác | Email đã được liên kết với một tài khoản Google khác. |
| 500 | Lỗi cấp phiên/tạo tài khoản | Lỗi khi cấp phiên làm việc. |

---

## 2. `GET /api/v1/auth/google` — bắt đầu luồng web (redirect)

**Query**

| Tham số | Bắt buộc | Ghi chú |
|---|---|---|
| `redirectUrl` | ✅ | Đích quay về sau khi hoàn tất (thuộc `server.front_end_origins`). Tên camelCase — không dùng `redirect_uri`. |

**Hành vi**
1. Validate `redirectUrl` (scheme + host chính xác, xem §4). Sai → **400 JSON**.
2. Sinh `state` (one-time, TTL) + PKCE verifier; lưu Redis `{ redirectUrl, codeVerifier }`.
3. **302** → Google authorization URL (`scope=openid email profile`, `redirect_uri` = callback backend).

**Lỗi**: `400` nếu `redirectUrl` không hợp lệ (không chuyển hướng).

---

## 3. `GET /api/v1/auth/google/callback` — Google gọi về backend

**Query** (do Google gửi): `code`, `state` (hoặc `error` nếu người dùng huỷ).

**Hành vi**

| Trường hợp | Kết quả |
|---|---|
| Thành công | Set cookie `accessToken`/`refreshToken`; **302 → redirectUrl** (lấy từ `state`) |
| `state` sai/hết hạn/đã dùng | **400 JSON** (không có đích tin cậy) |
| Người dùng huỷ / `error=access_denied` | **302 → redirectUrl?error=access_denied** |
| Code hết hạn / exchange thất bại | **302 → redirectUrl?error=exchange_failed** |
| Email Google chưa xác thực | **302 → redirectUrl?error=email_unverified** |
| Email trùng tài khoản chưa verify | **302 → redirectUrl?error=email_registered** |
| Đã liên kết Google khác | **302 → redirectUrl?error=account_linked** |
| Tài khoản bị khoá | **302 → redirectUrl?error=account_disabled** |
| Lỗi hệ thống | **302 → redirectUrl?error=server_error** |

> Web không cần đọc token từ URL. Sau khi quay về, FE gọi `GET /api/v1/me` kèm credentials để xác nhận phiên.

---

## 4. Quy tắc validate `redirectUrl`

- Parse URL; so **scheme + host chính xác** với một phần tử trong `server.front_end_origins`.
- Chỉ chấp nhận `https`; ngoại lệ `http://localhost[:port]` khi dev.
- **Không** cho subdomain; chuẩn hóa port và dấu chấm cuối trước khi so hostname.
- Cho phép path/query/fragment phía sau.
- Không dùng so khớp tiền tố/hậu tố chuỗi thô.

---

## 5. Endpoint dùng chung (không đổi)

| Endpoint | Ghi chú |
|---|---|
| `POST /api/v1/auth/refresh-token` | Cookie (web) hoặc body `oldRefreshToken` (mobile); rotation giữ nguyên |
| `POST /api/v1/auth/logout` | Web xoá cookie; mobile gửi `refreshToken` trong body |
| `GET /api/v1/me` | Xác nhận phiên sau redirect (web kèm credentials, mobile kèm Bearer) |

## 6. Constants

- `provider`: `local` \\| `google` (mới) — xem `identity/constants.md` (cập nhật khi implement).
- `roleSlug`: `user` \\| `admin` — xem [constants.md](../../../docs/api/identity/constants.md).
- `userStatus`: `active` \\| `inactive` \\| `suspended`.
