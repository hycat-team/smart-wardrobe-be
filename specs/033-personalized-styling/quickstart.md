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

- Chạy stack local (HTTP + worker) theo `make docker-dev-up` hoặc Compose. (`make dev` chỉ chạy
  `generate` + `build-all`, **không** khởi động service.) Worker phải chạy để kiểm chứng learning.

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

## 7. Kết quả kiểm chứng

Ghi tại phase close-out (T085), ngày 2026-10-27. Mục này chỉ ghi **việc đã thực sự chạy** và
**việc chưa chạy**, không suy diễn kết quả.

### 7.1 Môi trường

| Hạng mục | Giá trị |
|---|---|
| Go | `go1.26.5 windows/amd64` (`go.work` khai báo 1.26.0) |
| Container đang chạy | `postgres-closy-dev`, `rabbitmq-closy-dev`, `redis-closy-dev` (đều healthy) |
| Dịch vụ HTTP (account/commerce/ai) | **không chạy** |
| Worker learning | **không chạy** |
| `TEST_DATABASE_URL` | **chưa đặt**; các DB scratch hiện có (`smart_wardrobe_test`, `closy_test_027`, `verify_plan031`) **chưa** có bảng `fashion.user_style_profiles` / `fashion.style_signals`, tức chưa migrate schema 033 |
| AI provider | có key trong `.env` nhưng **không gọi** trong phiên này (không có service chạy) |

### 7.2 Đã chạy trong phase close-out

| Lệnh | Kết quả |
|---|---|
| `make SHELL=C:/Windows/System32/cmd.exe generate` (lần 1) | **FAIL** ở `swagger-combine`: `path /api/v1/me/style-profile: tag "Style Profile" missing from mapping` |
| Sửa `shared/tools/swagger-combine/module-mapping.json`: thêm tag `Style Profile` vào module `fashion` | — |
| `make SHELL=C:/Windows/System32/cmd.exe generate` (lần 2) | **PASS**: wire 3 service, swagger 3 service, combine (3 services / 9 modules), nginx-routes (43 locations) |
| `go test -race -count=3 ./services/commerce/internal/modules/fashion/...` | **PASS** toàn bộ package có test, gồm `personalization/{profile,feedback,learning,completion}`, `ai/recommendation{,...}`, `presentation/handler` |
| `go test -race -count=3 ./services/commerce/internal/modules/wardrobe/...` | **PASS**, gồm `usecase/outfit` (clone-on-save), `usecase/wardrobe/catalog` (catalog-init) |
| `go test -race -count=3 ./services/account/internal/modules/subscription/...` | **PASS** toàn bộ package có test |
| `go test ./shared/tools/swagger-combine/... ./shared/tools/nginx-routes/...` | **PASS** |
| `make SHELL=C:/Windows/System32/cmd.exe lint` | **PASS** (`fmt-check` + `go vet` toàn workspace) |
| `git status --porcelain` / `git diff --stat` | artifact sinh tự động đã đổi đúng phạm vi 033 (xem §7.5) |

### 7.3 Đã chạy ở phase trước, **không chạy lại** trong phase này

Các kết quả dưới đây là số liệu báo cáo của phase 2/phase 3, chép nguyên trạng; phase close-out
không tái lập được vì thiếu `TEST_DATABASE_URL` và DB scratch chưa migrate schema 033.

- Phase 2 — DB integration: **12/12** test `TestIntegration*` của Commerce (migration, outbox,
  repository mới).
- Phase 3 — DB integration repository hồ sơ gu: **4/4** test.
- Các unit test theo package đã xanh ở từng phase; phase close-out chạy lại toàn bộ scoped ở §7.2.

### 7.4 Chưa chạy và lý do

| Kịch bản | Vì sao chưa chạy |
|---|---|
| US1–US7 end-to-end (§3) | Cần Account + Commerce + AI chạy HTTP, worker learning, subscription `personalization-monthly`/plan có `personalization_enabled = true`, catalog seed và AI provider. Phiên này chỉ có Postgres/Redis/RabbitMQ. |
| US4 learning (event → `learned_styles` trong 60 s) | Cần worker `style_learning_queue` chạy cùng RabbitMQ và DB đã migrate. |
| US2 bước 4/5 (vượt hạn mức, catalog item bị gỡ) | Cần dữ liệu tủ/catalog thật và gọi HTTP; phần logic đã có unit test `usecase/outfit` + `usecase/wardrobe/catalog`. |
| SC-002/SC-003/SC-004/SC-013 (phân tích log 30 ngày) | Không có log 30 ngày của môi trường chạy thật. |
| SC-010 (p95) | Xem T090 — **NOT MEASURED**, lý do và lệnh tái lập ở `docs/development/technical-notes.md`. |
| SC-011 (client cũ chạy trên response mới) | Không chạy web/mobile runtime trong phiên này; đã đối chiếu tĩnh ở `frontend-guide.md` §15.5. |
| Kiểm chứng trên dữ liệu production | Ngoài phạm vi, bị cấm bởi §6. |

### 7.5 Artifact sinh tự động đã thay đổi (T086)

`make generate` chỉ tạo thay đổi thuộc 033:

- `services/{account,commerce}/api/swagger/{docs.go,swagger.json,swagger.yaml}` và
  `services/account/api/swagger/combined/**`: thêm route/style-profile, `personalizationEnabled`,
  các DTO `CompleteOutfit*`, `SubmitRecommendationFeedback*`, `StyleProfile*`.
- `deployments/nginx/local.conf` và 3 template (`aws/api-v2`, `services/vps-3service`,
  `services-redirect/vps-3service`): thêm `location ^~ /api/v1/me/style-profile` và
  `location ^~ /api/v1/ai/outfit-completions`.
- `shared/tools/swagger-combine/module-mapping.json`: thêm tag `Style Profile` (bắt buộc để
  `swagger-combine` chạy được; không phải thay đổi hành vi API).

Các file `docs/**` và `Makefile` đang `M` trong `git status` là thay đổi **có sẵn từ trước** phase
này, không do `make generate` sinh ra.

### 7.6 Lệnh tái lập

```bash
# Sinh artifact (wire + swagger + bundle + nginx)
make generate

# Unit test scoped, race, lặp 3 lần
go test -race -count=3 ./services/commerce/internal/modules/fashion/...
go test -race -count=3 ./services/commerce/internal/modules/wardrobe/...
go test -race -count=3 ./services/account/internal/modules/subscription/...

# Test của tool sinh artifact
go test ./shared/tools/swagger-combine/... ./shared/tools/nginx-routes/...

# Lint (Windows/PowerShell cần chỉ định shell)
make SHELL=C:/Windows/System32/cmd.exe lint
```

DB integration (cần scratch DB đã migrate đủ 3 service — xem
[local-setup.md](../../docs/development/local-setup.md) mục "Database-backed integration tests"):

```bash
export TEST_DATABASE_URL="postgres://admin:123456@localhost:5432/verify_it?sslmode=disable&search_path=public,identity,wardrobe,fashion,brand,subscription,aiexecutor,dashboard,shared,community"
export TEST_DATABASE_ALLOW_REGISTRY_ROLLBACK=true
make test-db-integration
```

## 8. Rà soát bảo mật (T089)

Phạm vi: các đường mới/đổi của 033. Kết luận dựa trên đọc mã nguồn + unit test hiện có; **không**
dựa trên pentest hay chạy E2E.

| # | Hạng mục | Kết luận | Bằng chứng |
|---|---|---|---|
| 1 | `POST /wardrobe-items/catalog-init` với ID item riêng tư của user khác | **ĐÃ VÁ — từ chối** | `services/commerce/internal/modules/wardrobe/application/usecase/wardrobe/catalog/catalog_uc.go` chỉ nhận template `item_type = SystemCatalogItem`, `status = InWardrobe`, chưa xoá mềm; ID không thoả → `ErrCatalogItemNotFound()` (404). Không clone một phần: toàn bộ batch được validate trước `BulkCreate`. Test: `TestInitClosetFromCatalogRejectsIneligibleTemplates`, `TestInitClosetFromCatalogRejectsMixedBatchWithoutPartialClone` |
| 2 | Quyền sở hữu khi feedback | **ĐẠT** | `feedback_uc.go` đọc snapshot theo `recommendationId`, so `snapshot.UserID` với user trong token; khác → 403 `FEEDBACK_FORBIDDEN`. Item/style lấy từ snapshot, **không** tin client. Test: `TestSubmitFeedbackRejectsForeignSnapshot` |
| 3 | Quyền sở hữu khi completions | **ĐẠT (fail-closed)** | `completion_uc.go` kiểm từng `fashionItemId` thuộc active wardrobe của user; thiếu một món → 400 `OUTFIT_ITEM_INVALID_OR_FORBIDDEN`; contract nil → từ chối. Test: `TestCompleteOutfitRejectsItemNotOwned` |
| 4 | PII trong log | **ĐẠT (có ghi chú)** | Log của 033 chỉ dùng `user_id`, `recommendation_id`, `signalId`, `outfit_id`, danh sách item id, `reason`, số đếm. Không log email/tên/số điện thoại/token/body. `workflow.go` log `prompt_len` (không log prompt) và `response_preview` = 600 ký tự đầu của **response LLM** (xem rủi ro 3) |
| 5 | Atomicity của clone khi lưu outfit | **ĐẠT trong một request** | `SaveOutfit`/`UpdateOutfit` chạy trong một `uow.Execute`: kiểm hạn mức → tạo outfit + item → `BulkCreate` bản sao → `TouchLastUsedAt` → publish outbox. Lỗi ở bất kỳ bước nào rollback cả transaction ⇒ không có bản sao mồ côi. Test: `outfit_uc_clone_test.go` |
| 6 | Race của hạn mức khi clone | **ĐÃ VÁ cho outfit/catalog; còn 2 đường khác (xem 8.1)** | `enforceOutfitCloneLimit` và `InitClosetFromCatalog` gọi `LockUserWardrobe` (`pg_advisory_xact_lock(hashtext(userID))`) trước `CountByUserID`, trong cùng UoW ⇒ hai request cùng user bị serialize. Test thứ tự lock→count trong `outfit_uc_clone_test.go` và `catalog_uc_test.go`. `CloneWardrobeItem` và `BatchUploadWardrobeItems` vẫn đếm ngoài transaction/không lock → `deferred.md` D-10. |
| 7 | Kill switch `personalization.enabled = false` | **ĐẠT** | `profile_uc.UpdateProfile/DeleteProfile` → 403 `PERSONALIZATION_DISABLED`; `GetProfile`/`SuggestStyles` vẫn đọc; `loadPersonalizationContext` trả context rỗng (bỏ qua gu đã lưu nhưng vẫn trả outfit + suggestions); `learning_uc.ApplySignal` ack và không ghi signal; completions không kiểm kill switch |

### 8.1 Rủi ro tồn dư và khuyến nghị

1. **Hạn mức tủ — đã đóng cho outfit/catalog (mức trung bình).** `SaveOutfit`/`UpdateOutfit` và
   `InitClosetFromCatalog` đã lấy `pg_advisory_xact_lock(hashtext($userID))` trước khi đếm.
   `CloneWardrobeItem` và `BatchUploadWardrobeItems` vẫn chưa lấy lock (đếm ngoài transaction) — đã ghi
   `deferred.md` D-10; cần một task riêng để nối hai đường này. Test đồng thời 2 goroutine trên DB thật
   vẫn nên chạy khi có `TEST_DATABASE_URL`.
2. **Oracle tồn tại recommendation (mức thấp).** Feedback trả 404 cho id không tồn tại và 403 cho
   id của user khác. Kẻ tấn công đã có token hợp lệ và đoán được UUID của người khác mới phân biệt
   được; UUID do server sinh và chỉ trả cho chủ sở hữu. Chấp nhận được.
3. **Nội dung free-text có thể vào log qua response của LLM (mức thấp).** `workflow.go` ghi
   `response_preview` = `stringutils.TruncateLogText(response, 600)` — 600 ký tự đầu của response
   LLM. Prompt không bị log, nhưng nếu model lặp lại `details` người dùng tự nhập vào câu trả lời
   thì đoạn text đó xuất hiện trong log. Tương tự, `profile_helper.go` log `zap.Error(err)` khi
   embedding thất bại; provider lỗi có thể trả nguyên văn input (nhãn style, không phải PII).
   Không log email/tên/token. Nếu log là nguồn chia sẻ rộng, cân nhắc hạ `response_preview` xuống
   mức debug hoặc mask.
4. **Kill switch làm mất signal đã phát.** Khi tắt, worker ack và không ghi signal (đúng hợp đồng),
   nên signal phát trong thời gian tắt không được học lại sau khi bật. Đây là hành vi đã chốt, cần
   vận hành biết khi bật/tắt.
5. **Catalog-init trả 404 cho cả ID không tồn tại và ID của user khác.** Có lợi cho bảo mật
   (không phân biệt được), nhưng client phải hiểu 404 ở endpoint này là "không phải món hệ thống
   hợp lệ", không phải "route sai".
