# Quickstart - Cá nhân hoá phối đồ và gợi ý bổ sung

**Feature**: `033-personalized-styling` | **Date**: 2026-10-27

Hướng dẫn này là kịch bản kiểm chứng end-to-end. Chi tiết schema ở [data-model.md](./data-model.md),
chi tiết request/response ở [contracts/personalization-api.md](./contracts/personalization-api.md).
Không chứa code triển khai; các bước code nằm ở `tasks.md`.

## 1. Chuẩn bị

- Go 1.25+, Docker, Make, `psql`, `buf` (nếu regenerate proto).
- Làm theo [local-setup.md](../../docs/development/local-setup.md): chạy `database/init/00-init.sql`,
  `01-schema.sql`, `02-indexes.sql`, `03-seed.sql`.
- Biến môi trường tối thiểu: `DATABASE_URL`, `REDIS_*`, `RABBITMQ_*`, AI provider cho embedding.
- Bật migration khi chạy hoặc chạy tay:

```bash
make migrate-up svc=account
make migrate-up svc=commerce
make wire svc=account
make wire svc=commerce
```

- Chạy stack local (HTTP + worker) theo `make dev` hoặc Compose. Worker phải chạy để kiểm chứng learning.

### Cấu hình cần có

```yaml
personalization:
  enabled: true
  max_styles: 4
  style_blend_weight: 0.30
  style_blend_weight_with_request: 0.10
  learned_style_weight: 0.10
  learned_embedding_weight: 0.20
  embedding_timeout_seconds: 2
  learning_decay: 0.9
  gap_score_threshold: 0.35
  suggestion_max_per_role: 1
  suggestion_max_total: 3

rag:
  recently_worn_penalty_days: 7        # nới từ 3
  recently_recommended_penalty_days: 3
  stale_candidate_slots: 4
  stale_eligible_days: 30
  cooldown_score_tolerance: 0.15
```

`personalization.enabled = false` là kill switch (tắt API ghi + áp dụng + learning).

## 2. Chuẩn bị dữ liệu

- Một user thuộc gói `premium-monthly` (hoặc plan có `personalization_enabled = true`).
- Ít nhất 5 active wardrobe item, có `fashion_items.style` và `embedding`.
- Có ít nhất một món trong tủ hệ thống (`wardrobe_items.item_type = 1`, `status = 0`).
- Lấy token user: `export TOKEN=...`.

Kiểm tra nhanh quyền lợi:

```bash
curl -s -H "Authorization: Bearer $TOKEN" \
  http://localhost:8080/api/v1/subscriptions/me
# Kỳ vọng: "personalizationEnabled": true
```

## 3. Kịch bản kiểm chứng

### US1 - Hồ sơ gu + gợi ý đúng gu (P1)

1. `GET /api/v1/me/style-profile` → mặc định `styles: []`, `hasEmbedding: false`.
2. `PUT /api/v1/me/style-profile`:

```bash
curl -s -X PUT -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
  -d '{"styles":["minimalist","elegant"],"excludedStyles":["streetwear"],"enabled":true}' \
  http://localhost:8080/api/v1/me/style-profile
```

Kỳ vọng: 200, `maxStyles: 4`, `hasEmbedding` = true nếu AI provider hoạt động; false nếu fail-soft.

3. `POST /api/v1/ai/outfit-recommendations` với body rỗng.
   Kỳ vọng: `recommendationId` có mặt; outfit có ít nhất một món khớp gu (nếu tủ có món phù hợp);
   explanation nhắc đến gu.
4. Gọi lại với `{"styleTarget":"streetwear"}`.
   Kỳ vọng: request style thắng; không ép minimalist; outfit vẫn hợp lệ.
5. Dùng user free gọi PUT → 403 `PERSONALIZATION_NOT_AVAILABLE`.

### US2 - Gợi ý bổ sung + clone khi lưu (P1)

1. Dùng user có tủ thiếu `giay` (footwear).
2. `POST /api/v1/ai/outfit-recommendations`.
   Kỳ vọng: `suggestions` có role `footwear`; không trùng món đã sở hữu; tối đa 3 món.
3. Lưu outfit có `fashionItemId` của món catalog và `itemContext: "system_catalog"`.
   Kỳ vọng: 201; `wardrobe_items` có row mới của user; outfit đọc lại thấy món thuộc user.
4. Đặt user gần `MaxWardrobeItems`, lưu outfit có 2 món catalog.
   Kỳ vọng: 400 `WARDROBE_LIMIT_EXCEEDED_FOR_OUTFIT`; không có row clone nào được tạo (kiểm tra SQL).
5. Gỡ catalog item khỏi tủ hệ thống rồi lưu → 400 `CATALOG_ITEM_UNAVAILABLE`.
6. Chạy `POST /api/v1/wardrobe-items/catalog-init` với ID item riêng tư của user khác
   → phải bị từ chối (fix bảo mật).

### US3 - Xoay vòng đồ lâu không dùng (P2)

1. Chọn một món, lưu vào outfit gần đây (trong 7 ngày).
2. Gọi recommendation 3 lần liên tiếp.
   Kỳ vọng: món đó không chiếm mọi lần; món lâu không dùng cùng role xuất hiện ít nhất một lần.
3. Kiểm tra log: có `stale_slots_added`, `cooldown_substitutions`,
   `recently_used_repeat_ratio`, `recently_recommended_repeat_ratio`, `returned_item_ids`,
   `stale_item_ids`, `stale_eligible_item_ids`, `cooldown_item_ids`.
4. Đổi tên outfit (không đổi item) → `last_used_at` của các món không đổi.
5. Tạo user có 2 bản sao cùng `fashion_item_id`; kiểm tra scoring dùng `max(last_used_at)`.

### US4 - Học gu từ outfit đã lưu (P2)

1. Ghi lại `learned_styles`/`learned_embedding` trước khi lưu.
2. Lưu outfit có các món style `minimalist`.
3. Trong vòng 60 giây, kiểm tra DB:

```bash
psql "$DATABASE_URL" -c \
"select learned_styles, learned_at from fashion.user_style_profiles where user_id = '$USER_ID';"
```

Kỳ vọng: `learned_styles.minimalist` tăng; `styles` (gu khai) không đổi.
4. Gọi recommendation → gu học ảnh hưởng thứ hạng (soft).
5. Phát lại cùng event (retry/duplicate) → `learned_styles` không cộng hai lần.
6. Update outfit chỉ đổi tên → không có signal mới; thêm một món → có signal chỉ cho món mới.

### US5 - Cold start (P2)

1. User premium chưa có hồ sơ gu, tủ có phân bố style rõ.
2. `GET /api/v1/me/style-profile/suggestions`.
   Kỳ vọng: top style + `itemCount` + `ratio` + `sampleSize`; không tự lưu.
3. `PUT` xác nhận một style → hồ sơ thay đổi.

### US6 - Feedback (P3)

1. Gọi recommendation, lấy `recommendationId`.
2. `POST /api/v1/ai/outfit-recommendations/feedback` với `{"recommendationId":"...","action":"like"}`.
   Kỳ vọng: 200; `learned_styles` tăng nhẹ.
3. Gửi lại cùng action → 200, không tạo signal thứ hai.
4. Dùng token user khác với cùng `recommendationId` → 403 `FEEDBACK_FORBIDDEN`.
5. Chờ snapshot hết hạn (hoặc xoá key Redis) → 404 `RECOMMENDATION_SNAPSHOT_NOT_FOUND`.

### US7 - Hoàn thiện bộ đồ (P3)

1. `POST /api/v1/ai/outfit-completions` với 2 món user sở hữu, thiếu footwear.
   Kỳ vọng: chỉ trả suggestion cho footwear; `remainingQuota` không đổi.
2. Gửi kèm một món của user khác → 400 `OUTFIT_ITEM_INVALID_OR_FORBIDDEN`.
3. Gửi tập đã đủ cấu trúc → `suggestions: []`.

### Bảo trì `.gitignore` (FR-067)

```bash
git check-ignore -v .specify/feature.json   # phải in ra .gitignore
git ls-files --error-unmatch .specify/feature.json  # phải báo không tracked
```

## 4. Kiểm tra tự động

Chạy scoped trong lúc phát triển:

```bash
make lint
go test -race -count=3 ./services/commerce/internal/modules/fashion/...
go test -race -count=3 ./services/commerce/internal/modules/wardrobe/...
go test -race -count=3 ./services/account/internal/modules/subscription/...
```

DB integration (migration + outbox + repo mới):

```bash
TEST_DATABASE_URL=... make test-db-integration
```

Trước khi push:

```bash
make buf-gen && make buf-lint
make migration-check
make check
```

## 5. Kết quả kỳ vọng (map SC)

| SC | Cách kiểm chứng |
|---|---|
| SC-001 | Đo thời gian thao tác PUT hồ sơ gu trên UI/API |
| SC-002 | Đếm `personal_style_match_ratio` trong log 30 ngày |
| SC-003 | Đo `recently_used_repeat_ratio` trong 30 ngày; mục tiêu ≤ 25% |
| SC-004 | Truy vấn `stale_item_ids` trong log 30 ngày; món >30 ngày không dùng xuất hiện ít nhất 1 lần |
| SC-005 | Kiểm tra `suggestions` đúng role, không trùng món sở hữu |
| SC-006 | Kịch bản vượt hạn mức: không có clone mồ côi |
| SC-007 | `learned_at` cập nhật trong 60 giây |
| SC-008 | So sánh `styles` trước/sau learning |
| SC-009 | Kịch bản catalog-init cross-user bị từ chối |
| SC-010 | Đo p95 của endpoint recommendation |
| SC-011 | Chạy client cũ (không gửi field mới) trên response mới |
| SC-012 | So `remainingQuota` trước/sau suggestions/completions |
| SC-013 | Truy vấn log 30 ngày tính được SC-002/SC-003/SC-004 |
| SC-014 | Test T049: suggestion thoả hard filter + gu và không trùng món sở hữu |

## 6. Dọn dẹp

- Xoá outfit/bản sao tạo trong test; hoặc dùng database test riêng.
- Xoá key Redis `personalization:*` nếu muốn reset trạng thái thử nghiệm.
- Không chạy quickstart trên dữ liệu production.
