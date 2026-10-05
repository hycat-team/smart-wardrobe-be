# Tasks: Nhóm chức năng cộng đồng (Community Social)

**Input**: Design documents from `/specs/022-community-social/`

**Prerequisites**: plan.md (required), spec.md (required), research.md, data-model.md, contracts/community-api.md, quickstart.md

**Tests**: Bắt buộc theo dự án (`make test-race`) — mỗi user story có task test cho logic cốt lõi.

**Compliance**: MỌI task phải tuân [compliance.md](./compliance.md), [AGENTS.md](../../AGENTS.md) và
[`.specify/memory/constitution.md`](../../.specify/memory/constitution.md). Không tự ý git mutation;
`make hygiene-scan` chỉ chạy khi dev cho phép.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Chạy song song được (khác file, không phụ thuộc task chưa xong)
- **[Story]**: US1..US6 tương ứng user story trong spec.md
- Đường dẫn file là chính xác (tương đối repo root)

## Path Conventions

Module mới: `services/commerce/internal/modules/community/`. Cross-service: `services/account/`,
`contracts/identity/v1/`. Shared: `shared/`.

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Khung module, config, migration, media, event — chưa có logic nghiệp vụ.

- [X] T001 [P] Đọc bắt buộc `specs/022-community-social/compliance.md`, `.specify/memory/constitution.md`, `docs/system-design/design-pattern.md`, `docs/development/code-hygiene.md`, `docs/system-design/architecture.md`
- [X] T002 [P] Tạo skeleton module `services/commerce/internal/modules/community/provider.go` (ProviderSet rỗng) và đăng ký vào `services/commerce/internal/di/wire.go` + regenerate `wire_gen.go`
- [X] T003 [P] Mở rộng struct `Community` trong `shared/config/config.go` (HotWindowDays, HotGravity, LikeWeight, CommentWeight, SearchContentPrefixChars, SearchUserCandidateCap=500, MaxMediaPerPost, MaxImageBytes, MaxVideoBytes, MaxVideoDurationSeconds, MaxTitleLength, MaxContentLength, MaxCommentLength, rate-limit fields) + `configs/config.yaml` + override env trong `shared/config/loader.go`
- [X] T004 [P] Thêm schema `community` vào `database.search_path` trong `configs/config.yaml` và `configs/config.production.yaml`
- [X] T005 [P] Thêm `ResourceType` vào `shared/application/dto/media_dtos.go` và cập nhật `shared/infrastructure/media/cloudinary_service.go` (ảnh + video)
- [X] T006 [P] Thêm topic `TopicOutfitDeleted`, queue `QueueCommunityOutfitCascade`, routing key + binding + DLQ vào `shared/application/constants/eventconstants/{eventconstants,queues}.go` và `shared/infrastructure/messaging/{rabbitmq_constants,topology}.go`
- [X] T007 [P] Thêm topic `TopicUserStatusChanged` (`identity.user.status.changed`), queue `QueueCommunityUserStatus`, routing key + binding + DLQ vào shared event constants + messaging
- [X] T008 Tạo `services/commerce/database/migrations/embed.go` và cập nhật `provideMigrationsFS()` trong `services/commerce/internal/di/providers.go` trả embed FS

**Checkpoint**: Khung + hạ tầng sẵn sàng.

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Enum, entity, migration, repository, helper, errors — CHẶN mọi user story.

**⚠️ CRITICAL**: Không bắt đầu user story nào đến khi xong phase này.

- [X] T009 [P] Enum `.../community/domain/constants/posttype/posttype.go` (`outfit`,`media`)
- [X] T010 [P] Enum `.../community/domain/constants/poststatus/poststatus.go` (`published`,`hidden`,`deleted`)
- [X] T011 [P] Enum `.../community/domain/constants/mediatype/mediatype.go` (`image`,`video`)
- [X] T012 [P] Enum `.../community/domain/constants/feedtype/feedtype.go` (`explore`,`following`)
- [X] T013 [P] Enum `.../community/domain/constants/feedsort/feedsort.go` (`hot`,`latest`)
- [X] T014 [P] Enum `.../community/domain/constants/followtype/followtype.go` (`following`,`followers`) và `.../hiddenreason/hiddenreason.go` (`admin`,`author_disabled`)
- [X] T015 [P] Entity `.../community/domain/entities/post.go` (gồm `moderated_by`, `hidden_reason`)
- [X] T016 [P] Entity `.../community/domain/entities/post_media.go`
- [X] T017 [P] Entity `.../community/domain/entities/post_like.go`
- [X] T018 [P] Entity `.../community/domain/entities/post_comment.go` (`parent_comment_id`, `root_comment_id`)
- [X] T019 [P] Entity `.../community/domain/entities/follow.go`
- [X] T020 [P] Entity `.../community/domain/entities/user_follow_stats.go` + `.../entities/table_name.go`
- [X] T021 Tạo migration `services/commerce/database/migrations/<ts>_create_community_schema.sql` (schema `community`, 6 bảng, CHECK lowercase, `hidden_reason`, FK user, index, `pg_trgm` + index search prefix, `-- +goose Up/Down`)
- [X] T022 [P] `.../community/domain/repositories/interfaces.go` (IPostRepository, IPostMediaRepository, IPostLikeRepository, IPostCommentRepository, IFollowRepository, IUserFollowStatsRepository)
- [X] T023 [P] `.../community/domain/repositories/readmodels.go` (PostFilter có `ViewerUserID`, PostListResult, CommentFilter, FollowFilter, FollowListResult, SearchUserRow)
- [X] T024 [P] Persistence `.../community/infrastructure/persistence/post_repo.go` (create, getByPublicID viewer-aware, list feed, update, soft delete, counters, admin list, hide/restore theo event user status)
- [X] T025 [P] Persistence `.../community/infrastructure/persistence/post_media_repo.go`
- [X] T026 [P] Persistence `.../community/infrastructure/persistence/post_like_repo.go`
- [X] T027 [P] Persistence `.../community/infrastructure/persistence/post_comment_repo.go`
- [X] T028 [P] Persistence `.../community/infrastructure/persistence/follow_repo.go`
- [X] T029 [P] Persistence `.../community/infrastructure/persistence/user_follow_stats_repo.go`
- [X] T030 [P] Helper sinh `public_id` 32 ký tự tại `.../community/application/usecase/shared/public_id_helper.go`
- [X] T031 [P] Domain errors `.../community/application/errors/errors.go` (tiếng Việt)
- [X] T032 Cập nhật `.../community/provider.go` đăng ký repo + helper (bổ sung usecase theo story)
- [X] T033 Tạo router skeleton `.../community/presentation/router/router.go` (nhóm public optional-auth, private user, admin) + đăng ký `communityRouter.Init(api)` trong `services/commerce/internal/bootstrap/router.go`
- [X] T034 Cập nhật `services/commerce/internal/bootstrap/worker.go` thêm field worker community (concrete pointer)

**Checkpoint**: Foundation xong.

---

## Phase 3: User Story 1 - Đăng bài và xem bảng tin (Priority: P1) 🎯 MVP

**Goal**: Tạo/sửa/xóa bài `outfit` (gắn outfit) và `media` (ảnh/video/không media); feed cơ bản; cascade khi outfit bị xóa; ẩn bài khi user bị khóa.

**Independent Test**: Tạo outfit post + 3 media post; xóa outfit → bài outfit biến mất ≤1 phút; khóa user → bài ẩn, mở → hiện.

### Implementation

- [X] T035 [P] [US1] DTO `.../community/application/dto/post.go` (CreatePostReq, UpdatePostReq, PostRes, PostMediaRes, OutfitBriefRes, GetFeedQueryReq)
- [X] T036 [P] [US1] Mapper `.../community/application/mapper/post.go`
- [X] T037 [US1] Usecase `.../community/application/usecase/post/` (CreatePost/UpdatePost/DeletePost/GetPostDetail/GetFeed latest+explore) — validate giới hạn, sở hữu outfit, `public_id`, UoW, **rate limit post**; viewer-aware (tác giả thấy bài hidden của mình)
- [X] T038 [US1] Handler `.../community/presentation/handler/post_handler.go` + swagger + message constants
- [X] T039 [US1] Route public `GET /posts`, `GET /posts/{postPublicID}`, `GET /posts/upload-signature`
- [X] T040 [US1] Route ghi `POST /posts`, `PUT /posts/{postPublicID}`, `DELETE /posts/{postPublicID}` (auth + role user)
- [X] T041 [US1] Publish `wardrobe.outfit.deleted` trong `services/commerce/internal/modules/wardrobe/application/usecase/outfit/outfit_uc.go` (DeleteOutfit) cùng UoW
- [X] T042 [US1] Cascade usecase + worker `.../community/presentation/worker/outfit_cascade_worker.go` (set `status=deleted` cho post outfit) + đăng ký bootstrap
- [X] T043 [US1] Publish `identity.user.status.changed` trong `services/account/internal/modules/admin/application/usecase/` (khi admin đổi status user) + usecase/worker `.../community/presentation/worker/user_status_worker.go` (ẩn/hiện với `hidden_reason=author_disabled`, tôn trọng ưu tiên admin) + đăng ký bootstrap
- [X] T044 [US1] Test usecase post + cascade + user-status (`.../post/*_test.go`) — validate rỗng, sở hữu outfit, soft delete terminal, viewer-aware, rate limit
- [X] T045 [US1] Cập nhật phần Post/CreatePost trong `specs/022-community-social/frontend-integration.md`

**Checkpoint**: US1 hoạt động và test độc lập.

---

## Phase 4: User Story 2 - Like và bình luận (Priority: P1)

**Goal**: Like/unlike idempotent, comment 2 cấp (reply vào cấp 2 tự gắn gốc), bộ đếm, thứ tự hiển thị.

**Independent Test**: Like lặp không đổi count; c0 → c1 → reply c1 gắn c0; xóa c0 còn reply hiển thị dấu.

### Implementation

- [X] T046 [P] [US2] DTO `.../community/application/dto/comment.go` (AddCommentReq, UpdateCommentReq, LikePostReq, CommentRes có `replyCount`/`isDeleted`, PostLikeUserRes)
- [X] T047 [P] [US2] Mapper `.../community/application/mapper/comment.go`
- [X] T048 [US2] Usecase `.../community/application/usecase/interaction/` (TogglePostLike idempotent, AddComment auto-root, Update/DeleteComment, GetPostComments gốc mới nhất trước + `replyCount`, GetCommentReplies cũ nhất trước, GetPostLikes) + counters UoW + **rate limit comment/like**
- [X] T049 [US2] Handler `.../community/presentation/handler/interaction_handler.go` + swagger
- [X] T050 [US2] Route `PUT /posts/{id}/like`, `GET /posts/{id}/likes`, `GET/POST/PUT/DELETE /posts/{id}/comments[/{commentID}/replies]` đúng auth
- [X] T051 [US2] Test usecase interaction (`.../interaction/*_test.go`) — idempotent, depth 2, xóa cha còn con, rate limit

**Checkpoint**: US1 + US2 hoạt động.

---

## Phase 5: User Story 3 - Follow và trang cá nhân công khai (Priority: P2)

**Goal**: Follow một chiều, danh sách follow (`type` optional trả cả hai + `relation`), profile công khai.

**Independent Test**: A follow B → counts đúng; profile B hiển thị tên/giới tính/avatar + isFollowing; self-follow bị chặn.

### Implementation

- [X] T052 [P] [US3] DTO `.../community/application/dto/follow.go` (FollowReq, FollowUserRes có `relation`) + `.../dto/profile.go`
- [X] T053 [P] [US3] Mapper `.../community/application/mapper/follow.go` + `.../mapper/profile.go`
- [X] T054 [US3] Usecase `.../community/application/usecase/follow/` (ToggleFollow idempotent, self-reject, cập nhật `user_follow_stats`, **rate limit follow**) + `.../usecase/profile/` (GetPublicProfile stitch `IUserBriefContract`, GetUserPosts latest, ListFollows `type=following|followers` optional + `relation`)
- [X] T055 [US3] Handler `.../community/presentation/handler/follow_handler.go` + `.../handler/profile_handler.go` + swagger
- [X] T056 [US3] Route `PUT /users/{username}/follow`, `GET /users/{username}`, `GET /users/{username}/posts`, `GET /users/{username}/follows`
- [X] T057 [US3] Test usecase follow/profile (`.../follow/*_test.go`, `.../profile/*_test.go`) — self-follow 400, idempotent, counts, relation

**Checkpoint**: US1–US3 hoạt động.

---

## Phase 6: User Story 4 - Tìm kiếm user + post (Priority: P2)

**Goal**: `GET /search?q=` trả 2 nhóm phân trang độc lập; post khớp 200 ký tự đầu; user khớp username + họ tên, sort theo follower với ứng viên cap N.

**Independent Test**: q khớp user và post; text sau ký tự 200 không khớp; q rỗng → 2 mảng rỗng; sort user theo follower trong N ứng viên.

### Implementation

- [X] T058 [P] [US4] Thêm message `SearchUserBriefsRequest` + rpc `SearchUserBriefs` vào `contracts/identity/v1/identity.proto`, chạy `make buf-gen`
- [X] T059 [US4] Account: repo `SearchByKeyword(limit)` trong `services/account/internal/modules/identity/domain/repositories/user_repo.go` + impl `.../infrastructure/persistence/user_repo_impl.go` (ILIKE username/first/last) + usecase + `contract/identity/user_brief_contract.go`/`_impl.go` + grpc server `.../presentation/grpc/server.go` + `contract/identity/transport/{mapper,to_proto}.go` + client `contracts/identity/v1/client/user_brief_client.go` (nhận limit = `SearchUserCandidateCap`)
- [X] T060 [P] [US4] DTO `.../community/application/dto/search.go` (SearchRes 2 nhóm phân trang, UserBriefRes)
- [X] T061 [US4] Usecase `.../community/application/usecase/search/` (posts ILIKE `left(content,200)` + escape; users tối đa N từ contract → enrich follower từ `user_follow_stats` → sort follower giảm dần rồi mới nhất → phân trang)
- [X] T062 [US4] Handler `.../community/presentation/handler/search_handler.go` + swagger
- [X] T063 [US4] Route `GET /search`
- [X] T064 [US4] Test usecase search (`.../search/*_test.go`) — prefix boundary 200, q rỗng, type filter, sort theo follower
- [X] T065 [US4] Cập nhật phần Search trong `frontend-integration.md`

**Checkpoint**: US1–US4 hoạt động.

---

## Phase 7: User Story 5 - Feed explore/following + hot ranking (Priority: P2)

**Goal**: `type=explore|following`, `sort=hot|latest`, default explore+hot; following chỉ bài của người đang theo dõi (không gồm bài của chính mình); guest → 401.

### Implementation

- [X] T066 [US5] Mở rộng `post_repo.go`: feed `following` (join `follows`) + điểm hot SQL theo config + cửa sổ `HotWindowDays`
- [X] T067 [US5] Cập nhật usecase GetFeed theo `feedtype`/`feedsort`, guest following → 401, viewer-aware
- [X] T068 [US5] Cập nhật handler/route `/posts` parse `type`,`sort`,`postType` + swagger
- [X] T069 [US5] Test usecase/repo feed hot vs latest + following filter (`.../post/*_test.go`)

**Checkpoint**: US1–US5 hoạt động.

---

## Phase 8: User Story 6 - Admin moderation (Priority: P3)

**Goal**: Admin xem/ẩn/xóa/khôi phục post + comment; audit; cascade comment; tác giả tự xóa không khôi phục.

### Implementation

- [X] T070 [P] [US6] DTO `.../community/application/dto/admin.go`
- [X] T071 [US6] Repo admin list posts/comments + update status/moderated_by + **cascade ẩn/hiện reply con** trong persistence tương ứng
- [X] T072 [US6] Usecase `.../community/application/usecase/admin/` (Hide/Restore/SoftDelete, chặn restore nội dung tác giả xóa, publish `admin.audit.log`)
- [X] T073 [US6] Handler `.../community/presentation/handler/admin_handler.go` + swagger
- [X] T074 [US6] Route `/admin/posts`, `/admin/comments` guard role `admin`
- [X] T075 [US6] Test usecase admin (`.../admin/*_test.go`) — restore author-deleted 400, audit, cascade comment

**Checkpoint**: Tất cả US hoạt động.

---

## Phase 9: Polish & Cross-Cutting Concerns

- [X] T076 [P] Worker reconcile counter (`.../community/presentation/worker/follow_stats_reconcile_worker.go`) + đăng ký bootstrap
- [X] T077 [P] Chạy `make swagger svc=commerce` + `make swagger-combine` + `make nginx-routes`
- [X] T078 [P] Hoàn thiện `specs/022-community-social/frontend-integration.md` (giữ nguyên + mới + thay đổi)
- [X] T079 Cập nhật ghi chú schema `community` trong `docs/system-design/architecture.md §VII`
- [X] T080 Chạy `make wire svc=commerce && make lint && make test-race && make migrate-up svc=commerce`
- [ ] T081 Đối chiếu `quickstart.md` + `spec.md §Success Criteria` (SC-001..SC-011)
- [X] T082 Ghi chú deferred ES hook (index `community_posts`) vào `research.md`/plan
- [ ] T083 [P] Đo hiệu năng ở tải tham chiếu (50 RPS): p95 ghi/danh sách < 500ms, search < 1s — đối chiếu SC-001

---

## Dependencies & Execution Order

### Phase Dependencies

- Setup → Foundational (CHẶN tất cả).
- US1 (MVP) sau Foundational.
- US2 sau Foundational (chạy sau US1 để có bài).
- US3 độc lập sau Foundational.
- US4 cần post repo (Foundational) + contract identity.
- US5 cần US1 (feed) + follow repo (Foundational) + US3 cho following.
- US6 cần post/comment repo.
- Polish sau các US mong muốn.

### Within Each User Story

- DTO/mapper [P] → usecase → handler → route → test.

### Parallel Opportunities

- T001–T007; T009–T020; T024–T031.
- Sau Foundational: US3, US4 có thể song song.

---

## Parallel Example: Foundational

```bash
# Enums:
- T009..T014
# Entities:
- T015..T020
# Persistence:
- T024..T029
```

---

## Implementation Strategy

### MVP First (US1 + US2)

1. Phase 1 Setup → Phase 2 Foundational.
2. Phase 3 US1 → Phase 4 US2.
3. **STOP & VALIDATE**: quickstart kịch bản 1–4.

### Incremental Delivery

1. Foundation → US1 → US2 (MVP).
2. + US3 (follow/profile) → + US4 (search) → + US5 (hot feed) → + US6 (admin).
3. Polish + tài liệu FE.

---

## Notes

- [P] = khác file, không phụ thuộc.
- Enum lowercase, JSON camelCase, message tiếng Việt, comment code tiếng Anh.
- Mỗi bảng/struct entity tách file riêng; mapper theo feature.
- Không tự ý git mutation; commit khi dev yêu cầu và `make wire && make lint && make test-race` xanh.
- Tài liệu FE (`frontend-integration.md`) là deliverable bắt buộc (FR-046/SC-010).
- **DI/wire**: sau mỗi thay đổi provider/DI/router (T002, T032, T033) chạy `make wire svc=commerce` để regenerate `wire_gen.go`, tránh lệch generated.
- **Task không map FR** (chấp nhận): T001 = tiền đề compliance cho mọi FR; T079 = cập nhật tài liệu kiến trúc (không có FR tương ứng); T082 = ghi chú deferred ES (mở rộng tương lai của FR-017/FR-018); T083 = kiểm chứng SC-001.

---

## Hậu kiểm sau bàn giao — bugfix counter & moderation (2026-10-06)

Sửa các lỗ hổng phát hiện khi đối chiếu FR-008, R12 và FR-036; không đổi endpoint/DTO nên không cần cập nhật `frontend-integration.md`.

- **Admin ẩn/xóa/khôi phục comment không cập nhật `comment_count`**: thêm `IPostRepository.RecountCommentCount` (khóa row `FOR UPDATE` + COUNT trong cùng transaction); `AdminUseCase` nhận `IUnitOfWork`, `setCommentStatus`/`RestoreComment` recount sau khi đổi trạng thái. Idempotent, không lệch khi thao tác lặp.
- **Unlike đồng thời trừ 2 lần**: `IPostLikeRepository.Delete` trả rows-affected; `TogglePostLike` chỉ decrement khi `affected == 1`.
- **`UpdatePost` ghi đè counter**: thay `Save` toàn entity bằng `UpdateEditableFields` (chỉ `title`/`content`/`outfit_id`).
- **Thiếu worker reconcile post counters (R12)**: thêm `IPostRepository.ReconcileCounters` (SQL drift-only) + `PostStatsReconcileWorker` (30 phút) + `ReconcilePostCounters` trong `IStatsSyncUseCase`; đăng ký provider/bootstrap và regenerate Wire.
- **FR-036 (phát hiện thêm)**: `UpdateStatusByRoot` không hồi sinh reply tác giả tự xóa (restore chỉ tác động reply `moderated_by IS NOT NULL`); chặn admin ẩn/xóa nội dung tác giả đã tự xóa bằng `ErrCannotModerateAuthorDeleted` (comment + post).
- **Kiểm chứng**: `make lint` xanh; `go test -race -count=3 ./internal/modules/community/...` xanh (services/commerce); `make wire svc=commerce` đã regenerate `wire_gen.go`.

