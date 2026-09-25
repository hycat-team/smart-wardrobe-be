# Phase 1 — Data Model: Đăng nhập Google

## 1. Thay đổi entity `identity.users`

File: `services/account/internal/modules/identity/domain/entities/user.go`

| Thuộc tính | Kiểu Go | mapstructure/GORM | Thay đổi | Ghi chú |
|---|---|---|---|---|
| `Provider` | `string` | `varchar(50);not null;default:'local'` | **Mới** | `local` \\| `google` |
| `ProviderID` | `*string` | `varchar(255)` | **Mới** | Google `sub`; NULL với tài khoản local |
| `EmailVerified` | `bool` | `boolean;not null;default:false` | **Mới** | true với tài khoản đăng ký OTP hoặc Google email_verified |
| `PasswordHash` | `*string` | `varchar(255)` (bỏ `not null`) | **Đổi** | Tài khoản Google-only có thể NULL |
| `Username` | `string` | giữ nguyên `not null`, unique `lower(username)` | Không đổi | Sinh tự động cho tài khoản Google |
| `Email` | `string` | giữ nguyên `not null`, unique `lower(email)` | Không đổi | Luôn chuẩn hóa lowercase |
| `RoleSlug` | `roleslug.RoleSlug` | giữ nguyên | Không đổi | Tài khoản Google luôn `user` |
| `Status` | `userstatus.UserStatus` | giữ nguyên | Không đổi | Tài khoản Google tạo với `Active` |

Hằng số mới: `domain/constants/authprovider/authprovider.go`

```go
package authprovider

type AuthProvider string

const (
    Local  AuthProvider = "local"
    Google AuthProvider = "google"
)
```

### Tác động tới code hiện có
- `SessionUseCase.Login`: `user.PasswordHash` giờ là con trỏ → guard `if user.PasswordHash == nil || *user.PasswordHash == ""` → `ErrInvalidCredentials`.
- `register_uc.go::ConfirmRegisterOtp`: tạo user với `Provider=local`, `EmailVerified=true`.
- `admin_uc.go::CreateUserByAdmin`: tạo user với `Provider=local`, `EmailVerified=false` (chưa từng verify).
- `password_recovery_uc.go::ResetPassword` & `profile_uc.go::ChangePassword`: thiết lập hash lần đầu cho tài khoản Google-only.
- Mapper/`UserRes`: không bắt buộc lộ `provider`/`emailVerified` ra API (có thể thêm sau, ngoài phạm vi).

## 2. Migration (Goose)

File: `services/account/database/migrations/20260925090000_add_google_auth_to_users.sql`

```sql
-- +goose Up
-- +goose StatementBegin
ALTER TABLE identity.users
    ADD COLUMN IF NOT EXISTS provider       varchar(50)  NOT NULL DEFAULT 'local',
    ADD COLUMN IF NOT EXISTS provider_id    varchar(255),
    ADD COLUMN IF NOT EXISTS email_verified boolean      NOT NULL DEFAULT false;

ALTER TABLE identity.users
    ALTER COLUMN password_hash DROP NOT NULL;

-- Mỗi định danh nhà cung cấp chỉ thuộc một tài khoản; bỏ qua tài khoản local (provider_id NULL).
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_provider_provider_id
    ON identity.users (provider, provider_id)
    WHERE provider_id IS NOT NULL;

-- Backfill: toàn bộ người dùng hiện hữu được coi là đã xác thực email (quyết định #8).
UPDATE identity.users
   SET email_verified = true
 WHERE provider = 'local';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS identity.idx_users_provider_provider_id;

ALTER TABLE identity.users
    ALTER COLUMN password_hash SET NOT NULL;

ALTER TABLE identity.users
    DROP COLUMN IF EXISTS provider,
    DROP COLUMN IF EXISTS provider_id,
    DROP COLUMN IF EXISTS email_verified;
-- +goose StatementEnd
```

> **Lưu ý Down**: `password_hash SET NOT NULL` sẽ fail nếu tồn tại tài khoản Google-only. Down migration chỉ dùng ở
> môi trường dev/drill; tài liệu ghi rõ cần xử lý dữ liệu trước khi rollback.

## 3. Giá trị tạm thời: `OAuthState`

Không phải bảng DB — lưu trên Redis qua `ICacheService`.

Value object: `application/vo/oauth_state.go`

```go
type OAuthState struct {
    RedirectURL  string `json:"redirectUrl"`  // đã validate, thuộc front_end_origins
    CodeVerifier string `json:"codeVerifier"` // PKCE
}
```

| Thuộc tính | Ý nghĩa | Ràng buộc |
|---|---|---|
| key | `oauth:google:state:<state>` | `state` ngẫu nhiên `crypto/rand`, one-time |
| TTL | `google.state_ttl_minutes` (mặc định 10) | hết hạn → callback từ chối |
| RedirectURL | đích quay về sau thành công/lỗi | phải vượt qua validate host/scheme |
| CodeVerifier | PKCE verifier | dùng khi exchange code |

## 4. Cấu hình mới (`shared/config`)

Block `google` trong `configs/config.yaml` / `config.production.yaml` (env-driven):

```yaml
google:
  client_ids: "${GOOGLE_CLIENT_IDS}"        # danh sách, phân tách bởi dấu phẩy
  client_secret: "${GOOGLE_CLIENT_SECRET}"
  redirect_uri: "${GOOGLE_REDIRECT_URI}"     # callback backend, đăng ký với Google
  state_ttl_minutes: 10
```

Struct `Google` (mapstructure) + override trong `loader.go`. Dùng `splitFrontEndOrigins`-style helper để tách
`client_ids` thành `[]string` dẫn xuất (nhất quán với pattern `FrontEndOrigins`).

### Biến môi trường

| Biến | Bắt buộc | Ghi chú |
|---|---|---|
| `GOOGLE_CLIENT_IDS` | ✅ | Danh sách client ID (web/iOS/Android), phẩy phân tách |
| `GOOGLE_CLIENT_SECRET` | Chỉ web redirect | Dùng cho Authorization Code flow |
| `GOOGLE_REDIRECT_URI` | ✅ | Callback backend, đăng ký với Google Cloud Console |
| `GOOGLE_STATE_TTL_MINUTES` | ❌ | Default 10 trong `configs/*.yaml` |

**Vị trí trong file env**: chèn section mới `8. GOOGLE OAUTH (Social Login)` ngay sau mục `7. JWT` trong
`.env.example`, `.env.production.example`, `.env.aws.example` (dồn số các mục sau), **không** append cuối file.

## 5. Quy tắc validate (tóm tắt)

| Trường | Quy tắc |
|---|---|
| `email` (Google) | lowercase + trim; unique không phân biệt hoa thường |
| `provider_id` | non-empty với Google; unique `(provider, provider_id)` khi khác NULL |
| `username` | `^[a-zA-Z0-9._]{4,20}$`, không `[_.]` liên tiếp, không `.`/`_` đầu/cuối; tự sinh khi đăng nhập Google |
| `redirectUrl` | parse URL; scheme `https` (dev: `http://localhost[:port]`); host ∈ `front_end_origins` (exact, không subdomain) |
| `aud` (ID token) | thuộc `google.client_ids` |
| `email_verified` (Google claim) | phải `true` |
| `password_hash` | nullable; verify bỏ qua khi NULL/rỗng |

## 6. Chuyển trạng thái (tài khoản Google)

```text
[chưa tồn tại] --google login (email verified)--> [Active, provider=google, email_verified=true, password=NULL]
[Active, provider=local, email_verified=true] --google login--> [provider=google, provider_id=sub] (giữ password)
[Active, provider=local, email_verified=false] --google login--> [từ chối: đăng nhập mật khẩu]
[Active, provider=google, sub khác] --google login--> [từ chối: đã liên kết Google khác]
[Inactive/Suspended/Deleted/blacklist] --google login--> [từ chối: account disabled]
[Active, provider=google, password=NULL] --forgot/change password--> [password set]
```
