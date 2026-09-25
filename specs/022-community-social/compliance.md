# Compliance & Governance (MANDATORY) — Spec 022 Community Social

**Feature**: 022-community-social | **Date**: 2026-09-25
**Mục đích**: nhắc lại các quy tắc **bắt buộc** phải tuân thủ khi triển khai spec 022, để tránh
vi phạm kiến trúc/naming/workflow. Đây là checklist companion cho `spec.md`, `plan.md`, `tasks.md`.

> **Thứ tự ưu tiên khi xung đột tài liệu**: Constitution > system-design > development > other docs.
> Nếu tài liệu và mã nguồn không khớp: **DỪNG**, nêu xung đột và hỏi dev, không đoán.

## 1. Nguồn bắt buộc đọc trước khi code

| Tài liệu | Vai trò |
|---|---|
| [AGENTS.md](../../AGENTS.md) | Quy tắc agent: navigation, workflow, git, code compliance |
| [.specify/memory/constitution.md](../../.specify/memory/constitution.md) | Hiến pháp — tối cao về kiến trúc/nguyên tắc |
| `docs/development/agent-workflow.md` | Quy trình Spec-kit, core principles §IV |
| `docs/development/git-workflow.md` | Quy tắc git (không tự ý mutate) |
| `docs/development/code-hygiene.md` | Ngưỡng cấu trúc file/function, struct-per-file |
| `docs/system-design/design-pattern.md` | Convention triển khai (error, pagination, DTO, worker, rate limit) |
| `docs/system-design/architecture.md` | Kiến trúc, module boundaries, schema |
| `docs/system-design/event-driven-design.md` + `docs/development/messaging-guide.md` | Outbox/RabbitMQ cho cascade outfit |

## 2. Quy tắc chung (từ AGENTS.md)

- **Constitution Compliance**: đọc và tuân `.specify/memory/constitution.md` cho **mọi** hành động.
- **Workflow**: task lớn theo Spec-kit; luôn gợi ý lệnh/skill kế tiếp sau mỗi bước.
- **Git — no autonomous**: KHÔNG tự chạy `commit/push/tag/branch/checkout/merge/rebase/reset/amend`
  nếu không có chỉ đạo rõ ràng của dev. Chỉ đọc `status/log/diff`. Commit message tiếng Anh,
  Conventional Commits; trước khi đề xuất commit: `make wire && make lint && make test-race` phải xanh.
- **Makefile**: ưu tiên target Makefile (`make wire svc=commerce`, `make migrate-up svc=commerce`, …).
- **Language**: giao tiếp với dev bằng **tiếng Việt**; message/validation/Swagger **tiếng Việt**;
  comment code/log/identifier **tiếng Anh**.
- **specs/ tiếng Việt**: mọi tài liệu trong `specs/` viết tiếng Việt.
- **Relative paths**: dùng đường dẫn tương đối trong docs/spec/link (không absolute/`file:///`).
- **.env**: mọi comment trong file `.env*` phải viết **tiếng Anh**.
- **hygiene-scan**: `make hygiene-scan` CHỈ chạy khi dev cho phép/yêu cầu; không tự chạy.

## 3. Constitution gates liên quan (tóm tắt cho 022)

| Điều | Ràng buộc cho 022 | Cách đáp ứng trong plan |
|---|---|---|
| §I Services & module boundaries | Module `community` nằm trong Commerce; không import implementation Account | Chỉ dùng `IUserBriefContract` |
| §II Contracts & DTO | Cross-module qua contract DTO; cross-service qua proto | RPC `SearchUserBriefs`; stitching `sliceutils.IndexBy/GroupBy` |
| §III Async event-driven | Write side effect xuyên module qua event + outbox | Topic `wardrobe.outfit.deleted` + worker |
| §IV Clean architecture | `presentation → application → domain ← infrastructure` | Layout chuẩn, handler thin |
| §V Transaction & error | `apperror`, `IUnitOfWork.Execute` | domain errors tiếng Việt |
| §VII Design patterns | Tuân `design-pattern.md` | xem §5 dưới |
| §VIII Portability | Không vendor lock-in; ES optional | DB search mặc định, ES sau |

## 4. Quy tắc theo feature 022

- **Schema/migration**: Goose per-service; migration mới ở `services/commerce/database/migrations/`;
  không sửa `init-db`; bật `embed.go` + `provideMigrationsFS` cho Commerce.
- **Schema separation**: FK tới `identity.users` được phép; **KHÔNG** FK `posts → wardrobe.outfits`;
  liên kết outfit enforce tầng ứng dụng; cascade qua event.
- **Enum lowercase**: `outfit|media`, `published|hidden|deleted`, `image|video`, `explore|following`,
  `hot|latest`, `following|followers` (query) và `following|follower` (relation).
- **JSON camelCase** cho mọi DTO frontend; contract DTO giữ camelCase.
- **Tương thích FE**: giữ path/envelope/DTO cũ cho post/like/comment/upload; bỏ field resale.
- **Deliverable FE**: `frontend-integration.md` phải đầy đủ (giữ nguyên + mới + thay đổi).

## 5. Convention triển khai (theo `design-pattern.md`)

- **Error**: repository trả plain `error`; use case dịch sang domain error ở `application/errors/`;
  handler trả `error`; middleware global map HTTP. Dùng `apperror.New*`/sentinel.
- **Pagination**: `shared_dto.PaginationQuery` + `PaginationResult[T]`; repo `Count` trước `Find`,
  `ApplyPagination`; handler `validation.BindQuery`.
- **Handler**: `func (h *XxxHandler) Method(c *gin.Context) error` + `shared_pres.WrapHandler`;
  message constants khai báo đầu file.
- **Repository**: `IGenericRepository[T,ID]`, preload at construction, `GetQueryWithPreload`, `GetDB(ctx)`.
- **Unit of Work**: `uow.Execute(ctx, fn)` cho thao tác nhiều repo (post + counter, follow + stats).
- **DTO naming**: `CreateXxxReq`, `UpdateXxxReq`, `GetXxxQueryReq`, `XxxRes`, `XxxBriefRes`,
  `XxxListRes`, `XxxContractDTO`, `XxxEventPayload`; một struct một vai trò; enum ở `domain/constants/`.
- **Mapper**: đặt trong `application/mapper/`, không để trong use case; helper một-lần dùng hậu tố `_helper.go`.
- **Contract**: chỉ DTO; impl delegate use case; không import entity/repo/persistence của module khác.
- **Worker**: `presentation/worker/`, nhúng `*BaseWorker`, `RegisterTypedConsumer`, thin; đăng ký ở
  `bootstrap.Workers` với kiểu con trỏ cụ thể.
- **Rate limit**: quota nghiệp vụ dùng `IRateLimiter` (Layer 2), không dùng middleware global.
- **Migration**: `make migrate-create svc=commerce name=...`; `-- +goose Up/Down`.

## 6. Code hygiene (theo `code-hygiene.md`)

- Giữ trong ngưỡng độ dài file/function; mỗi struct một file khi là entity DTO chính; tách mapper theo nhóm.
- Không tạo helper vô nghĩa; tái dùng `shared/pkg/utils` (`sliceutils`, `stringutils`, `timeutils`…).
- Chỉ chạy `make hygiene-scan` khi dev cho phép.

## 7. Kiểm thử & gate trước khi báo hoàn thành

```bash
make wire svc=commerce
make buf-gen                 # nếu đổi proto identity
make lint                    # fmt-check + go vet
make test-race               # unit + race
# và migration:
make migrate-up svc=commerce
```

- Viết unit test cho use case (hot ranking, comment depth, follow idempotent, search prefix 200).
- Đối chiếu `spec.md §Success Criteria` và `quickstart.md`.

## 8. Checklist trước commit (khi dev yêu cầu commit)

- [ ] `make wire && make lint && make test-race` xanh.
- [ ] Không có vi phạm layer boundary / contract purity.
- [ ] Enum lowercase, JSON camelCase, message tiếng Việt, comment code tiếng Anh.
- [ ] Migration goose Up/Down đầy đủ; không sửa init-db.
- [ ] `.env*` comment tiếng Anh (nếu chạm).
- [ ] `frontend-integration.md` cập nhật khớp API thực tế.
- [ ] Không tự ý chạy git mutation khi chưa được yêu cầu.
