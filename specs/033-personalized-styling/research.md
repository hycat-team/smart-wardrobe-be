# Phase 0 Research - Cá nhân hoá phối đồ và gợi ý bổ sung

**Feature**: `033-personalized-styling` | **Date**: 2026-10-27

Mọi điểm chưa rõ trong `plan.md` đã được giải quyết. Tài liệu này ghi lại quyết định, lý do và phương án
bị loại. Tham chiếu code là trạng thái hiện tại của repository.

## R1 - Vị trí hồ sơ gu: Commerce `fashion`, không phải Account `identity`

**Quyết định**: bảng `fashion.user_style_profiles`, 1 row/user, do Commerce sở hữu.

**Lý do**:
- Account service hiện **không có dependency AI nào**. Đặt hồ sơ gu ở Account sẽ buộc phải thêm
  `shared/infrastructure/ai` hoặc gọi chéo service để sinh embedding.
- Commerce đã có `IAIOrchestrator`, config `RecommendationEmbeddingDimension = 768`, pgvector và toàn bộ
  consumer của gu (recommendation, chat, learning worker).
- ADR 015 đã xoá `identity.user_style_profiles` vì feed bị gỡ. Feature này mở lại phạm vi **recommendation**
  (không phải feed), nên cần ADR mới ghi rõ phạm vi thay vì tái tạo bảng cũ.

**Phương án loại bỏ**:
- Thêm cột `style_preferences` vào `users.body_profile` (JSONB): đơn giản nhưng semantic sai, Account phải
  gọi AI, và `body_profile` đang được dùng cho số đo cơ thể.
- Tái tạo bảng trong `identity`: mở lại quyết định 015 mà không có lý do mới; Account vẫn thiếu AI.
- Lưu trong Redis: mất dữ liệu khi restart, không phù hợp dữ liệu người dùng.

## R2 - Quyền lợi gói: boolean trên plan, đi qua chuỗi contract hiện có

**Quyết định**: `subscription_plans.personalization_enabled boolean NOT NULL DEFAULT false`; migration
bật `true` **một lần** cho plan active có `price > 0` như backfill cho premium hiện có; gói giá 0/gói
mới mặc định tắt và do admin bật thủ công. Cờ runtime là nguồn sự thật. Field đi qua
`BuildPlanBenefitSnapshot` → `UserSubscriptionOverviewRes` → `UserSubscriptionContractDTO` →
`SubscriptionOverviewCacheEntry` → proto `UserSubscriptionOverview` (field 11) → Commerce.

**Lý do**:
- Đây là pattern hiện có cho `MaxWardrobeItems`/`MaxOutfits`: plan → snapshot → contract → cache → proto.
  Không tạo nguồn sự thật thứ hai.
- Commerce đã có `userSubContract.GetUserSubscriptionOverview` trong recommendation
  (`usecase.go`), nên không thêm call mới.
- Cache overview TTL 60 giây là backstop chấp nhận được khi admin bật/tắt.

**Phương án loại bỏ**:
- Dùng `plan.Price > 0` làm entitlement **runtime**: không cross-service được (Commerce không thấy
  price), sai với gói khuyến mãi giá 0. Backfill migration dùng `price > 0` chỉ để chọn premium hiện có
  tại thời điểm triển khai; đây không phải quy tắc runtime.
- Thêm middleware gating: repo không có middleware plan; mọi gate hiện nằm trong usecase.
- Gọi Account mỗi lần gợi ý để hỏi quyền lợi: thêm round-trip, trong khi overview đã được lấy cho quota.

## R3 - Style embedding: sinh lúc lưu, blend lúc gợi ý

**Quyết định**:
- PUT hồ sơ gu chuẩn hoá dữ liệu, enrich các style đã biết bằng lexicon nội bộ, gọi
  `IAIOrchestrator.GenerateEmbeddings` một lần với timeout ngắn, lưu `style_embedding` + `embedding_hash`.
- Lỗi/timeout: vẫn lưu `styles`, để vector NULL; recommendation dùng text-level style.
- Khi gợi ý: `q = normalize(q)`, sau đó `q = normalize((1-w)·q + w·normalize(styleVector))`.
- Normalize thủ công sau `AdjustVectorLength` vì hàm này chỉ truncate/pad
  (`shared/pkg/utils/sliceutils/sliceutils.go`).

**Lý do**:
- Không thêm provider call vào hot path; embedding operation đang always-free.
- `embedding_hash` tránh gọi lại khi user bấm lưu nhiều lần mà không đổi gu.
- Fail-soft đúng FR-010/FR-011.

**Phương án loại bỏ**:
- Embed tại request: thêm 100-300ms + một failure mode vào mỗi lần gợi ý.
- Chỉ nhét style text vào semantic query: mất trọng số, bị ngữ cảnh request lấn át, không tận dụng
  được vector.
- Lưu vector không có hash: không biết khi nào cần re-embed sau khi đổi style hoặc đổi model.

## R4 - Catalog suggestions trong ranh giới module

**Quyết định**:
- Wardrobe contract thêm `GetSystemCatalogFashionItemIDs(ctx, limit)` — pure, chỉ đọc repo Wardrobe,
  trả danh sách `fashion_item_id` của `item_type = SystemCatalogItem`, `status = InWardrobe`,
  `is_deleted = false`.
- Fashion cache danh sách này trong Redis key `personalization:catalog:v1`, TTL 5 phút, rồi gọi
  `IFashionItemRepository.GetHybridCandidates` với tập ID đó.
- Loại các `fashion_item_id` người dùng đã sở hữu, lấy qua `GetOwnedFashionItemIDs` (mọi row
  `wardrobe_items` chưa xoá, bất kỳ status).
- Gap detection: role không có món nào hoặc mọi ứng viên dưới `gap_score_threshold`.
- Trả `suggestions` tối đa 1 món/role, 3 món/response.

**Lý do**:
- Không join `wardrobe_items` từ Fashion (vi phạm §II: cấm cross-module DB join).
- `GetHybridCandidates` hiện nhận `fashionItemIDs` — đúng shape để tái sử dụng.
- Catalog ít thay đổi; cache 5 phút đủ giảm tải. Nếu cache miss, gọi contract trực tiếp.

**Phương án loại bỏ**:
- Join trực tiếp `wardrobe_items` trong Fashion: vi phạm ranh giới module.
- Đưa catalog vào prompt: LLM có thể ghép bộ toàn catalog, tăng prompt budget, phá cam kết "phối từ tủ".
- Bảng projection `fashion.system_catalog_index`: đồng bộ khi admin thêm/xoá món, phức tạp không cần.
- Quét toàn bộ catalog mỗi request: tốn DB, không cần khi đã có HNSW trên `fashion_items.embedding`.

## R5 - Clone catalog trong cùng transaction với outfit

**Quyết định**:
- `SaveOutfitItemReq` thêm `itemContext` optional.
- Catalog hợp lệ → tạo row `wardrobe_items` mới (`UserID`, `FashionItemID`, `Status = InWardrobe`,
  `ItemType = UserItem`, copy `Price`).
- Outfit item dùng cùng `fashion_item_id`, `item_context = user_wardrobe`.
- Toàn bộ trong `IUnitOfWork`; kiểm tra `MaxWardrobeItems` trước khi clone.
- Fix `InitClosetFromCatalog` chỉ nhận `item_type = SystemCatalogItem` và `is_deleted = false`.

**Lý do**:
- `outfit_items` lưu `fashion_item_id`, không lưu `wardrobe_item_id`; clone tái sử dụng đúng fashion item.
- Ownership được bảo đảm bằng row `wardrobe_items` của user; `outfit_items.item_context` vẫn là
  `user_wardrobe`, không cần thêm giá trị mới vào CHECK constraint.
- UoW tránh bản sao mồ côi và outfit thiếu món.

**Phương án loại bỏ**:
- Thêm `item_context = system_catalog` vào `outfit_items`: phải sửa toàn bộ read/delete path và CHECK
  constraint, rủi ro cao hơn nhiều.
- Clone sau khi lưu outfit: dữ liệu dở dang nếu clone lỗi.
- Tái sử dụng `InitClosetFromCatalog`: đang thiếu validate `item_type`/`is_deleted`, là lỗ hổng
  cross-user; phải fix và không dùng cho luồng save.

## R6 - Learning qua outbox + worker

**Quyết định**:
- Topic `wardrobe.outfit.saved`; queue `style_learning_queue`; routing key tương ứng.
- Payload: `signalId`, `userId`, `outfitId`, `fashionItemIds`, `source`, `occurredAt`.
- Outbox record ghi trong cùng transaction với outfit.
- Worker `StyleLearningWorker` insert `fashion.style_signals` (PK `id` = payload `signalId`,
  ON CONFLICT DO NOTHING) rồi cập nhật hồ sơ trong cùng transaction.
- `learned_styles` decay 0.9, top-K 10; `learned_embedding` decay 0.9.
- `UpdateOutfit` chỉ phát item mới thêm.

**Lý do**:
- §III yêu cầu cross-module write side effect qua event; outfit (Wardrobe) và learning (Fashion) là hai
  module.
- At-least-once của outbox + PK signal_id = exactly-once về hiệu lực.
- Worker tách khỏi HTTP, không làm chậm/không làm hỏng save.

**Phương án loại bỏ**:
- Cập nhật đồng bộ trong `SaveOutfit`: thêm latency và coupling rollback.
- Database trigger: không thấy metadata style/embedding và khó test.
- Derive lazy mỗi lần gợi ý: đọc nhiều embedding, khó cache, chậm.

## R7 - Rotation: tín hiệu, stale slots, cooldown

**Quyết định**:
- Thêm `last_recommended_at`.
- Dedupe theo `fashion_item_id`, lấy `max(last_used_at)`.
- Vùng làm mát: dùng trong 7 ngày hoặc gợi ý trong 3 ngày.
- Nới `recently_worn_penalty_days` 3 → 7, penalty tối đa 0.45; thêm penalty 0.12 cho gợi ý gần đây.
- "Món mới" theo `created_at`; long-unworn bonus không phụ thuộc style/occasion.
- `stale_candidate_slots` (4) nằm trong pool 20, không cộng thêm.
- `ApplyRecencyCooldown`: ưu tiên món ngoài vùng làm mát trong ngưỡng 0.15.

**Lý do**: sửa đúng các nguyên nhân đã phân tích: cửa sổ quá ngắn, bonus bất đối xứng, chỉ rerank, không
nhớ đã gợi ý, touch sai khi update metadata, bản sao trùng, LLM không được dặn.

**Phương án loại bỏ**:
- Chỉ tăng penalty: đồ cũ vẫn không vào retrieval pool.
- Hard rotation: có thể tạo outfit kém, phá structure.
- Thêm API "đã mặc": đã chốt dùng save làm proxy trong phạm vi này.

## R8 - Feedback: snapshot Redis + signal deterministic

**Quyết định**:
- Mỗi recommendation sinh `recommendationId`; ghi snapshot Redis TTL 7 ngày.
- Feedback chỉ nhận `recommendationId + action`; backend đọc snapshot, xác thực `userId`, lấy item IDs.
- Signal ID deterministic = hash(userId + recommendationId + action), chống trùng.
- Feedback gọi cùng `StyleLearningUseCase.ApplySignal`.

**Lý do**: tránh bảng snapshot tăng trưởng; không tin item IDs từ client; Redis đã có.

**Phương án loại bỏ**:
- Bảng snapshot DB: tăng trưởng theo lượt gợi ý, không cần audit ở v1.
- Token HMAC: thêm phức tạp cho FE, không cần khi đã có Redis.
- Không xác thực: feedback giả có thể bẻ gu người khác.

## R9 - Cold start: phân bố tủ đồ, không auto-apply

**Quyết định**: `GET /me/style-profile/suggestions` dùng active items + `fashionItemRepo.GetByIDs` để đếm
style, normalize nhãn đã biết, trả top style + `itemCount` + `ratio` + `sampleSize`. Không tự lưu.

**Lý do**: dữ liệu sẵn có, không cần AI; FR-046 cấm tự áp dụng. Vector suy ra từ tủ đồ (inferred
embedding) bị **deferred**: áp dụng nó trước khi người dùng xác nhận sẽ mâu thuẫn với tinh thần "gợi ý
không tự áp dụng".

**Phương án loại bỏ**:
- Tự sinh vector trung bình và dùng làm prior ngay: vi phạm FR-046.
- Dùng LLM sinh style labels: tốn chi phí, không cần.

## R10 - Observability: log + stats, không dashboard

**Quyết định**: mở rộng `RerankStats` (`StaleSlotsAdded`, `CooldownSubstitutions`,
`RecentlyUsedRepeatRatio`, `RecentlyRecommendedRepeatRatio`, `PersonalStyleMatchRatio`,
`ReturnedItemIDs`, `StaleItemIDs`, `StaleEligibleItemIDs`, `CooldownItemIDs`); log structured mỗi lần
gợi ý với `recommendationId`, internal user ID, `personalized`, đầy đủ counter và danh sách ID trên.

**Lý do**: đủ tính SC-003/SC-004 bằng truy vấn log; không mở rộng scope vận hành; không log PII.

**Phương án loại bỏ**: dashboard admin mới (ngoài phạm vi); chỉ log cũ (không có counter).

## R11 - Tương thích proto/cache

**Quyết định**: proto field 11 additive; cache entry thêm field `omitempty`; client cũ đọc được; cache cũ
thiếu field → `false`.

**Lý do**: backward compatibility; rollback an toàn.

**Phương án loại bỏ**: đổi field cũ; backfill `last_recommended_at` (không có nguồn dữ liệu).

## R12 - `.specify/feature.json`

**Quyết định**: thêm vào `.gitignore` và `git rm --cached` (file vẫn còn trên disk).

**Lý do**: file cục bộ của công cụ, đã bị track; chỉ thêm ignore sẽ không có tác dụng.

**Phương án loại bỏ**: chỉ thêm ignore (file vẫn hiện trong `git status`).
