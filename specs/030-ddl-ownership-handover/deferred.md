# Deferred — Bàn giao quyền sở hữu DDL (030)

Work discovered while building Bàn giao quyền sở hữu DDL (030) that is **not this feature's job**.
Nothing here blocks closing 030; nothing here is a task of 030.

This file exists because an unchecked box in `tasks.md` makes a different claim.
`- [ ]` means *this feature is unfinished*. Work belonging to another feature, or
to the test infrastructure itself, is not this feature's unfinished business, and
recording it as such either keeps a closed feature looking open or pressures the
exit condition into being falsified.

## Rules for this file

- Every entry states **what it is**, **why it is out of scope**, and **what would
  unblock it**. An entry that only names a symptom is not an entry.
- An entry that undermines an already-ticked task's evidence **must say so and name
  that task ID**. Otherwise `[X]` reads as unconditional when it is not.
- If an entry turns out to belong to this feature after all, it moves back into
  `tasks.md` as a real task. **This file is not a parking lot.**
- Report every entry in the skill's final report. Deferring silently defeats the
  purpose of recording it.

## Entries

### D-01 — Drift `subscription.subscription_plans.plan_kind` + `tier_rank` (thiếu `DEFAULT 0` trên live)

- **Vấn đề**: Database dựng trắng từ init hiện tại + history mới (V1) sinh `subscription.subscription_plans.plan_kind` và `tier_rank` CÓ `DEFAULT 0`, trong khi database live (dev, và bản sao rehearsal V2 sau handover) KHÔNG có default ở cả hai cột. Nguyên văn từ `verification-log.md` (T023-bước 6): "`subscription.subscription_plans.plan_kind` + `tier_rank`: V1 có `DEFAULT 0`, V2 (và dev live) KHÔNG có default."
- **Vì sao ngoài phạm vi**: Drift có sẵn từ trước spec 030, không do handover gây ra — chứng minh: V2 == dev-live ở điểm này, trong khi các file handover chứa 0 DDL `subscription.*` (gate T018/T020 đã chứng minh cây Account sạch + 2 baseline chỉ chạm `aiexecutor`/`dashboard`); câu lệnh định nghĩa các cột này nằm trong file không đụng tới `20260620_payment_lifecycle.sql`. Thuộc về lịch sử/init cũ, không phải trách nhiệm của spec bàn giao ownership.
- **Gỡ bằng cách nào**: Precaution trước giờ G production: so schema prod-live với DB trắng V1 ở phạm vi `subscription`; drift ngoài handover-scope được chấp nhận cho cutover nếu bản sao restore (V2) == prod-live ở đúng điểm đó. Không sửa init/history trong spec 030.
- **Ảnh hưởng tới task đã tick**: T016-white-replay và T023-V1-vs-V2 diff — bằng chứng "đồng nhất" của hai task này mang caveat đúng 4 điểm D-01..D-04 (raw `pg_dump` diff), trong khi đồng nhất catalog ở handover-scope (10 bảng / 156 cột / 48 constraint / 27 index) không bị ảnh hưởng.

### D-02 — Drift `deposit_transactions_gateway_reference_key` (UNIQUE chỉ có trên live)

- **Vấn đề**: `CONSTRAINT deposit_transactions_gateway_reference_key UNIQUE (gateway_reference)` trên `subscription.deposit_transactions` CÓ trên V2 (và dev live), KHÔNG có trên V1 (trắng từ init + history mới). Nguyên văn từ `verification-log.md` (T023-bước 6): "`CONSTRAINT deposit_transactions_gateway_reference_key UNIQUE (gateway_reference)` trên `subscription.deposit_transactions`: V2 (và dev live) CÓ, V1 KHÔNG có."
- **Vì sao ngoài phạm vi**: Như D-01 — drift có sẵn (V2 == dev-live), handover chứa 0 DDL `subscription.*`, thuộc lịch sử/init cũ.
- **Gỡ bằng cách nào**: Cùng precaution pre-cutover prod diff như D-01; không sửa trong spec 030.
- **Ảnh hưởng tới task đã tick**: T016-white-replay và T023-V1-vs-V2 diff (cùng caveat raw-diff như D-01).

### D-03 — Drift `idx_webhook_inbox_processing` (index chỉ có trên V1)

- **Vấn đề**: `CREATE INDEX idx_webhook_inbox_processing ON subscription.provider_webhook_inbox ...` CÓ trên V1, KHÔNG có trên V2 (và dev live). Nguyên văn từ `verification-log.md` (T023-bước 6): "`CREATE INDEX idx_webhook_inbox_processing ON subscription.provider_webhook_inbox ...`: V1 CÓ, V2 (và dev live) KHÔNG có."
- **Vì sao ngoài phạm vi**: Như D-01 — drift có sẵn (V2 == dev-live), handover chứa 0 DDL `subscription.*`, thuộc lịch sử/init cũ.
- **Gỡ bằng cách nào**: Cùng precaution pre-cutover prod diff như D-01; không sửa trong spec 030.
- **Ảnh hưởng tới task đã tick**: T016-white-replay và T023-V1-vs-V2 diff (cùng caveat raw-diff như D-01).

### D-04 — Drift `subscription.user_subscriptions.started_at` (thiếu `DEFAULT now()` trên live)

- **Vấn đề**: `subscription.user_subscriptions.started_at` CÓ `DEFAULT now()` trên V1, KHÔNG có default trên V2 (và dev live). Nguyên văn từ `verification-log.md` (T023-bước 6): "`subscription.user_subscriptions.started_at`: V1 có `DEFAULT now()`, V2 (và dev live) KHÔNG có default."
- **Vì sao ngoài phạm vi**: Như D-01 — drift có sẵn (V2 == dev-live), handover chứa 0 DDL `subscription.*`, thuộc lịch sử/init cũ.
- **Gỡ bằng cách nào**: Cùng precaution pre-cutover prod diff như D-01; không sửa trong spec 030.
- **Ảnh hưởng tới task đã tick**: T016-white-replay và T023-V1-vs-V2 diff (cùng caveat raw-diff như D-01).

### D-05 — Drift dữ liệu dev sửa tay không qua migration (premium 30000000000, model flash-lite, embeddings 0.00)

- **Vấn đề**: Dữ liệu dev hiện tại lệch khỏi giá trị chain migration ở 4 điểm (chỉ đọc, phát hiện ở T014): `ai_cost_policies` gói premium có `hard_cost_micro_vnd = 30000000000` (chain seed `25000000000`); thừa model `gemini-3.5-flash-lite`; 2 output embedding là `0.00` (chain `0.20`/`0.15`); `ai_cost_policy_audit_logs` đếm = 0 dòng nên các lệch trên là sửa tay không qua migration. Baseline AI seed theo giá trị chain có ghi nhận, KHÔNG ghi đè DB cũ.
- **Vì sao ngoài phạm vi**: Đây là drift *dữ liệu*, không phải *schema* — SC-002 (đồng nhất schema) không bị ảnh hưởng; đếm dòng pre/post handover bằng nhau tuyệt đối (T023a, V5) nên SC-005 không bị ảnh hưởng. Đối chiếu/sửa dữ liệu seed thuộc về quản trị dữ liệu, không phải spec bàn giao ownership DDL.
- **Gỡ bằng cách nào**: Trước giờ G, đối chiếu 4 giá trị trên giữa prod-live và chain seed; nếu prod mang giá trị sửa-tay có chủ ý thì giữ (baseline không ghi đè DB cũ nên an toàn), nếu cần seed trắng khớp prod thì sửa ở spec dữ liệu riêng, không sửa baseline 030 sau khi đã hội tụ.
- **Ảnh hưởng tới task đã tick**: T014 (baseline seed theo chain — quyết định đã ghi nhận) và T023-đếm-dòng (pre/post bằng nhau nên kết luận 0-mất-mát giữ nguyên).
- **Cập nhật 2026-10-20 (sau cutover prod)**: prod xác nhận `ai_model_configs = 7` (cùng drift dev,
  tức cũng có `gemini-3.5-flash-lite`); giá trị data trên prod là sửa tay có chủ ý. Chủ hệ thống
  **chấp nhận drift, không sửa** — baseline không ghi đè DB cũ nên prod/dev giữ nguyên, còn môi
  trường dựng trắng sẽ theo chain seed. Ghi nhận là drift đã chấp nhận, không còn chờ xử lý.
