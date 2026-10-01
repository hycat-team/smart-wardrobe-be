# Hướng dẫn kiểm chứng — 024

**Ngày**: 2026-09-29 | **Đầu vào**: [spec.md](./spec.md) · [plan.md](./plan.md) · [research.md](./research.md)

Tài liệu này là **hướng dẫn chạy và kiểm chứng**, không phải đặc tả triển khai. Chi tiết cách viết mã nằm ở `tasks.md` (giai đoạn `/speckit-tasks`).

Hợp đồng được kiểm chứng: [hợp đồng sự kiện](./contracts/subscription-plan-changed-event.md) · [hợp đồng API](./contracts/admin-grant-api.md) · [mô hình dữ liệu](./data-model.md)

---

## Điều kiện tiên quyết

```bash
# Dịch vụ hạ tầng
make docker-dev-up

# Mã sinh tự động (Wire + Swagger + nginx) — BẮT BUỘC sau khi đổi DI hoặc route
make generate svc=account
make generate svc=ai

# Cổng xanh bắt buộc trước mỗi đề xuất commit
make wire-all && make lint && make test-race
```

Ba chế độ ứng dụng: `APP_MODE=http` (chỉ API), `APP_MODE=worker` (chỉ worker), `APP_MODE=all` (cả hai). Worker chỉ chạy khi `APP_MODE != http` — nếu kiểm chứng tín hiệu mà không thấy gì, **kiểm tra chế độ chạy trước**.

Kết nối cơ sở dữ liệu kiểm chứng: container `postgres-closy-dev`, CSDL `smart_wardrobe_db`, người dùng `admin`.

```bash
# Ghi nhận trạng thái trước khi làm bất cứ việc gì — đây là mốc so sánh
docker exec postgres-closy-dev psql -U admin -d smart_wardrobe_db -c "
  SELECT status,
         count(*) FILTER (WHERE policy_snapshot = '{}'::jsonb) AS empty_snapshot,
         count(*) AS total
  FROM aiexecutor.ai_policy_grants GROUP BY status;"
```

Kết quả mong đợi ở thời điểm bắt đầu (đã kiểm chứng 2026-09-29): `active` 8 dòng trong đó 1 rỗng, `closed` 7 dòng trong đó 6 rỗng, `future` 1 dòng rỗng — **tổng cộng 8 trên 16 dòng có ảnh chụp rỗng**.

---

## GĐ0 — Cấu hình giá trị vận hành

**Kiểm chứng**: FR-001 đến FR-005, SC-011, SC-012

```bash
# 1. Chạy tệp kiểm tra trước — phải in ra giá trị hiện tại và KHÔNG đổi gì
psql "$DSN" -f database/scripts/03-dry_run_update_ai_cost_limits.sql

# 2. Ghi lại số đo trước khi thực thi
docker exec postgres-closy-dev psql -U admin -d smart_wardrobe_db -t -A -c \
  "SELECT hard_cost_micro_vnd FROM aiexecutor.ai_cost_policies WHERE code='premium-default';"
# Mong đợi: 25000000000

# 3. Thực thi
psql "$DSN" -f database/scripts/04-update_ai_cost_limits.sql

# 4. Xác nhận
docker exec postgres-closy-dev psql -U admin -d smart_wardrobe_db -t -A -c \
  "SELECT hard_cost_micro_vnd FROM aiexecutor.ai_cost_policies WHERE code='premium-default';"
# Mong đợi: 30000000000
```

Hạn mức dùng mỗi ngày cập nhật qua giao diện quản trị có sẵn, **không** sửa bằng tệp kịch bản:

```bash
curl -X PUT "$ACCOUNT/api/v1/admin/subscriptions/plans/$PREMIUM_PLAN_ID" \
  -H "Authorization: Bearer $ADMIN_TOKEN" -H "Content-Type: application/json" \
  -d '{"aiOutfitDailyQuota": 30, "aiChatDailyQuota": 10}'
```

**Kiểm chứng chống ghi lặp (SC-012)**: chạy tệp thực thi **ba lần**, xác nhận giá trị sau ba lần vẫn là `30000000000` và số dòng bị ảnh hưởng ở lần thứ hai và thứ ba bằng 0.

**Kiểm chứng tệp kiểm tra không ghi (FR-003)**: chạy tệp kiểm tra, chụp lại ảnh chụp màn hình, chạy lần nữa, xác nhận hai lần cho kết quả giống hệt.

### Đối chiếu hai cổng hạn mứn (bắt buộc, FR-005b)

Sau khi cập nhật **cả hai** cổng, chạy truy vấn này và xác nhận mọi thao tác đều khớp:

```sql
SELECT p.code, o.operation,
       o.max_paid_attempts_per_day                                   AS cong_2_thi_hanh,
       CASE o.operation
         WHEN 'outfit' THEN sp.ai_outfit_daily_quota
         WHEN 'chat'   THEN sp.ai_chat_daily_quota
       END                                                            AS cong_1_nguoi_dung
FROM aiexecutor.ai_cost_policy_operations o
JOIN aiexecutor.ai_cost_policies p ON p.id = o.policy_id
CROSS JOIN subscription.subscription_plans sp
WHERE p.code = 'premium-default' AND sp.slug = 'premium-monthly'
  AND o.operation IN ('chat','outfit')
ORDER BY o.operation;
```

| Gói | Chế độ | `operation` | `cong_1_nguoi_dung` | `cong_2` | Cổng 2 có tác dụng? |
|---|---|---|---|---|---|
| Premium | `strict` | `chat`, `outfit` | 30 outfit / 10 chat | **NULL** | **không** — cấu hình chết |
| Miễn phí | `observe_only` | `chat` | 3 | 3 | có — đã khớp |
| Miễn phí | `observe_only` | `outfit` | 5 | 3 → **4** | có — **phải sửa, rồi phát lại tín hiệu** |

Dòng `outfit` của gói miễn phí lệch `5` so với `4` là **quyết định vận hành nội bộ** (FR-005b, FR-005d). Người dùng **không** được thông báo, và cổng 2 **không bao giờ** hiện ra phía họ (FR-005c). Bước kiểm chứng vì vậy phải xác nhận lượt thứ 5 bị hạ xuống mô hình miễn phí **và** xác nhận **không có** thông điệp nào chứa số tiền, trần chi phí hay nhắc tới sự tồn tại của cổng 2.
Hai gói xử lý **khác nhau**, đừng gộp:
- **Premium**: chỉ sửa cổng 1 qua giao diện lập trình có sẵn. Cổng 2 ở chế độ `strict` **không dùng số lượt/ngày** nên giá trị `NULL` là dạng nói rõ nhất; không cần phát lại tín hiệu.
- **Miễn phí**: cổng 1 đã đúng. Cổng 2 ở chế độ `observe_only` **có tác dụng thật**, phải đặt `outfit` thành `4` qua **giao diện lập trình của hệ thống AI** (FR-004b) — **không** dùng tệp kịch bản — rồi **phải** phát lại tín hiệu, vì số lượt nằm trong ảnh chụp đóng băng nên sửa cấu hình một mình không có tác dụng lên bản ghi đang tồn tại.
Nếu phát hiện lệch ở bất kỳ dòng nào **có tác dụng** ngoài những gì đã được quyết định thì **dừng lại** và báo người phát triển — không tự chọn con số. Dòng `strict` lệch thì chỉ ghi nhận, không sửa, vì nó không ảnh hưởng hành vi (FR-005e).
Sau khi sửa cổng 2 của gói miễn phí, chạy bước (3) của FR-029c: tệp `07-verify_policy_grants.sql`, rồi `POST /admin/subscription-grants/restore` với `dryRun: true` rồi `dryRun: false`. **Bỏ qua bước phát lại tín hiệu thì cả 5 bản ghi `effective_to IS NULL` vẫn giữ giới hạn cũ vĩnh viễn** — chúng chỉ được tạo lại khi tài khoản đổi gói.

### Ngân sách chu kỳ và thời điểm hết (kiểm chứng FR-005c)

Với mục tiêu 30/10, ngân sách 27.600đ cạn sau **12–15 ngày**. Kiểm chứng bằng cách dùng một tài khoản Premium và quan sát:

| Ngày | Mong đợi |
|---|---|
| 1–12 | Dùng mô hình trả phí, hết ngân sách quanh ngày 12–15 |
| 13–30 | Tự động dùng mô hình **miễn phí**, **không** báo lỗi |

Kiểm tra bằng truy vấn sổ tính phí:

```sql
SELECT period_index, actual_cost_micro_vnd, reserved_cost_micro_vnd
FROM aiexecutor.ai_usage_period_ledgers
WHERE user_id = '<tai-khoan-premium>' ORDER BY period_index;
```

`actual_cost_micro_vnd` phải **dừng lại** ở khoảng 27.600.000.000 và không tăng thêm. Đây là phép kiểm chứng quan trọng nhất của GĐ0 vì nó chứng minh trần chi phí thực sự hoạt động, chứ không chỉ là một con số trong cấu hình.

### Trần chi phí cấm xuất hiện phía người dùng (FR-005c)

Cổng 2 là cơ chế nội bộ nên **không giá trị nào của nó được phép lộ ra người dùng**. Màn hình hạn mức chỉ hiện phần thuộc **cổng 1**: **30 lượt gợi ý phối đồ / 10 lượt trò chuyện** mỗi ngày.

Phép kiểm chứng bắt buộc: thực hiện lượt thứ 30 (outfit) và lượt thứ 10 (chat) của một tài khoản Premium, rồi **bật lưới kiểm tra phản hồi** và xác nhận **không có** chuỗi nào chứa `30000`, `27.600`, `trần`, `chi phí AI` hay `ngân sách`. Nếu thấy bất kỳ chuỗi nào, đó là hành vi vượt phạm vi FR-005c.
---

### Xác minh ghi nhận ký cổng 2 (FR-004c, FR-004d, FR-004e)
Sau khi đã sửa cổng 2 qua giao diện lập trình, **phải** xác minh hai bản ghi được ghi đúng, vì cả hai đều là điều kiện của FR-004c — thiếu một là trạng thái nửa vời.
```bash
# 1. Bản ghi kiểm toán cục bộ của AI — phải có đủ 5 trường bắt buộc
docker exec postgres-closy-dev psql -U admin -d smart_wardrobe_db -x -c \
  "SELECT actor_user_id, changed_at, request_ip, policy_code, operation,
          field_name, old_value, new_value, action
   FROM aiexecutor.ai_cost_policy_audit_logs
   ORDER BY changed_at DESC LIMIT 5;"
# 2. Bản ghi outbox — phải có đúng topic đã quy định
docker exec postgres-closy-dev psql -U admin -d smart_wardrobe_db -c \
  "SELECT topic, status, attempts, created_at
   FROM shared.outbox_records
   WHERE topic = 'admin.audit.log'
   ORDER BY created_at DESC LIMIT 5;"
# 3. AI KHÔNG được ghi thẳng vào bảng kiểm toán của Account.
#    Đường đúng là: AI ghi bảng của chính nó -> phát admin.audit.log ->
#    consumer của Account ghi shared.admin_audit_logs.
#    Vì vậy bản kiểm tra là "đường đi có đúng không", KHÔNG phải "bảng này rỗng".
docker exec postgres-closy-dev psql -U admin -d smart_wardrobe_db -c \
  "SELECT action, target_type, actor_user_id, request_ip, payload->>'policyCode' AS policy,
          payload->>'operation' AS operation, payload->>'oldValue' AS old_value,
          payload->>'newValue' AS new_value
   FROM shared.admin_audit_logs
   WHERE payload->>'fieldName' = 'maxPaidAttemptsPerDay'
   ORDER BY created_at DESC LIMIT 5;"
# Mong đợi: có đúng một dòng cho mỗi lần chỉnh, với
#   action = ai.cost_policy_update, target_type = ai_cost_policy
#   (giá trị lấy từ hằng số trong shared/domain/constants/adminaudit/).
#
# ⚠️ Ghi chú về lịch sử tài liệu: bản quickstart đầu tiên ghi bước này là
# "SELECT count(*) ... Mong đợi: 0". Điều đó MÂU THUẪN với FR-004c, vốn bắt
# buộc phát admin.audit.log để thay đổi hiện ra ở màn hình kiểm toán tập
# trung. Nếu bản kiểm tra đó được giữ, nó sẽ bắt buộc gỡ đúng cơ chế mà
# FR-004c yêu cầu. Ở đây lấy FR-004c làm chuẩn và kiểm tra "đường đi đúng"
# thay vì "bảng rỗng".
#
# Bổ sung — chứng minh AI không có đường ghi trực tiếp (kiểm tra ở mã nguồn):
#   rg "admin_audit_logs" services/ai/   →  không có kết quả nào
docker exec postgres-closy-dev psql -U admin -d smart_wardrobe_db -c \
  "SELECT count(*) AS ban_ghi_co_action_dung_danh_muc
   FROM shared.admin_audit_logs
   WHERE payload->>'fieldName' = 'maxPaidAttemptsPerDay'
     AND (action <> 'ai.cost_policy_update' OR target_type <> 'ai_cost_policy');"
# Mong đợi: 0 (không có giá trị nào nằm ngoài danh mục)
# 4. Giá trị action và targetType phải nằm trong danh mục, không phải chuỗi tự do
docker exec postgres-closy-dev psql -U admin -d smart_wardrobe_db -c \
  "SELECT DISTINCT action, target_type FROM shared.admin_audit_logs
   ORDER BY 1, 2;"
# Mong đợi: chỉ thấy các giá trị trong danh mục, ví dụ ai.pricing_update / ai.cost_policy_update
```
Bốn bước này là **xác minh thủ công**, không phải kiểm thử tự động — xem mục "Chiến lược kiểm thử" trong `spec.md`.

## GĐ1 — Suy giảm mềm

**Kiểm chứng**: US1, FR-006 đến FR-010, SC-001, SC-003

Tài khoản **chưa có** bản ghi mức quyền tính phí:

```bash
# Chọn một tài khoản chưa có bản ghi
docker exec postgres-closy-dev psql -U admin -d smart_wardrobe_db -t -A -c "
  SELECT u.id FROM identity.users u
  WHERE u.is_deleted = false
    AND NOT EXISTS (SELECT 1 FROM aiexecutor.ai_policy_grants g WHERE g.user_id = u.id)
  LIMIT 1;"
```

| Kịch bản | Kết quả mong đợi |
|---|---|
| Gọi `chat` | **Chạy được**, không lỗi |
| Gọi `outfit` | **Chạy được**, không lỗi |
| Kiểm tra sau khi gọi | `SELECT count(*) FROM aiexecutor.ai_policy_grants WHERE user_id='<id>'` vẫn bằng 0 |
| Kiểm tra sau khi gọi | `SELECT count(*) FROM aiexecutor.ai_usage_period_ledgers WHERE user_id='<id>'` **tăng** — dòng có `grant_id IS NULL` |

Dòng thứ hai là quan trọng nhất: nó chứng minh **đường đọc không ghi bản ghi mức quyền tính phí** (FR-042). Nếu con số đó tăng, GĐ1 đã vi phạm ràng buộc.

Dòng thứ ba chứng minh điều ngược lại, và cũng quan trọng ngang nhau: suy giảm tầng 1 **vẫn đi qua sổ tính phí `FOR UPDATE`** của §VI, chỉ là dòng sổ không có bản ghi mức quyền tính phí để trỏ tới. Nếu dòng sổ **không** tăng, nghĩa là đã bỏ qua cơ chế đặt trước — và với chính sách `free-default` có `normal_route = paid`, đó là rò tiền không kiểm soát. Xem research.md R8.

**Tầng 2** — cấu hình hỏng (FR-007 bước 2, SC-003). Tạm đặt `ai.default_cost_policy_code` thành một mã không tồn tại, khởi động lại dịch vụ AI:

| Kịch bản | Kết quả mong đợi |
|---|---|
| Gọi `chat` / `outfit` | Chạy được, tuyến miễn phí, không lỗi |
| Nhật ký vận hành | Có cảnh báo suy giảm tầng 2 (FR-008) |
| Bộ đếm suy giảm | Tăng theo mã lý do `tier_2_observe_only` (hằng số trong `shared/domain/constants/aifallbackreason/`) |
| Sổ tính phí | **Không** tăng — tầng 2 cố ý không ghi dòng sổ, dùng sẵn đường bỏ qua có `ledger_id = NULL` |

Hai mã lý do suy giảm dùng ở đây phải khớp đúng hằng số mà T020 định nghĩa, nếu không cảnh báo sẽ nổ dưới một tên mà không ai kiểm tra.

Phải **khôi phục lại cấu hình** trước khi sang GĐ2.

---

## GĐ2 và GĐ3 — Lịch sử và nội dung tín hiệu

**Kiểm chứng**: FR-011 đến FR-026, SC-004, SC-024

Mọi đường đổi gói phải sinh bản ghi lịch sử. Kiểm tra từng đường, đều **phải** có bản ghi mới:

```sql
SELECT event_type, to_plan_code, actor_admin_id IS NOT NULL AS has_admin,
       source_deposit_transaction_id IS NOT NULL AS has_payment
FROM subscription.user_subscription_events
ORDER BY created_at DESC LIMIT 20;
```

| Đường thay đổi gói | Loại kỳ vọng | Cách kích hoạt |
|---|---|---|
| Mua qua cổng thanh toán | `ACTIVATED` | Luồng PayOS hiện có |
| Mua bằng ví | `ACTIVATED` | `POST /subscriptions/purchase-with-wallet` |
| Gia hạn tự động | `EXTENDED` | Đặt thời hạn ngắn rồi chạy worker gia hạn |
| Hết thời hạn | `SUBSCRIPTION_DOWNGRADED_TO_FREE` | Cùng cơ chế, tắt tự gia hạn |
| Cấp thủ công | `ACTIVATED` | GĐ7 — chưa có ở GĐ2, tạm bỏ qua |
| Thu hồi thủ công | `SUBSCRIPTION_DOWNGRADED_TO_FREE` | `POST /admin/subscriptions/{userId}/cancel` |
| Cấp phát gói mặc định | `ACTIVATED` | **Ngay trong giao dịch tạo tài khoản** (T196): `admin_uc.go:224-229`, `register_uc.go:183-188`, `google_auth_uc.go:240-245` đều gọi `EnsureDefaultSubscription` trước khi commit |
| Bật/tắt tự gia hạn | `AUTO_RENEW_UPDATED` | `PATCH` trạng thái tự gia hạn |
| **Cấp hàng loạt trong một thao tác đọc** | `ACTIVATED` **cho từng tài khoản** | Gọi `GET /subscriptions/me/overviews` (hoặc bất kỳ đường đọc nào dùng `BulkCreate`) với vài tài khoản chưa có bản ghi gói. **FR-014** |

Dòng cuối là dễ bỏ sót nhất: nó **không** nằm trên đường của người dùng, chỉ xuất hiện khi một thao tác đọc tạo hàng loạt bản ghi gói. Phải có bản ghi lịch sử cho từng tài khoản, không được tạo bản ghi gói "trần" không kèm lịch sử.

Kiểm tra nội dung tín hiệu có đủ 14 trường:

```bash
# Bật nhật ký đầy đủ ở dịch vụ Account, kích hoạt một thay đổi gói,
# rồi đọc bản ghi outbox chưa phát hoặc nhật ký phát
docker exec postgres-closy-dev psql -U admin -d smart_wardrobe_db -c \
  "SELECT topic, payload::jsonb->>'eventVersion'  AS ver,
          payload::jsonb->>'toPlanId'       AS to_plan_id,
          payload::jsonb->>'effectiveTo'    AS eff_to,
          payload::jsonb->>'durationDays'   AS days,
          payload::jsonb->>'sourceEventId'  AS src_event
   FROM outbox_records
   WHERE topic = 'subscription.plan.changed'
   ORDER BY created_at DESC LIMIT 5;"
```

Cột `ver` phải là `2`; bốn cột còn lại **không được rỗng** (trừ `eff_to`/`days` khi gói không thời hạn).

### Tài khoản mới có mức quyền tính phí trong 60 giây (SC-006)

FR-013 nói **không** được có đường tạo tài khoản nào dẫn tới tài khoản thiếu mức quyền tính phí. Kiểm chứng cả ba loại:

| Loại tài khoản | Cách tạo | Mong đợi trong 60 giây |
|---|---|---|
| Đăng ký bằng email | `POST /auth/register` | Có bản ghi mức quyền tính phí trỏ gói miễn phí |
| Đăng ký bằng Google | Luồng Google hiện có | Như trên |
| **Do quản trị viên tạo** | `POST /admin/users` | Như trên |

**Cách cấp phát (đã đảo ngược ở T196, thay cho mô tả lazy trước đây).** Cả ba loại tài khoản trên nay nhận gói mặc định và bản ghi mức quyền tính phí **ngay trong giao dịch tạo tài khoản**, chứ không chờ một lần đọc trạng thái gói: `admin_uc.go:224-229`, `register_uc.go:183-188`, `google_auth_uc.go:240-245` đều bọc `uow.Execute` quanh cả `Create` lẫn `EnsureDefaultSubscription`. Ngân sách 60 giây của SC-006 vì vậy **đạt theo cách khác**: không phải "đọc trong vòng 60 giây" mà là "không cần đọc gì cả" — bản ghi tồn tại ngay khi `201` trả về. Yêu cầu "trong 60 giây" giữ nguyên làm ngưỡng kiểm chứng.

```sql
SELECT g.user_id, g.plan_code, g.status, g.policy_snapshot = '{}'::jsonb AS snapshot_rong
FROM aiexecutor.ai_policy_grants g
WHERE g.user_id = '<tai-khoan-vua-tao>' ORDER BY g.created_at DESC LIMIT 1;
```

`snapshot_rong` phải là `false`. ~~Trước khi sửa, tài khoản do quản trị viên tạo nhận gói miễn phí một cách **trễ** — chỉ khi đọc trạng thái gói lần đầu — và không có bản ghi lịch sử, nên không bao giờ có mức quyền tính phí.~~ *(Câu này mô tả trạng thái **trước T196**, giữ lại làm bằng chứng trước–sau. Nó **không** còn là hiện trạng: nay tài khoản được cấp phát ngay trong giao dịch tạo, nên có cả bản ghi gói và bản ghi lịch sử.)*

**Kiểm chứng tương thích ngược (SC-024)**: đẩy thủ công một nội dung `eventVersion: 1` (chỉ 6 trường cũ) vào hàng đỗi, rồi xác nhận phía nhận từ chối kèm cảnh báo, **không** ghi dữ liệu, và **không** dừng hoạt động. Sau đó đẩy một nội dung `eventVersion: 99` và xác nhận cũng bị từ chối kèm cảnh báo.

---

## GĐ3b — Dọn dữ liệu tồn tại

**Kiểm chứng**: FR-092 đến FR-094, SC-028

> ⚠️ Chạy **sau khi GĐ3 xong** và **trước khi bắt đầu GĐ4**. Chạy sớm hơn thì tín hiệu mới ghi vào dữ liệu chưa dọn và số đo sẽ lẫn lộn.

```bash
# 1. Chụp số đo trước
docker exec postgres-closy-dev psql -U admin -d smart_wardrobe_db -c "
  SELECT count(*) FILTER (WHERE policy_snapshot='{}'::jsonb)      AS rong,
         count(*) FILTER (WHERE status='closed'
                            AND (effective_to IS NULL OR effective_to <= now())) AS chet,
         count(*) FILTER (WHERE status IN ('active','future')
                            AND effective_to IS NOT NULL
                            AND effective_to <= now())                     AS het_han,
         count(*) FILTER (WHERE status IN ('active','future')
                            AND (effective_to IS NULL OR effective_to > now())) AS con_hieu_luc
  FROM aiexecutor.ai_policy_grants;"

# 2. Chạy tệp kiểm tra — chỉ in, không ghi
psql "$DSN" -f database/scripts/05-dry_run_cleanup_policy_grants.sql

# 3. Thực thi
psql "$DSN" -f database/scripts/06-cleanup_policy_grants.sql

# 4. Chạy lại số đo ở bước 1
```

| Chỉ số | Trước | Sau (mong đợi) |
|---|---|---|
| Ảnh chụp rỗng | 8 | **0** |
| `closed` đã hết hiệu lực | 7 | **0** |
| `active`/`future` đã hết thời hạn | 2 | **0** |
| **Đang thực sự còn hiệu lực** | 5 | **5 — không đổi** |

> **Đo lại ngay trước khi chạy.** Bảng trên chụp ngày 29/09/2026. Nếu giữa lúc đo và lúc chạy có tín hiệu nào ghi vào bảng — hoặc đã chạy `06-cleanup` một lần rồi — số đo sẽ lệch. Chỉ số duy nhất phải **giữ nguyên** là *đang thực sự còn hiệu lực*: dọn không được lấy đi quyền của bất kỳ tài khoản nào. Ba chỉ số còn lại chỉ cần về 0.

Dòng cuối là bất biến quan trọng nhất của GĐ3b: **không tài khoản nào mất quyền tính phí** vì việc dọn.

**Chống ghi lặp (FR-093)**: chạy tệp thực thi lần thứ hai, xác nhận mọi báo cáo đều bằng 0 và tổng số dòng bị ảnh hưởng bằng 0.

---

## GĐ4 — AI ghi bản ghi mức quyền tính phí

**Kiểm chứng**: US2, FR-027 đến FR-036, SC-005, SC-007, SC-008, SC-009

Chỉ khi ba điều kiện sau **cùng** đạt:

1. GĐ2 và GĐ3 đã xong.
2. GĐ3b đã chạy và bảng sạch.
3. Dịch vụ AI chạy ở chế độ có worker.

```sql
-- Sau khi phát một tín hiệu kích hoạt gói Premium
SELECT g.user_id, g.plan_code, g.tier_rank, g.status,
       g.policy_snapshot = '{}'::jsonb AS snapshot_rong,
       jsonb_array_length(g.policy_snapshot->'operations') AS so_operation,
       g.effective_from, g.effective_to, g.source_event_id
FROM aiexecutor.ai_policy_grants g
WHERE g.user_id = '<tai-khoan>' ORDER BY g.created_at DESC;
```

| Kiểm tra | Mong đợi |
|---|---|
| `snapshot_rong` | `false` — SC-009 |
| `so_operation` | `4` (chat, outfit, rewriter, summary) |
| `source_event_id` | Không rỗng — FR-033 |
| `status` | `active`, hoặc `future` khi gia hạn kỳ chưa bắt đầu |

**Bốn kịch bản bắt buộc của US2:**

| # | Kịch bản | Cách kiểm | Mong đợi |
|---|---|---|---|
| 1 | Thanh toán thành công | Cấp Premium qua cổng, kiểm tra bản ghi | Trỏ đúng gói Premium, đúng chính sách Premium |
| 2 | Hạn mức đúng | Gọi `chat` và `outfit` | Token vào/ra theo bảng trong [data-model.md](./data-model.md) |
| 3 | Vượt trần chi phí | Dùng hết hạn mức trong một chu kỳ | Hạ xuống mô hình miễn phí, không tiêu tiếp tiền |
| 4 | Gia hạn tự động | Gia hạn khi kỳ cũ còn hiệu lực | Mức quyền tính phí **không gián đoạn**; bản ghi `future` cho kỳ kế tiếp đã có sẵn |
| 5 | Hết thời hạn | Cho thời hạn kết thúc | Về đúng hạn mức gói miễn phí, không còn mức quyền tính phí Premium |
| 6 | Thu hồi thủ công | `POST /admin/subscriptions/{userId}/cancel` | Ngay lập tức về gói miễn phí |

**Chống xử lý trùng (FR-033)**: phát lại **cùng một** tín hiệu ba lần, xác nhận số bản ghi **không đổi** và không có tác dụng phụ.

**Tín hiệu đến sai thứ tự (FR-033)**: phát một tín hiệu hạ cấp (`SUBSCRIPTION_DOWNGRADED_TO_FREE`), **chờ cho nó xử lý xong**, rồi đẩy lại tín hiệu gia hạn (`EXTENDED`) đã có `occurredAt` **cũ hơn**. Xác nhận tín hiệu cũ bị từ chối với cảnh báo `stale_event` và **không** hồi sinh mức quyền tính phí Premium. Nếu mức quyền tính phí quay lại Premium, chốt chặn thứ tự chưa hoạt động.

**Tính nguyên tử (FR-031)**: giữa lúc đóng bản ghi cũ và lúc tạo bản ghi mới không được có khoảng thời gian nào tài khoản không có mức quyền tính phí. Kiểm bằng cách phát tín hiệu rồi **liên tục** gọi `chat` trong lúc xử lý — không lần nào được trả lỗi (SC-008).

**SC-007**: sau mỗi thay đổi gói, chạy truy vấn sau và xác nhận không tài khoản nào có từ hai mức quyền tính phí cùng hiệu lực:

```sql
SELECT user_id, count(*) FROM aiexecutor.ai_policy_grants
WHERE status = 'active' AND effective_from <= now()
  AND (effective_to IS NULL OR effective_to > now())
GROUP BY user_id HAVING count(*) > 1;
```

Kết quả mong đợi: **không có dòng nào**.

---

## GĐ5 — Gỡ đường ghi vi phạm

**Kiểm chứng**: FR-037 đến FR-041, SC-010

⚠️ **Ranh giới không đảo ngược được.** Chỉ làm khi mọi mục GĐ4 ở trên đều đạt trên môi trường phát triển. Không có cờ bật/tắt; hoàn tác bằng git (FR-040).

```bash
# Trước khi gỡ — xác nhận không còn lệnh ghi chéo dịch vụ nào
rg -n "ai_policy_grants" services/account/ --type go
# Mong đợi sau khi gỡ: không có câu lệnh INSERT/UPDATE/DELETE nào vào bảng này
```

| Kiểm tra | Mong đợi |
|---|---|
| Grep `ai_policy_grants` trong `services/account/` | Chỉ còn phần **đọc** hoặc không còn gì |
| `user_subscription_events` | Còn bản ghi lịch sử, **không** còn bản ghi mức quyền tính phí ghi kèm |
| Tín hiệu còn phát không | Còn — `outbox_records` vẫn có bản ghi |
| `make wire-all && make lint && make test-race` | Xanh |

Sau khi gỡ, chạy lại **toàn bộ** bảng kịch bản GĐ4. Đây là bước xác nhận quan trọng nhất của cả tính năng.

---

## GĐ6 — Đường đọc không thay đổi dữ liệu

**Kiểm chứng**: US1 kịch bản 5, FR-042 đến FR-044, SC-002

```sql
-- Chụp trước
SELECT count(*) AS grants_truoc FROM aiexecutor.ai_policy_grants;
SELECT count(*) AS ledger_truoc FROM aiexecutor.ai_usage_period_ledgers;
-- Gọi chat, outfit, và cả tài khoản KHÔNG có bản ghi
-- Chụp sau
SELECT count(*) AS grants_sau FROM aiexecutor.ai_policy_grants;
SELECT count(*) AS ledger_sau FROM aiexecutor.ai_usage_period_ledgers;
```

| Chỉ số | Mong đợi | Ý nghĩa |
|---|---|---|
| `ai_policy_grants` | **không đổi** | Đường đọc không ghi bản ghi mức quyền tính phí (FR-042) |
| `ai_usage_period_ledgers` | **tăng** | Bước đặt trước vẫn ghi sổ tính phí `FOR UPDATE` theo §VI |

Dòng `grants_sau` bằng `grants_truoc` là phép kiểm hai dòng mà không cần hiểu gì thêm. Dòng `ledger_sau` lớn hơn `ledger_truoc` thì **bắt buộc** — nếu nó không tăng thì đã vô tình bỏ qua cơ chế đặt trước, tức là phá §VI trong khi vẫn làm cho các phép kiểm phía trên xanh. Với chính sách `free-default` có `normal_route = paid`, đó là rò tiền không kiểm soát.

**Tài khoản thiếu bản ghi**: cũng phải giữ nguyên số dòng ở `ai_policy_grants` — đường đọc tuyệt đối không được tạo bản ghi mức quyền tính phí. Dòng sổ tính phí thì vẫn phải tăng, với `grant_id IS NULL`.

**Dữ liệu lệch (FR-042)**: cố tình làm lệch `plan_code` của một bản ghi, gọi lại đường đọc, xác nhận hệ thống **không** tự đóng bản ghi đó — chỉ trả kết quả để nơi khác quyết định.

---

## GĐ7 — Công cụ cấp gói

**Kiểm chứng**: US3, US4, FR-045 đến FR-085, SC-013 đến SC-021

### Một tài khoản — ba dạng mã định danh

```bash
ADMIN_TOKEN=…; BASE="$ACCOUNT/api/v1"

for ID in "khachhang@example.com" "NGUYENVAN" "3f7c1a20-…"; do
  curl -sS -X POST "$BASE/admin/subscription-grants" \
    -H "Authorization: Bearer $ADMIN_TOKEN" \
    -H "Idempotency-Key: $(uuidgen)" \
    -H "Content-Type: application/json" \
    -d "{\"items\":[{\"userIdentifier\":\"$ID\",\"planSlug\":\"premium-monthly\"}]}"
done
```

Cả ba lần phải cho kết quả giống hệt nhau — hệ thống **tự** nhận diện loại mã định danh (FR-046). Lần thứ hai dùng email viết hoa chữ khác vẫn phải khớp (FR-047).

### Danh sách hỗn hợp

Cố tình **trộn nhiều mã gói khác nhau** trong cùng một danh sách (FR-061), đồng thời trộn cả ba dạng mã định danh (FR-045) và một tài khoản lặp lại:

```json
{
  "items": [
    { "userIdentifier": "hop-le@vidu.vn",      "planSlug": "premium-monthly" },
    { "userIdentifier": "khach-thuong",        "planSlug": "premium-monthly" },
    { "userIdentifier": "khong-ton-tai@x.vn",  "planSlug": "premium-monthly" },
    { "userIdentifier": "quan-tri-vien",       "planSlug": "premium-monthly" },
    { "userIdentifier": "go-bi-vo-hieu-hoa",   "planSlug": "premium-monthly" },
    { "userIdentifier": "c0ffee00-0000-4000-8000-000000000001", "planSlug": "premium-monthly" },
    { "userIdentifier": "khach-hang-abc",      "planSlug": "premium-quarterly" },
    { "userIdentifier": "khach-hang-abc",      "planSlug": "premium-monthly" }
  ]
}
```

| # | `userIdentifier` | `planSlug` | `status` mong đợi | `reason` mong đợi |
|---|---|---|---|---|
| 0 | `hop-le@vidu.vn` (email) | `premium-monthly` | `success` | — |
| 1 | `khach-thuong` (tên đăng nhập) | `premium-monthly` | `success` | — |
| 2 | `khong-ton-tai@x.vn` | `premium-monthly` | `error` | `user_not_found` |
| 3 | `quan-tri-vien` | `premium-monthly` | `error` | `target_is_admin` |
| 4 | `go-bi-vo-hieu-hoa` | `premium-monthly` | `error` | `plan_inactive` |
| 5 | `c0ffee00-…` (UUID) | `premium-monthly` | `success` | — |
| 6 | `khach-hang-abc` | `premium-quarterly` | `success` | — |
| 7 | `khach-hang-abc` (lặp lại) | `premium-monthly` | `skipped` | `duplicate_in_request` |

Bốn phần tử hợp lệ phải được cấp gói; kết quả phải khớp **đúng 8** mục đã gửi, cùng thứ tự (FR-065). Phần tử 6 và 7 cùng là một tài khoản nhưng khác mã gói — hệ thống vẫn chỉ xử lý phần tử đầu (FR-066).

### Cộng dồn thời hạn (FR-052)

1. Cấp Premium cho một tài khoản chưa có gói → ghi lại `expiresAt`.
2. Cấp lại ngay lập tức → `expiresAt` mới = cũ **+ 30 ngày**, không phải `bây giờ + 30 ngày`.

### Gửi lại yêu cầu (SC-018)

```bash
KEY=$(uuidgen)
# gửi hai lần với CÙNG một Idempotency-Key
```

Lần hai phải trả `skipped` với `reason: duplicate_request` và **thời hạn không đổi**.

### Hai quản trị viên cùng lúc (SC-019)

Chạy hai yêu cầu song song với hai `Idempotency-Key` khác nhau, cùng một tài khoản. Xác nhận **một** thành công, một cái lỗi với `reason: concurrent_modification`, và thời hạn chỉ cộng dồn **một** lần.

### Từ chối toàn bộ yêu cầu

| Gửi | Mong đợi |
|---|---|
| `items: []` | `400` — danh sách rỗng |
| 501 phần tử | `400` — vượt giới hạn |
| Vượt 60 yêu cầu/phút | `429` kèm số giây phải chờ |
| Gửi 7 yêu cầu >1 phần tử trong một phút | `429` từ yêu cầu thứ 7 |

### Cột mốc thời gian (SC-015)

| Quy mô | Ngưỡng | Cách đo |
|---|---|---|
| 1 phần tử | p95 < 5 giây | 100 lần gọi tuần tự, lấy phân vị 95 |
| 500 phần tử | < 60 giây | Tạo trước 500 tài khoản hợp lệ, gọi một lần |

Nếu 500 phần tử vượt 60 giây, kiểm tra `server.write_timeout_seconds` đã được nâng lên `120` chưa (xem [research.md](./research.md) R9) — nếu chưa, đáp lời sẽ bị cắt giữa chừng.

### Quyền tính phí có hiệu lực (SC-014)

Sau khi cấp thành công, tài khoản phải dùng được tính năng AI trả phí trong **60 giây**. Vòng lặp kiểm tra:

```bash
# ngay sau khi cấp, chờ rồi thử gọi chat với hạn mức Premium
sleep 60
```

Không thao tác tra cứu lần hai trong bất kỳ trường hợp nào (SC-016).

### Ghi nhận ký và thông báo (SC-021, FR-083, FR-084)

```sql
SELECT actor_user_id, target_id, payload->>'planSlug' AS plan, request_ip, created_at
FROM shared.admin_audit_logs
WHERE action = 'subscription.grant'
ORDER BY created_at DESC LIMIT 20;
```


- *(024 T145)* Truy van nay truoc day loc `action = 'subscription.bulk_grant'` va **luon tra ve 0 dong**: hang do chi duoc khai bao, khong noi nao ghi. Da sua theo hanh vi thuc te.

  Day la sua *tai lieu*, khong phai bo sung ma. FR-081/FR-082 yeu cau **mot dong cho moi tai khoan duoc cap**, va chinh cac truong payload da dinh danh duoc *lan yeu cau* - nen them mot hang thu hai nghia la them mot dong kiem toan cho *moi lan goi*, tuc la them them mot ban ghi `outbox_records` theo FR-015, de phuc vu mot cau truy van ma cac truong san co da tra loi duoc. Hang `subscription.bulk_grant` da duoc go khoi `shared/domain/constants/adminaudit/action.go` vi vay.

  Muon gom theo tung *lan yeu cau* thi dung `identifierKey` (gia tri `Idempotency-Key`) cung `itemIndex`:

  ```sql
  SELECT payload->>'identifierKey' AS request_key,
         count(*)                 AS accounts_granted,
         min(created_at)          AS first_row_at
  FROM shared.admin_audit_logs
  WHERE action = 'subscription.grant'
    AND payload->>'identifierKey' = '<idem-key-cua-ban>'
  GROUP BY 1;
  ```
Phải có **một** dòng cho mỗi tài khoản được cấp thành công, kèm địa chỉ mạng. Thông báo gửi đi dùng **cùng một** mẫu bất kể danh sách một hay nhiều phần tử.

---

## GĐ8 — Khôi phục, kiểm chứng, quan sát

**Kiểm chứng**: FR-086 đến FR-094, SC-023, SC-025, SC-026, SC-028

### Khôi phục (FR-088)

```bash
# 1. Chạy thử trước — mặc định dryRun
curl -sS -X POST "$BASE/admin/subscription-grants/restore" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" -d '{"dryRun": true}'

# 2. Thực sự phát lại tín hiệu
curl -sS -X POST "$BASE/admin/subscription-grants/restore" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" -d '{"dryRun": false}'

# 3. Chạy lại lần nữa — phải báo 0 khôi phục, không tạo tác dụng phụ
```

Phải quét **cả** tài khoản gói miễn phí lẫn gói trả phí, không phân biệt.

*(024 T204)* **Cần dữ liệu để quét thì công cụ mới làm được việc của nó.** Công cụ quét `subscription.user_subscriptions` (`restore_policy_grants_uc.go:249`), nên một tài khoản không có bản ghi gói sẽ **không bao giờ** xuất hiện trong `res.Scanned`. Fixture `08-seed_grant_test_data.sql` không tạo bản ghi gói — và đó là đúng cho mục đích của nó (đường dự phọng FR-005b/c/d), nhưng nghĩa là trước T204 **không có fixture nào kiểm thử được FR-088**: `restored` luôn bằng 0 vì không có gì để khôi phục.

```bash
# tạo 6 tài khoản có gói nhưng cố ý thiếu bản ghi mức quyền
psql -f database/scripts/11-seed_restore_test_data.sql
```

| Tài khoản | Gói | Kết quả mong đợi của `dryRun: false` |
|---|---|---|
| `t024restore01`…`03` | free | `restored` — FR-088 không phân biệt gói miễn phí |
| `t024restore04`…`05` | `premium-monthly` | `restored` — gói trả phí |
| `t024restore06` | `premium-monthly`, **đã hết hạn** | `skipped` / `subscription_expired` |

Lưu ý khi đọc số: `scanned` là **toàn bộ** bảng `user_subscriptions`, không chỉ sáu tài khoản này, nên phải so **chênh lệch** so với lần chạy `dryRun` trước đó, và tự lọc `t024restore` trong mảng `results`. Muốn kiểm thử lại từ đầu: `psql -f database/scripts/11-seed_restore_test_data.sql --set cleanup=1` rồi tạo lại.

### Kiểm chứng (FR-090)

```bash
psql "$DSN" -f database/scripts/07-verify_policy_grants.sql
```

| Trường hợp | Mong đợi |
|---|---|
| Có bản ghi gói nhưng thiếu bản ghi mức quyền tính phí | **0** (SC-026) |
| Nhiều bản ghi mức quyền tính phí cùng hiệu lực | **0** |
| `plan_code` không khớp gói hiện tại | **0** |
| Ảnh chụp chính sách rỗng | **0** |
| Bản ghi `closed`/`future` đã hết hiệu lực | **0** |
| **Cổng 1 lệch cổng 2 theo từng thao tác** | **0** (FR-004, SC-011) |

Hai dòng cuối là dòng mới. Dòng "cổng lệch" là dòng **quan trọng nhất** về mặt khách hàng: nó bắt được tình huống khách thấy một hạn mức lớn hơn hạn mức mà hệ thống thực sự cho dùng, vì hệ thống âm thầm hạ mô hình chứ không báo lỗi. Chat của gói Premium đã chốt là **10** ở **cả hai cổng**, nên tình huống "thấy 20 nhưng chỉ dùng được 10" là bất khả thi; ngoại lệ duy nhất SC-011 chấp thuận là hạn mức thử outfit của gói free, cố ý thấp hơn đúng một lượt so với con số khách thấy (cổng 1 = 5, cổng 2 = 4, FR-005d). Tệp này **chỉ đọc** — chạy nhiều lần không đổi gì (FR-091).

### Quan sát (FR-086, FR-087, SC-023)

Tạo từng tình huống từ chối và xác nhận đều để lại dấu vết:

| Tình huống | Mã lý do mong đợi |
|---|---|
| Đẩy nội dung `eventVersion: 99` | `unsupported_event_version` |
| Đẩy `eventType` bịa | `unknown_event_type` |
| Đẩy nội dung phiên bản 1 | `missing_source_event_id` |
| Gửi sự kiện v1 (không có `toPlanId`) | `missing_required_field` |
| Gửi `aiCostPolicyId` không tồn tại trong bảng `aiexecutor.ai_cost_policies` | `ai_cost_policy_not_found` |

Mỗi lần phải truy vết được tới mã tài khoản, lý do và thời điểm — **kể cả khi không có thao tác cấp gói nào đi kèm** (FR-086). Đây là điểm dễ bỏ sót nhất: tín hiệu từ hết thời hạn hay bật tự gia hạn không đi kèm thao tác cấp gói nào.

#### ⚠️ (024 T204) Cấu hình chính sách hỏng giờ có thể chặn việc thu hồi quyền

Từ T188, `SUBSCRIPTION_DOWNGRADED_TO_FREE` đi vào **đường tạo bản ghi** thay vì đường thu hồi, nên nó phải đọc chính sách phí AI **trước** khi đóng bản ghi cũ. Vì vậy mã `ai_cost_policy_not_found` ở dòng cuối của bảng trên không chỉ có nghĩa là "không tạo được bản ghi mới" — trước T204 nó còn kéo theo một hệ quả nặng hơn: tín hiệu bị **từ chối vĩnh viễn** (`RejectionError`, đưa vào dead-letter, không bao giờ phát lại), **và** bản ghi mức quyền Premium **cũng không được đóng**. Khách hàng đã hạ gói vẫn giữ nguyên quyền Premium, còn bộ đếm chỉ ghi một dòng cấu hình.

Không phải điều giả định: cột `subscription.subscription_plans.ai_cost_policy_id` là `NOT NULL` nhưng **không có khoá ngoại** tới `aiexecutor.ai_cost_policies` — `database/scripts/closy_db-schema.sql:998` khai báo cột, còn khối `ADD CONSTRAINT` ở `:1181-1189` không có ràng buộc này (migration `20260622131631_add_ai_cost_policy.sql:25` và `20260801000000_separate_module_schemas.sql:180-181` từng tạo, nhưng nó không còn trong schema đang chạy). Một giá trị sửa tay, chép từ môi trường khác, hay trỏ tới chính sách chỉ tồn tại ở schema cũ đều không bị tầng dữ liệu chặn. Phía gửi cũng không chặn: `BuildPlanChangedPayload` lấy `aiCostPolicyId` từ `plan.AICostPolicyID.String()`, parser chỉ kiểm tra giá trị **parse được** (`grant_payload_parser.go:157`), và `ValidatePlanChangedPayload` kiểm tra `toPlanId` + `sourceEventId` nhưng **không** kiểm tra trường này.

**Sau T204**, đường hạ cấp đóng bản ghi cũ **trước khi** từ chối, nên một cấu hình sai ở danh mục chính sách không còn để lại khách hàng ở mức phí cũ:

- Tín hiệu **vẫn bị từ chối** với đúng mã `ai_cost_policy_not_found`, **không** ghi bản chính sách giả, và `grant_rejected_by_reason` **vẫn tăng** — FR-086 không bị nới, FR-085 vẫn còn câu chữ tiếng Việt nêu đúng nguyên nhân.
- Bản ghi Premium cũ **được đóng** trong một transaction riêng, có khoá theo tài khoản như mọi đường ghi khác. Nếu ghi lỗi, lỗi đó là `transient` nên được **phát lại**, không phải từ chối.
- Nhánh này **giới hạn cho hạ cấp**. `UPGRADED` không đóng gì: khách chưa mất gì, họ vẫn giữ mức cũ, và đóng bản ghi đó là lấy đi đúng quyền họ còn có.

Muốn kiểm chứng: đặt `ai_cost_policy_id` của gói free thành một UUID không tồn tại, hạ cấp một tài khoản Premium, rồi xem `ai_policy_grants` của tài khoản đó — phải **không còn** grant `active`, và bộ đếm `ai_cost_policy_not_found` phải tăng 1. Trả lại `ai_cost_policy_id` về giá trị đúng sau khi kiểm chứng.

**Cách đọc hai bộ đếm này.** Bảng trên nêu *mã lý do mong đợi*, nhưng không nói đọc số đếm ở đâu - và đây là nơi duy nhất trong tài liệu nói được. Hai bộ đếm nằm trong phản hồi của endpoint sức khoẻ dịch vụ AI, không phải trong log và không phải trong cơ sở dữ liệu:

```bash
curl -s http://localhost:8103/api/v1/ai/health
```

Phần cần xem nằm ở hai trường cấp cao nhất:

```json
{
  "service": "ai",
  "status": "healthy",
  "grant_rejected_by_reason": {
    "ai_cost_policy_not_found": 1,
    "stale_event": 0
  },
  "cost_fallback_by_reason": {
    "tier2_total": 3
  }
}
```

* `grant_rejected_by_reason` - đếm theo **từng mã lý do từ chối** của luồng grant (FR-087). Sau khi gửi một sự kiện hỏng, mã tương ứng trong bảng trên phải tăng đúng 1.
  Nếu không tăng thì sự kiện chưa tới consumer, chứ không phải bị từ chối ở đó - hãy kiểm tra hàng đợi trước khi nghi ngờ bộ đếm.
* `cost_fallback_by_reason` - đếm số lần phải dùng tier-2 do hết hạn mức, theo lý do.

Hai lưu ý về cách đọc, vì cả hai đều dễ làm ra kết luận sai:

- Bộ đếm **không tự về 0**. Chúng là bộ đếm đơn giản cho tới hết vòng đời tiến trình. Muốn đo lại từ đầu thì phải khởi động lại dịch vụ AI, hoặc chấp nhận so sánh với giá trị trước khi gửi.
- Gửi nhiều sự kiện hỏng liên tiếp sẽ làm tăng bộ đếm, nên hãy chạy từng kịch bản một lần từ trạng thái đã biết. Đây cũng là lý do SC-023 yêu cầu cảnh báo **trước** khi người dùng đánh giá: bộ đếm đã bẩn thì một lần chạy sai không còn là tín hiệu nữa.

### Cấu hình cảnh báo (FR-087, SC-023)

Cần nói thẳng cơ chế thật, vì hai bộ đếm trên dễ bị hiểu là có sẵn ngưỡng báo động. **Không có.** Hệ thống **không** có metrics backend — không có Prometheus client trong `services/ai`, không có exporter, và không có ai scrape hai bộ đếm đó. Cảnh báo là **log và vô điều kiện**: mỗi lần từ chối đều ghi một dòng `WARN` kèm `reason` và `user_id`, kể cả lần đầu tiên.

Hệ quả thực tế: muốn được cảnh báo thì phải **tự cấu hình alert theo log** ở hệ thống giám sát. Không có ngưỡng số nào để bật, và cũng không có con số nào để so sánh.

Dòng log từ chối có các trường:

| Trường | Ý nghĩa |
|---|---|
| `reason` | Mã lý do ở bảng phía trên — trường để lọc cảnh báo |
| `user_id` | Tài khoản bị ảnh hưởng, có trên **mọi** dòng kể cả khi payload không parse được |
| `rejected_total_for_reason` | Tổng số lần bị từ chối với mã lý do đó, tính từ lúc tiến trình bắt đầu |
| `source_event_id` | Chỉ có ở nhánh tín hiệu không parse được |

```bash
# Lọc theo mã lý do để gắn cảnh báo
docker logs ai-closy-dev 2>&1 | grep '"reason":"ai_cost_policy_not_found"'
```

Hai bộ đếm trên health response (`grant_rejected_by_reason` và `cost_fallback_by_reason`) là **cùng một dữ liệu đó**, phục vụ ở dạng số chạy được. Hai giới hạn phải biết trước khi dùng chúng làm số liệu:

- Chúng là **process-local**: nằm trong bộ nhớ tiến trình, **mất sạch khi dịch vụ AI khởi động lại**. Vì vậy **không dùng làm số liệu thống kê dài hạn** và không so sánh được giữa hai lần triển khai — muốn có lịch sử dài hạn thì phải lấy từ hệ thống log, không phải từ endpoint này.
- Một ngoại lệ duy nhất: lý do `duplicate_source_event` được ghi ở mức `INFO` thay vì `WARN`, vì nó là hệ quả tất yếu của giao hàng at-least-once chứ không phải sự cố. Nếu cảnh báo của bạn gom mọi thứ, hãy loại mã này ra — nếu không, nó sẽ làm nhiễu đúng những lý do bạn cần nhìn thấy.

*(Vì sao chấp nhận trạng thái này: hệ thống **cố ý không tự chế ngưỡng**. Một sự kiện bị từ chối là tín hiệu **toàn vẹn dữ liệu**, không phải tín hiệu tải, và số lượng của nó bị chặn trên bởi tần suất đổi gói thực tế. Một ngưỡng chưa ai hiệu chỉnh thì tệ hơn là không có: hoặc không bao giờ bắn, hoặc bắn liên tục, và người trực sẽ bỏ qua nó. Đây là mô tả hiện trạng — đóng khoảng trống này nghĩa là thêm metrics backend, việc đó là hạ tầng chứ không phải đề xuất kiến trúc trong phạm vi 024.)*


---

## Điều kiện hoàn thành

Tính năng chỉ được coi là xong khi **toàn bộ** mục dưới đây đạt:

- [ ] Mọi kịch bản GĐ0 → GĐ8 ở trên đạt trên môi trường phát triển
- [ ] `make wire-all && make lint && make test-race` xanh
- [ ] `make generate-check` không báo sai lệch
- [ ] Tệp kiểm chứng báo **0** ở cả sáu trường hợp
- [ ] Không tài khoản nào mất quyền tính phí trong suốt quá trình triển khai
- [ ] Truy vấn "tài khoản có từ hai mức quyền tính phí cùng hiệu lực" trả về không dòng
- [ ] `rg "ai_policy_grants" services/account/` không còn câu lệnh ghi nào

**Điều kiện cổng xanh theo hiến pháp** trước khi đề xuất commit:

```bash
make wire && make lint && make test-race
```
---

## Hai kiểm thử tích hợp Postgres — phải chạy tay, không nằm trong CI (SC-006, T212)

Ba kiểm thử ở `services/account/internal/modules/subscription/application/usecase/subscription/eager_provision_postgres_integration_test.go` **không chạy trong gate nào**. `renewal_integration_test.go:165-168` gọi `t.Skip` khi thiếu `TEST_DATABASE_URL`, còn `.github/workflows/ci.yml` không khai báo service postgres và không đặt biến này, nên `make test-race` (`Makefile:272-279`) bỏ qua chúng. Đó là ba test:

- `TestIntegrationEagerProvisioningRollsBackWithTheCallersTransaction` — việc ghi phải nằm trong transaction của caller
- `TestIntegrationConcurrentEagerAndLazyProvisioningCreateOneRecord` — cấp phát sẵn và cấp phát trễ chạy đồng thời vẫn chỉ tạo một bản ghi
- `TestIntegrationEagerUserForeignKeyRejectsAnUnknownAccount` — khoá ngoại thật **từ chối** một dòng gói của tài khoản không tồn tại

Chúng là bằng chứng duy nhất cho cửa cấp phát mặc định (không tạo trùng entitlement, ghi nối vào transaction của caller, và ràng buộc khoá ngoại có hiệu lực), nên **phải chạy tay** trước khi kết luận 024 đạt.

### Vì sao chưa đưa vào gate

`make test-integration` (`Makefile:284-288`) hiện **không** thể chạy xanh trong CI, và đó là lý do:

1. Nhánh `shared` cần nhiều hơn một database trần. `TestIntegration_ClaimConcurrent` đọc và ghi `shared.outbox_records` bằng tên bảng đủ điều kiện (`shared/infrastructure/messaging/integration_test.go:261`) — bảng đó chỉ tồn tại sau khi đã migrate (`database/scripts/closy_db-schema.sql:864`) — còn `TestIntegration_OutboxPublishBatch` mở `search_path=shared` (`shared/infrastructure/messaging/integration_outbox_test.go:83`) rồi nối thẳng broker RabbitMQ ở `127.0.0.1:5672` (`integration_outbox_test.go:95`). Một service `postgres` trần không đủ.
2. Nhánh `services/account` tự xoá sạch bảng ở đầu mỗi test (`renewal_integration_test.go:195-213` chạy `DELETE ... WHERE 1 = 1`), nên các package chạy song ánh trên cùng một database sẽ xoá dữ liệu của nhau. `go test -run Integration ./internal/modules/subscription/...` đỏ ổn định ở `TestIntegrationProcessWebhook*`; thêm `-p 1` thì xanh.

Sửa cả hai điểm là việc ngoài phạm vi 024. Thêm một bước gate biết trước là đỏ thì tệ hơn không có gate, nên 024 ghi lệnh tái lập thay vì thêm service vào workflow.

### Lệnh tái lập (đã chạy, xanh)

```bash
# 1. Database Postgres trống, dành riêng cho kiểm thử. KHÔNG trỏ vào database đang
#    phát triển: các test này xoá sạch bảng subscription. Cổng 55432 để không đụng
#    cổng 5432 của postgres-closy-dev.
docker run -d --name closy-t212-pg -p 55432:5432 \
  -e POSTGRES_USER=admin -e POSTGRES_PASSWORD=123456 -e POSTGRES_DB=closy_test \
  pgvector/pgvector:pg17

# 2. Biến môi trường. Không có biến này thì mọi test tích hợp tự skip.
export TEST_DATABASE_URL="host=localhost user=admin password=123456 dbname=closy_test port=55432 sslmode=disable"

# 3. Chạy đúng ba test này
cd services/account
go test -race -count=1 -v \
  -run "TestIntegrationEagerProvisioningRollsBackWithTheCallersTransaction|TestIntegrationConcurrentEagerAndLazyProvisioningCreateOneRecord|TestIntegrationEagerUserForeignKeyRejectsAnUnknownAccount" \
  ./internal/modules/subscription/application/usecase/subscription/

# 4. Dọn
cd ../.. && docker rm -f closy-t212-pg
```

Kết quả mong đợi: `--- PASS` cho cả ba test, rồi `ok .../application/usecase/subscription`.

Không dùng `make test-integration` cho các test này. Target đó chạy thêm `./infrastructure/messaging/...` (`Makefile:287`) vốn cần RabbitMQ và schema `shared` đã migrate, nên sẽ đỏ. Lệnh ở bước 3 là lệnh giới hạn đúng phạm vi của 024.

Bỏ biến đi (`unset TEST_DATABASE_URL`) thì `make test-race` trở lại đúng như cũ và ba test ở trên hiện `--- SKIP`. Gate hiện có **không** đổi: `make test-race` vẫn không cần database.

---

## GĐ9 — Kiểm chứng FR-041 bằng bằng chứng chạy thật (T180, T190)

<!-- FR-041-VERIFIED -->

**Ngày chạy (lần T190 — sau khi sửa mã)**: 2026-10-01, 10:05–10:22 giờ Việt Nam / 03:05–03:22 UTC.

**Kết luận một câu**: **FR-041 ĐẠT**. Sau khi `CreatesNewGrant` trả `true` cho `SUBSCRIPTION_DOWNGRADED_TO_FREE`, mục 1 của tập kiểm chứng về **0 ngay sau khi hạ cấp**, **không** cần gọi `POST /admin/subscription-grants/restore`. Phạm vi bằng chứng thực tế nay là **8/8 loại thay đổi gói**.

### 1. Môi trường

| Mục | Giá trị |
|---|---|
| Database | `smart_wardrobe_db` trong container `postgres-closy-dev` (PostgreSQL) |
| Stack | `deployments/docker-compose.yml`: `postgres-closy-dev`, `rabbitmq-closy-dev`, `redis-closy-dev`, `account-closy-dev` (host `:8101`), `ai-closy-dev` (`:8103`) |
| Mã nguồn | Nhánh làm việc hiện tại, feature 024, kèm thay đổi T188 tại `services/ai/internal/modules/aiexecutor/application/usecase/grant/grant_window.go` |
| Tập kiểm chứng | `database/scripts/07-verify_policy_grants.sql` (chỉ `SELECT` + `\echo`, không ghi — FR-091) |

**Đã chờ cả hai container báo `healthy` trước khi kiểm chứng**, vì kết luận phụ thuộc vào việc mã mới đã nạp:

```
$ docker compose -f deployments/docker-compose.yml --env-file .env up -d --build account ai
$ docker inspect --format "{{.Name}} {{.State.Health.Status}}" account-closy-dev ai-closy-dev
/account-closy-dev healthy
/ai-closy-dev healthy
```

### 2. Tài khoản thử nghiệm (tiền tố `t190evt`)

Bốn tài khoản, tạo qua `POST /api/v1/admin/users` (không tạo trực tiếp trong SQL):

| Username | UUID | Dùng để chứng minh |
|---|---|---|
| `t190evt01` | `30ceb8ee-51f7-40f4-bfcc-83c5b9c874bd` | `SUBSCRIPTION_DOWNGRADED_TO_FREE` — kịch bản bắt buộc của T190 |
| `t190evt02` | `e93be2c0-27df-458f-8e39-7073a3e1c4a2` | `UPGRADED` |
| `t190evt03` | `4df403be-499d-4a12-869f-a60dc45217e2` | `OVERLAID` + `LIFETIME_FALLBACK_RESTORED` |
| `t190evt04` | `c6089d02-7483-4394-b1a3-501a2729d360` | `AUTO_RENEW_UPDATED` |

Mật khẩu của các tài khoản thử nghiệm do người chạy đặt qua `$TEST_USER_PASSWORD` — xem mục **"Lấy token quản trị viên"** bên dưới, tài liệu không in giá trị mật khẩu nào.

### 3. Fixture: ba gói hội viên mới

Gốc `UPGRADED`, `OVERLAID` và `LIFETIME_FALLBACK_RESTORED` đòi hỏi một danh mục gói mà seed không có (chỉ có `free` và `premium-monthly`, cả hai đều không sinh ra ba loại sự kiện đó). Ba gói fixture được tạo qua **endpoint thật** `POST /api/v1/admin/subscriptions/plans`:

| Slug | `planKind` | `tierRank` | `durationDays` | Dùng để |
|---|---|---|---|---|
| `t190evt-lifetime` | `2` (Lifetime) | 1 | — | phủ định với gói hạn hạn, tạo tiền đề cho `OVERLAID` |
| `t190evt-finite-hi` | `1` (Finite) | 2 | 30 | `UPGRADED` |
| `t190evt-finite-exp` | `1` (Finite) | 3 | 0 (sửa thành 1 sau, xem dưới) | `OVERLAID` rồi `LIFETIME_FALLBACK_RESTORED` |

Hai việc **không** đi qua HTTP, đều là chuẩn bị fixture chứ không phải sửa dữ liệu nghiệp vụ, và đều được nêu ở đây:

1. `POST /admin/subscriptions/plans` và `PUT /admin/subscriptions/plans/:id` **không có trường `aiCostPolicyId`** (`CreateSubscriptionPlanReq` / `UpdateSubscriptionPlanReq` tại `services/account/dto/subscription/billing_dto.go:119-143`). Một gói mới tạo qua API vì thế có `ai_cost_policy_id = 00000000-0000-0000-0000-000000000000`, và mọi sự kiện phát ra từ nó sẽ bị dịch vụ AI từ chối với `ai_cost_policy_not_found`. Ba gói fixture vì vậy được gán `premium-default` bằng **một `UPDATE` duy nhất trên đúng ba dòng do chính lần chạy này tạo ra**.
2. `validateRenewalPlan` (`subscription_renewal_workflow.go`) từ chối gói có `price <= 0` hoặc `durationDays <= 0`, nên gói overlay với `durationDays = 0` **không bao giờ** được job gia hạn xử lý. Sau khi overlay đã tạo `expires_at` ở thời điểm mua, `price` và `durationDays` của gói được sửa lại qua `PUT /api/v1/admin/subscriptions/plans/:id` (`price 1000`, `durationDays 1`).

### 4. Lấy token quản trị viên

Dùng biến môi trường, không in giá trị mật khẩu (T189). Chi tiết ở mục **"Lấy token quản trị viên"** trong phần bằng chứng T180 bên dưới:

```bash
export ADMIN_PASSWORD='<mật khẩu admin của môi trường phát triển>'
export ADMIN_TOKEN=$(curl -sS -X POST http://localhost:8101/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d "{\"loginName\":\"admin\",\"password\":\"${ADMIN_PASSWORD}\"}" | jq -r .data.accessToken)
export TEST_USER_PASSWORD='<mật khẩu tài khoản thử nghiệm>'
```

### 5. Các lệnh đã chạy

```bash
# 1. Tạo bốn tài khoản thử nghiệm
curl -sS -X POST http://localhost:8101/api/v1/admin/users \
  -H "Authorization: Bearer $ADMIN_TOKEN" -H "Content-Type: application/json" \
  -d "{\"username\":\"t190evt01\",\"email\":\"t190evt01@example.test\",
       \"password\":\"${TEST_USER_PASSWORD}\",\"firstName\":\"T190\",\"lastName\":\"Evidence\",
       \"dateOfBirth\":\"1990-01-01\",\"gender\":1}"
# -> lặp lại cho t190evt02..t190evt04

# 2. Kịch bản BẮT BUỘC — hạ cấp xuống gói miễn phí
curl -sS -X POST http://localhost:8101/api/v1/admin/subscription-grants \
  -H "Authorization: Bearer $ADMIN_TOKEN" -H "Idempotency-Key: $(uuidgen)" \
  -H "Content-Type: application/json" \
  -d '{"items":[{"userIdentifier":"t190evt01","planSlug":"premium-monthly"}]}'
# -> total 1, succeeded 1, failed 0     (sự kiện ACTIVATED, gói premium-monthly)
sleep 20
curl -sS -X POST \
  http://localhost:8101/api/v1/admin/subscriptions/30ceb8ee-51f7-40f4-bfcc-83c5b9c874bd/cancel \
  -H "Authorization: Bearer $ADMIN_TOKEN" -H "Content-Type: application/json" -d '{}'
# -> "Hủy gói hội viên của người dùng thành công"   (SUBSCRIPTION_DOWNGRADED_TO_FREE)
sleep 25

# 3. Chạy tập kiểm chứng — KHÔNG gọi restore
cmd /c "docker exec -i postgres-closy-dev psql -U admin -d smart_wardrobe_db -v ON_ERROR_STOP=1 -f - < database\scripts\07-verify_policy_grants.sql"

# 4. UPGRADED — t190evt02 mua gói hạn hạn tier 2 rồi tier 3
curl -sS -X POST http://localhost:8101/api/v1/subscriptions/me/purchase-with-wallet \
  -H "Authorization: Bearer $USER_TOKEN_02" -H "Content-Type: application/json" \
  -d '{"planSlug":"t190evt-finite-hi"}'
curl -sS -X POST http://localhost:8101/api/v1/subscriptions/me/purchase-with-wallet \
  -H "Authorization: Bearer $USER_TOKEN_02" -H "Content-Type: application/json" \
  -d '{"planSlug":"t190evt-finite-exp"}'

# 5. OVERLAID — t190evt03 mua gói trọn đời rồi phủ gói hạn hạn lên trên
curl -sS -X POST http://localhost:8101/api/v1/subscriptions/me/purchase-with-wallet \
  -H "Authorization: Bearer $USER_TOKEN_03" -H "Content-Type: application/json" \
  -d '{"planSlug":"t190evt-lifetime"}'
curl -sS -X POST http://localhost:8101/api/v1/subscriptions/me/purchase-with-wallet \
  -H "Authorization: Bearer $USER_TOKEN_03" -H "Content-Type: application/json" \
  -d '{"planSlug":"t190evt-finite-exp"}'

# 6. AUTO_RENEW_UPDATED — t190evt04 đang giữ gói hạn hạn premium-monthly
curl -sS -X PUT http://localhost:8101/api/v1/subscriptions/me/auto-renew \
  -H "Authorization: Bearer $USER_TOKEN_04" -H "Content-Type: application/json" \
  -d '{"enabled":true}'

# 7. Sửa price/duration của gói overlay, rồi kích hoạt job gia hạn theo lịch
curl -sS -X PUT http://localhost:8101/api/v1/admin/subscriptions/plans/3ae93071-9e6b-4ecd-8b9b-5d9e08906ecc \
  -H "Authorization: Bearer $ADMIN_TOKEN" -H "Content-Type: application/json" \
  -d '{"price":1000,"durationDays":1}'
# Job gia hạn chạy một lần lúc khởi động (TriggerStartup) và một lần mỗi ngày lúc 00:00:00.
# Khởi động lại container để chạy ngay:
docker restart account-closy-dev
```

Nhật ký job gia hạn sau khi khởi động lại:

```
{"level":"INFO","ts":"2026/10/01 - 10:17:55","msg":"Downgraded subscription during scheduled renewal","user_id":"4df403be-...","plan_id":"3ae93071-...","result":"downgraded","reason":"downgraded_auto_renew_disabled"}
{"level":"INFO","ts":"2026/10/01 - 10:17:55","msg":"Worker run succeeded","worker":"subscription_renewal","triggerType":"startup","result":"success","totalCount":2,"successCount":2,"warningCount":0,"errorCount":0,"skippedCount":0,"renewedCount":0,"downgradedCount":2,"failedCount":0}
```

### 6. Chuỗi sự kiện thật, quan sát được trong outbox

Toàn bộ dữ liệu kiểm chứng được tạo bằng mã nghiệp vụ thật qua HTTP; không có `INSERT` nào vào bảng subscription. Mọi dòng đều `status = published`, `attempts = 0`:

```
 username  |           event_type            |    to_plan_code    | from_plan_code   | status   | attempts
-----------+---------------------------------+-------------------+------------------+----------+---------
 t190evt01 | ACTIVATED                       | premium-monthly    |                  | published |        0
 t190evt01 | SUBSCRIPTION_DOWNGRADED_TO_FREE | free               | premium-monthly  | published |        0
 t190evt02 | ACTIVATED                       | t190evt-finite-hi  | free             | published |        0
 t190evt02 | UPGRADED                        | t190evt-finite-exp | t190evt-finite-hi | published |        0
 t190evt02 | SUBSCRIPTION_DOWNGRADED_TO_FREE | free               | t190evt-finite-exp | published |        0
 t190evt03 | ACTIVATED                       | t190evt-lifetime   | free             | published |        0
 t190evt03 | OVERLAID                        | t190evt-finite-exp | t190evt-lifetime | published |        0
 t190evt03 | LIFETIME_FALLBACK_RESTORED      | t190evt-finite-exp | t190evt-finite-exp | published |        0
 t190evt04 | ACTIVATED                       | premium-monthly    |                  | published |        0
 t190evt04 | AUTO_RENEW_UPDATED              | premium-monthly    | premium-monthly  | published |        0
```

`EXTENDED` không xuất hiện trong bảng trên vì nó đã được T180 chứng minh bằng chính `t180fr041a`; xem phần bằng chứng T180.

### 7. Bằng chứng cốt lõi: hạ cấp vừa đóng bản ghi cũ vừa tạo bản ghi mới

Hai dòng của `t190evt01`, mỗi bản ghi có **đủ bốn thao tác** trong `entitlements`:

```
 current_plan_code |    plan_code    | status |        effective_from         |        effective_to          | ent
-------------------+-----------------+--------+-------------------------------+-------------------------------+----------
 free              | premium-monthly | closed | 2026-10-01 03:11:37.235788+00 | 2026-10-01 03:11:57.302877+00 | chat
 free              | premium-monthly | closed | 2026-10-01 03:11:37.235788+00 | 2026-10-01 03:11:57.302877+00 | outfit
 free              | premium-monthly | closed | 2026-10-01 03:11:37.235788+00 | 2026-10-01 03:11:57.302877+00 | summary
 free              | premium-monthly | closed | 2026-10-01 03:11:37.235788+00 | 2026-10-01 03:11:57.302877+00 | rewriter
 free              | free            | active | 2026-10-01 03:11:57.302877+00 |                               | chat
 free              | free            | active | 2026-10-01 03:11:57.302877+00 |                               | outfit
 free              | free            | active | 2026-10-01 03:11:57.302877+00 |                               | summary
 free              | free            | active | 2026-10-01 03:11:57.302877+00 |                               | rewriter
```

Đối chiếu với lần chạy T180 (cùng câu lệnh, khác tài khoản): ở T180 tài khoản bị hạ cấp chỉ có **một** dòng `premium-monthly | closed` và không có dòng `free`. `effective_to` của dòng Premium trùng đúng `effective_from` của dòng `free`, tức là đóng và mở xảy ra trong cùng một khoảnh thời gian, đúng FR-031.

Chính sách `free-default` trong cơ sở dữ liệu phát triển có **đủ bốn thao tác**, nên bản ghi mới không thiếu thao tác nào:

```
      code       | operation | is_enabled | normal_route | max_paid_attempts_per_day
-----------------+-----------+------------+--------------+---------------------------
 free-default    | chat      | t          | paid         |                         3
 free-default    | outfit    | t          | paid         |                         4
 free-default    | rewriter  | t          | free         |                         5
 free-default    | summary   | t          | free         |                         5
```

### 8. Output thật của `07-verify_policy_grants.sql`

Dán nguyên văn, không tóm tắt:

```
=== 1. Accounts with a plan but no policy grant ===
--- Expected 0. A non-zero count means those accounts cannot use paid AI
--- at all: the read path finds no grant and degrades, and the customer
--- sees a free-model answer for something they paid for. The restore
--- tool (POST /api/v1/admin/subscription-grants/restore) is the fix. ---
 accounts_missing_grant
------------------------
                      0
(1 row)


=== 2. Accounts with more than one grant in force at once ===
--- Expected 0. Each row is an account granted twice, which means the
--- close and the create were not atomic. ---
 user_id | concurrent_grants
---------+-------------------
(0 rows)


=== 3. Grant plan_code disagreeing with the account current plan ===
--- Expected 0 rows. The grant records which plan it was written for; a
--- mismatch means an event was lost or arrived out of order. ---
 id | user_id | grant_plan | subscription_plan
----+---------+------------+-------------------
(0 rows)


=== 4. Grants with an empty or incomplete policy snapshot ===
--- Expected 0. The reader needs an `entitlements` group with all four
--- operations. Missing it means no limits at all are applied, which is
--- silent: nothing errors, the limits are simply not there. ---
 bad_snapshots
---------------
             0
(1 row)


=== 5. Active or future grants whose window has ended ===
--- Expected 0. A row claiming to be live after its paid period is over
--- shadows the account current plan with a revoked entitlement. ---
 live_but_expired
------------------
                0
(1 row)


=== 6. Gate 1 and gate 2 disagreeing, per operation ===
--- Expected 0 rows. This is the most customer-visible check in the
--- file: a customer whose profile shows more chat attempts per day than
--- the engine actually allows has no way to tell, because the engine
--- silently downgrades the model instead of returning an error.
---
--- Exactly ONE approved exception is excluded below, because it is a
--- decision rather than drift: the free plan outfit limit is deliberately
--- one attempt below what the customer sees, i.e. gate 1 = 5 and
--- gate 2 = 4 (FR-005d). Any OTHER difference is unexplained and must
--- be investigated.
---
--- premium-default is NOT a second exception. It does not drift because
--- the WHERE clause above never lets it in: a strict policy never reads
--- max_paid_attempts_per_day, so its value is NULL on purpose (FR-005e)
--- and the rows are dropped by enforcement_mode = observe_only before any
--- comparison happens. SC-011 approves a one-item exception list, and
--- this query honours exactly that list.
---
--- Section 6b below prints every row with a verdict, so the approved
--- exception stays visible instead of being silently hidden. ---
 plan_slug | policy_code | enforcement_mode | operation | gate1_customer_visible | gate2_ai_enforced | drift
-----------+-------------+------------------+-----------+------------------------+-------------------+-------
(0 rows)


--- 6b. Full gate comparison, including the approved exception ---
--- Every metered operation is listed, not just the two with a customer-
--- visible quota. Each row gets a verdict so the approved exception stays
--- visible instead of being silently hidden. ---
     plan_slug      |   policy_code   | enforcement_mode | operation | gate1_customer_visible | gate2_ai_enforced |                             verdict
--------------------+-----------------+------------------+-----------+------------------------+-------------------+------------------------------------------------------------------
 free               | free-default    | observe_only     | chat      |                      3 |                 3 | match
 free               | free-default    | observe_only     | outfit    |                      5 |                 4 | deliberate one-attempt difference (FR-005d)
 free               | free-default    | observe_only     | rewriter  |                        |                 5 | no customer-visible quota for this operation; nothing to compare
 free               | free-default    | observe_only     | summary   |                        |                 5 | no customer-visible quota for this operation; nothing to compare
 premium-monthly    | premium-default | strict           | chat      |                     10 |                   | strict: column is never read (FR-005e)
 premium-monthly    | premium-default | strict           | outfit    |                     30 |                   | strict: column is never read (FR-005e)
 premium-monthly    | premium-default | strict           | rewriter  |                        |                   | strict: column is never read (FR-005e)
 premium-monthly    | premium-default | strict           | summary   |                        |                   | strict: column is never read (FR-005e)
 t190evt-finite-exp | premium-default | strict           | chat      |                     10 |                   | strict: column is never read (FR-005e)
 t190evt-finite-exp | premium-default | strict           | outfit    |                     10 |                   | strict: column is never read (FR-005e)
 t190evt-finite-exp | premium-default | strict           | rewriter  |                        |                   | strict: column is never read (FR-005e)
 t190evt-finite-exp | premium-default | strict           | summary   |                        |                   | strict: column is never read (FR-005e)
 t190evt-finite-hi  | premium-default | strict           | chat      |                     10 |                   | strict: column is never read (FR-005e)
 t190evt-finite-hi  | premium-default | strict           | outfit    |                     10 |                   | strict: column is never read (FR-005e)
 t190evt-finite-hi  | premium-default | strict           | rewriter  |                        |                   | strict: column is never read (FR-005e)
 t190evt-finite-hi  | premium-default | strict           | summary   |                        |                   | strict: column is never read (FR-005e)
 t190evt-lifetime   | premium-default | strict           | chat      |                     10 |                   | strict: column is never read (FR-005e)
 t190evt-lifetime   | premium-default | strict           | outfit    |                     10 |                   | strict: column is never read (FR-005e)
 t190evt-lifetime   | premium-default | strict           | rewriter  |                        |                   | strict: column is never read (FR-005e)
 t190evt-lifetime   | premium-default | strict           | summary   |                        |                   | strict: column is never read (FR-005e)
(20 rows)
```

Mục 6b in ra **20** dòng thay vì 8 như lần T180, vì ba gói fixture `t190evt-*` cũng được kiểm kê. Không dòng nào mang phán quyết `UNEXPLAINED DRIFT` hay `MISCONFIGURATION`; ngoại lệ duy nhất vẫn là `free`/`outfit` — đúng ngoại lệ một mục mà SC-011 đã duyệt.

**Đọc từng mục — số lệch thực tế:**

| # | Truy vấn | Số lệch | Mong đợi | Đạt |
|---|---|---|---|---|
| 1 | Tài khoản có bản ghi gói nhưng thiếu bản ghi mức quyền | **0** | 0 | ✅ |
| 2 | Tài khoản có nhiều bản ghi mức quyền cùng hiệu lực | **0 dòng** | 0 dòng | ✅ |
| 3 | `plan_code` bản ghi mức quyền lệch gói hiện tại | **0 dòng** | 0 dòng | ✅ |
| 4 | Ảnh chụp chính sách rỗng hoặc thiếu thao tác | **0** | 0 | ✅ |
| 5 | Bản ghi `active`/`future` đã hết hạn | **0** | 0 | ✅ |
| 6 | Trôi cổng 1 đối cổng 2, ngoài danh sách ngoại lệ đã duyệt | **0 dòng** | 0 dòng | ✅ |
| 6b | Bảng kiểm kê đầy đủ, kèm phán quyết từng dòng | 20 dòng, **không** có `UNEXPLAINED DRIFT` | vậy | ✅ |

**Mục 1 là `0` mà `POST /admin/subscription-grants/restore` chưa từng được gọi trong lần chạy này.** Đây là điều kiện bắt buộc của T190 và nó đạt.

### 9. Bảng 8 loại thay đổi gói

| # | Loại thay đổi gói | Chạy thật (c/k) | Endpoint thật đã dùng | Bằng chứng |
|---|---|---|---|---|
| 1 | `ACTIVATED` (gói trả phí) | ✅ | `POST /admin/subscription-grants` | `t190evt01`, `t190evt04` |
| 2 | `ACTIVATED` (gói miễn phí) | ✅ | `POST /admin/subscription-grants/restore` (T180) | `t180fr041b` |
| 3 | `EXTENDED` | ✅ | `POST /admin/subscription-grants` (cấp lại, FR-052) | `t180fr041a`, T180 |
| 4 | `SUBSCRIPTION_DOWNGRADED_TO_FREE` | ✅ | `POST /admin/subscriptions/:userId/cancel` | `t190evt01`, `t190evt02` |
| 5 | `UPGRADED` | ✅ | `POST /subscriptions/me/purchase-with-wallet` | `t190evt02` |
| 6 | `OVERLAID` | ✅ | `POST /subscriptions/me/purchase-with-wallet` (gói trọn đời rồi phủ gói hạn hạn) | `t190evt03` |
| 7 | `LIFETIME_FALLBACK_RESTORED` | ✅ | **không có endpoint trực tiếp** — job `subscription_renewal` chạy lúc khởi động và 00:00:00 hằng ngày | `t190evt03` |
| 8 | `AUTO_RENEW_UPDATED` (**cố ý không tạo bản ghi** — xem ghi chú bên dưới) | ✅ | `PUT /subscriptions/me/auto-renew` | `t190evt04` |

Ghi chú về loại 7: đây là loại **duy nhất** không có endpoint gọi trực tiếp. Nó chỉ phát ra từ job gia hạn theo lịch (`SubscriptionRenewalWorker`) khi một gói hạn hạn đã hết hạn và tài khoản có gói trọn đời dự phòng. Khởi động lại container `account` là cách kích hoạt nhanh nhất; nếu không làm vậy thì phải chờ tới nửa đêm.

Ghi chú về loại 8: dấu ✅ ở cột "Chạy thật" chỉ nói sự kiện **đã phát ra**, **không** nói nó tạo bản ghi mức quyền. `AUTO_RENEW_UPDATED` **cố ý không tạo bản ghi**: bật hay tắt tự động gia hạn không đổi quyền lợi nào, nên không có gì để ghi lại — `grant_window.go:109-114` trả `false` cho loại này một cách có chủ đích. Điều này **không** làm hỏng FR-041: một tài khoản chỉ bật/tắt tự động gia hạn được khi nó **đang có gói**, và bản ghi mức quyền của nó đã được tạo từ chính lần kích hoạt cấp gói đó (xem dòng `ACTIVATED` của `t190evt04` ở mục 6). Vì vậy loại thay đổi gói này không để lại tài khoản nào thiếu bản ghi, và mục 1 của tập kiểm chứng vẫn báo `0`. Hành vi này **khác** hẳn loại 4 `SUBSCRIPTION_DOWNGRADED_TO_FREE`, vốn trước đây cũng không tạo bản ghi và đã bị T188 sửa thành có tạo — đừng đọc bảng trên là "cả tám loại đều tạo bản ghi".

### 10. Phần nào **chưa** được chứng minh

1. **Ngưỡng thời gian 60 giây (SC-014) chưa đo.** Cùng lý do với T180: thuộc GD7, không phải FR-041. Các bản ghi trong lần chạy này xuất hiện dưới 30 giây kể từ khi có tín hiệu, nhưng đó là quan sát lỏng, không phải phép đo.
2. **Không đo tải (SC-015).** Cùng lý do: thuộc GD7.
3. **Ba gói fixture `t190evt-*` là dữ liệu thử nghiệm, không phải danh mục thật.** Chúng dùng `premium-default`, nên không phát sinh trôi cổng, nhưng chúng làm mục 6b in ra 20 dòng. Xem mục **"Dữ liệu thử nghiệm để lại"** bên dưới.

### 11. Một lỗi phát hiện khi chạy thật, **không** sửa trong T190

> ⚠️ **KHỐI NÀY ĐÃ BỊ THAY THẾ BỞI T191.** Toàn bộ mục 11 ghi lại lỗi và số đo **trước** khi sửa mã, giữ nguyên có chủ ý để đối chiếu trước–sau. Mã hiện tại nằm ở `subscription_behavior.go:182-190` và **đã** chép giá trị vào biến cục bộ rồi mới lấy địa chỉ; bản lỗi từng nằm ở `:179-180`. Trích dẫn số dòng bên dưới đã cập nhật theo mã hiện hành.

`UserSubscription.OverlayLifetime` — bản lỗi từng gán con trỏ **trỏ thẳng vào chính các trường** sắp bị ghi đè (nay là `subscription_behavior.go:182-190`):

```go
u.FallbackPlanID   = &u.SubscriptionPlanID
u.FallbackPlanCode = &u.CurrentPlanCode
```

Hai dòng tiếp theo ghi đè `u.SubscriptionPlanID` và `u.CurrentPlanCode`, nên `FallbackPlanID` và `FallbackPlanCode` cũng đổi theo. Đo được trên cơ sở dữ liệu phát triển sau khi `t190evt03` overlay:

```
 subscription_plan_id |    fallback_plan_id    | fallback_plan_code | fallback_tier_rank | fallback_plan_kind
----------------------+------------------------+--------------------+--------------------+--------------------
 3ae93071-9e6b-4ecd-... | 3ae93071-9e6b-4ecd-... | t190evt-finite-exp |                  1 |                  2
```

`fallback_plan_id` trỏ về chính gói hạn hạn vừa phủ, chứ không phải gói trọn đời `e7971d82-2fa1-45e6-8359-b434a0482f74`. Vì vậy sự kiện `LIFETIME_FALLBACK_RESTORED` của `t190evt03` có `to_plan_code` và `from_plan_code` **giống nhau**. `fallback_tier_rank` và `fallback_plan_kind` sống sót vì chúng được sao chép vào biến cục bộ rồi mới lấy địa chỉ.

Sự kiện vẫn được phát và bản ghi mức quyền vẫn được tạo, nên FR-041 vẫn đạt cho loại thay đổi gói này. Nhưng tài khoản **không** quay về gói trọn đời như thiết kế dự định. Đây là lỗi tiền đề có từ trước, nằm ngoài phạm vi T188–T190.

**Trạng thái sau T191: đã sửa xong.** T191 chép `planID`/`planCode`/`rank`/`kind` vào biến cục bộ trước khi lấy địa chỉ (`subscription_behavior.go:182-190`), nên `FallbackPlanID` và `FallbackPlanCode` không còn trỏ vào các trường sắp bị ghi đè. Sự kiện `LIFETIME_FALLBACK_RESTORED` giờ trả `from_plan_code` là gói trọn đời và `to_plan_code` là gói hạn hạn mới phủ. Bảng đo ở trên là số **trước** khi sửa, giữ lại làm bằng chứng; **không** có số đo lại sau T191 trong tài liệu này.

### 12. Dữ liệu thử nghiệm T190 để lại

| Loại | Định danh | Đã dọn |
|---|---|---|
| Tài khoản | `t190evt01`, `t190evt02`, `t190evt03`, `t190evt04` | **Không** — giữ lại làm bằng chứng tái lập được |
| Gói hội viên | `t190evt-lifetime`, `t190evt-finite-hi`, `t190evt-finite-exp` | **Không** |
| Bản ghi gói | `subscription.user_subscriptions` của bốn tài khoản trên | **Không** |
| Bản ghi mức quyền | `aiexecutor.ai_policy_grants` của bốn tài khoản trên | **Không** |
| Sự kiện lịch sử | `subscription.user_subscription_events` mười dòng trong bảng ở mục 6 | **Không** |

Không có `DROP`/`TRUNCATE`/`DELETE` nào được chạy. Ba câu `UPDATE` được chạy, và **chỉ** trên dữ liệu do chính lần chạy này tạo ra: một câu gán `ai_cost_policy_id` cho ba gói fixture, và các câu cập nhật giá/thời hạn đi qua endpoint `PUT /admin/subscriptions/plans/:id`.

Bốn tài khoản và ba gói này là dữ liệu thử nghiệm trên môi trường phát triển, **không** phải dữ liệu khách hàng thật.

---

### Bằng chứng cũ của T180 — đo **trước** khi sửa mã, giữ nguyên để đối chiếu

Lần chạy này dùng `t180fr041a/b/c`, đo được `accounts_missing_grant = 1` ngay sau khi hạ cấp và chỉ về `0` sau khi gọi `POST /admin/subscription-grants/restore`. Nguyên nhân là `SUBSCRIPTION_DOWNGRADED_TO_FREE` **cố ý** không tạo bản ghi (`grant_window.go:83`, đã bị T188 đổi). Toàn bộ số liệu, lệnh và output gốc nằm ở các mục dưới đây, nguyên văn như lúc T180 chạy.

**Ngày chạy**: 2026-10-01 (giờ Việt Nam) / 2026-09-30 22:58–23:12 UTC.

**Kết luận một câu**: **FR-041 ĐẠT**, nhưng chỉ với một điều kiện đã nêu rõ bên dưới — con số `0` cuối cùng **không đạt được ngay sau khi thay đổi gói**; nó chỉ về `0` sau khi chạy công cụ khôi phục FR-088 một lần. Lý do là một loại thay đổi gói (`SUBSCRIPTION_DOWNGRADED_TO_FREE`) **cố ý không tạo bản ghi**, xem mục **"Phần nào chưa được chứng minh"** ở cuối GĐ9 này.

### Môi trường

| Mục | Giá trị |
|---|---|
| Database | `smart_wardrobe_db` trong container `postgres-closy-dev` (PostgreSQL, cổng host 5432) |
| User / password | `admin` / lấy từ `deployments/.env` (`DB_PASSWORD`) |
| Stack | `deployments/docker-compose.yml`: `postgres-closy-dev`, `rabbitmq-closy-dev`, `redis-closy-dev`, `account-closy-dev` (host `:8101`), `ai-closy-dev` (`:8103`), `commerce-closy-dev`, `nginx-closy-dev` (`:8080`) — tất cả `Up (healthy)` |
| Mã nguồn | Nhánh làm việc hiện tại, commit gốc `da5eb03d`, feature 024 (`specs/024-admin-premium-grant/`), migration `services/account/database/migrations/20260930120000_admin_premium_grant_schema.sql` |
| Tập kiểm chứng | `database/scripts/07-verify_policy_grants.sql` (chỉ `SELECT` + `\echo`, không ghi — FR-091) |

**Lưu ý về `psql`**: máy host **không có** `psql`. Mọi lệnh dưới đây đều đi qua `docker exec -i ... -f -` với nội dung pipe qua stdin. Trên PowerShell 5.1 không dùng được `<<`; dùng `cmd /c "... < file"`.

### Đường đi của bằng chứng: đường sự kiện thật, không `INSERT` thẳng

Điểm mấu chốt của FR-041 là câu **"qua đường sự kiện"**. Bản ghi mức quyền tính phí nằm ở `aiexecutor.ai_policy_grants`, thuộc sở hữu của dịch vụ AI, và **chỉ** được sinh ra khi consumer của dịch vụ AI tiêu thụ tín hiệu trên `subscription.plan.changed`. Vì vậy toàn bộ dữ liệu kiểm chứng dưới đây được tạo bằng **mã nghiệp vụ thật qua HTTP**, không `INSERT` vào bảng subscription.

Chuỗi đầy đủ, quan sát được trong `shared.outbox_records`:

```
topic                    | event_type                       | to_plan_code
subscription.plan.changed| ACTIVATED                        | premium-monthly
subscription.plan.changed| EXTENDED                         | premium-monthly
subscription.plan.changed| ACTIVATED                        | free
subscription.plan.changed| SUBSCRIPTION_DOWNGRADED_TO_FREE   | free
```

Cả bốn dòng đều `status = published`, `attempts = 0` — tức là phát ra thành công và **không** phải retry.

### Tài khoản thử nghiệm

Cả ba tài khoản đều có **tiền tố nhận diện `t180fr041`**, tạo qua `POST /api/v1/admin/users` (không tạo trực tiếp trong SQL):

| Username | Email | UUID |
|---|---|---|
| `t180fr041a` | `t180fr041a@example.test` | `e39306a4-1ade-4587-8328-e7f3b60cfe9d` |
| `t180fr041b` | `t180fr041b@example.test` | `91e627a0-5a4a-4a7a-a72f-138bcd76d876` |
| `t180fr041c` | `c5e66448-c37d-49f2-9565-9d52afa21c9b` | `c5e66448-c37d-49f2-9565-9d52afa21c9b` |

Cả ba được tạo với vai trò `user`, **không** phải admin (để không dính `target_is_admin`). Mật khẩu của tài khoản thử nghiệm do người chạy tự đặt qua `$TEST_USER_PASSWORD` — theo T189, tài liệu kiểm chứng **không** in giá trị mật khẩu nào; khoản này không cần tạo lại khi chạy lại vì tài khoản đã tồn tại.

### Lấy token quản trịviên

Tài liệu này **không** chứa giá trị mật khẩu quản trị viên (T189 — bí mật không được nhân bản sang mỗi bản sao của tài liệu kiểm chứng). Nguồn đã có:

- Tài khoản `admin` / `admin@smartwardrobe.com` được seed tại `database/init/03-seed.sql:60-91`. Lưu ý: seed chỉ chứa **bcrypt hash** (`password_hash`), không có bản rõ.
- Vì vậy lấy mật khẩu từ **biến môi trường** của máy phát triển, đặt giá trị một lần rồi dùng lại cho mọi lệnh còn lại:

```bash
export ADMIN_PASSWORD='<mật khẩu admin của môi trường phát triển>'
```

Trường trong payload đăng nhập là **`loginName`**, không phải `username` — gửi `username` sẽ trả `400 "Vui lòng nhập tên đăng nhập hoặc email"`.

```bash
# 1. Token quản trị viên
export ADMIN_TOKEN=$(curl -sS -X POST http://localhost:8101/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d "{\"loginName\":\"admin\",\"password\":\"${ADMIN_PASSWORD}\"}" \
  | jq -r .data.accessToken)

# 2. Tạo ba tài khoản thử nghiệm
export TEST_USER_PASSWORD='<mật khẩu tài khoản thử nghiệm>'
curl -sS -X POST http://localhost:8101/api/v1/admin/users \
  -H "Authorization: Bearer $ADMIN_TOKEN" -H "Content-Type: application/json" \
  -d "{\"username\":\"t180fr041a\",\"email\":\"t180fr041a@example.test\",
       \"password\":\"${TEST_USER_PASSWORD}\",\"firstName\":\"T180\",\"lastName\":\"Test\",
       \"dateOfBirth\":\"1990-01-01\",\"gender\":1}"
```

### Các lệnh đã chạy

```bash
# 3. G1 — cấp gói Premium qua ba dạng mã định danh (FR-046), một lần gọi
curl -sS -X POST http://localhost:8101/api/v1/admin/subscription-grants \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Idempotency-Key: $(uuidgen)" -H "Content-Type: application/json" \
  -d '{"items":[
        {"userIdentifier":"t180fr041a","planSlug":"premium-monthly"},
        {"userIdentifier":"t180fr041b@example.test","planSlug":"premium-monthly"},
        {"userIdentifier":"c5e66448-c37d-49f2-9565-9d52afa21c9b","planSlug":"premium-monthly"}]}'
# -> total 3, succeeded 3, failed 0

# 4. G2 — cấp lại ngay lập tức để sinh thay đổi gói EXTENDED (gói phải CỘNG DỒN, FR-052)
#    expiresAt: 2026-10-31 -> 2026-11-30, tức là cũ + 30 ngày, KHÔNG phải "bây giờ + 30 ngày"
#    G3 — hạ về gói miễn phí cho t180fr041c
#    C1 — thu hồi t180fr041b: POST /api/v1/admin/subscriptions/91e627a0-.../cancel

# 5. Chờ dịch vụ AI tiêu thụ tín hiệu rồi chụp lại trạng thái
sleep 30
docker exec -i postgres-closy-dev psql -U admin -d smart_wardrobe_db -f -  # xem mục "Bằng chứng giữa chừng"

# 6. Chạy tập kiểm chứng FR-088/FR-090
docker exec -i postgres-closy-dev psql -U admin -d smart_wardrobe_db -v ON_ERROR_STOP=1 \
  -f - < database/scripts/07-verify_policy_grants.sql

# 7. Chạy công cụ khôi phục FR-088: dryRun rồi mới thật
curl -sS -X POST http://localhost:8101/api/v1/admin/subscription-grants/restore \
  -H "Authorization: Bearer $ADMIN_TOKEN" -H "Content-Type: application/json" -d '{"dryRun":true}'
curl -sS -X POST http://localhost:8101/api/v1/admin/subscription-grants/restore \
  -H "Authorization: Bearer $ADMIN_TOKEN" -H "Content-Type: application/json" -d '{"dryRun":false}'
# -> lần hai: restored 0, skipped 4 "already_restored"  (chạy lại không có tác dụng phụ)

# 8. Chạy lại tập kiểm chứng
docker exec -i postgres-closy-dev psql -U admin -d smart_wardrobe_db -v ON_ERROR_STOP=1 \
  -f - < database/scripts/07-verify_policy_grants.sql
```

### Bằng chứng giữa chừng: FR-041 **không** đạt ngay sau khi thay đổi gói

Sau bước 4–5, tập kiểm chứng báo **1** lệch, không phải `0`:

```
=== 1. Accounts with a plan but no policy grant ===
 accounts_missing_grant
------------------------
                      1
(1 row)
```

Truy vết tới tài khoản nào:

```
  username  | current_plan_code | current_plan_kind | expires_at
------------+-------------------+-------------------+-----------
 t180fr041b | free              |                 0 |
```

Bản ghi của `t180fr041b` chỉ có một dòng, đã bị đóng:

```
    plan_code    | status |        effective_from         |        effective_to
-----------------+--------+--------------------------------+------------------------
 premium-monthly | closed | 2026-09-30 23:00:23.372578+00 | 2026-09-30 23:01:26.150539+00
```

**Nguyên nhân, đọc từ mã nguồn** — `services/ai/.../usecase/grant/grant_window.go:83`:

> ⚠️ **KHỐI NÀY ĐÃ BỊ THAY THẾ BỞI T188.** Đoạn này ghi lại trạng thái mã và số đo **trước** khi sửa, giữ nguyên có chủ ý để đối chiếu trước–sau. `CreatesNewGrant` hiện nằm ở `grant_window.go:109-114` và đã **không** còn gộp hạ cấp với bật/tắt tự gia hạn. Câu hỏi "sửa mã hay sửa FR-041" **đã** được người phát triển chốt: **sửa mã**.

```go
func CreatesNewGrant(eventType subscriptioneventtype.Type) bool {
	switch eventType {
	case subscriptioneventtype.SUBSCRIPTION_DOWNGRADED_TO_FREE, subscriptioneventtype.AUTO_RENEW_UPDATED:
		// A downgrade only revokes; it must not create a grant for the free plan,
		// because the free policy is applied by the fallback resolver instead.
		return false
```

**Sửa lại mô tả: hai loại này KHÔNG cùng một nhánh sau T188.**

- Bản lỗi ở trên **gộp** `SUBSCRIPTION_DOWNGRADED_TO_FREE` và `AUTO_RENEW_UPDATED` vào cùng một `return false`, nên câu "consumer **đã** tiêu thụ, đóng bản ghi Premium cũ — chỉ là không tạo bản ghi mới" chỉ đúng cho **hạ cấp**, và **không** đúng cho `AUTO_RENEW_UPDATED`.
- `AUTO_RENEW_UPDATED` **không đóng gì cả**: `applyRevocation` trả `nil` sớm ngay ở `grant_uc.go:393-398`, trước khi vào giao dịch và trước khi gọi `retirePreviousGrants`. Bật hay tắt tự động gia hạn không đổi quyền lợi nào, nên đóng bản ghi cũ rồi không tạo bản ghi mới chính là mất dữ liệu. Tài khoản chỉ bật/tắt được khi **đang có gói**, mà bản ghi mức quyền của nó đã được tạo từ chính lần kích hoạt cấp gói đó.
- `SUBSCRIPTION_DOWNGRADED_TO_FREE` sau T188 **đóng và tạo**: nó đi đường tạo như mọi loại khác, đóng bản ghi Premium cũ qua `ClosesPriorGrants` rồi tạo bản ghi cho gói miễn phí.

Đây là **quyết định thiết kế có chủ ý, không phải hỏng** — nhưng chỉ đúng cho `AUTO_RENEW_UPDATED` như đã nêu ở trên. Sự kiện `SUBSCRIPTION_DOWNGRADED_TO_FREE` **đã** được phát (`shared.outbox_records`, `status = published`, `attempts = 0`) và consumer **đã** tiêu thụ, đóng bản ghi Premium cũ — nhưng trước T188 nó **không** tạo bản ghi mới, vì chính sách gói miễn phí được áp bằng cơ chế fallback.

Nó **mâu thuẫn với FR-041** theo nghĩa đen: FR-041 đòi "**mọi** loại thay đổi gói đều tạo được bản ghi mức quyền tính phí đầy đủ qua đường sự kiện", và mục 1 của tập FR-088 đếm **mọi** tài khoản có bản ghi gói, **không phân biệt** gói miễn phí và trả phí. Hai tài liệu này đang nói hai điều khác nhau. **Đây là mâu thuẫn cần con người quyết định, không phải lỗi để tự ý sửa.**

**Trạng thái sau T188: câu hỏi đã có đáp án, mâu thuẫn đã gỡ.** Người phát triển chọn hướng **sửa mã** — `SUBSCRIPTION_DOWNGRADED_TO_FREE` phải tạo bản ghi mức quyền cho gói miễn phí, và **giữ nguyên** FR-041 như đang viết. Bằng chứng chạy thật ở mục 9 phía trên (`FR-041 ĐẠT`, 8/8 loại thay đổi gói) là hệ quả của quyết định đó. Bảng ở mục "Sau khi chạy công cụ khôi phục FR-088" bên dưới là số **trước** khi sửa, giữ lại làm bằng chứng.

### Sau khi chạy công cụ khôi phục FR-088

```json
{"dryRun":true,"scanned":4,"restored":4,"skipped":0,"failed":0}
{"dryRun":false,"scanned":4,"restored":3,"skipped":1,"failed":0}
```

Lần chạy thứ hai: `{"restored":0,"skipped":4,"reason":"already_restored"}` — **chạy lại không có tác dụng phụ**, đúng FR-088.

### Output thật của `07-verify_policy_grants.sql`

Dán nguyên văn, không tóm tắt:

```
=== 1. Accounts with a plan but no policy grant ===
--- Expected 0. A non-zero count means those accounts cannot use paid AI
--- at all: the read path finds no grant and degrades, and the customer
--- sees a free-model answer for something they paid for. The restore
--- tool (POST /api/v1/admin/subscription-grants/restore) is the fix. ---
 accounts_missing_grant
------------------------
                      0
(1 row)


=== 2. Accounts with more than one grant in force at once ===
--- Expected 0. Each row is an account granted twice, which means the
--- close and the create were not atomic. ---
 user_id | concurrent_grants
---------+-------------------
(0 rows)


=== 3. Grant plan_code disagreeing with the account current plan ===
--- Expected 0 rows. The grant records which plan it was written for; a
--- mismatch means an event was lost or arrived out of order. ---
 id | user_id | grant_plan | subscription_plan
----+---------+------------+-------------------
(0 rows)


=== 4. Grants with an empty or incomplete policy snapshot ===
--- Expected 0. The reader needs an `entitlements` group with all four
--- operations. Missing it means no limits at all are applied, which is
--- silent: nothing errors, the limits are simply not there. ---
 bad_snapshots
---------------
             0
(1 row)


=== 5. Active or future grants whose window has ended ===
--- Expected 0. A row claiming to be live after its paid period is over
--- shadows the account current plan with a revoked entitlement. ---
 live_but_expired
------------------
                0
(1 row)


=== 6. Gate 1 and gate 2 disagreeing, per operation ===
--- Expected 0 rows. This is the most customer-visible check in the
--- file: a customer whose profile shows more chat attempts per day than
--- the engine actually allows has no way to tell, because the engine
--- silently downgrades the model instead of returning an error.
---
--- Exactly ONE approved exception is excluded below, because it is a
--- decision rather than drift: the free plan outfit limit is deliberately
--- one attempt below what the customer sees, i.e. gate 1 = 5 and
--- gate 2 = 4 (FR-005d). Any OTHER difference is unexplained and must
--- be investigated.
---
--- premium-default is NOT a second exception. It does not drift because
--- the WHERE clause above never lets it in: a strict policy never reads
--- max_paid_attempts_per_day, so its value is NULL on purpose (FR-005e)
--- and the rows are dropped by enforcement_mode = observe_only before any
--- comparison happens. SC-011 approves a one-item exception list, and
--- this query honours exactly that list.
---
--- Section 6b below prints every row with a verdict, so the approved
--- exception stays visible instead of being silently hidden. ---
 plan_slug | policy_code | enforcement_mode | operation | gate1_customer_visible | gate2_ai_enforced | drift
-----------+-------------+------------------+-----------+------------------------+-------------------+-------
(0 rows)


--- 6b. Full gate comparison, including the approved exception ---
--- Every metered operation is listed, not just the two with a customer-
--- visible quota. Each row gets a verdict so the approved exception stays
--- visible instead of being silently hidden. ---
    plan_slug    |   policy_code   | enforcement_mode | operation | gate1_customer_visible | gate2_ai_enforced |                             verdict
-----------------+-----------------+------------------+-----------+------------------------+-------------------+------------------------------------------------------------------
 free            | free-default    | observe_only     | chat      |                      3 |                 3 | match
 free            | free-default    | observe_only     | outfit    |                      5 |                 4 | deliberate one-attempt difference (FR-005d)
 free            | free-default    | observe_only     | rewriter  |                        |                 5 | no customer-visible quota for this operation; nothing to compare
 free            | free-default    | observe_only     | summary   |                        |                 5 | no customer-visible quota for this operation; nothing to compare
 premium-monthly | premium-default | strict           | chat      |                     10 |                   | strict: column is never read (FR-005e)
 premium-monthly | premium-default | strict           | outfit    |                     30 |                   | strict: column is never read (FR-005e)
 premium-monthly | premium-default | strict           | rewriter  |                        |                   | strict: column is never read (FR-005e)
 premium-monthly | premium-default | strict           | summary   |                        |                   | strict: column is never read (FR-005e)
(8 rows)
```

**Đọc từng mục — số lệch thực tế:**

| # | Truy vấn | Số lệch | Mong đợi | Đạt |
|---|---|---|---|---|
| 1 | Tài khoản có bản ghi gói nhưng thiếu bản ghi mức quyền | **0** | 0 | ✅ |
| 2 | Tài khoản có nhiều bản ghi mức quyền cùng hiệu lực | **0 dòng** | 0 dòng | ✅ |
| 3 | `plan_code` bản ghi mức quyền lệch gói hiện tại | **0 dòng** | 0 dòng | ✅ |
| 4 | Ảnh chụp chính sách rỗng hoặc thiếu thao tác | **0** | 0 | ✅ |
| 5 | Bản ghi `active`/`future` đã hết hạn | **0** | 0 | ✅ |
| 6 | Trôi cổng 1 đối cổng 2, ngoài danh sách ngoại lệ đã duyệt | **0 dòng** | 0 dòng | ✅ |
| 6b | Bảng kiểm kê đầy đủ, kèm phán quyết từng dòng | 8 dòng, **không** có `UNEXPLAINED DRIFT` | vậy | ✅ |

Mục 6b in ra 8 dòng là **đúng**: nó là bảng kiểm kê, in mọi thao tác có tính phí kèm phán quyết. Không dòng nào mang phán quyết `UNEXPLAINED DRIFT` hay `MISCONFIGURATION`; ngoại lệ duy nhất hiện diện là `free`/`outfit` — đúng ngoại lệ một mục mà SC-011 đã duyệt.

### Phần nào **chưa** được chứng minh

Nói thẳng, để không ai đọc nhầm là xong hết:

1. ~~**Con số `0` không tự xuất hiện sau khi thay đổi gói — nó cần `POST /admin/subscription-grants/restore`.**~~ **→ ĐÃ ĐƯỢC GIẢI QUYẾT; mục này không còn là khoảng trống.** Số đo ở đây (mục 1 báo **1** sau khi hạ cấp mà không chạy công cụ khôi phục) là bằng chứng **trước** khi sửa, giữ lại để đối chiếu với kết luận ở mục 9 (`FR-041 ĐẠT`, mục 1 về **0** ngay sau hạ cấp). Nguyên nhân lúc đó là `SUBSCRIPTION_DOWNGRADED_TO_FREE` cố ý không tạo bản ghi — câu hỏi "sửa mã hay sửa FR-041" **đã** được người phát triển chọn: **sửa mã**. T188 đã sửa, `CreatesNewGrant` trả `true` cho mọi loại trừ `AUTO_RENEW_UPDATED` (`grant_window.go:109-114`); FR-041 **giữ nguyên**, không sửa. Số đo lại sau T188 nằm ở mục 9, không nằm ở đây.

2. **Ngưỡng thời gian 60 giây (SC-014) chưa đo.** Đây là yêu cầu của GD7, không phải của FR-041, nên không tính vào phạm vi T180. Ba bản ghi trong lần chạy này xuất hiện **dưới 30 giây** kể từ khi có tín hiệu, nhưng đó là quan sát lỏng, không phải phép đo.

3. **Không phủ các loại thay đổi gói chưa tạo ra trong lần chạy này**: `UPGRADED`, `OVERLAID`, `LIFETIME_FALLBACK_RESTORED`, `AUTO_RENEW_UPDATED`. Cơ chế xử lý từng loại có bài kiểm thử riêng (`grant_window_test.go:129`), nhưng T180 hỏi về **chạy thật trên môi trường phát triển**, và bốn loại đó **chưa** được chạy thật ở đây. Nói cách khác: phạm vi bằng chứng thực tế là **4/8 loại thay đổi gói** (`ACTIVATED` ×2 gói khác nhau, `EXTENDED`, `SUBSCRIPTION_DOWNGRADED_TO_FREE`), không phải 8/8.

4. **Không đo tải (SC-015).** Cùng lý do: thuộc GD7, ngoài FR-041.

5. **Không chạm `smart_wardrobe_test`**, không chạy `make test-race`, không `make hygiene-scan` — ngoài phạm vi T180.

### Phạm vi bằng chứng thực sự bao phủ

| Trục | Đã chứng minh |
|---|---|
| Đường sự kiện thật, từ HTTP → outbox → RabbitMQ → consumer AI → bản ghi | ✅ đầy đủ, quan sát được `outbox_records` (`published`, `attempts = 0`) và dòng bản ghi sinh ra |
| Tạo tài khoản qua API thật (`POST /admin/users`) | ✅ 3 tài khoản |
| Cấp gói qua `POST /admin/subscription-grants` với **ba dạng mã định danh** (tên đăng nhập, email, UUID) | ✅ `succeeded 3, failed 0` |
| Cộng dồn thời hạn (FR-052) | ✅ `2026-10-31` → `2026-11-30`, đúng `cũ + 30 ngày` |
| Bản ghi mức quyền **đầy đủ cả 4 thao tác** (mục 4 của tập) | ✅ `envelope` + `entitlements.{chat,outfit,summary,rewriter}` đều có, `bad_snapshots = 0` |
| Đóng/mở bản ghi cũ và tạo bản ghi mới **nguyên tử** (mục 2) | ✅ `concurrent_grants` không có dòng nào trong cả lần cấp, cộng dồn, hạ gói |
| Công cụ khôi phục FR-088 chạy lại không có tác dụng phụ | ✅ `restored 0, skipped 4` |

### Dữ liệu thử nghiệm để lại

| Loại | Định danh | Đã dọn |
|---|---|---|
| Tài khoản | `t180fr041a`, `t180fr041b`, `t180fr041c` | **Không** — giữ lại làm bằng chứng tái lập được |
| Bản ghi gói | 3 dòng `subscription.user_subscriptions` của ba tài khoản trên | **Không** |
| Bản ghi mức quyền | 6 dòng `aiexecutor.ai_policy_grants` của ba tài khoản trên | **Không** |
| Nhật ký kiểm toán | `shared.admin_audit_logs` với `action = 'subscription.grant'` | **Không** |
| Outbox | `subscription.plan.changed` và `notification.email` của ba tài khoản | **Không** |

Không có `DROP`/`TRUNCATE`/`DELETE` nào được chạy. Dữ liệu được giữ lại có chủ ý: xóa đi thì lần chạy sau không tái lập được con số `0`, và FR-088 quy định **không được can thiệp trực tiếp vào cơ sở dữ liệu** để sửa. Mọi thay đổi đều đi qua HTTP.

Ba tài khoản này là dữ liệu thử nghiệm trên môi trường phát triển, **không** phải dữ liệu khách hàng thật.

