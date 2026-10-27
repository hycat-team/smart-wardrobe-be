# Phase 1 Data Model - Cá nhân hoá phối đồ và gợi ý bổ sung

**Feature**: `033-personalized-styling` | **Date**: 2026-10-27

Mô hình dữ liệu này là additive. Không đổi tên/kiểu/ý nghĩa của bất kỳ cột hiện có nào. Hai migration
thuộc hai service theo ownership contract 030:

- Account: `services/account/database/migrations/` — `subscription.subscription_plans`.
- Commerce: `services/commerce/database/migrations/` — `fashion.*` và `wardrobe.wardrobe_items`.

Cross-schema FK tới `identity.users(id)` được phép theo ownership contract §2 (ngoại lệ
`REFERENCES identity.users(`).

## 1. Account delta - `subscription.subscription_plans`

| Column | Type | Constraints | Notes |
|---|---|---|---|
| `personalization_enabled` | `boolean` | `NOT NULL DEFAULT false` | Quyền lợi cá nhân hoá; free mặc định tắt |

Migration Up:

```sql
ALTER TABLE subscription.subscription_plans
    ADD COLUMN IF NOT EXISTS personalization_enabled boolean NOT NULL DEFAULT false;

UPDATE subscription.subscription_plans
SET personalization_enabled = true,
    updated_at = now()
WHERE is_active = true
  AND price > 0;
```

Migration Down: `ALTER TABLE ... DROP COLUMN IF EXISTS personalization_enabled;`

Đây là **backfill một lần** cho các gói premium đang tồn tại tại thời điểm triển khai. Gói khuyến mãi
giá 0 và gói tạo sau khi triển khai mặc định tắt; admin bật thủ công. Cờ runtime là nguồn sự thật, không
phải `price > 0`.

Không backfill `user_subscriptions.current_benefit_snapshot`: overview được dựng từ **plan row** hiện tại
(`BuildUserSubscriptionOverviewRes(sub, plan)`), không đọc field này từ snapshot. Snapshot cũ thiếu key là
chấp nhận được; lần grant kế tiếp sẽ có. Nếu sau này AI engine cần đọc từ snapshot, phải thêm backfill
riêng.

## 2. Commerce - `fashion.user_style_profiles`

Một row/user. Không dùng HNSW index vì vector gu chỉ là **query vector**, không phải dữ liệu được tìm
kiếm tương tự.

| Column | Type | Constraints | Notes |
|---|---|---|---|
| `user_id` | `uuid` | PK, FK `identity.users(id) ON DELETE CASCADE` | 1 row/user; xoá tài khoản xoá gu |
| `styles` | `jsonb` | `NOT NULL DEFAULT '[]'::jsonb` | Mảng chuỗi phong cách yêu thích |
| `excluded_styles` | `jsonb` | `NOT NULL DEFAULT '[]'::jsonb` | Mảng chuỗi phong cách muốn tránh |
| `enabled` | `boolean` | `NOT NULL DEFAULT true` | Người dùng có thể tắt mà không xoá dữ liệu |
| `style_embedding` | `vector(768)` | NULL | Vector của tập style; sinh khi PUT |
| `embedding_hash` | `text` | NULL | hash(styles + excluded + model); bỏ qua embed nếu không đổi |
| `embedding_model` | `varchar(100)` | NULL | model đã dùng; phục vụ re-embed khi đổi model |
| `learned_styles` | `jsonb` | `NOT NULL DEFAULT '{}'::jsonb` | Map `style -> weight` do hệ thống suy ra |
| `learned_embedding` | `vector(768)` | NULL | Vector trung bình có suy giảm từ outfit đã lưu |
| `learned_at` | `timestamptz` | NULL | Lần cập nhật learned gần nhất |
| `created_at` | `timestamptz` | `NOT NULL DEFAULT now()` | |
| `updated_at` | `timestamptz` | `NOT NULL DEFAULT now()` | |

Checks:

```sql
CHECK (jsonb_typeof(styles) = 'array')
CHECK (jsonb_typeof(excluded_styles) = 'array')
CHECK (jsonb_typeof(learned_styles) = 'object')
CHECK (jsonb_array_length(styles) <= 50)
CHECK (jsonb_array_length(excluded_styles) <= 50)
```

Giới hạn 50 ở DB chỉ là safety net; usecase áp dụng giới hạn cấu hình (mặc định 4) và trả lỗi thân thiện.

## 3. Commerce - `fashion.style_signals`

Append-only, dùng chung cho outfit save + feedback.

| Column | Type | Constraints | Notes |
|---|---|---|---|
| `id` | `uuid` | PK | `signalId`; outbox retry dùng lại đúng ID |
| `user_id` | `uuid` | `NOT NULL`, FK `identity.users(id) ON DELETE CASCADE` | |
| `source` | `smallint` | `NOT NULL` | `1 outfit_saved`, `2 feedback_like`, `3 feedback_dislike`, `4 feedback_saved` |
| `outfit_id` | `uuid` | NULL | Không FK: outfit có thể bị xoá nhưng signal là lịch sử |
| `recommendation_id` | `uuid` | NULL | Chỉ có với feedback |
| `fashion_item_ids` | `uuid[]` | `NOT NULL` | Snapshot món tham gia; không FK để giữ lịch sử |
| `style_snapshot` | `jsonb` | NULL | Snapshot gu tại thời điểm signal (audit/debug) |
| `created_at` | `timestamptz` | `NOT NULL DEFAULT now()` | |

Checks và indexes:

```sql
CHECK (source IN (1, 2, 3, 4))
CHECK (cardinality(fashion_item_ids) > 0)

CREATE INDEX idx_style_signals_user_created
    ON fashion.style_signals (user_id, created_at DESC);

CREATE UNIQUE INDEX ux_style_signals_feedback
    ON fashion.style_signals (user_id, recommendation_id, source)
    WHERE recommendation_id IS NOT NULL;
```

`ux_style_signals_feedback` chống trùng feedback; outfit save không có `recommendation_id` nên dựa vào
PK `id` (sinh một lần trong payload outbox).

Trọng số học theo nguồn (dùng trong usecase, không lưu DB): `outfit_saved = 1.0`, `feedback_like = 0.5`,
`feedback_saved = 0.25`, `feedback_dislike = -0.5`.

## 4. Commerce delta - `wardrobe.wardrobe_items`

| Column | Type | Constraints | Notes |
|---|---|---|---|
| `last_recommended_at` | `timestamptz` | NULL | Lần gần nhất món được trả về trong recommendation |

Không backfill (không có nguồn). Không thêm index: truy vấn stale lọc theo `user_id`/`status`/`item_type`
(đã có `idx_wardrobe_items_user_status`) và sắp xếp tối đa vài trăm row/user; đo lại nếu SC-010 có vấn đề.

Migration Down: drop column.

## 5. Redis keys

| Key | Value | TTL | Mục đích |
|---|---|---|---|
| `personalization:rec:{recommendationId}` | JSON `{userId, fashionItemIds, styles, createdAt}` | 7 ngày | Xác thực feedback + lấy item IDs |
| `personalization:catalog:v1` | JSON `{ids: [uuid...], fetchedAt}` | 5 phút | Cache fashion item ID của tủ hệ thống |

Cache lỗi/timeout MUST degrade thành miss (gọi contract/repo), không làm hỏng request.

## 5b. Bản ghi quan sát gợi ý (không có bảng)

Spec §Key Entities mô tả "Bản ghi quan sát gợi ý"; feature này **không** lưu thành bảng. Nó là một dòng
structured log cho mỗi recommendation (xem plan D11), gồm `recommendationId`, internal user id,
`StaleSlotsAdded`, `CooldownSubstitutions`, `RecentlyUsedRepeatRatio`, `RecentlyRecommendedRepeatRatio`,
`PersonalStyleMatchRatio`, `ReturnedItemIDs`, `StaleItemIDs`, `StaleEligibleItemIDs`,
`CooldownItemIDs`. Log đủ để tính
SC-002/SC-003/SC-004 bằng truy vấn vận hành; không chứa PII ngoài internal user id.

## 6. Quan hệ

- `fashion.user_style_profiles` 1:1 `identity.users`.
- `fashion.style_signals` N:1 `identity.users`; N:1 `wardrobe.outfits` (logical, không FK);
  N:1 recommendation (logical, qua `recommendation_id`).
- `wardrobe.wardrobe_items` N:1 `fashion.fashion_items` (hiện không FK sau 030; giữ nguyên).
- `wardrobe.outfit_items` N:1 `fashion.fashion_items` (hiện không FK; clone tái sử dụng cùng fashion item).

## 7. Quy tắc validate (từ spec)

- `styles` và `excluded_styles`: trim, bỏ rỗng, dedupe không phân biệt hoa thường, mỗi giá trị ≤ 50 ký tự,
  tổng mỗi danh sách ≤ `personalization.max_styles` (mặc định 4).
- Danh sách `styles` rỗng hợp lệ (nghĩa là "không có gu ưu tiên"); `excluded_styles` vẫn được áp dụng.
- `enabled` mặc định true; tắt không xoá dữ liệu.
- Không validate giá trị theo từ vựng; giá trị lạ đi qua nhánh embedding/text-level.
- Feedback `action` chỉ nhận `like|dislike|saved`; `saved` là tín hiệu nhẹ để tránh cộng hai lần với
  outfit save.
- `recommendationId` phải tồn tại trong Redis và thuộc đúng user; nếu không → từ chối.
- Catalog clone chỉ nhận item `SystemCatalogItem`, `InWardrobe`, chưa xoá, có `fashion_item_id`.
- "Sở hữu" cho mục đích dedupe/clone = tồn tại row `wardrobe_items` chưa xoá của user, **bất kỳ status**
  (kể cả Selling/Sold/Processing); nếu đã sở hữu thì tái sử dụng row, không clone thêm.

## 8. State transitions

| Đối tượng | Trạng thái | Chuyển |
|---|---|---|
| Hồ sơ gu | Chưa có | Tạo lazy khi PUT hoặc khi nhận signal đầu tiên |
| Hồ sơ gu | Bật | Người dùng tắt → `enabled = false`; dữ liệu giữ nguyên |
| Quyền lợi | Bật | Plan hết hạn/hạ gói → đọc được, không sửa, không áp dụng; dữ liệu giữ nguyên |
| Learned | Rỗng | Mỗi signal: decay + cộng/trừ; top-K 10; không reset trong v1 |
| Signal | — | Append-only; xoá tài khoản cascade |
| `last_recommended_at` | NULL | Cập nhật best-effort sau mỗi recommendation; lỗi ghi được log, lần gợi ý kế tiếp thử lại (FR-054) |
| `last_used_at` | NULL | Cập nhật khi lưu outfit (item mới) hoặc thêm vào outfit |
| Recommendation snapshot | — | Tạo khi trả kết quả; hết hạn sau 7 ngày |

## 9. Migration plan

### Account

`<ts>_add_plan_personalization.sql`:

```sql
-- +goose Up
-- +goose StatementBegin
ALTER TABLE subscription.subscription_plans
    ADD COLUMN IF NOT EXISTS personalization_enabled boolean NOT NULL DEFAULT false;

UPDATE subscription.subscription_plans
SET personalization_enabled = true,
    updated_at = now()
WHERE is_active = true
  AND price > 0;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE subscription.subscription_plans
    DROP COLUMN IF EXISTS personalization_enabled;
-- +goose StatementEnd
```

### Commerce

`<ts>_add_personalization.sql`:

- `CREATE TABLE IF NOT EXISTS fashion.user_style_profiles ...`
- `CREATE TABLE IF NOT EXISTS fashion.style_signals ...`
- `ALTER TABLE wardrobe.wardrobe_items ADD COLUMN IF NOT EXISTS last_recommended_at timestamptz;`
- Indexes + checks + FKs.
- Down: drop tables (theo thứ tự signal → profile), drop column.

Migration chạy online: thêm cột nullable/default và bảng mới, không rewrite bảng lớn. Thứ tự deploy:
Account trước (proto field mới), Commerce sau. Nếu Commerce chạy trước, proto field mặc định `false` → tính
năng tắt, không lỗi.

## 10. Backfill

- Plan: bật cờ cho plan active có giá > 0 (trong migration).
- Không backfill hồ sơ gu, signal, `last_recommended_at`.
- Không học hồi tố từ outfit cũ; spec đã ghi rõ ngoài phạm vi.

## 11. Rollback

- Down migration drop đúng cột/bảng/index mới.
- `personalization.enabled = false` trong config là kill switch: tắt API ghi, tắt áp dụng gu, tắt learning
  mà không cần rollback schema; dữ liệu vẫn giữ.
- Rollback Commerce trước, Account sau (thứ tự ngược deploy). Proto field additive nên client cũ không bị
  ảnh hưởng.

## 12. Data volume & retention

- 200k user → 200k profile rows (nhỏ).
- Signal: giả định 10 outfit save/user/tháng → ~2M rows/tháng ở quy mô 200k; index
  `(user_id, created_at DESC)` phục vụ đọc theo user. Signal giữ theo vòng đời tài khoản (cascade); nếu
  tăng trưởng vượt dự kiến, cân nhắc partition theo tháng ở feature sau.
- Learned data nằm trong profile row, không tăng theo signal.
- Redis snapshot: TTL 7 ngày; số key tỉ lệ với recommendation của user đang hoạt động, không tăng vĩnh viễn.
- Catalog cache: 1 key, TTL 5 phút, kích thước vài nghìn UUID.
