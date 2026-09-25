# Implementation Plan: Nhóm chức năng cộng đồng (Community Social)

**Branch**: `022-community-social` | **Date**: 2026-09-25 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `/specs/022-community-social/spec.md`

## Summary

Bổ sung nhóm chức năng cộng đồng (community) cho SmartWardrobe Backend: đăng bài (outfit
post gắn outfit, cascade khi outfit bị xóa; media post ảnh/video/không media), like, comment
2 cấp, follow một chiều, danh sách follow/follower, trang cá nhân công khai theo username,
search `q` (user + post, 200 ký tự đầu), feed `explore`/`following` với hot ranking query-time,
và admin moderation cho post/comment.

Cách tiếp cận kỹ thuật: tạo module mới `community` trong **Commerce Core** (dùng chung
outfit/wardrobe, tái dùng `IUserBriefContract` sẵn có), schema DB `community`, migration
Goose thuộc quyền Commerce (bật `provideMigrationsFS` lần đầu), cascade outfit qua sự kiện
outbox, search mặc định bằng truy vấn DB (Elasticsearch để sau), và một tài liệu tích hợp
frontend bao phủ toàn bộ spec (giữ nguyên + mới + thay đổi).

## Technical Context

**Language/Version**: Go 1.25+ (multi-module workspace `go.work`)

**Primary Dependencies**: Gin, GORM, Google Wire, Goose (migrations), RabbitMQ (outbox),
Postgres (+ `pg_trgm` cho search dự phòng), Cloudinary (`IMediaService`), Redis (`IRateLimiter`)

**Storage**: PostgreSQL — schema mới `community` (thêm vào `database.search_path`)

**Testing**: `go test ./...` / `make test-race` (unit + integration theo pattern hiện có), test
cho use case và repository; không thêm framework mới

**Target Platform**: Linux (container 3-service: Account/Commerce/AI); module nằm trong Commerce

**Project Type**: Web service (REST API backend)

**Performance Goals**: 95% thao tác đăng bài/like/comment/follow < 2 giây dưới tải thông thường;
feed hot trong cửa sổ 30 ngày không full-table scan

**Constraints**: Phân trang theo `PaginationQuery` (page/limit); hot ranking query-time không
snapshot table; cascade outfit bất đồng bộ đạt nhất quán ≤ 1 phút; ES không bắt buộc

**Scale/Scope**: ~15 endpoint mới trong 1 module; 6 bảng cộng thêm 1 bảng stats; 1 thay đổi
proto identity (thêm RPC search user); 1 topic/queue mới cho cascade outfit

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

> Triển khai bắt buộc tuân thủ [AGENTS.md](../../AGENTS.md) + [`.specify/memory/constitution.md`](../../.specify/memory/constitution.md)
> và các doc trong `docs/development/`, `docs/system-design/`. Checklist đầy đủ: [compliance.md](./compliance.md).
> Không tự ý git mutation; `make hygiene-scan` chỉ chạy khi dev cho phép.

| Gate (Constitution) | Đánh giá | Trạng thái |
|---|---|---|
| §I — Independently Deployable Services | Không thêm service; module mới nằm trong Commerce. Không import implementation module Account | PASS |
| §I — Module boundaries | Chỉ dùng `IUserBriefContract` (Account) qua gRPC; outfit/wardrobe qua contract in-service trong Commerce | PASS |
| §II — Strict boundaries & DTO contracts | Cross-module chỉ qua contract DTO; cross-service qua protobuf mới `SearchUserBriefs`; stitching bằng `sliceutils.IndexBy/GroupBy` | PASS |
| §III — Async event-driven | Cascade outfit dùng outbox + worker consumer; không gọi đồng bộ xuyên module cho write side effect | PASS |
| §IV — Clean Architecture layering | Layout chuẩn `domain/application/infrastructure/presentation`; handler thin, use case orchestrates | PASS |
| §V — Transaction & Error handling | Dùng `apperror`; thao tác nhiều repo dùng `IUnitOfWork.Execute`; message tiếng Việt | PASS |
| §VI — AI pipeline | Không liên quan (không gọi AI) | N/A |
| §VII — Design patterns | Tuân `design-pattern.md`: DTO naming, generic repo, pagination, worker pattern, rate limit Layer 2 | PASS |
| §VIII — Portability / no vendor lock-in | Chỉ dùng Postgres/Redis/RabbitMQ/Cloudinary; ES optional & disableable, mặc định DB search | PASS |
| Tech stack — Goose migrations per service | Migration tạo trong `services/commerce/database/migrations/` + bật embed FS của Commerce | PASS (xem Complexity Tracking) |
| Naming — JSON camelCase, constants lowercase | DTO camelCase; enum `outfit`/`media`, `published`/`hidden`/`deleted`, `image`/`video`, `explore`/`following`, `hot`/`latest` (lowercase) | PASS |
| Code hygiene | Tách file theo feature/mapper; không vượt ngưỡng; hygiene-scan chỉ chạy khi được yêu cầu | PASS |

**Kết luận gate**: PASS — không có vi phạm cần biện minh ngoài việc Commerce lần đầu có migration
(đúng chủ trương per-service, FR-035).

## Project Structure

### Documentation (this feature)

```text
specs/022-community-social/
├── plan.md                    # This file
├── spec.md                    # Feature specification
├── research.md                # Phase 0 output
├── data-model.md              # Phase 1 output
├── quickstart.md              # Phase 1 output
├── compliance.md              # Checklist tuân thủ Constitution + AGENTS.md (MANDATORY)
├── frontend-integration.md    # Deliverable FR-046 (old/new/changed cho FE)
├── contracts/
│   └── community-api.md       # Hợp đồng HTTP API đầy đủ
├── checklists/
│   └── requirements.md
└── tasks.md                   # Phase 2 (/speckit-tasks)
```

### Source Code (repository root)

```text
services/commerce/
├── database/migrations/
│   ├── embed.go                                  # MỚI: embed FS (bật auto-migrate Commerce)
│   └── 2026XXXXXXXXXX_create_community_schema.sql# MỚI: schema + bảng + index + trigram
└── internal/
    ├── di/
    │   ├── providers.go                          # SỬA: provideMigrationsFS trả embed FS
    │   ├── app.go                                # SỬA: thêm Community module provider set
    │   └── wire.go / wire_gen.go                 # SỬA: đăng ký module + contract search
    ├── bootstrap/
    │   ├── router.go                             # SỬA: NewEngine nhận + Init communityRouter
    │   └── worker.go                             # SỬA: Workers thêm worker community
    └── modules/
        ├── wardrobe/
        │   ├── application/usecase/outfit/outfit_uc.go   # SỬA: publish outfit.deleted
        │   └── domain/entities/outfit.go                  # (không đổi cấu trúc)
        └── community/                            # MỚI — module mới (flat provider style)
            ├── provider.go
            ├── domain/
            │   ├── constants/
            │   │   ├── posttype/posttype.go             # outfit|media
            │   │   ├── poststatus/poststatus.go         # published|hidden|deleted
            │   │   ├── mediatype/mediatype.go           # image|video
            │   │   ├── feedtype/feedtype.go             # explore|following
            │   │   ├── feedsort/feedsort.go             # hot|latest
            │   │   └── followtype/followtype.go         # following|followers (filter); relation following|follower
            │   ├── entities/
            │   │   ├── post.go
            │   │   ├── post_media.go
            │   │   ├── post_like.go
            │   │   ├── post_comment.go
            │   │   ├── follow.go
            │   │   ├── user_follow_stats.go
            │   │   └── table_name.go
            │   └── repositories/
            │       ├── interfaces.go
            │       └── readmodels.go
            ├── application/
            │   ├── constants/                            # cấu hình hành vi (limit/weight)
            │   ├── dto/
            │   │   ├── post.go                            # Req/Res + Query
            │   │   ├── comment.go
            │   │   ├── follow.go
            │   │   ├── profile.go
            │   │   ├── search.go
            │   │   ├── admin.go
            │   │   ├── event_payload.go
            │   │   └── contract_dto.go                    # community contract (nếu expose)
            │   ├── errors/errors.go
            │   ├── interface/usecase/                    # interfaces
            │   ├── mapper/                               # post/comment/follow/profile/search
            │   └── usecase/
            │       ├── post/          (create/update/delete/get/feed)
            │       ├── interaction/   (like/comment)
            │       ├── follow/        (toggle/list)
            │       ├── profile/       (public profile/posts)
            │       ├── search/        (combined search)
            │       ├── admin/         (moderation)
            │       └── outfitcascade/ (consumer of outfit.deleted)
            ├── infrastructure/persistence/               # GORM repos
            └── presentation/
                ├── handler/       (post, interaction, follow, profile, search, admin)
                ├── router/router.go
                └── worker/        (outfit_cascade_worker.go, user_status_worker.go)

services/account/                                 # thay đổi cross-service cho search user + status event
├── internal/modules/identity/
│   ├── application/usecase/                        # MỚI: SearchUserBriefs use case
│   ├── domain/repositories/user_repo.go            # SỬA: thêm SearchByKeyword
│   ├── infrastructure/persistence/user_repo_impl.go# SỬA: ILIKE username/name
│   └── presentation/grpc/server.go                 # SỬA: thêm RPC handler
├── internal/modules/admin/application/usecase/     # SỬA: publish identity.user.status.changed khi đổi status user
└── contract/identity/
    ├── user_brief_contract.go                      # SỬA: thêm SearchUserBriefs
    ├── user_brief_contract_impl.go                 # SỬA
    └── transport/mapper.go + to_proto.go           # SỬA: mapping mới

contracts/identity/v1/
├── identity.proto                                  # SỬA: message + rpc SearchUserBriefs
├── gen/... (buf-gen)                               # TÁI SINH
└── client/user_brief_client.go                     # SỬA: thêm client method

shared/
├── application/dto/media_dtos.go                   # SỬA: thêm ResourceType
├── application/media/interfaces.go                 # (giữ; hành vi mở rộng qua params)
├── infrastructure/media/cloudinary_service.go      # SỬA: resource type cho video
├── config/config.go                                # SỬA: struct Community mở rộng
├── config/loader.go                                # SỬA: env override mới
├── application/constants/eventconstants/            # SỬA: topic/queue outfit.deleted + identity.user.status.changed
└── infrastructure/messaging/
    ├── rabbitmq_constants.go                       # SỬA: routing key mới
    └── topology.go                                 # SỬA: binding + DLQ
```

**Structure Decision**: Chọn module `community` trong `services/commerce/internal/modules/`
(full canonical layout, flattened `provider.go` theo kiểu wardrobe). Migration thuộc Commerce
để đúng chủ trương per-service (constitution FR-035), lần đầu bật `provideMigrationsFS`.
Thay đổi identity chỉ giới hạn ở thêm một RPC search user (cross-service), không thay đổi DTO
`UserBriefContractDTO` hiện có.

## Complexity Tracking

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| Commerce lần đầu có migration riêng (`provideMigrationsFS` từ nil → embed FS) | Schema `community` thuộc Commerce; constitution yêu cầu per-service Goose | Tập trung vào Account lệch kiến trúc per-service và trộn quyền sở hữu schema |
| Thêm RPC `SearchUserBriefs` vào proto identity | Search user cần dữ liệu thuộc Account; Commerce không được đọc DB Account | Nhân bản username sang Commerce gây drift và đồng bộ phức tạp; search client-side không khả thi |
| Bảng `community.user_follow_stats` (counter lưu sẵn) | Trang profile cần follower/following count nhanh; spec chọn denormalized + reconcile | COUNT mỗi request chấp nhận được ở quy mô nhỏ nhưng không theo quyết định denormalized; có worker reconcile |

## Cross-cutting decisions (Phase 0/1 summary)

- **Hot ranking**: SQL query-time, không snapshot. Công thức:
  `score = (like_count + 2*comment_count + 1) / POWER(age_hours + 2, gravity)`, lọc
  `created_at >= now() - HotWindowDays`; `gravity=1.5`, `HotWindowDays=30` (cấu hình).
- **Search**: Post dùng `ILIKE` trên biểu thức `left(content, SearchContentPrefixChars)` + index
  trigram; user qua RPC `SearchUserBriefs` (ILIKE username/first/last). Chuẩn hoá escape ký tự
  `%`/`_`. Trả 2 `PaginationResult` độc lập.
- **Cascade outfit**: `DELETE /outfits/:id` publish `wardrobe.outfit.deleted` trong UoW; worker
  community consume → set `status=deleted` cho mọi post `outfit` tham chiếu.
- **Admin moderation**: route `/api/v1/admin/posts|comments` đăng ký trong community router,
  guard `RolesAuthorize(admin)`; hành động ghi audit qua topic `admin.audit.log`.
- **Rate limit**: dùng `IRateLimiter` (Layer 2) tại use case.
- **Tài liệu FE**: `frontend-integration.md` là artifact bắt buộc (FR-046), mô tả giữ nguyên/mới/thay đổi.
- **Trạng thái user (FR-035)**: Account publish `identity.user.status.changed` khi admin đổi trạng thái;
  community worker ẩn/hiện bài với `status=hidden` + `hidden_reason=author_disabled` (không join xuyên module).
- **Pagination comment**: comment/reply giữ mảng (tương thích FE); phân trang cho feed/like/follow/search.
- **Nhãn follow**: query `type=following|followers`; `relation=following|follower`.
- **Search user**: Account trả tối đa `Community.SearchUserCandidateCap` (mặc định 500) ứng viên;
  community enrich follower count rồi sort + phân trang.
- **Viewer-aware**: tác giả thấy bài `hidden` của mình; người khác 404. Admin xóa/ẩn comment cascade reply con.
- **Rate limit** áp ngay trong từng user story.

## Phase status

- Phase 0 Research → `research.md` (hoàn tất, không còn NEEDS CLARIFICATION).
- Phase 1 Design → `data-model.md`, `contracts/community-api.md`, `quickstart.md`,
  `frontend-integration.md` (hoàn tất).
- Phase 2 Tasks → `/speckit-tasks` (chưa thực hiện).
