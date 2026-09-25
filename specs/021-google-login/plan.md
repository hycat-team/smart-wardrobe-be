# Implementation Plan: Đăng nhập Google (Google Login)

**Branch**: `021-google-login` | **Date**: 2026-09-25 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `/specs/021-google-login/spec.md`

## Summary

Bổ sung đăng nhập/đăng ký bằng Google vào module `identity` (Account service) theo hai luồng:
**web** dùng Authorization Code + redirect (FE truyền `redirectUrl`, backend validate rồi nhúng vào
`state`), **mobile/app** dùng Google ID token. Hệ thống tự động liên kết (auto-link) danh tính Google
với tài khoản hiện hữu khi email Google đã xác thực trùng email đã xác thực của tài khoản; từ chối các
trường hợp chưa xác thực email, tài khoản bị vô hiệu hóa, hoặc email đã gắn danh tính Google khác.
Phiên đăng nhập dùng lại nguyên cơ chế access/refresh token hiện có. Cần migration bổ sung
`provider`, `provider_id`, `email_verified` và cho `password_hash` nullable; bổ sung cấu hình Google
client. Kèm **deliverable tài liệu hướng dẫn frontend** (web + mobile) tại `docs/api/identity/`.

## Technical Context

**Language/Version**: Go 1.26 (multi-module workspace `go.work`)

**Primary Dependencies**: Gin, `github.com/golang-jwt/jwt/v4` (HS256 session hiện hữu), GORM, Google Wire.
Bổ sung mới: `golang.org/x/oauth2` (+ `google.golang.org/api/idtoken` để xác minh ID token) cho luồng Google; Redis qua `ICacheService` shared cho `state`.

**Storage**: PostgreSQL (schema `identity`), Redis (state chống CSRF của redirect flow)

**Testing**: `go test`, `testify`, GoMock/kế thừa fake hiện có, `migration_integration_test.go` cho migration

**Target Platform**: Linux server — Account service (`services/account`)

**Project Type**: web-service (backend) + tài liệu hợp đồng tích hợp frontend

**Performance Goals**: Hoàn tất đăng nhập Google < 15 giây (SC-001); không thêm độ trễ cho luồng đăng nhập mật khẩu

**Constraints**: tái sử dụng phiên HS256 hiện có; cookie `SameSite=Strict`; không thêm RoleSlug; audience/client ID cấu hình qua env; không dùng vendor-exclusive (Firebase/Supabase Auth)

**Scale/Scope**: ~3 endpoint mới (`POST /auth/google`, `GET /auth/google`, `GET /auth/google/callback`), 1 migration, 2 file config YAML + 3 file env + 1 tài liệu vận hành, 1 tài liệu FE; phục vụ toàn bộ người dùng hiện hữu

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Gate | Yêu cầu | Trạng thái | Ghi chú |
|------|---------|------------|---------|
| §I Independently Deployable | Thay đổi nằm trong Account service, module `identity` | PASS | Không đụng Commerce/AI |
| §II Module Boundaries | Chỉ dùng contract/DTO/constants hợp lệ; không import entity module khác | PASS | Chỉ nội bộ `identity` |
| §III Event-Driven | Side-effect ghi ưu tiên event; outbox khi cần | PASS | Google login chỉ tạo user/refresh token; không cần event mới |
| §IV Clean Architecture | presentation → application → domain ← infrastructure | PASS | Verifier là interface `application/interface/security`, impl ở `infrastructure/security` |
| §V Transaction & Error | `apperror` + UoW cho thao tác đa repository | PASS | Create user + refresh token dùng UoW khi cần |
| §VII Design Patterns | Tuân `design-pattern.md` | PASS | Handler thin, `WrapHandler`, DTO camelCase |
| §VIII Portability / No Lock-in | Không phụ thuộc vendor-exclusive; config-driven | PASS | Google OAuth là IdP chuẩn OIDC, wrap sau interface riêng; client ID/secret qua env |
| Tech Stack | Goose migration, Wire, Swagger, message tiếng Việt | PASS | Migration per-service; swagger cập nhật |
| RBAC | Chỉ 2 RoleSlug | PASS | Tài khoản Google luôn `user` |

**Kết luận**: Không có vi phạm; không cần mục Complexity Tracking.

## Project Structure

### Documentation (this feature)

```text
specs/021-google-login/
├── plan.md              # This file (/speckit-plan command output)
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
├── contracts/           # Phase 1 output
│   └── google-login-api.md
├── checklists/
│   └── requirements.md
└── tasks.md             # Phase 2 output (/speckit-tasks - NOT created here)
```

**Deliverable bổ sung (theo yêu cầu dev)**: tài liệu hướng dẫn frontend cho cả **web** và **mobile**
được tạo tại `docs/api/identity/google-login-frontend-guide.md` và phải được giữ đồng bộ với
`contracts/google-login-api.md`. File này là outcome bắt buộc của quá trình thực thi.

### Source Code (repository root)

```text
services/account/
├── database/migrations/
│   └── <ts>_add_google_auth_to_users.sql        # migration mới
├── dto/identity/auth.go                          # thêm DTO Google (GoogleLoginReq)
└── internal/modules/identity/
    ├── application/
    │   ├── errors/errors.go                      # thêm lỗi Google/liên kết
    │   ├── interface/
    │   │   ├── security/google_token_verifier.go # interface mới (port)
    │   │   ├── identity/oauth_state_store.go     # port state store (redirect flow)
    │   │   └── usecase/google_auth_uc.go         # interface usecase mới
    │   ├── usecase/
    │   │   ├── google_auth_uc.go                 # usecase mới
    │   │   ├── google_username_helper.go         # sinh username từ tiền tố email
    │   │   ├── google_redirect_helper.go         # validate redirectUrl
    │   │   ├── session_uc.go                     # tách issueSession dùng chung
    │   │   ├── profile_uc.go                     # nhánh set-password lần đầu
    │   │   └── register_uc.go                    # set email_verified + map lỗi trùng
    │   ├── provider.go                           # bind usecase mới
    │   └── vo/oauth_state.go                     # value object state (redirectUrl)
    ├── domain/
    │   ├── constants/authprovider/authprovider.go # hằng số provider (local/google)
    │   ├── entities/user.go                      # provider/provider_id/email_verified, password nullable
    │   └── repositories/user_repo.go             # GetByProviderID (nếu cần)
    ├── infrastructure/
    │   ├── security/google_token_verifier.go     # adapter xác minh ID token
    │   ├── caching/redis_oauth_state.go          # lưu/đọc state (Redis)
    │   └── provider.go                           # bind adapter
    └── presentation/
        ├── handler/auth_handler.go               # handlers Google
        ├── router/router.go                      # route mới
        └── provider.go                           # wire
services/account/internal/di/wire_gen.go          # regenerate sau khi thêm provider
shared/config/config.go                           # struct Google client (client ID list, secret, redirect)
shared/config/loader.go                           # env overrides cho Google config

docs/api/identity/
├── google-login-frontend-guide.md                # DELIVERABLE tài liệu FE (web + mobile)
└── auth-api.md                                   # cập nhật tham chiếu luồng Google

configs/config.yaml, configs/config.production.yaml  # thêm block google
.env.example, .env.production.example, .env.aws.example  # section "8. GOOGLE OAUTH" sau mục "7. JWT"
docs/operations/shared-reference.md                 # §2.3 bổ sung biến GOOGLE_*
```

**Structure Decision**: Giữ toàn bộ logic trong module `identity` của Account service. Port
`IGoogleTokenVerifier` + `IOAuthStateStore` đặt ở `application/interface` (hướng vào trong), adapter
đặt ở `infrastructure` (thay thế được, không khoá vendor). `issueSession` được tách trong usecase để
Google login và password login dùng chung một cơ chế phiên.

## Phase 0 — Research

Xem [research.md](./research.md). Các quyết định chính: cách xác minh ID token, lưu `state` trên Redis,
thuật toán resolve/auto-link tài khoản, sinh username, nullable password + set password lần đầu, validate
`redirectUrl`, chống race, và mô hình tích hợp frontend.

## Phase 1 — Design & Contracts

- [data-model.md](./data-model.md): thay đổi entity `User`, migration, value object `OAuthState`.
- [contracts/google-login-api.md](./contracts/google-login-api.md): hợp đồng API 3 endpoint + mã lỗi.
- [quickstart.md](./quickstart.md): kịch bản kiểm chứng end-to-end.
- [docs/api/identity/google-login-frontend-guide.md](../../docs/api/identity/google-login-frontend-guide.md):
  hướng dẫn frontend web + mobile (deliverable).

## Post-Design Constitution Re-check

| Gate | Trạng thái sau thiết kế |
|------|-------------------------|
| §II Module Boundaries | PASS — không có import chéo module/service |
| §IV Clean Architecture | PASS — port/adapter tách đúng tầng |
| §V Error Handling | PASS — mọi lỗi qua `apperror`, thông báo tiếng Việt |
| §VIII Portability | PASS — client ID/secret cấu hình; adapter thay thế được |
| Tech Stack (Goose/Wire/Swagger) | PASS — migration + wire + swagger được cập nhật |

## Complexity Tracking

Không có vi phạm Constitution cần biện minh.
