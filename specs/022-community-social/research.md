# Phase 0 Research: Nhóm chức năng cộng đồng (Community Social)

**Feature**: 022-community-social | **Date**: 2026-09-25

Tất cả điểm mơ hồ trong spec đã được chốt tại `## Clarifications`. Dưới đây là các quyết định
kỹ thuật (Decision / Rationale / Alternatives) để triển khai.

## R1. Vị trí module: `services/commerce/internal/modules/community`

- **Decision**: Đặt module trong Commerce Core, layout chuẩn, `provider.go` kiểu flattened
  (như wardrobe).
- **Rationale**: Outfit/wardrobe thuộc Commerce → cascade và render outfit dùng contract in-service,
  không cần gRPC mới từ Commerce. Commerce đã consume `IUserBriefContract` (Account) qua gRPC
  (`CommerceContracts.UserBrief`), tái dùng để lấy thông tin tác giả.
- **Alternatives**: (a) Account — mỗi render outfit phải gọi chéo Commerce; (b) service mới —
  thừa, vi phạm §I (chỉ 3 service).

## R2. Migration Commerce lần đầu

- **Decision**: Tạo `services/commerce/database/migrations/` với file goose
  `<ts>_create_community_schema.sql` + `embed.go`; sửa `provideMigrationsFS()` trả embed FS
  thay vì `nil`; giữ `migration_table` của Commerce.
- **Rationale**: Constitution yêu cầu migration per-service; schema `community` do Commerce sở hữu.
  Goose version table riêng đã cấu hình sẵn (`goose_db_version_commerce`).
- **Alternatives**: Đặt migration trong Account (de-facto cũ) — lệch kiến trúc, trộn quyền schema.

## R3. Schema & tên bảng

- **Decision**: Schema `community`; bảng `posts`, `post_media`, `post_likes`, `post_comments`,
  `follows`, `user_follow_stats`. Thêm `community` vào `database.search_path`.
- **Rationale**: Theo pattern schema-per-module (`architecture.md §VII`); FK tới `identity.users`
  được phép (giữ ~30 FK user hiện có), không thêm FK xuyên module khác.
- **Alternatives**: Dùng `public` — phá vỡ schema separation đã thiết lập.

## R4. Hot ranking query-time

- **Decision**: Tính điểm bằng SQL khi truy vấn:
  `score = (like_count + 2*comment_count + 1) / POWER(age_hours + 2, gravity)`; lọc cửa sổ
  `created_at >= now() - HotWindowDays`. `gravity=1.5`, `HotWindowDays=30`, `like×1`, `comment×2`
  cấu hình được. `latest` dùng `created_at DESC`.
- **Rationale**: Bản cũ (`post_score_snapshots` + dirty flag + worker refresh) bị đánh giá là
  overengineer; query-time loại bỏ hoàn toàn hạ tầng điểm, vẫn bảo đảm ranking engagement + decay.
  Cửa sổ thời gian chặn full scan.
- **Alternatives**: (a) Snapshot table + worker — bị loại; (b) materialized view — thêm hạ tầng,
  user chọn bản nhẹ.
- **Note**: Cần đảm bảo `age_hours` không âm và mẫu > 0; index `(created_at DESC, id)`.

## R5. Search dự phòng bằng DB

- **Decision**: Post search dùng `content ILIKE '%q%'` trên biểu thức `left(content, N)` (N=200
  config), hỗ trợ index trigram (`pg_trgm`) trên `lower(left(content, N))`. User search gọi RPC
  `SearchUserBriefs` sang Account (ILIKE username/first_name/last_name). Escape `%`/`_`.
- **Rationale**: ES optional & mặc định tắt; DB fallback đơn giản, đủ cho MVP. Postgres `pg_trgm`
  tăng tốc ILIKE. User data thuộc Account → bắt buộc qua contract.
- **Alternatives**: (a) Bật ES ngay — thêm adapter + sync worker, chưa cần; (b) nhân bản username
  sang Commerce — drift. Để hook ES sau theo pattern `IWardrobeSearchService` (index `community_posts`).

## R6. Mở rộng proto identity cho user search

- **Decision**: Thêm message `SearchUserBriefsRequest { string keyword = 1; int32 limit = 2; }` và
  RPC `SearchUserBriefs` vào `IdentityService`; cập nhật transport mapper, gRPC server (Account),
  client (`contracts/identity/v1/client`), và `IUserBriefContract` + impl. Chạy `make buf-gen`.
- **Rationale**: Chỉ mở rộng một contract sẵn có; không tạo service/gRPC mới. Giữ mapping wire⇄DTO
  ở owner (Account `contract/identity/transport`).
- **Alternatives**: Dựng gRPC contract mới từ Commerce để Account gọi post search — nhiều bề mặt hơn.

## R7. Cascade outfit qua event

- **Decision**: Thêm topic `TopicOutfitDeleted = "wardrobe.outfit.deleted"` + routing key + queue
  `community_outfit_cascade_queue` (+ DLQ). Trong `OutfitUseCase.DeleteOutfit`, sau khi xoá thành
  công trong UoW, publish payload `{ outfitId }`. Worker `OutfitCascadeWorker` consume → use case
  set `status=deleted` cho các post `outfit` có `outfit_id` tương ứng.
- **Rationale**: §III ưu tiên event cho write side effect; outbox bảo đảm at-least-once. Lưu ý
  routing key `wardrobe.event.#` (ES sync) KHÔNG khớp `wardrobe.outfit.deleted` nên không lẫn queue.
- **Alternatives**: Gọi contract cộng đồng đồng bộ từ `DeleteOutfit` — coupling & không bền vững
  khi broker/DB tách.

## R8. Media upload hỗ trợ video

- **Decision**: Thêm `ResourceType` (`image`|`video`) vào `UploadSignatureParams`/`UploadSignatureResult`
  và tham số `resourceType` cho endpoint `GET /posts/upload-signature`; `CloudinaryService` trả
  kèm `ResourceType` để FE chọn đúng endpoint upload. Dùng `Cloudinary.PostFolder`.
- **Rationale**: Cloudinary ký cùng tham số cho cả 2 loại, nhưng endpoint upload khác
  (`/<resource_type>/upload`); FE cần biết loại.
- **Alternatives**: Chỉ ảnh — không đáp ứng yêu cầu video.

## R9. Rate limiting

- **Decision**: Dùng `IRateLimiter` (Layer 2, Redis fixed window) ở use case: post 10/giờ,
  comment 60/giờ, follow 200/giờ, like 600/giờ (config). Khoá theo user.
- **Rationale**: `design-pattern.md` yêu cầu business quota đi qua `IRateLimiter`, không dùng
  middleware global.
- **Alternatives**: Middleware token bucket — dành cho shield per-IP, không hợp quota nghiệp vụ.

## R10. Admin moderation & audit

- **Decision**: Route `/api/v1/admin/posts` và `/api/v1/admin/comments` đăng ký trong community
  router, guard `RolesAuthorize(roleslug.Admin)`. Ẩn/xóa/khôi phục từng mục; ghi audit qua publish
  topic `admin.audit.log` (payload `RecordAuditLogInput` hiện có).
- **Rationale**: Module admin nằm ở Account, dữ liệu ở Commerce; giữ route local tránh gRPC mới.
- **Alternatives**: Expose gRPC community cho admin Account gọi — nặng hơn; hoặc để admin Account
  gọi DB Commerce — vi phạm boundary.

## R11. Phân trang

- **Decision**: Dùng `shared_dto.PaginationQuery` + `PaginationResult[T]`; repository
  `Count` trước `Find`, `ApplyPagination`. Danh sách comment, like, follow, search đều phân trang
  (trừ comment/reply giữ mảng như FE cũ nếu cần tương thích — xem contract).
- **Rationale**: Chuẩn dự án; user chốt theo `design-pattern.md`.
- **Alternatives**: Cursor/keyset — chưa cần, phá tương thích FE.

## R12. Counter lưu sẵn & đối soát

- **Decision**: `posts.like_count`/`posts.comment_count` cập nhật cùng transaction khi like/comment;
  `community.user_follow_stats(user_id, follower_count, following_count)` cập nhật cùng transaction
  khi follow/unfollow. Một worker reconcile định kỳ (cron) đối soát lại từ bảng gốc.
- **Rationale**: Hiển thị nhanh; spec chọn denormalized + reconcile.
- **Alternatives**: COUNT on-the-fly — chậm hơn khi list lớn; bỏ counter → nhiều query.

## R13. Tương thích API frontend cũ

- **Decision**: Giữ path/envelope/DTO cũ cho post/like/comment/upload. Bỏ `items`, `totalPrice`,
  `globalHotnessScore`, `finalFeedScore`, transfer fields. Thêm `outfit`, `isFollowingAuthor`.
  `postType` đổi `SALE`→`MEDIA`. Thêm endpoint mới cho follow/profile/search/admin lite.
- **Rationale**: Giảm chi phí frontend; frontend vốn không dùng `/transfers/*`.
- **Alternatives**: Thiết kế lại toàn bộ path — tốn công FE.

## R14. Tài liệu tích hợp frontend

- **Decision**: Tạo `specs/022-community-social/frontend-integration.md` mô tả 3 nhóm: giữ nguyên,
  mới, thay đổi; kèm bảng endpoint + ví dụ request/response + checklist migration FE.
- **Rationale**: FR-046/SC-010 là deliverable bắt buộc.
- **Alternatives**: Chỉ dựa vào swagger — không đủ phần "changed vs FE".

## R15. Ẩn bài theo trạng thái user (giải C1)

- **Decision**: Account publish topic `identity.user.status.changed` (payload `{ userId, status }`) khi admin đổi
  trạng thái user. Community worker consume → set `posts.status=hidden` + `hidden_reason=author_disabled`
  khi user bị vô hiệu hóa/khóa; restore `published` + clear reason khi hoạt động lại.
- **Rationale**: Tránh join `identity.users` xuyên module (vi phạm §II); event bất đồng bộ đúng §III.
  `hidden_reason` phân biệt với ẩn do admin để restore đúng.
- **Alternatives**: (a) Gọi contract mỗi lần đọc — N+1, chậm; (b) bỏ FR-035 — giảm scope không mong muốn.

## R16. Comment/reply giữ mảng (giải C2)

- **Decision**: `GET /posts/{id}/comments` và `.../replies` giữ `CommentRes[]`; phân trang chỉ cho
  feed/like/follow/search. Sửa FR-023.
- **Rationale**: Tương thích FE cũ, giảm chi phí frontend; khối lượng comment 1 bài thường nhỏ.

## R17. Nhãn follow thống nhất (giải C3)

- **Decision**: Query `type=following|followers`; trường `relation=following|follower`; endpoint
  `GET /users/{username}/follows`.
- **Rationale**: Đồng nhất hướng nhìn từ người xem; tránh lẫn `followee`/`following`.
- **Alternatives**: `followee/follower` — kém trực quan cho FE.

## R18. Sort search user theo follower (giải C4)

- **Decision**: Account `SearchUserBriefs(keyword, limit=N)` trả tối đa N ứng viên (config
  `Community.SearchUserCandidateCap`, default 500). Community enrich `follower_count` từ
  `user_follow_stats`, sort follower giảm dần rồi mới nhất, phân trang in-memory.
- **Rationale**: Giữ đúng FR-043 với dữ liệu tách owner; N giới hạn chi phí.
- **Trade-off**: Ngoài N ứng viên, kết quả xếp hạng có thể thiếu; chấp nhận ở MVP (ghi rõ).
- **Alternatives**: (a) bỏ sort follower — đổi yêu cầu; (b) projection user trong community — thêm sync.

## R19. Rate limit trong từng user story (giải C9)

- **Decision**: Áp `IRateLimiter` ngay khi xây usecase post/comment/follow/like (không dồn về Polish).
- **Rationale**: MVP đã được bảo vệ; tránh nợ kỹ thuật.

## R20. Viewer-aware cho bài ẩn (giải C5)

- **Decision**: Repo/query feed/detail nhận `viewerUserID`: tác giả thấy bài `hidden` của mình
  (kèm trạng thái); người khác không; admin thấy qua endpoint admin.
- **Rationale**: Đúng FR-036.

## R21. Cascade comment theo cây (giải C8)

- **Decision**: Admin ẩn/xóa comment gốc → ẩn/hiện các reply con tương ứng; restore mở lại theo cha.
- **Rationale**: Đúng US6 AC3.

## Outstanding / Deferred

- Bật Elasticsearch cho post/user search (index `community_posts`) — để sau, dùng DB fallback đợt này.
- Thông báo (notification) like/comment/follow — ngoài scope.
- Bio, privacy, block/report, đồng bộ khi outfit đổi — ngoài scope.
