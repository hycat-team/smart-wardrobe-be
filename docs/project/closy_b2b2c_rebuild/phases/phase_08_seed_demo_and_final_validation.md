# Phase 08 - Seed Demo and Final Validation

## Mục tiêu

Tạo dữ liệu demo B2B2C tối thiểu và chạy validation end-to-end để đảm bảo các phase trước hoạt động cùng nhau.

## Không làm trong phase này

```text
- Không tạo brand subscription/B2B billing.
- Không tạo campaign.
- Không tạo brand_orders.
- Không fake logic bằng cách bypass permission.
```

## Seed data cần có

### Users

```text
- 1 admin/demo user ACTIVE, phone verified.
- 1 brand owner user ACTIVE.
- 1 brand manager user ACTIVE.
- 1 normal B2C user ACTIVE.
- 1 offline customer user UNVERIFIED/BRAND_CREATED.
```

Không dùng số điện thoại thật của người dùng ngoài đời. Dùng số demo có pattern rõ.

### Brand

```text
- 1 brand ACTIVE.
- brand_members owner/manager.
- brand_customers for normal B2C user and offline customer.
```

### Loyalty

```text
loyalty_program:
- amount_per_point = 10000
- rounding_mode = FLOOR
- point_expiry_days = NULL hoặc 365 tùy demo

loyalty_tiers:
- Bronze min_total_spend = 0
- Silver min_total_spend = 1000000
- Gold min_total_spend = 5000000
```

### Benefits

```text
- SAMPLE_MIX_ACCESS as TIER_PRIVILEGE required Gold
- optional BRAND_ITEM_RECOMMENDATION as TIER_PRIVILEGE required Bronze or no gating depending implemented rule
- optional PRIORITY_BRAND_CHAT for Gold
```

### Fashion/wardrobe/brand items

```text
- vài wardrobe_items cho B2C user.
- vài brand_items PRODUCT.
- vài brand_items SAMPLE.
```

Brand items phải có `fashion_items` metadata đủ để AI/retrieval chạy.

### Chat

```text
- 1 conversation open giữa ACTIVE user và brand.
- vài messages demo.
```

Không tạo chat cho UNVERIFIED user.

## End-to-end validation cases

### Case 1: Offline loyalty acquisition

```text
Brand staff nhập phone + customer_name + purchase_amount
Nếu phone chưa tồn tại -> create UNVERIFIED user
create brand_customer
create loyalty_account
create EARN transaction
update total_spend/current_points/current_tier
```

Expected:

```text
- user.status = UNVERIFIED
- user.registration_source = BRAND_CREATED
- phone_verified_at = NULL
- loyalty visible in brand portal
- user chưa dùng app/chat đầy đủ
```

### Case 2: User claim offline account

```text
User verify phone bằng Redis OTP flow hiện có
```

Expected:

```text
- user.status = ACTIVE
- phone_verified_at set
- user sees loyalty points/tier in app
```

### Case 3: Wardrobe-only AI recommendation

```text
include_brand_items = false
```

Expected:

```text
- only USER_WARDROBE outfit items
- no brand contract call if observable
```

### Case 4: Brand item AI recommendation

```text
include_brand_items = true
```

Expected:

```text
- eligible PRODUCT brand items may appear
- SAMPLE only appears if SAMPLE_MIX_ACCESS rule passes
- outfit_items item_context correct
```

### Case 5: Benefit redemption

```text
User redeems point-based benefit
```

Expected:

```text
- REDEEM transaction inserted
- current_points decreases
- total_spend unchanged
- benefit_redemption created
```

### Case 6: Brand chat

```text
ACTIVE user opens chat with brand
brand staff replies
```

Expected:

```text
- conversation/message created
- last_message_at updates
- staff of other brand cannot view
```

### Case 7: Privacy

Try Brand Portal calls:

```text
- get raw wardrobe of customer
- get outfit not containing brand item
- get AI chat private data
```

Expected:

```text
- API does not exist or returns forbidden/not found
```

## Final technical validation

Run:

```bash
go test ./...
```

Run app locally/dev compose and verify:

```text
- migrations apply cleanly on empty DB
- migrations apply cleanly on DB with old wardrobe data
- app starts
- swagger generated if repo requires
- health endpoint still works
```

## Demo script output

Create or update:

```text
docs/project/closy_b2b2c_rebuild/demo_validation_script.md
```

Include:

```text
- seed command
- API call sequence
- expected responses
- screenshots needed for presentation if any
```

## Acceptance checklist

- [ ] Seed data exists and is safe.
- [ ] Offline loyalty flow works.
- [ ] Phone claim flow works with existing Redis OTP.
- [ ] AI recommendation works with/without brand items.
- [ ] Loyalty ledger is append-only.
- [ ] Benefit feature access works.
- [ ] Brand chat works for ACTIVE users.
- [ ] Privacy restrictions hold.
- [ ] Campaign and brand subscription not implemented.
