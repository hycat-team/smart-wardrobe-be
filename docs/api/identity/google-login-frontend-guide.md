# Hướng dẫn Frontend — Đăng nhập Google (Web & Mobile)

> **Trạng thái**: Đây là hợp đồng tích hợp cho dev frontend của tính năng "Đăng nhập Google" (spec
> `021-google-login`). Backend đang được triển khai theo hợp đồng tại
> [`specs/021-google-login/contracts/google-login-api.md`](../../../specs/021-google-login/contracts/google-login-api.md).
> Khi backend hoàn tất, tài liệu này là điểm tham chiếu chuẩn cho FE; mọi thay đổi hợp đồng phải cập nhật cả hai nơi.

Tài liệu này hướng dẫn **frontend phải làm gì** để tích hợp đăng nhập Google, cho **cả web và mobile**.
Không mô tả chi tiết nội bộ backend.

---

## 0. Chuẩn bị chung

- **API base**: `{API_BASE}` (ví dụ `https://api.closy.hycat.online`).
- **Google Client ID** (lấy từ team backend / Google Cloud Console):
  - Web: `WEB_CLIENT_ID`
  - iOS: `IOS_CLIENT_ID`
  - Android: `ANDROID_CLIENT_ID`
  - Nếu mobile dùng "server client ID", dùng `WEB_CLIENT_ID`.
- **Điều kiện tiên quyết**:
  - Web và API **cùng registrable domain** (ví dụ FE `closy.hycat.online`, API `api.closy.hycat.online`) để cookie `SameSite=Strict` hoạt động.
  - Mọi request cần phiên phải gửi kèm credentials (`fetch(..., { credentials: 'include' })`).
- **Endpoint liên quan**:

| Việc | Method + Path |
|---|---|
| Web bắt đầu đăng nhập (redirect) | `GET {API_BASE}/api/v1/auth/google?redirectUrl=<URL>` |
| Mobile (và web tuỳ chọn) đăng nhập bằng ID token | `POST {API_BASE}/api/v1/auth/google` |
| Kiểm tra phiên / lấy hồ sơ | `GET {API_BASE}/api/v1/me` |
| Xoay vòng token | `POST {API_BASE}/api/v1/auth/refresh-token` |
| Đăng xuất | `POST {API_BASE}/api/v1/auth/logout` |

---

## 1. Luồng WEB — Authorization Code + redirect (khuyến nghị)

Browser rời khỏi app sang Google rồi quay về. Không cần nạp script Google.

### Các bước

1. **Nút "Đăng nhập bằng Google"** → điều hướng trình duyệt tới:

   ```
   {API_BASE}/api/v1/auth/google?redirectUrl={encodeURIComponent(FE_RETURN_URL)}
   ```

   Trong đó `FE_RETURN_URL` là URL trong app FE mà bạn muốn quay về sau khi đăng nhập, **cùng host với
   `front_end_origin`** đã đăng ký (ví dụ `https://closy.hycat.online/auth/callback`). Có thể kèm path/query.

   ```js
   const API_BASE = import.meta.env.VITE_API_BASE; // hoặc cấu hình tương đương
   function loginWithGoogle() {
     const returnUrl = `${window.location.origin}/auth/callback`;
     window.location.href =
       `${API_BASE}/api/v1/auth/google?redirectUrl=${encodeURIComponent(returnUrl)}`;
   }
   ```

2. Backend kiểm tra `redirectUrl` rồi chuyển hướng sang Google. Người dùng đăng nhập/đồng ý.

3. Google gọi callback backend; backend tạo phiên và **đặt cookie `accessToken`/`refreshToken`
   (HttpOnly)** rồi chuyển hướng về `FE_RETURN_URL`.

4. Ở trang `FE_RETURN_URL`, FE **kiểm tra URL có `?error=` không**:
   - Không có `error` → gọi `GET /me` để xác nhận phiên:

     ```js
     async function loadSession() {
       const res = await fetch(`${API_BASE}/api/v1/me`, {
         credentials: 'include', // BẮT BUỘC: gửi cookie HttpOnly
       });
       if (res.status === 401) return null; // phiên không hợp lệ/hết hạn
       const body = await res.json();
       return body.data;
     }
     ```
   - Có `error` → hiển thị thông báo tương ứng (bảng §4).

> **Lưu ý bảo mật**: Không cố đọc cookie `accessToken`/`refreshToken` từ JS — chúng là HttpOnly.
> Không nhét token vào URL. Web dựa hoàn toàn vào cookie + `credentials: 'include'`.
>
> **Lưu ý cookie domain**: Ở production, backend set cookie phiên với `Domain=closy.hycat.online`
> (biến `COOKIE_DOMAIN`) vì FE chạy ở `closy.hycat.online` còn API ở `api.closy.hycat.online` —
> nhờ vậy cookie được gửi cho cả hai subdomain. Ở local để `COOKIE_DOMAIN` rỗng (host-only).

### Xử lý lỗi trên trang callback

```js
const params = new URLSearchParams(window.location.search);
const error = params.get('error');
if (error) {
  showError(mapErrorToMessage(error)); // xem §4
  // ví dụ: history.replaceState để xoá query
} else {
  const user = await loadSession();
  if (user) redirectToHome(); else showError('Không xác nhận được phiên đăng nhập.');
}
```

---

## 2. Luồng WEB — Google Identity Services lấy ID token (tuỳ chọn)

Nếu muốn tránh chuyển hướng, web có thể dùng GIS để lấy ID token rồi gọi cùng endpoint như mobile.

```html
<script src="https://accounts.google.com/gsi/client" async defer></script>
```

```js
google.accounts.id.initialize({
  client_id: WEB_CLIENT_ID,
  callback: async (response) => {
    const res = await fetch(`${API_BASE}/api/v1/auth/google`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      credentials: 'include', // vẫn cần để nhận cookie phiên
      body: JSON.stringify({ idToken: response.credential, deviceName: 'Web' }),
    });
    const body = await res.json();
    if (!res.ok) {
      showError(body.message);
      return;
    }
    // Web: cookie đã được set; gọi /me để lấy hồ sơ
    const user = await loadSession();
    redirectToHome(user);
  },
});
google.accounts.id.prompt();
```

**Chọn luồng nào?** Redirect (§1) không phụ thuộc JS Google và ổn định hơn; GIS (§2) cho trải nghiệm liền mạch
(One Tap) nhưng phụ thuộc script Google. Ưu tiên §1, bổ sung §2 nếu cần One Tap.

---

## 3. Luồng MOBILE (Android / iOS)

Mobile dùng SDK Google Sign-In để lấy **ID token**, rồi đổi sang phiên Closy qua backend.

### 3.1. Android (Kotlin, Google Sign-In)

```kotlin
// 1. Lấy ID token từ Google Sign-In (Credential Manager / GoogleSignIn)
val idToken: String = googleIdToken

// 2. Đổi lấy phiên Closy
val json = JSONObject().apply {
    put("idToken", idToken)
    put("deviceName", Build.MODEL)
}
// POST {API_BASE}/api/v1/auth/google  -> { data: { accessToken, refreshToken } }
```

### 3.2. iOS (Swift, Google Sign-In)

```swift
// 1. Lấy idToken từ GIDSignIn
let idToken = user.authentication.idToken

// 2. POST {API_BASE}/api/v1/auth/google
// body: { "idToken": idToken, "deviceName": UIDevice.current.name }
// response.data: { accessToken, refreshToken }
```

### 3.3. Sau khi có token (mobile)

- Lưu `accessToken` và `refreshToken` trong **secure storage** (Android Keystore / iOS Keychain).
  **Không** lưu ở nơi không an toàn.
- Gọi API kèm header: `Authorization: Bearer <accessToken>`.
- **Xoay vòng token** khi access token hết hạn:

  ```
  POST {API_BASE}/api/v1/auth/refresh-token
  Body: { "oldRefreshToken": "<refreshToken>", "deviceName": "<...>" }
  → data: { accessToken, refreshToken }  (refresh token cũ bị thu hồi — lưu token mới)
  ```

- **Đăng xuất**:

  ```
  POST {API_BASE}/api/v1/auth/logout
  Headers: Authorization: Bearer <accessToken>
  Body: { "refreshToken": "<refreshToken>" }
  ```

> Mobile **không** dùng cookie; luôn truyền token trong body/header.

---

## 4. Bảng mã lỗi (`error` query trên web / `message` từ API)

| Mã (`?error=`) | HTTP (mobile) | Ý nghĩa | Gợi ý UI |
|---|---|---|---|
| `access_denied` | — | Người dùng huỷ đồng ý tại Google | Quay lại màn đăng nhập, không báo lỗi nặng |
| `email_unverified` | 400 | Email Google chưa xác thực | "Email Google chưa được xác thực." |
| `email_registered` | 400 | Email trùng tài khoản chưa verify (admin tạo) | "Email đã được đăng ký. Vui lòng đăng nhập bằng mật khẩu." |
| `account_linked` | 409 | Email đã liên kết một Google khác | "Email đã liên kết với tài khoản Google khác." |
| `account_disabled` | 403 | Tài khoản bị khoá/vô hiệu hoá | Thông báo liên hệ CSKH |
| `exchange_failed` | 400 | Code hết hạn / đổi code thất bại | "Không hoàn tất được đăng nhập, vui lòng thử lại." |
| `server_error` | 500 | Lỗi hệ thống | "Có lỗi xảy ra, vui lòng thử lại sau." |
| (token invalid) | 400 | ID token sai/hết hạn | "Phiên Google không hợp lệ, thử đăng nhập lại." |

> Lưu ý: nếu `state` sai/hết hạn/đã dùng, backend trả **JSON 400** (không redirect), nên sẽ không xuất hiện `?error=` trên URL.

Backend luôn trả message tiếng Việt; FE có thể hiển thị trực tiếp hoặc map sang thông báo riêng.

---

## 5. Checklist QA cho Frontend

**Web**
- [ ] Nút đăng nhập điều hướng đúng `GET /auth/google?redirectUrl=...`.
- [ ] `redirectUrl` là URL cùng host FE (kèm path tuỳ ý).
- [ ] Mọi fetch tới API dùng `credentials: 'include'`.
- [ ] Trang callback đọc `?error=` và hiển thị thông báo phù hợp.
- [ ] Sau đăng nhập gọi `GET /me` để lấy hồ sơ; 401 → coi như chưa đăng nhập.
- [ ] Không truy cập cookie HttpOnly bằng JS.

**Mobile**
- [ ] Lấy được `idToken` từ Google Sign-In SDK với đúng client ID nền tảng.
- [ ] `POST /auth/google` trả về `accessToken`/`refreshToken`; lưu vào secure storage.
- [ ] Gắn `Authorization: Bearer` cho các API cần xác thực.
- [ ] Xử lý 401 → refresh token; refresh thất bại → đăng xuất và quay lại màn đăng nhập.
- [ ] Logout gọi `/auth/logout` với `refreshToken` và xoá token cục bộ.

**Chung**
- [ ] Tài khoản chỉ-Google vẫn có thể đặt mật khẩu qua luồng quên mật khẩu (nếu FE hỗ trợ).
- [ ] Người dùng đã có mật khẩu đăng nhập Google vẫn vào đúng tài khoản cũ (auto-link).

---

## 6. Tham chiếu

- Hợp đồng API đầy đủ: [`specs/021-google-login/contracts/google-login-api.md`](../../../specs/021-google-login/contracts/google-login-api.md)
- Mô hình dữ liệu: [`specs/021-google-login/data-model.md`](../../../specs/021-google-login/data-model.md)
- Kiểm chứng nhanh: [`specs/021-google-login/quickstart.md`](../../../specs/021-google-login/quickstart.md)
- Hằng số chung: [constants.md](./constants.md)
