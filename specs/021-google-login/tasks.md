---

description: "Task list for Google Login implementation"
---

# Tasks: Đăng nhập Google (Google Login)

**Input**: Design documents from `/specs/021-google-login/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/google-login-api.md, quickstart.md

**Tests**: Có bao gồm task test — theo quy ước dự án (`docs/development/local-setup.md` §12: unit test ở usecase/repository, `make test-race`).

**Organization**: Task nhóm theo user story để có thể implement/kiểm thử độc lập.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Có thể chạy song song (khác file, không phụ thuộc)
- **[Story]**: User story tương ứng (US1, US2, US3, US4)
- Mọi task kèm đường dẫn file cụ thể

## Path Conventions

Multi-module Go workspace. Feature nằm trong `services/account` (module `identity`) và `shared/config`.

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Cấu hình, dependency và kiểu dùng chung.

- [x] T001 Thêm block cấu hình Google vào `shared/config/config.go` (struct `Google{ClientIDs []string, ClientSecret, RedirectURI, StateTTLMinutes}`), override env trong `shared/config/loader.go` (kèm helper tách `client_ids` thành slice), validate trong `shared/config/validation.go`, và khai báo placeholder trong `configs/config.yaml` + `configs/config.production.yaml`. Đồng thời chèn section mới `8. GOOGLE OAUTH (Social Login)` gồm `GOOGLE_CLIENT_IDS`, `GOOGLE_CLIENT_SECRET`, `GOOGLE_REDIRECT_URI`, `GOOGLE_STATE_TTL_MINUTES` **ngay sau mục `7. JWT`** trong `.env.example`, `.env.production.example`, `.env.aws.example` — dồn số các mục sau (Email→9, PayOS→10, Cloudinary→11, AI→12; Ngrok→13 chỉ ở `.env.example`), KHÔNG append vào cuối file
- [x] T002 Thêm dependency `golang.org/x/oauth2` và `google.golang.org/api/idtoken` vào `services/account/go.mod` rồi chạy `go mod tidy` trong `services/account`
- [x] T003 [P] Thêm hằng số provider trong `services/account/internal/modules/identity/domain/constants/authprovider/authprovider.go` (`local`, `google`)
- [x] T004 [P] Thêm DTO Google trong `services/account/dto/identity/auth.go`: `GoogleLoginReq` (body) và `GoogleStartReq` bind query `redirectUrl` (`form:"redirectUrl"`, required) cho `GET /auth/google`
- [x] T005 [P] Thêm error constructors trong `services/account/internal/modules/identity/application/errors/errors.go` (token không hợp lệ, email chưa xác thực, email đã đăng ký dùng mật khẩu, đã liên kết Google khác, cấu hình Google thiếu)

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Hạ tầng bắt buộc cho mọi user story.

**⚠️ CRITICAL**: Không bắt đầu user story nào trước khi xong phase này.

- [x] T006 Tạo migration `services/account/database/migrations/20260925090000_add_google_auth_to_users.sql` (thêm `provider`, `provider_id`, `email_verified`; `password_hash` nullable; unique partial index `(provider, provider_id)`; backfill `email_verified=true`)
- [x] T007 Cập nhật entity `services/account/internal/modules/identity/domain/entities/user.go` (`Provider`, `ProviderID *string`, `EmailVerified bool`, `PasswordHash *string`) và điều chỉnh `user_behavior.go` nếu cần
- [x] T008 Thêm `GetByProviderID(ctx, provider, providerID)` vào `services/account/internal/modules/identity/domain/repositories/user_repo.go` và implement trong `.../infrastructure/persistence/user_repo_impl.go`
- [x] T009 Định nghĩa port `IGoogleTokenVerifier` trong `services/account/internal/modules/identity/application/interface/security/google_token_verifier.go` và `IOAuthStateStore` trong `.../application/interface/identity/oauth_state_store.go`
- [x] T010 [P] Implement `GoogleTokenVerifier` (verify chữ ký/`aud` trong danh sách/`iss`/`exp`/`email_verified`) trong `services/account/internal/modules/identity/infrastructure/security/google_token_verifier.go`
- [x] T011 [P] Implement `RedisOAuthStateStore` trong `services/account/internal/modules/identity/infrastructure/caching/redis_oauth_state.go` và value object `OAuthState` trong `.../application/vo/oauth_state.go`
- [x] T012 Tách helper `issueSession(ctx, user, device)` trong `services/account/internal/modules/identity/application/usecase/session_uc.go`, cho `Login` dùng lại
- [x] T013 Cập nhật `ConfirmRegisterOtp` trong `services/account/internal/modules/identity/application/usecase/register_uc.go`: set `Provider=local`, `EmailVerified=true`; map lỗi unique email sang `ErrEmailExists`
- [x] T014 [P] Cập nhật `CreateUserByAdmin` trong `services/account/internal/modules/identity/application/usecase/admin_uc.go`: set `Provider=local`, `EmailVerified=false`
- [x] T015 Đăng ký provider mới vào Wire: `.../identity/application/provider.go`, `.../identity/infrastructure/provider.go`, rồi chạy `make wire` để cập nhật `services/account/internal/di/wire_gen.go`

**Checkpoint**: Hạ tầng sẵn sàng — bắt đầu user story.

---

## Phase 3: User Story 1 - Đăng nhập/đăng ký bằng Google ID token (Priority: P1) 🎯 MVP

**Goal**: `POST /api/v1/auth/google` xác thực ID token, tạo tài khoản mới hoặc đăng nhập tài khoản Google đã tồn tại.

**Independent Test**: Gửi ID token hợp lệ của người dùng mới và người dùng đã tồn tại; cả hai nhận phiên hợp lệ; ID token `email_verified=false` bị từ chối; email trùng tài khoản local trả lỗi rõ ràng (auto-link kiểm ở US2).

### Implementation for User Story 1

- [x] T016 [US1] Thêm interface `IGoogleAuthUseCase` (+ method `LoginWithIDToken`) trong `services/account/internal/modules/identity/application/interface/usecase/google_auth_uc.go`
- [x] T017 [US1] Implement `GoogleAuthUseCase` trong `services/account/internal/modules/identity/application/usecase/google_auth_uc.go`: verify token, lookup theo provider/sub, tạo mới nếu chưa có, check eligibility + blacklist, gọi `issueSession`. Nếu không thấy theo `sub` thì tra email: nếu email thuộc tài khoản local đã verify → trả lỗi `ErrEmailRegisteredLoginPassword` (auto-link nằm ở US2); nếu email chưa tồn tại → tạo mới (đảm bảo US1 không lỗi unique-key khi chỉ implement MVP)
- [x] T018 [P] [US1] Implement helper sinh `username` từ tiền tố email (chuẩn hóa theo `IsUserName`, xử lý trùng) trong `services/account/internal/modules/identity/application/usecase/google_username_helper.go`
- [x] T019 [US1] Áp dụng UoW khi tạo user + refresh token; bắt lỗi unique-key (`lower(email)`, `(provider, provider_id)`), tra lại theo `sub`/email và cấp phiên idempotent (chống race first-login) trong `google_auth_uc.go`
- [x] T020 [US1] Thêm handler `GoogleLogin` (`POST`) + swagger annotation trong `services/account/internal/modules/identity/presentation/handler/auth_handler.go` (set cookie)
- [x] T021 [US1] Thêm route `POST /auth/google` trong `services/account/internal/modules/identity/presentation/router/router.go` và bind usecase vào `.../presentation/provider.go`
- [x] T022 [US1] Unit test `google_auth_uc_test.go`: tạo mới, đăng nhập lại, `email_verified=false`, tài khoản inactive/bị khoá, token không hợp lệ
- [x] T023 [US1] Unit test sinh username (`google_username_helper_test.go`): tiền tố dài/ngắn/ký tự đặc biệt/trùng

**Checkpoint**: US1 hoạt động và kiểm thử độc lập (MVP).

---

## Phase 4: User Story 2 - Tự động liên kết tài khoản email/mật khẩu (Priority: P1)

**Goal**: Google login với email đã xác thực trùng tài khoản local đã verify → auto-link vào tài khoản cũ.

**Independent Test**: Tạo tài khoản local đã verify, rồi Google login cùng email → chỉ còn một tài khoản, cả hai cách đăng nhập đều hoạt động.

### Implementation for User Story 2

- [x] T024 [US2] Bổ sung nhánh resolve theo email trong `services/account/internal/modules/identity/application/usecase/google_auth_uc.go`: auto-link khi `provider=local` & `email_verified=true`; từ chối khi `email_verified=false`; từ chối khi đã gắn sub khác
- [x] T025 [US2] Chỉnh nhánh auto-link dùng transaction: gán `provider`/`provider_id` cho tài khoản local đã verify, bắt lỗi unique và cấp phiên idempotent trong `google_auth_uc.go`
- [x] T026 [US2] Đảm bảo so sánh email lowercase ở mọi nhánh (dùng `GetByEmail` sẵn có) trong `google_auth_uc.go`
- [x] T027 [US2] Unit test `google_auth_uc_test.go`: auto-link thành công, account admin-tạo bị từ chối, email đã gắn sub khác bị từ chối, email khác hoa thường coi là cùng, race tạo trùng

**Checkpoint**: US1 + US2 cùng hoạt động độc lập.

---

## Phase 5: User Story 3 - Đăng nhập Google trên web bằng redirect (Priority: P2)

**Goal**: Luồng Authorization Code: FE → `GET /auth/google?redirectUrl=...` → Google → callback backend → set cookie → quay về FE.

**Independent Test**: Bắt đầu luồng trong trình duyệt, hoàn tất consent, quay về FE ở trạng thái đã đăng nhập; `redirectUrl` giả mạo bị 400; huỷ consent quay về kèm `?error=`.

### Implementation for User Story 3

- [x] T028 [P] [US3] Implement helper validate `redirectUrl` (parse URL, exact host/scheme, localhost dev, chuẩn hóa port/dấu chấm cuối) trong `services/account/internal/modules/identity/application/usecase/google_redirect_helper.go`
- [x] T029 [US3] Thêm method usecase `StartGoogleRedirect` và `HandleGoogleCallback` trong `services/account/internal/modules/identity/application/usecase/google_auth_uc.go` (dùng `IOAuthStateStore`, PKCE, đổi code, gọi resolve/link dùng chung)
- [x] T030 [US3] Thêm handler `GoogleStart` (`GET`, trả 302 Google) và `GoogleCallback` (`GET`, set cookie + 302 về `redirectUrl?error=...` khi lỗi) trong `services/account/internal/modules/identity/presentation/handler/auth_handler.go`, kèm swagger annotation cho cả 2 endpoint
- [x] T031 [US3] Thêm route `GET /auth/google` và `GET /auth/google/callback` trong `services/account/internal/modules/identity/presentation/router/router.go`
- [x] T032 [US3] Unit test `google_redirect_helper_test.go`: host hợp lệ (path/query), `.hehe`, `origin@evil.com`, subdomain, `http` ngoài localhost, `localhost` dev
- [x] T033 [US3] Unit test state store + callback (`google_auth_uc_test.go`): state sai/hết hạn, huỷ consent, exchange fail, đích redirect lấy từ state

**Checkpoint**: Cả 3 luồng (mobile/web) hoạt động độc lập.

---

## Phase 6: User Story 4 - Thiết lập mật khẩu cho tài khoản Google-only (Priority: P3)

**Goal**: Tài khoản chỉ có Google có thể đặt mật khẩu lần đầu; đăng nhập mật khẩu khi chưa có mật khẩu bị từ chối hợp lý.

**Independent Test**: Tạo tài khoản Google-only → quên mật khẩu → đặt mật khẩu → đăng nhập email+mật khẩu thành công.

### Implementation for User Story 4

- [x] T034 [US4] Thêm guard trong `services/account/internal/modules/identity/application/usecase/session_uc.go::Login`: nếu `PasswordHash == nil || *PasswordHash == ""` → `ErrPasswordNotSet` (thông báo hướng dẫn Google / Quên mật khẩu); sai mật khẩu thì giữ `ErrInvalidCredentials`
- [x] T035 [US4] Cho phép đổi mật khẩu lần đầu trong `services/account/internal/modules/identity/application/usecase/profile_uc.go::ChangePassword` (bỏ qua verify `OldPassword` khi hash NULL)
- [x] T036 [US4] Đảm bảo `ResetPassword` trong `services/account/internal/modules/identity/application/usecase/password_recovery_uc.go` hoạt động với tài khoản hash NULL
- [x] T037 [US4] Unit test: login khi chưa có mật khẩu, change-password lần đầu, reset-password cho Google-only (`session_uc_test.go`, `password_recovery_uc_test.go`)

**Checkpoint**: Toàn bộ user story hoạt động độc lập.

---

## Phase 7: Polish & Cross-Cutting Concerns

**Purpose**: Tài liệu, swagger, kiểm chứng end-to-end.

- [x] T038 [P] Cập nhật `docs/api/identity/auth-api.md` (luồng Google) và `docs/api/identity/constants.md` (provider)
- [x] T039 [P] Đồng bộ `docs/api/identity/google-login-frontend-guide.md` với hợp đồng cuối cùng (web + mobile)
- [x] T040 [P] Regenerate swagger cho service account (`make swagger svc=account` hoặc target tương đương)
- [x] T041 Thêm test tích hợp migration cho cột mới trong `services/account/database/migrations/migration_integration_test.go`
- [x] T042 Chạy `make wire && make lint && make test-race` toàn workspace
- [ ] T043 Kiểm chứng theo `specs/021-google-login/quickstart.md` (các kịch bản backend/web/mật khẩu), bao gồm đo thời gian hoàn tất đăng nhập < 15s (SC-001)
- [x] T044 [P] Cập nhật `docs/operations/shared-reference.md` §2.3: bổ sung 4 biến `GOOGLE_*` và cập nhật bảng profile/nguồn env cho cả VPS + box ngoài
- [x] T045 [P] Xác nhận/điều chỉnh rate limit cho các endpoint Google dùng shield `/auth` hiện có (`shared/presentation/middleware/ratelimit.go`): key theo IP + định danh cho `GET`/callback, bổ sung test, ghi chú trong `contracts/google-login-api.md` (FR-022)
- [x] T046 [P] Ghi chú chính sách anonymize email khi xoá tài khoản (FR-021, deferred) vào tài liệu liên quan; không implement cho tới khi có luồng xoá

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: Không phụ thuộc — bắt đầu ngay
- **Foundational (Phase 2)**: Phụ thuộc Setup — CHẶN mọi user story
- **User Stories (Phase 3–6)**: Phụ thuộc Foundational
- **Polish (Phase 7)**: Phụ thuộc các user story mong muốn hoàn tất

### User Story Dependencies

- **US1 (P1)**: Sau Foundational; MVP, không phụ thuộc story khác
- **US2 (P1)**: Sau Foundational; mở rộng nhánh resolve của US1 (T024 nối tiếp T017), vẫn kiểm thử độc lập
- **US3 (P2)**: Sau Foundational; dùng lại resolve/issueSession của US1/US2
- **US4 (P3)**: Sau Foundational; độc lập với US3

### Within Each User Story

- Helper/model trước usecase; usecase trước handler/route; core trước integration
- Test viết cùng task tính năng tương ứng

### Parallel Opportunities

- Setup: T003, T004, T005 song song
- Foundational: T010, T011, T014 song song (khác file)
- US1: T018 song song; US3: T028 song song
- Polish: T038, T039, T040, T044, T045, T046 song song

---

## Parallel Example: User Story 1

```bash
# Có thể chạy song song (khác file):
Task: "T018 Implement username helper in .../application/usecase/google_username_helper.go"
Task: "T016 Add IGoogleAuthUseCase in .../application/interface/usecase/google_auth_uc.go"
```

---

## Implementation Strategy

### MVP First (User Story 1)

1. Phase 1 Setup → Phase 2 Foundational (bắt buộc)
2. Phase 3 US1 (mobile ID token login/register)
3. **STOP & VALIDATE**: test US1 độc lập, deploy/demo

### Incremental Delivery

1. Setup + Foundational → nền tảng sẵn sàng
2. US1 → test → demo (MVP)
3. US2 → test → demo (auto-link)
4. US3 → test → demo (web redirect)
5. US4 → test → demo (set password)

---

## Notes

- [P] = khác file, không phụ thuộc
- [Story] map task ↔ user story để truy vết
- FR-019 (không đồng bộ avatar Google) được thoả bằng việc không implement — không cần task
- T022/T027/T033 cùng file `google_auth_uc_test.go` nên chạy tuần tự theo phase (không đánh dấu [P])
- FR-021 là deferred, theo dõi ở T046
- Không commit tự động; mỗi task/nhóm logic có thể tạo commit khi dev cho phép
- Verify `make wire && make lint && make test-race` trước khi đề xuất commit
- Tránh task mơ hồ hoặc trùng file gây xung đột

---

## Phase 8: Convergence

- [x] T047 [US1] Phân biệt lỗi trùng email khi tạo user trong `services/account/internal/modules/identity/application/usecase/register_uc.go::ConfirmRegisterOtp` và map sang `ErrEmailExists` thay vì `ErrAccountCreationFailed` (xử lý race với Google login) per FR-018/T013 (partial)
- [x] T048 [US3] Bổ sung unit test cho luồng web trong `services/account/internal/modules/identity/application/usecase/google_auth_uc_test.go`: `StartRedirect`/`HandleCallback` (state sai/hết hạn, huỷ đồng ý, exchange thất bại, đích redirect lấy từ state) per T033 (partial)
- [x] T049 [US3] Bổ sung test/điều chỉnh rate-limit cho các endpoint Google (shield `/auth`) và xác nhận key theo IP cho `GET`/callback trong `shared/presentation/middleware/ratelimit.go` (+ test) per FR-022/T045 (partial)
- [x] T050 [US3] Sửa `docs/api/identity/google-login-frontend-guide.md` §4: bỏ `state_invalid` khỏi bảng `?error=` vì state sai/hết hạn trả JSON 400 (không redirect) per FR-023 (contradicts)
