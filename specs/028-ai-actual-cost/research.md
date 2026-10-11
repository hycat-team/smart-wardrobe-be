# Research: Chi phí AI thật phục vụ báo cáo lợi nhuận (028)

**Ngày**: 2026-10-11 | **Spec**: [spec.md](./spec.md) | **Seed tay**: [manual-seed.md](./manual-seed.md)

## R1 — Không cần cột `billed`, không cần migration schema

- **Decision**: Giữ nguyên schema `ai_usage_events` / `ai_usage_period_ledgers`. Không thêm cột `billed_cost_micro_vnd`. Số tiền tính cho user suy ra được: `SUM(actual_cost_micro_vnd) WHERE logical_route='paid'`; tuyến free luôn có billed = 0 theo logic hiện tại.
- **Rationale**: Clarify Q1 đã chốt "không thêm cột/bảng". Logic reserve (`cost/usecase.go:107-110` chỉ reserve khi route paid), hard-cap và chuyển tuyến không đụng tới `actual` — tách biệt tự nhiên, đúng yêu cầu "không đổi cách thu tiền user".
- **Alternatives considered**: Thêm cột billed riêng (bị loại bởi Q1 và vì suy ra được từ `logical_route`, thêm cột chỉ tạo nguồn lệch số liệu thứ hai).

## R2 — Chỗ duy nhất bóp méo số liệu là `repo.Confirm`

- **Decision**: Sửa `AICostRepository.Confirm` (`services/ai/.../infrastructure/persistence/ai_cost_repo_impl.go:212-243`): bỏ dòng ép `cost = 0` khi route != paid; ledger luôn cộng `actual_cost_micro_vnd` (cả paid lẫn free); giữ nguyên phân `paid_*/free_*_tokens`.
- **Rationale**: `costUC.Confirm` (`cost/usecase.go:283-289`) đã tính đúng `microCost` theo pricing của model thật — nhưng kết quả bị ép 0 ở tầng repo trước khi ghi. Mọi báo cáo (`GetAdminAIUsageStats`, `GetAIUsageByOperation`, snapshot commerce) đều `SUM(actual_cost_micro_vnd)` nên tự đúng theo mà **không cần sửa phía commerce**.
- **Alternatives considered**: Sửa ở tầng usecase (không đủ — ép 0 nằm ở repo, và ledger cũng cần cộng dồn).

## R3 — Vision lấy token thật từ `usageMetadata`; embedding đếm inline bằng `countTokens`

- **Decision**:
  - Vision (`google.CallVision`): response `generateContent` luôn kèm `usageMetadata` nhưng code hiện tại không parse (`google_vision_embed.go:71-79`). Parse thêm và đưa số liệu lên qua hàm mới `CallVisionWithUsage`; giữ hàm cũ cho caller commerce (`analyze_uc.go`, `fashion_helper.go`) không bị ảnh hưởng.
  - Embedding: `batchEmbedContents` không trả usage, và spike T001 chứng minh `countTokens` 400 (`invalid request`) với cả hai model → **không thực hiện call đếm** (tránh latency ~200–600ms vô ích cho mọi lượt embedding). Orchestrator `Confirm` với token 0 → nhánh `usage_unreported`, actual = 0 (Q2-A). Nếu sau này Google hỗ trợ `countTokens` cho embedding, chỉ cần bật lại đúng 1 điểm (xem T015 cũ) mà không đổi thiết kế.
- **Rationale**: Vision có số liệu thật miễn phí (không thêm API call). Embedding không có nguồn số liệu nào trong response; đếm inline là cách duy nhất trung thực mà không cần hạ tầng mới — text đang sẵn trong RAM nên không vướng bài toán "worker quét backlog không có gì để đếm" (xem R7). Estimator nội bộ chỉ là heuristic đếm rune (`token_estimator.go`) nên không được dùng để tính tiền.
- **Alternatives considered**: Giữ nguyên `usage_unreported` cho embedding (bị loại sau quyết định inline countTokens — chỉ còn là nhánh fallback, không phải đường chính).
- **Điều kiện tiên quyết**: spike 1 call `countTokens` thật với key production cho `gemini-embedding-2` và `embedding-001` (cộng đồng báo 001 không được hỗ trợ chính thức — nếu cả hai đều 400 thì embedding ở lại `usage_unreported`). Xem R7.
- **Kết quả spike T001 (11/10/2026)**: cả `gemini-embedding-2` và `gemini-embedding-001` đều 400 (`invalid request`) qua `CountGoogleTokensByRequest` với 3 chunk mẫu → T015 chỉ còn nhánh fail-open `usage_unreported`.

## R4 — Hai mã `error_code` mới, hiện qua drill-down (thêm field proto cộng thêm)

- **Decision**: `pricing_config_missing` (thiếu giá model, Q1) và `usage_unreported` (không có số token, Q2-A). Ghi trên event giữ `status=Confirmed` nên không làm tăng chỉ số lỗi (error = Released/Unknown/Expired). Để admin thấy được, thêm field cộng thêm `error_code = 7` vào message `AIEventDrillDown` (`contracts/ai/v1/ai.proto:80-87`) + regen + mapper hai đầu. Không đổi endpoint, không đổi semantics các API khác.
- **Rationale**: `GetAIErrorBreakdown` chỉ đếm `status=Released` nên không thấy được các lượt Confirmed này — drill-down là kênh duy nhất, mà message hiện tại không có chỗ cho `error_code`.
- **Alternatives considered**: Nới error-breakdown để đếm cả Confirmed có error_code (làm bẩn ngữ nghĩa "lỗi", tăng TotalErrors — loại).

## R5 — Thứ tự triển khai bắt buộc: seed giá trước khi verify

- **Decision**: `manual-seed.md` phải được chạy trên môi trường verify **trước** khi kiểm thử vision (`gemini-3.5-flash-lite` chưa có giá trong DB → nếu không seed, vision rơi vào nhánh `pricing_config_missing`).
- **Rationale**: `PricingUseCase.GetActivePricingConfig` trả lỗi khi thiếu row; hiện tại lỗi này bị `_ =` nuốt ở orchestrator nên vision âm thầm ghi 0 — sau tính năng, lỗi thiếu giá vẫn không chặn user nhưng phải hiện đúng mã để đối soát.

## R6 — Chuẩn kiểm thử và gate

- **Decision**: Unit test cho 5 nhánh `costUC.Confirm` (paid giữ nguyên, free có token → actual>0, free thiếu pricing → `pricing_config_missing` kể cả provider/model rỗng, free thiếu token → `usage_unreported`), repo test cho cộng dồn ledger, orchestrator test vision có usage + embedding inline count (mock countTokens thành công/thất bại), drill-down test field `error_code`. Gate: `make lint` + `go test -race -count=3` scoped packages chạm tới (theo `docs/development/git-workflow.md`).
- **Rationale**: Tính năng chạm tiền (dù chỉ báo cáo) nên mỗi nhánh ghi số liệu phải có test khóa lại.

## R7 — Vì sao không làm worker riêng để đếm batch (quyết định hậu clarify)

- **Decision**: Không làm worker/queue cho việc đếm token (đã hỏi ý kiến, chốt inline). Cơ chế batch embedding hiện tại: `CallEmbeddingBatch` cắt chunks thành sub-batch ≤100, mỗi sub-batch 1 call `batchEmbedContents` (`google_vision_embed.go:93-170`).
- **Rationale**:
  1. `countTokens` chỉ trả tổng, không tách theo chunk — muốn ghi đúng từng event thì mỗi event cần đúng 1 call riêng; gom nhiều event vào 1 call không chia ngược được. Worker-batch chỉ dàn đều tải chứ không giảm tổng số call.
  2. `ai_usage_events` không lưu input text — worker quét backlog không có gì để đếm. Muốn worker chạy được phải enqueue kèm text lúc request (thêm hạ tầng RabbitMQ/outbox + xử lý trùng/backlog/delay số liệu) hoặc thêm cột lưu text (thêm schema). Inline lúc chunks còn trong RAM không vướng gì.
  3. Mọi luồng vision/embedding (kể cả fashion worker commerce) đều đi qua `IAIOrchestrator` contract → orchestrator UC có cost event — phủ 100% volume cần đếm, không sót nhánh.
- **Rủi ro đã thấy (xử lý ở implement)**:
  - `countTokens` có hỗ trợ embedding model trên AI Studio không — spike trước (R3).
  - Fallback model thắng nhưng giá tính theo `plan.Model` (primary) — lệch nhỏ, hiếm, ghi nhận là giới hạn đã biết.
  - `gemini-embedding-001` có thể đã sunset (cộng đồng ghi 14/07/2026) mà `.env.production` vẫn để fallback — kiểm tra ops riêng, nếu xác nhận thì ghi `deferred.md`, ngoài scope tính năng.
