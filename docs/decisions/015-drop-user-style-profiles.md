# ADR 015 - Xoá bảng `user_style_profiles`

> Đặt tên file theo `NNN-ten-kebab.md`; quy ước đầy đủ ở [README.md](README.md).

*   **Status**: Accepted
*   **Decision Date**: 2026-10-11
*   **Decision Maker**: Dev
*   **Supersedes**: —

## 1. Context

Bảng `identity.user_style_profiles` lưu vector gu thời trang (`taste_embedding`, `vector(768)` kèm HNSW index `vector_cosine_ops`) và `preferred_colors` để phục vụ xếp hạng feed cá nhân hoá. Tính năng feed đã bị gỡ khỏi MVP theo [ADR 002](002-remove-community-resale.md) (2026-06-28).

Sau khi feed bị gỡ, toàn bộ đường dẫn còn lại của bảng là:

-   `UserRepository.GetStyleProfile` (account, identity) — **không có HTTP handler, không expose gRPC/proto, không có bất kỳ caller nào** trong workspace;
-   `UserUseCase.GetStyleProfile` + `IUserUseCase.GetStyleProfile` — chỉ là lớp vỏ gọi xuống repo, cũng không ai gọi;
-   Association `User.StyleProfile` trong entity — gây preload `StyleProfile` trên mọi query user generic;
-   `preferred_colors`/`taste_embedding` **không hề được code ứng dụng ghi** (chỉ có đọc qua nhánh chết); dữ liệu seed từng bị truncate trong cleanup feature 006 (`database/scripts/seed_feature_006_cleanup.sql`).

Nói cách khác: bảng + index pgvector + ~5 lớp code trung gian không còn phục vụ bất cứ hành vi nào của hệ thống.

## 2. Decision

Xoá hoàn toàn bảng và mọi dây chuyền code:

1.  Migration `20261011090000_drop_user_style_profiles.sql`: `DROP TABLE identity.user_style_profiles CASCADE` (index HNSW và FK mất theo bảng). Phần Down tạo lại đúng shape production.
2.  Xoá code: struct `UserStyleProfile` + association trên `User` + `TableName()`, `relations := []string{"StyleProfile"}`, `GetStyleProfile` trên cả 3 tầng (repo/interface/impl của use case), read model `UserStyleProfileRes` và alias `dto/identity/profile.go`.
3.  Dọn fresh-install artifacts: `database/init/01-schema.sql`, `database/init/02-indexes.sql`, `database/scripts/closy-db-schema.sql`, `database/scripts/12-dry_run_preprod_cleanup.sql`.
4.  Cập nhật `docs/system-design/architecture.md` (số bảng 53 → 52, danh sách HNSW index).

## 3. Consequences

-   **Tích cực**: bớt 1 bảng + 1 index HNSW tốn bộ nhớ (pgvector HNSW index khá nặng); mọi query user không còn preload vô ích; bề mặt schema đúng với thực tế sử dụng.
-   **Tiêu cực**: nếu tương lai cần lại feed gu thời trang phải thiết kế lại từ đầu (embedding chưa bao giờ được backend sinh ra, nên rollback ADR 002 không khôi phục được dữ liệu cũ).
-   Type dùng chung `entities.Vector`/`PreferredColors` được **giữ** (fashion module vẫn dùng cho `fashion_items.embedding`).

## 4. Bổ sung (tuỳ chọn, [YYYY-MM-DD])
