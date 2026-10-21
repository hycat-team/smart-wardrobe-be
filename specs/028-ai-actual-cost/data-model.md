# Data Model: Chi phí AI thật phục vụ báo cáo lợi nhuận (028)

> Không thay đổi schema (xem `research.md` R1). Tài liệu này mô tả ngữ nghĩa
> mới của các trường **hiện có** và hai giá trị `error_code` mới.

## 1. `ai_usage_events` (ngữ nghĩa sau tính năng)

| Trường (hiện có) | Ngữ nghĩa mới |
|---|---|
| `prompt_tokens`, `output_tokens`, `thinking_tokens` | Số token thật. Vision: từ `usageMetadata`. Embedding: từ `countTokens` đếm inline khi thành công; 0 kèm `usage_unreported` chỉ khi đếm thất bại. |
| `actual_cost_micro_vnd` | **Chi phí thật = `microCost(tokens, pricing model thật)` cho mọi capability billed**, kể cả khi `logical_route='free'` (ví dụ vision). Bằng 0 khi: capability nằm trong `AI_UNBILLED_CAPABILITIES` (key miễn phí), model giá 0, thiếu giá, hoặc thiếu số token. |
| `logical_route` | Không đổi. `paid` = có tính cho user; `free` = miễn phí cho user (billed = 0). Billed suy ra từ đây, không lưu riêng. Lưu ý: route không quyết định chi phí thật — key/capability mới quyết định. |
| `error_code` | Thêm 2 giá trị mới (chỉ khi `status=Confirmed`): xem §2. |
| `pricing_config_id`, `input/output_usd_per_million`, `usd_to_vnd` | Vẫn snapshot giá tại thời điểm reserve/confirm cho capability billed. Capability unbilled bỏ trống các cột này vì không tra pricing. |

## 2. Giá trị `error_code` mới (khi `status = Confirmed`)

| Giá trị | Khi nào | `actual` | Hiện ở đâu |
|---|---|---|---|
| `pricing_config_missing` | Thiếu row giá cho (provider, model) tại thời điểm confirm | 0 | Drill-down (field mới), không vào error-breakdown |
| `usage_unreported` | Có giá nhưng không có số token (hiện tại: embedding) | 0 | Như trên |

Cả hai đều KHÔNG thuộc tập trạng thái lỗi (`Released`, `UnknownUsage`,
`ExpiredUnverified`) nên không làm tăng `AIErrorCount` / `TotalErrors`.

## 3. `ai_usage_period_ledgers` (ngữ nghĩa sau tính năng)

| Trường (hiện có) | Ngữ nghĩa mới |
|---|---|
| `actual_cost_micro_vnd` | Cộng dồn actual của các capability billed, không phân biệt route (trước đây chỉ paid). Capability unbilled cộng 0. Đây là nguồn của `NetAIMargin`. |
| `reserved_cost_micro_vnd` | Không đổi — chỉ reserve khi route paid. |
| `paid_*/free_*_tokens` | Không đổi — vẫn phân theo route. |

## 4. `ai_model_configs` (không đổi schema, seed tay)

Đủ 7 model theo `manual-seed.md`. Lưu ý embedding: `output_usd_per_million = 0`
(theo cách thu tiền của Google). Unique hiệu lực là
`(provider, model, effective_from)` + cửa sổ `effective_to` — seed dùng ngữ
nghĩa gán đè dòng đang hiệu lực, không dùng `ON CONFLICT (provider, model)`.

## 5. Quy tắc validation (cho implement)

- `Confirm` với provider/model rỗng → coi như thiếu giá: actual 0 + `pricing_config_missing` (đồng bộ với FR-007 — mọi lượt thiếu giá đều phải đối soát được).
- `Confirm` của capability unbilled → actual 0, không tra pricing, không set `error_code`; token/provider/model vẫn ghi để báo cáo lượt dùng. Danh sách capability unbilled đọc từ `AI_UNBILLED_CAPABILITIES` (YAML alias `${AI_UNBILLED_CAPABILITIES}`), mặc định rỗng = tất cả billed.
- `Confirm` thiếu pricing → actual 0 + `pricing_config_missing`, KHÔNG trả lỗi chặn user.
- `Confirm` đủ pricing nhưng tổng token = 0 → actual 0 + `usage_unreported`.
- `Confirm` đủ cả hai → actual = `microCost`, không ép 0 theo route.
- `Prepare`/`reserveAgainstPolicy`: route free không tra pricing và không reserve (reserve vốn = 0), nên FREE model không cần row giá. Hard-cap, chuyển tuyến, `PrepareFree` → không đụng.
