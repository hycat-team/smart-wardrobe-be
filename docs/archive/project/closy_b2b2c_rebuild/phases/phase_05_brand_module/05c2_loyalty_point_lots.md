# Sub Phase 5c2 — Loyalty Point Lots & Expiry-Safe Redeem

## Mục tiêu

Bổ sung cơ chế quản lý điểm loyalty theo “point lots” để xử lý redeem và expiry chính xác, không cần replay toàn bộ ledger.

Phase 5c đã triển khai loyalty point transaction ledger. Phase 5c2 này bổ sung thêm bảng vận hành `loyalty_point_lots`.

Quyết định domain mới:

```text
loyalty_point_transactions = append-only ledger, dùng để audit/hiển thị lịch sử điểm
loyalty_point_lots = mutable operational state, dùng để biết điểm EARN nào còn bao nhiêu và hết hạn khi nào
```

Không dùng cách replay toàn bộ `loyalty_point_transactions` để tính điểm hết hạn trong MVP vì dễ sai và nặng.

## Không làm trong phase này

Không làm:

- Không dùng `remaining_points` trong `loyalty_point_transactions`.
- Không sửa transaction cũ.
- Không replay toàn bộ ledger để tính expiry.
- Không làm worker expiry đầy đủ nếu phase này chỉ tập trung vào schema + usecase.
- Không làm `brand_orders`.
- Không làm campaign.
- Không làm brand subscription.
- Không làm yearly tier reset.
- Không đổi tier khi redeem/expire điểm.

Nếu phase 5c đã lỡ tạo field `remaining_points` trong `loyalty_point_transactions`, không dùng field đó trong code mới. Có thể để tạo migration drop field.

---

## 1. Migration cần tạo

Tạo migration mới, ví dụ:

```text
20260628xxxxxx_create_loyalty_point_lots.sql
```

### Up migration

```sql
-- +goose Up

CREATE TABLE IF NOT EXISTS loyalty_point_lots (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    loyalty_account_id UUID NOT NULL REFERENCES loyalty_accounts(id) ON DELETE CASCADE,
    brand_id UUID NOT NULL REFERENCES brands(id) ON DELETE CASCADE,
    brand_customer_id UUID NOT NULL REFERENCES brand_customers(id) ON DELETE CASCADE,
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,

    earn_transaction_id UUID NOT NULL REFERENCES loyalty_point_transactions(id) ON DELETE RESTRICT,

    earned_points INT NOT NULL,
    remaining_points INT NOT NULL,

    expires_at TIMESTAMP WITH TIME ZONE,
    status VARCHAR(50) NOT NULL DEFAULT 'ACTIVE',

    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_loyalty_point_lots_earned_positive
        CHECK (earned_points > 0),

    CONSTRAINT chk_loyalty_point_lots_remaining_non_negative
        CHECK (remaining_points >= 0),

    CONSTRAINT chk_loyalty_point_lots_remaining_lte_earned
        CHECK (remaining_points <= earned_points),

    CONSTRAINT chk_loyalty_point_lots_status
        CHECK (status IN ('ACTIVE', 'CONSUMED', 'EXPIRED'))
);

CREATE UNIQUE INDEX IF NOT EXISTS ux_loyalty_point_lots_earn_transaction
ON loyalty_point_lots(earn_transaction_id);

CREATE INDEX IF NOT EXISTS idx_loyalty_point_lots_account_active_expiry
ON loyalty_point_lots(loyalty_account_id, expires_at)
WHERE status = 'ACTIVE' AND remaining_points > 0;

CREATE INDEX IF NOT EXISTS idx_loyalty_point_lots_expiry_worker
ON loyalty_point_lots(expires_at)
WHERE status = 'ACTIVE' AND remaining_points > 0 AND expires_at IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_loyalty_point_lots_brand_customer
ON loyalty_point_lots(brand_customer_id);

-- Optional dev backfill.
-- Chỉ dùng nếu phase 5c đã có EARN transaction test data nhưng chưa có lots.
-- Nếu database đang sạch thì block này không ảnh hưởng.
INSERT INTO loyalty_point_lots (
    loyalty_account_id,
    brand_id,
    brand_customer_id,
    user_id,
    earn_transaction_id,
    earned_points,
    remaining_points,
    expires_at,
    status,
    created_at,
    updated_at
)
SELECT
    t.loyalty_account_id,
    t.brand_id,
    t.brand_customer_id,
    t.user_id,
    t.id,
    t.points_delta,
    t.points_delta,
    t.expires_at,
    CASE
        WHEN t.expires_at IS NOT NULL AND t.expires_at <= NOW() THEN 'EXPIRED'
        ELSE 'ACTIVE'
    END,
    t.created_at,
    NOW()
FROM loyalty_point_transactions t
WHERE t.transaction_type = 'EARN'
  AND t.points_delta > 0
  AND NOT EXISTS (
      SELECT 1
      FROM loyalty_point_lots l
      WHERE l.earn_transaction_id = t.id
  );
```

Lưu ý backfill:

- Nếu DB dev có redeem cũ trước khi có lots, backfill này không thể tự biết redeem đã ăn vào lot nào.
- Vì đây là nhánh rebuild/dev và dữ liệu cũ đã backup, chấp nhận dùng backfill đơn giản.
- Nếu có dữ liệu test phức tạp, reset seed hoặc truncate loyalty test data trước khi chạy.

### Down migration

```sql
-- +goose Down

DROP INDEX IF EXISTS idx_loyalty_point_lots_brand_customer;
DROP INDEX IF EXISTS idx_loyalty_point_lots_expiry_worker;
DROP INDEX IF EXISTS idx_loyalty_point_lots_account_active_expiry;
DROP INDEX IF EXISTS ux_loyalty_point_lots_earn_transaction;

DROP TABLE IF EXISTS loyalty_point_lots;
```

---

## 2. Entity / Model cần thêm

Thêm entity/domain model:

```text
LoyaltyPointLot
```

Fields:

```text
ID
LoyaltyAccountID
BrandID
BrandCustomerID
UserID nullable
EarnTransactionID
EarnedPoints
RemainingPoints
ExpiresAt nullable
Status
CreatedAt
UpdatedAt
```

Status constants:

```text
ACTIVE
CONSUMED
EXPIRED
```

Rule:

- `earned_points > 0`
- `remaining_points >= 0`
- `remaining_points <= earned_points`
- `remaining_points = 0` thì status không được là ACTIVE lâu dài; phải chuyển sang `CONSUMED` hoặc `EXPIRED` tùy usecase.
- `earn_transaction_id` phải trỏ tới transaction loại `EARN`.

---

## 3. Repository cần thêm/cập nhật

Thêm repository cho lots, hoặc thêm method vào loyalty repository hiện tại.

Required methods:

```text
CreatePointLot(ctx, lot) error

ListRedeemableLotsForUpdate(ctx, loyaltyAccountID, now) ([]LoyaltyPointLot, error)

ListExpiredLotsForUpdate(ctx, loyaltyAccountID, now) ([]LoyaltyPointLot, error)

UpdateLotRemainingAndStatus(ctx, lotID, remainingPoints, status) error

ListAccountsWithExpiredLots(ctx, now, limit) ([]LoyaltyAccountID, error)
```

Important:

- Các method `List...ForUpdate` phải lock row để tránh race condition.
- Dùng `FOR UPDATE` khi lấy lots để redeem/expire.

Redeemable lots query logic:

```sql
SELECT *
FROM loyalty_point_lots
WHERE loyalty_account_id = $1
  AND status = 'ACTIVE'
  AND remaining_points > 0
  AND (expires_at IS NULL OR expires_at > $2)
ORDER BY
  expires_at ASC NULLS LAST,
  created_at ASC
FOR UPDATE;
```

Expired lots query logic:

```sql
SELECT *
FROM loyalty_point_lots
WHERE loyalty_account_id = $1
  AND status = 'ACTIVE'
  AND remaining_points > 0
  AND expires_at IS NOT NULL
  AND expires_at <= $2
ORDER BY expires_at ASC, created_at ASC
FOR UPDATE;
```

---

## 4. Update usecase EARN points

Khi cộng điểm từ purchase/manual earn:

Flow bắt buộc:

```text
Start DB transaction
Lock loyalty_account row
Calculate earned_points
Insert loyalty_point_transactions type EARN
Create loyalty_point_lots linked to EARN transaction
Update loyalty_accounts current_points/lifetime_points/total_spend/current_tier_id
Commit
```

Pseudo:

```text
earned_points = calculate from purchase_amount or input points_delta

transaction:
- transaction_type = EARN
- points_delta = earned_points
- balance_after = old_current_points + earned_points
- spend_amount = purchase_amount nullable
- expires_at = now + loyalty_program.point_expiry_days, or NULL
- idempotency_key

lot:
- earn_transaction_id = transaction.id
- earned_points = earned_points
- remaining_points = earned_points
- expires_at = transaction.expires_at
- status = ACTIVE
```

Important:

- Nếu idempotency key đã tồn tại thì không tạo transaction/lot mới.
- Nếu brand không cấu hình expiry, `expires_at = NULL`.
- `lifetime_points` chỉ tăng khi EARN.
- `total_spend` chỉ tăng khi có purchase_amount.
- Tier update dựa trên `total_spend`, không dựa trên points.

---

## 5. Update usecase REDEEM points

Trước khi redeem, phải expire due lots của account trong cùng transaction hoặc cùng service flow.

Flow bắt buộc:

```text
Start DB transaction
Lock loyalty_account row
ExpireDueLotsForAccount(accountID, now)
Reload/check current_points
List redeemable lots FOR UPDATE
Allocate redeem amount from lots by FEFO
Insert loyalty_point_transactions type REDEEM
Update loyalty_accounts.current_points
Commit
```

FEFO rule:

```text
FEFO = First Expired, First Out
Lots có expires_at gần nhất bị trừ trước
Lots expires_at NULL xếp sau lots có hạn
Nếu cùng expires_at thì created_at cũ hơn trừ trước
```

Pseudo allocation:

```text
remaining_to_redeem = required_points

for lot in redeemable_lots:
    take = min(lot.remaining_points, remaining_to_redeem)
    lot.remaining_points -= take

    if lot.remaining_points == 0:
        lot.status = CONSUMED

    update lot

    remaining_to_redeem -= take

    if remaining_to_redeem == 0:
        break

if remaining_to_redeem > 0:
    rollback, return insufficient points
```

Sau khi allocate:

```text
insert transaction REDEEM:
- points_delta = -required_points
- balance_after = old_current_points - required_points
- transaction_type = REDEEM
```

Update account:

```text
current_points -= required_points
```

Không update:

- `lifetime_points`
- `total_spend`
- `current_tier_id`

---

## 6. Expire due lots usecase

Tạo internal usecase:

```text
ExpireDueLotsForAccount(ctx, loyaltyAccountID, now) -> expiredPoints
```

Flow:

```text
Lock loyalty_account row if caller has not locked it
List expired lots FOR UPDATE
Sum remaining_points
If sum = 0 return 0
Set expired lots remaining_points = 0, status = EXPIRED
Insert loyalty_point_transactions type EXPIRE with points_delta = -sum
Update loyalty_accounts.current_points -= sum
Return sum
```

Rules:

- Có thể gom nhiều lots hết hạn thành một transaction `EXPIRE` cho mỗi account.
- Không tạo EXPIRE nếu không có điểm hết hạn.
- Không giảm `lifetime_points`.
- Không giảm `total_spend`.
- Không đổi `current_tier_id`.
- Không sửa transaction EARN cũ.
- Chỉ update lots.

Transaction EXPIRE:

```text
transaction_type = EXPIRE
points_delta = -expired_points
balance_after = old_current_points - expired_points
reason = "Expired loyalty points"
reference_type = "POINT_EXPIRY"
reference_id = nullable
```

Safety:

- `expired_points` không được lớn hơn `loyalty_accounts.current_points`.
- Nếu dữ liệu lệch, cap ở `current_points` và log warning/error.
- Tuy nhiên trong luồng chuẩn, lots và current_points phải luôn khớp.

---

## 7. Worker expiry

Nếu phase này có triển khai worker, tạo subcomponent trong brand module hoặc worker layer hiện có:

```text
LoyaltyPointExpiryWorker
```

Worker chạy định kỳ, ví dụ mỗi ngày hoặc mỗi giờ tùy config.

Flow:

```text
Find accounts with expired active lots
For each account:
    call ExpireDueLotsForAccount(accountID, now)
```

Query tìm account:

```sql
SELECT DISTINCT loyalty_account_id
FROM loyalty_point_lots
WHERE status = 'ACTIVE'
  AND remaining_points > 0
  AND expires_at IS NOT NULL
  AND expires_at <= NOW()
LIMIT $1;
```

Worker constraints:

- Idempotent.
- Có thể chạy lại nhiều lần.
- Nếu lot đã EXPIRED hoặc remaining_points = 0 thì không xử lý lại.
- Không hard delete lots.
- Không hard delete transactions.
- Không dùng cron ngoài nếu project chưa có worker scheduler; có thể expose internal worker runner hoặc register vào existing background job pattern.

Nếu project chưa có scheduler chuẩn, phase này có thể chỉ implement service `ExpireDueLotsForAccount` và để worker scheduling sang subphase 5g. Nhưng redeem phải gọi expiry on-demand trước.

---

## 8. Cập nhật benefit redemption

Usecase redeem benefit phải đổi sang dùng lots.

Flow:

```text
User redeem benefit
→ check brand_customer/user relationship
→ check benefit status
→ check required_points
→ call ExpireDueLotsForAccount
→ allocate points from active lots
→ create REDEEM transaction
→ create benefit_redemptions
→ update loyalty account
```

Nếu sau expire user không đủ điểm:

```text
Return insufficient points
Do not create benefit_redemptions
Do not create REDEEM transaction
```

---

## 9. API behavior không đổi

Không cần đổi public API nếu phase 5c đã có:

```text
POST /api/v1/brand-portal/brands/:brandId/loyalty/points
POST /api/v1/brands/:brandId/benefits/:benefitId/redeem
```

Nhưng internal implementation phải thay đổi:

- EARN tạo thêm lots.
- REDEEM trừ lots.
- EXPIRE xử lý lots.

Response loyalty account nên vẫn trả:

- current_points
- lifetime_points
- total_spend
- current_tier

Optional response có thể thêm:

```text
next_expiring_points
next_expiring_at
```

nếu dễ query từ lots, nhưng không bắt buộc trong phase này.

---

## 10. Tests bắt buộc

### EARN creates lot

Given:

- loyalty program amount_per_point = 10000
- purchase_amount = 50000

Expect:

- transaction EARN +5
- point lot earned_points = 5
- remaining_points = 5
- current_points +5
- lifetime_points +5
- total_spend +50000

### EARN with expiry

Given:

- point_expiry_days = 30

Expect:

- EARN transaction expires_at = now + 30 days
- lot expires_at same as transaction

### Redeem consumes nearest-expiring lot first

Given:

- Lot A remaining 100 expires tomorrow
- Lot B remaining 100 expires next month
- redeem 150

Expect:

- Lot A remaining 0 status CONSUMED
- Lot B remaining 50 status ACTIVE
- transaction REDEEM -150
- current_points reduced by 150

### Redeem fails if insufficient after expiry

Given:

- Lot A remaining 100 expired yesterday
- Lot B remaining 100 valid
- current_points = 200
- redeem 150

Expect:

- ExpireDueLots creates EXPIRE -100
- Lot A status EXPIRED
- current_points becomes 100
- redeem 150 fails
- no REDEEM transaction
- no benefit redemption

### Expiry worker groups expired lots

Given:

- Lot A expired remaining 30
- Lot B expired remaining 20
- same account

Expect:

- one EXPIRE transaction -50
- both lots status EXPIRED
- current_points reduced by 50

### Expiry does not affect tier

Given:

- total_spend qualifies Gold
- current_tier = Gold
- points expire

Expect:

- current_tier remains Gold
- total_spend unchanged
- lifetime_points unchanged

### Idempotency

Given:

- Same idempotency_key submitted twice for EARN

Expect:

- only one transaction
- only one lot
- account points only updated once

---

## 11. Acceptance checklist

Phase 5c2 is complete when:

- [ ] Migration creates `loyalty_point_lots`.
- [ ] EARN transaction creates corresponding point lot.
- [ ] Redeem uses lots and FEFO allocation.
- [ ] Expired lots are processed before redeem.
- [ ] EXPIRE transaction is created by expiry usecase/worker, not by staff endpoint.
- [ ] `loyalty_point_transactions` remains append-only audit ledger.
- [ ] No code uses `remaining_points` on `loyalty_point_transactions`.
- [ ] No replay-ledger algorithm is used for expiry in MVP.
- [ ] `current_points` matches sum of active lot `remaining_points`.
- [ ] Expire/redeem does not change total_spend, lifetime_points, or tier.
- [ ] All point mutation operations are atomic and row-locked.
- [ ] Tests above pass.

---

## 12. Common mistakes to avoid

Do not:

- Do not update old loyalty_point_transactions.
- Do not store remaining_points in loyalty_point_transactions.
- Do not redeem from expired lots.
- Do not rely only on scheduled worker before redeem.
- Do not allow staff endpoint to create EXPIRE manually.
- Do not reduce lifetime_points on redeem/expire.
- Do not reduce total_spend on redeem/expire.
- Do not recalculate tier from current_points.
- Do not create lots for REDEEM/EXPIRE/ADJUST transactions.
- Do not hard delete lots.
- Do not hard delete transactions.
