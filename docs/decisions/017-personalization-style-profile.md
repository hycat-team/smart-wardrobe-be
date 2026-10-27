# ADR 017 - Hồ sơ gu cá nhân hoá và xoay vòng đồ

> Đặt tên file theo `NNN-ten-kebab.md`; quy ước đầy đủ ở [README.md](README.md).

*   **Status**: Accepted
*   **Decision Date**: 2026-10-27
*   **Decision Maker**: Dev
*   **Supersedes**: —
*   **Superseded by**: —

## 1. Context

Feature 033 mở lại phạm vi **recommendation** (gợi ý phối đồ), không phải feed. Người dùng trả phí cần
một hồ sơ gu phong cách để kết quả phối đồ phản ánh sở thích cá nhân, thay vì chỉ dựa vào `styleTarget`
của từng request. Đồng thời pipeline hiện tại có ba vấn đề quan sát được:

1.  Đồ lâu không dùng gần như không bao giờ quay lại pool ứng viên: cửa sổ phạt recency quá ngắn, bonus
    "món mới" bất đối xứng và hệ thống không nhớ món nào đã được gợi ý.
2.  Sau khi tạo outfit, người dùng không được gợi ý món còn thiếu; nếu tự tìm được món trong tủ hệ thống
    thì không có đường lưu hợp lệ (không tự tạo bản sao thuộc người dùng).
3.  Hành vi lưu outfit không được dùng để cải thiện gợi ý lần sau.

ADR [015](015-drop-user-style-profiles.md) đã xoá bảng `identity.user_style_profiles` vì feed bị gỡ và
bảng đó không còn consumer. Việc tái sử dụng bảng cũ sẽ mở lại quyết định 015 mà không có lý do mới, và
đặt embedding vào service `identity` buộc Account phải có dependency AI — điều hiện không tồn tại.

Các ràng buộc bắt buộc:

-   Hiến pháp §II: chỉ giao tiếp qua contract, không import entity/repo chéo module.
-   Hiến pháp §III: side effect chéo module phải đi qua event/outbox, không gọi đồng bộ trong hot path.
-   §VI: không thêm LLM call vào hot path, chi phí kiểm soát được, fail-soft.
-   Hồ sơ gu là **ưu tiên mềm**, không được phá cấu trúc outfit hay biến thành hard filter.

## 2. Decision

### 2.1 Hồ sơ gu thuộc `fashion` (Commerce), không thuộc `identity`

Tạo `fashion.user_style_profiles` (1 row/user) với `styles`, `excluded_styles`, `enabled`,
`style_embedding vector(768)`, `embedding_hash`, `embedding_model`, `learned_styles`, `learned_embedding`,
`learned_at`. Lý do:

-   Commerce đã có `IAIOrchestrator`, config embedding dimension 768, pgvector và toàn bộ consumer của gu.
-   Account không có dependency AI nào; đặt ở Account sẽ buộc thêm embedding provider hoặc sinh embedding
    bất đồng bộ qua service khác.
-   Vector gu chỉ là **query vector** (không dùng để tìm láng giềng), nên không cần index HNSW; đây là
    khác biệt bản chất so với bảng feed đã xoá ở ADR 015.
-   Loại bỏ phương án thêm cột vào `identity.users`/`body_profile` (semantic sai) và tái tạo
    `identity.user_style_profiles` (mở lại ADR 015 vô căn cứ).

Embedding được sinh **một lần khi lưu hồ sơ** (`PUT /me/style-profile`), không sinh lúc gợi ý. Provider
lỗi ⇒ lưu `styles` với `style_embedding = NULL` và `embedding_hash = NULL`, recommendation chuyển sang so
khớp từ khoá.

### 2.2 Quyền lợi đi theo chuỗi plan → snapshot → contract → cache → proto

Thêm `subscription.subscription_plans.personalization_enabled boolean NOT NULL DEFAULT false`.
Migration bật `true` **một lần** cho plan `is_active = true AND price > 0`; gói khuyến mãi giá 0 và gói
tạo sau triển khai mặc định tắt, admin bật thủ công. Cờ runtime là nguồn sự thật, **không** dùng
`price > 0` để quyết định quyền lợi.

Field đi qua hai nhánh của cùng một plan row: (a) `BuildPlanBenefitSnapshot` lúc grant; (b)
`BuildUserSubscriptionOverviewRes` → `UserSubscriptionContractDTO` → `SubscriptionOverviewCacheEntry` →
proto `UserSubscriptionOverview` field 11 → Commerce. Overview đọc plan row hiện tại, không đọc lại
snapshot cũ.

Hệ quả vận hành: overview được cache Redis **60 giây**, nên bật/tắt quyền lợi có dung sai tối đa 60 giây.
Đây là dung sai đã biết (FR-004), không cần sweep cache. Cache entry cũ thiếu field được đọc là `false`
(an toàn); proto field additive nên server cũ trả `default false`, không lỗi. Khi thu hồi quyền lợi, hồ sơ
gu vẫn đọc được nhưng không được áp dụng; dữ liệu gu/signal giữ nguyên.

### 2.3 Áp dụng gu như ưu tiên mềm

-   Blend vector query (không hard filter SQL):
    `q = normalize((1-w)·q + w·normalize(styleVector))`, `w = 0.30` mặc định; hạ về `0.10` khi request có
    `styleTarget` vì request thắng gu lưu sẵn.
-   Scoring: `personal-style-match` (+0.25 khi không có request style, +0.10 khi có), `learned-style-match`
    (+0.10 cấu hình được), `avoid-style` (−0.30).
-   Prompt thêm `personal_style`, `learned_style`, `avoid_styles` và rule chỉ ưu tiên gu khi chất lượng
    ứng viên tương đương.
-   HSL fallback dùng cùng intent/scoring để AI lỗi vẫn giữ mức cá nhân hoá.
-   `enabled = false` trên hồ sơ ⇒ bỏ qua toàn bộ gu, kể cả khi gói còn quyền lợi.
-   Loại bỏ hard filter theo style: phá outfit, phải sửa fallback và structure guard, và đi ngược precedent
    "category là soft filter".

### 2.4 Học gu qua outbox + worker, idempotent

-   Mỗi lần lưu outfit, Wardrobe ghi `wardrobe.outfit.saved` vào outbox trong **cùng transaction**.
    Worker `StyleLearningWorker` consume `style_learning_queue`, insert `fashion.style_signals` với PK
    `id = signalId` (`ON CONFLICT DO NOTHING`) rồi cập nhật hồ sơ trong cùng transaction.
-   `learned_styles = topK(decay·old + sourceWeight·normalizedStyleCounts)`, decay 0.9, K = 10. Trọng số
    nguồn: `outfit_saved = 1.0`, `feedback_like = 0.5`, `feedback_saved = 0.25`, `feedback_dislike = -0.5`.
-   `learned_embedding = normalize(decay·old + (1−decay)·mean(item embeddings))`.
-   Gu **khai báo** không bao giờ bị ghi đè; learned có trọng số thấp hơn và không tạo hard filter.
-   Feedback `like/dislike/saved` dùng chung `ApplySignal`; signal ID deterministic để chống trùng.
    Snapshot recommendation nằm ở Redis `personalization:rec:{recommendationId}` (TTL 7 ngày), server đọc
    item IDs từ snapshot, không tin client.
-   Loại bỏ cập nhật đồng bộ trong `SaveOutfit` (latency + rollback coupling), database trigger (không thấy
    metadata/style) và derive lazy mỗi lần gợi ý (đọc nhiều embedding, khó cache).

### 2.5 Xoay vòng đồ lâu không dùng

-   Thêm `wardrobe.wardrobe_items.last_recommended_at`, cập nhật **best-effort** sau khi trả kết quả
    (primary + alternatives); lỗi ghi chỉ log, lần sau thử lại (FR-054).
-   `last_used_at` chỉ touch cho item **mới thêm** vào outfit.
-   Dedupe theo `fashion_item_id`, lấy `max(last_used_at)` trước scoring.
-   Cửa sổ làm mát: 7 ngày kể từ lần dùng (nới từ 3), 3 ngày kể từ lần được gợi ý; penalty tối đa 0.45 và
    0.12.
-   Dành `stale_candidate_slots = 4` trong pool 20 cho item `last_used_at NULLS FIRST,
    last_recommended_at NULLS FIRST` thoả hard filter (`stale_eligible_days = 30`); không cộng thêm vào
    prompt budget.
-   `ApplyRecencyCooldown`: trong ngưỡng `cooldown_score_tolerance = 0.15`, đưa món ngoài vùng làm mát lên
    trước; nếu cả vai trò đều trong vùng làm mát thì giữ nguyên.
-   Loại bỏ hard rotation và API "đã mặc" (ngoài phạm vi; dùng sự kiện lưu outfit làm proxy).

### 2.6 Kill switch và rollback

`personalization.enabled = false` là kill switch duy nhất: chặn `PUT`/`DELETE` hồ sơ, bỏ qua gu trong
recommendation, worker ack nhưng không ghi. `GET` hồ sơ và catalog suggestions/completions vẫn chạy.
Down migration chỉ drop đúng bảng/cột/index mới; dữ liệu plan hiện có không bị xoá. Proto/cache/JSON đều
additive nên client cũ không bị ảnh hưởng.

## 3. Consequences

**Tích cực**

-   Gu cá nhân ảnh hưởng retrieval mà không phá cấu trúc outfit và không thêm LLM call vào hot path.
-   Đồ lâu không dùng quay lại pool qua stale slots + cooldown, có tín hiệu đo được (`RerankStats`).
-   Học gu bất đồng bộ, không làm chậm/không làm hỏng thao tác lưu outfit; outbox at-least-once nhưng
    idempotent nhờ `signalId`.
-   Quyền lợi đi theo đúng chuỗi plan hiện có, không tạo nguồn sự thật thứ hai.

**Tiêu cực / đánh đổi**

-   Dung sai cache 60 giây khi admin thu hồi quyền lợi là hành vi đã biết, phải ghi tài liệu vận hành.
-   Learning có thể thu hẹp gu (rich-get-richer) nếu decay/top-K sai; giảm thiểu bằng decay 0.9, K = 10,
    trọng số learned thấp và gu khai báo luôn thắng.
-   Stale slots chiếm chỗ ứng viên tốt trong pool 20; cần đo SC-003/SC-004 để phát hiện suy giảm.
-   Snapshot feedback hết hạn sau 7 ngày; feedback muộn trả lỗi rõ ràng, không ảnh hưởng learning từ outfit
    save.
-   Catalog ID cache có TTL 5 phút nên có thể gợi ý món vừa bị gỡ; khi lưu outfit, item vẫn được validate
    lại và bị từ chối nếu không còn hợp lệ.
