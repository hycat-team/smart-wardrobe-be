# Phase 05b - Loyalty Schema

## Mục tiêu

Tạo schema loyalty tối thiểu để brand custom rule tích điểm, custom tier theo tổng chi tiêu, lưu point balance projection, và lưu ledger bất biến.

## Không làm trong phase này

```text
- Không tạo loyalty_point_lots.
- Không tạo remaining_points.
- Không tạo brand_orders.
- Không tạo rolling yearly tier reset.
- Không tạo multiple active loyalty programs per brand.
```

## Schema target

### loyalty_programs

```text
id UUID PK
brand_id UUID FK brands(id)
name VARCHAR(255)
amount_per_point DECIMAL(12,2)
point_expiry_days INT NULL
rounding_mode VARCHAR(50)
is_active BOOLEAN
created_at
updated_at
```

Rounding mode:

```text
FLOOR
ROUND
CEIL
```

Rule:

```text
- MVP chỉ một active loyalty_program mỗi brand.
- amount_per_point > 0.
- point_expiry_days NULL nghĩa là điểm không hết hạn.
```

Khuyến nghị constraint/index:

```sql
CREATE UNIQUE INDEX ... ON loyalty_programs(brand_id)
WHERE is_active = true;
```

### loyalty_tiers

```text
id UUID PK
brand_id UUID FK brands(id)
name VARCHAR(255)
rank INT
min_total_spend DECIMAL(12,2)
description TEXT NULL
created_at
updated_at
```

Rule:

```text
- Tier dựa trên total_spend.
- Không dựa trên current_points.
- rank tăng dần theo hạng.
- min_total_spend không âm.
```

Khuyến nghị:

```text
unique(brand_id, rank)
unique(brand_id, name)
```

### loyalty_accounts

```text
id UUID PK
brand_id UUID FK brands(id)
brand_customer_id UUID FK brand_customers(id)
user_id UUID FK users(id) NULL
current_points INT
lifetime_points INT
total_spend DECIMAL(12,2)
current_tier_id UUID FK loyalty_tiers(id) NULL
created_at
updated_at
```

Meaning:

```text
brand_customer_id = identity chính của loyalty trong brand
user_id = nullable, chỉ có khi brand customer đã linked Closy account
current_points = số dư point hiện tại, projection đọc nhanh
lifetime_points = tổng point từng earn, không giảm khi redeem
total_spend = tổng chi tiêu lifetime dùng xét tier
current_tier_id = tier hiện tại theo total_spend
```

Unique/index gợi ý:

```sql
CREATE UNIQUE INDEX ... ON loyalty_accounts(brand_customer_id);

CREATE UNIQUE INDEX ... ON loyalty_accounts(brand_id, user_id)
WHERE user_id IS NOT NULL;
```

Khi brand customer được claim/link, update `loyalty_accounts.user_id = current_user.id`.

### loyalty_point_transactions

Append-only ledger:

```text
id UUID PK
loyalty_account_id UUID FK loyalty_accounts(id)
brand_id UUID FK brands(id)
brand_customer_id UUID FK brand_customers(id)
user_id UUID FK users(id) NULL
points_delta INT
balance_after INT
transaction_type VARCHAR(50)
reason VARCHAR(255) NULL
spend_amount DECIMAL(12,2) NULL
reference_type VARCHAR(100) NULL
reference_id UUID NULL
expires_at TIMESTAMP NULL
idempotency_key VARCHAR(100) NULL
created_by_user_id UUID FK users(id) NULL
created_at TIMESTAMP
```

Transaction type:

```text
EARN
REDEEM
ADJUST
EXPIRE
REFUND
```

Important:

```text
- Không có remaining_points.
- Không update transaction cũ.
- EARN là points_delta dương.
- REDEEM là points_delta âm.
- EXPIRE là points_delta âm.
- ADJUST có thể dương hoặc âm.
- REFUND tùy business, nhưng phải rõ reason/reference.
```

Index/idempotency:

```sql
CREATE UNIQUE INDEX ... ON loyalty_point_transactions(brand_id, idempotency_key)
WHERE idempotency_key IS NOT NULL;

CREATE INDEX ... ON loyalty_point_transactions(loyalty_account_id, created_at);
CREATE INDEX ... ON loyalty_point_transactions(brand_customer_id, created_at);
CREATE INDEX ... ON loyalty_point_transactions(brand_id, user_id);
CREATE INDEX ... ON loyalty_point_transactions(expires_at)
WHERE expires_at IS NOT NULL;
```

### brand_customer_claims

Claim/link account bằng token, không cần SMS OTP.

```text
id UUID PK
brand_customer_id UUID FK brand_customers(id)
claim_token_hash VARCHAR(255)
expires_at TIMESTAMP
consumed_at TIMESTAMP NULL
created_at TIMESTAMP
```

Rule:

```text
- Token/code phải lưu dạng hash, không lưu raw token.
- Token dùng một lần.
- Token có hạn dùng.
- Claim thành công thì set consumed_at.
- Chỉ cho claim nếu brand_customers.user_id IS NULL.
- Claim thành công thì set brand_customers.user_id, brand_customers.claimed_at và loyalty_accounts.user_id.
- Không cần OTP phone cho claim trong MVP.
```

## Expiry model without remaining_points

`expires_at` chỉ lưu trên transaction EARN để biết điểm earn đó có hạn hay không.

Khi muốn tính điểm hết hạn chính xác mà không mutate transaction cũ, dùng ledger projection:

```text
- Đọc toàn bộ transactions của loyalty_account theo created_at/id.
- Tạo in-memory buckets từ EARN transactions, mỗi bucket có points_delta và expires_at.
- Với các transaction âm như REDEEM/EXPIRE/ADJUST âm, allocate âm vào buckets theo expires_at gần nhất trước.
- Điểm bucket nào còn lại và expires_at <= now là điểm cần EXPIRE.
- Insert thêm EXPIRE transaction âm.
- Không update EARN transaction cũ.
```

MVP có thể chưa chạy expiry job nếu chưa cần demo, nhưng schema không được dùng `remaining_points`.

## Tests

Schema tests:

- Chỉ một active loyalty_program mỗi brand.
- Unique loyalty account theo brand_customer_id.
- Unique loyalty account theo brand/user khi user_id non-null.
- Transaction idempotency key unique theo brand khi non-null.
- Có thể tạo transaction âm/dương.
- Claim token không lưu raw token và chỉ dùng một lần.

Domain tests:

- Tier chọn theo total_spend cao nhất thỏa min_total_spend.
- Redeem điểm không làm đổi total_spend.
- current_points là projection sau transactions.

## Acceptance checklist

- [ ] Loyalty schema tạo đủ.
- [ ] `brand_customer_id` là identity chính của loyalty account.
- [ ] `user_id` nullable cho offline customer.
- [ ] Có `brand_customer_claims`.
- [ ] Không có `loyalty_point_lots`.
- [ ] Không có `remaining_points`.
- [ ] `loyalty_point_transactions` append-only.
- [ ] Tier theo total_spend.
- [ ] Idempotency key có index hoặc logic tương đương.
