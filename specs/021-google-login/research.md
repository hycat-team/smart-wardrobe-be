# Phase 0 — Research: Đăng nhập Google

Tài liệu ghi lại các quyết định thiết kế (Decision / Rationale / Alternatives) đã chốt, dựa trên
spec `spec.md` và ràng buộc trong `.specify/memory/constitution.md`.

---

## R1. Cách xác minh Google ID token

**Decision**: Đặt port `IGoogleTokenVerifier` ở `application/interface/security/`, adapter
`infrastructure/security/google_token_verifier.go` dùng `google.golang.org/api/idtoken` để xác minh
ID token (chữ ký qua JWKS của Google, `iss`, `exp`), sau đó tự kiểm tra `aud` nằm trong **danh sách
client ID** cấu hình và `email_verified == true`. Dùng `golang.org/x/oauth2` (với `google.Endpoint`)
để đổi authorization code ở luồng web.

**Rationale**: Thư viện chính thức xử lý đúng JWKS/rotation/clock; giữ logic nghiệp vụ (audience list,
`email_verified`) trong adapter của mình. Đặt sau interface để có thể thay bằng OIDC verifier khác mà
không đổi usecase (tuân §IV/§VIII).

**Alternatives considered**:
- `github.com/coreos/go-oidc/v3` (generic OIDC) — tốt nhưng thêm dependency lớn, không cần thiết khi chỉ có Google.
- Tự fetch JWKS + verify bằng `golang-jwt/v4` — nhiều code, dễ sai cache key rotation.

---

## R2. Lưu `state` chống CSRF của redirect flow

**Decision**: Port `IOAuthStateStore` (application interface), adapter Redis dùng `ICacheService` sẵn có.
Key `oauth:google:state:<state>` → JSON `{ "redirectUrl": <đã validate>, "codeVerifier": <PKCE verifier> }`,
TTL ngắn (mặc định 10 phút, cấu hình được). `state` sinh ngẫu nhiên bằng `crypto/rand`; đọc xong **xóa ngay**
(dùng một lần). Không lưu `state` trong cookie.

**Rationale**: Cookie `SameSite=Strict` hiện tại không gửi kèm khi Google redirect cross-site về backend.
Lưu server-side vừa tránh vấn đề cookie, vừa cho phép lưu `codeVerifier` (PKCE) và `redirectUrl` đã validate
mà không tin dữ liệu client gửi lại ở callback.

**Alternatives considered**:
- State ký HMAC chứa `redirectUrl` (stateless) — không lưu được `codeVerifier`, khó thu hồi/dùng-một-lần.
- Cookie state `SameSite=Lax` — nới lỏng bảo mật, phụ thuộc hành vi trình duyệt.

---

## R3. Thuật toán resolve & auto-link tài khoản

**Decision** (áp dụng chung cho cả web và mobile):
1. Xác minh ID token → `{sub, emailLower, emailVerified}`.
2. Nếu `emailVerified == false` → từ chối (`ErrGoogleEmailUnverified`).
3. Tìm theo định danh nhà cung cấp (`GetByProviderID('google', sub)`):
   - Tìm thấy → dùng tài khoản đó (nếu eligible).
4. Không tìm thấy → tìm theo `emailLower`:
   - **Không có** → tạo tài khoản mới: `provider='google'`, `provider_id=sub`, `email_verified=true`,
     `username` sinh tự động, `role_slug='user'`, `status=Active`, `password_hash=NULL`.
   - **Có và `provider='local'` + `email_verified=true`** → auto-link: gán `provider='google'`,
     `provider_id=sub` (giữ nguyên mật khẩu cũ) → cấp phiên.
   - **Có và `provider='local'` + `email_verified=false`** (vd tài khoản admin tạo) → từ chối,
     hướng dẫn đăng nhập bằng mật khẩu (`ErrEmailRegisteredLoginPassword`).
   - **Có và `provider='google'` + `provider_id != sub`** → từ chối (`ErrAccountLinkedOtherGoogle`).
5. Kiểm tra eligible giống đăng nhập thường: `ensureUserEligibleForLogin` + kiểm tra blacklist.

**Rationale**: Khớp các quyết định #2/#3 trong spec. Khóa định danh theo `sub` (ổn định), không theo email
(email có thể đổi chủ). Chỉ auto-link khi cả hai phía đã xác thực email.

**Alternatives considered**: Auto-link mọi trường hợp theo email (rủi ro takeover với account admin-tạo/chưa
verify); từ chối mọi trùng email (UX kém, dễ mất tài khoản).

---

## R4. Sinh `username` cho tài khoản Google

**Decision**: Lấy **tiền tố trước `@`** của email Google, rồi chuẩn hóa để thỏa validator `username` hiện có
(`shared/pkg/utils/validation/helper.go::IsUserName`): `^[a-zA-Z0-9._]{4,20}$`, không có `[_.]` liên tiếp,
không bắt đầu/kết thúc bằng `.` hoặc `_`.
Quy trình: lowercase → thay ký tự không hợp lệ → gộp `[_.]` liên tiếp → cắt `[._]` đầu/cuối → nếu < 4 ký tự
thì đệm/nối hậu tố → cắt còn 20. Khi trùng, thêm hậu tố `_<4 hex>` (giữ ≤ 20), thử tối đa N lần; cuối cùng dùng
`user_<8 hex>`. Unique index `lower(username)` là chốt cuối.

**Rationale**: Đúng quyết định #5 của spec; tái dùng luật validate sẵn có để không phát sinh luật mới.

**Alternatives considered**: Dùng toàn bộ email làm username (vượt 20 ký tự, chứa `@`); cho user tự đặt ngay
(thêm bước, trái mục tiêu giảm ma sát).

---

## R5. `password_hash` nullable & thiết lập mật khẩu lần đầu

**Decision**: Cho `password_hash` nullable. Đăng nhập mật khẩu tách 2 nhánh: nếu hash rỗng/NULL →
trả `ErrPasswordNotSet` (thông báo hướng dẫn đăng nhập bằng Google hoặc dùng Quên mật khẩu để thiết lập
mật khẩu, không gọi verify); nếu có hash nhưng sai → `ErrInvalidCredentials` (không gọi nói rõ tài khoản
nào tồn tại). Cho phép thiết lập mật khẩu lần đầu qua **quên mật khẩu** (đã có sẵn,
chỉ cần đảm bảo hoạt động khi hash NULL) và qua **đổi mật khẩu** (`profile_uc.go`): nếu tài khoản chưa có mật khẩu
thì bỏ qua bước xác minh `OldPassword` (request đã xác thực bằng JWT).

**Rationale**: Đúng quyết định #4. Tái dùng luồng recovery sẵn có, giảm code mới. Báo "sai mật khẩu" cho
một tài khoản chưa từng có mật khẩu là thông báo sai nguyên nhân, dẫn người dùng đi sai hướng. Mức lộ
thông tin chấp nhận được: nhánh riêng chỉ tiết lộ "tài khoản này chưa thiết lập mật khẩu", không nói gì
về mật khẩu hay trạng thái tài khoản.

**Alternatives considered**: Không cho set mật khẩu (kém linh hoạt); thêm endpoint `set-password` riêng (dư thừa).

---

## R6. Validate `redirectUrl` (chống open redirect)

**Decision**: Parse URL và so **scheme + host chính xác** với một trong `server.front_end_origins`:
- Bắt buộc scheme `https`; ngoại lệ duy nhất `http://localhost[:port]` khi môi trường dev.
- So `u.Hostname()` (đã lowercase, bỏ dấu chấm cuối; port xét riêng theo cấu hình) với hostname hợp lệ; **không cho subdomain**.
- Cho phép path/query/fragment phía sau.
- **Không** dùng `HasPrefix`/`HasSuffix` trên chuỗi thô (chống userinfo trick `https://origin@evil.com` và `.hehe`).

**Rationale**: Quyết định Q1 trong `## Clarifications`. So host sau khi parse chống được mọi biến thể chuỗi;
`url.Parse` tách `User`/`Host` đúng.

**Alternatives considered**: So cả chuỗi origin (dính suffix); whitelist path cụ thể (quá chặt, FE cần nhiều path).

---

## R7. Hành vi luồng lỗi web

**Decision**: Validate `redirectUrl` fail → HTTP 400 JSON (không redirect). Mọi lỗi sau khi đã có đích hợp lệ
trong `state` (user hủy, state sai/hết hạn, code hết hạn, exchange fail, tài khoản không hợp lệ) → `302` về đích
kèm `?error=<code>`; không cấp phiên.

**Rationale**: Quyết định Q3. Không redirect khi chưa có đích tin cậy (chống open redirect), nhưng giữ UX tốt
sau khi đã có đích.

**Alternatives considered**: Luôn JSON (user mắc kẹt ở trang API); luôn redirect về origin mặc định (che giấu đích).

---

## R8. Chống race / tạo trùng

**Decision**: Dựa vào ràng buộc DB làm chốt: unique `lower(email)` và unique `(provider, provider_id)`.
Khi tạo tài khoản gặp lỗi unique, bắt lỗi, tra lại theo `sub`/email và cấp phiên (idempotent). Thao tác tạo user
+ refresh token nằm trong `uow.Execute`. `ConfirmRegisterOtp` map lỗi unique email thành `ErrEmailExists` thay vì
lỗi nội bộ chung.

**Rationale**: Đúng FR-018; transaction + unique index là cách chắc chắn nhất.

**Alternatives considered**: Khóa mềm bằng Redis distributed lock (thêm phức tạp, không cần thiết).

---

## R9. Tái sử dụng phiên

**Decision**: Tách hàm `issueSession(ctx, user, device)` từ `SessionUseCase.Login` để `GoogleAuthUseCase` dùng
chung (tạo access + refresh JWT HS256, lưu `identity.refresh_tokens`). Giữ nguyên rotation, logout, blacklist.

**Rationale**: FR-011; đảm bảo Google login và password login cho ra phiên giống hệt, tránh nhánh logic song song.

**Alternatives considered**: Viết riêng luồng phát hành token cho Google (trùng lặp, dễ lệch bảo mật).

---

## R10. Client ID / audience

**Decision**: Cấu hình **danh sách** client ID hợp lệ (web/iOS/Android) qua env; backend chấp nhận nếu `aud`
thuộc danh sách. Google config gồm: `client_ids` (list), `client_secret` (cho code exchange), `redirect_uri`
(callback backend, phải đăng ký với Google), `state_ttl_minutes`.

**Rationale**: Quyết định Q4. Đa nền tảng, không sửa code khi thêm client mới.

**Alternatives considered**: Một client ID (không hỗ trợ được nhiều nền tảng); hardcode (trái §VIII).

---

## R11. Mô hình tích hợp frontend

**Decision**: Web dùng **Google Identity Services** có thể ở 2 dạng: (a) **Authorization Code + redirect** qua
`GET /auth/google` (spec chốt), hoặc (b) GIS lấy ID token rồi `POST /auth/google` (tuỳ chọn, giống mobile).
Mobile dùng Google Sign-In SDK lấy ID token rồi `POST /auth/google`. Sau đăng nhập:
- Mobile: lưu token trả về JSON, gọi `/me` (Bearer).
- Web: cookie HttpOnly đã set từ response; FE gọi `/me` kèm credentials.
Tài liệu FE bắt buộc: `docs/api/identity/google-login-frontend-guide.md` (web + mobile).

**Rationale**: Yêu cầu bổ sung của dev; đảm bảo FE có hợp đồng rõ ràng. Tài liệu nằm cạnh `auth-api.md`.

**Alternatives considered**: Chỉ viết trong `contracts/` (dev FE khó tìm, không theo cấu trúc docs API hiện tại).

---

## R12. Ngoài phạm vi (đã xác nhận)

- Không đồng bộ avatar Google (quyết định #7).
- Không endpoint link/unlink thủ công (quyết định #11).
- Không thêm RoleSlug (quyết định #12 / constitution).
- Chính sách anonymize email khi xoá tài khoản (quyết định #9) — ghi nhận, chưa có luồng xoá để gắn vào.

## R13. Email không phân biệt hoa thường

**Decision**: Chuẩn hóa `strings.ToLower(strings.TrimSpace(email))` trước mọi so sánh/tạo. DB đã có unique index
`lower(email)` (migration `20260621084228_case_insensitive_users.sql`).

**Rationale**: Quyết định #6; index hiện có đảm bảo tính duy nhất không phân biệt hoa thường.
