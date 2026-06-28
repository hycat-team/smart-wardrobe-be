# Phase 05d - Benefits and Feature Access

## Mục tiêu

Tạo hệ thống quyền lợi của brand. Benefit không chỉ là voucher/discount mà còn có thể mở khóa feature trong app như sample mix access, brand item recommendation, priority chat.

## Không làm trong phase này

```text
- Không tạo complex rule engine.
- Không tạo campaign rewards.
- Không tạo brand subscription.
- Không để styling tự check loyalty tables.
```

## Schema target

### brand_benefits

```text
id UUID PK
brand_id UUID FK brands(id)
name VARCHAR(255)
description TEXT NULL
benefit_type VARCHAR(50)
unlock_type VARCHAR(50)
required_points INT NULL
required_tier_id UUID FK loyalty_tiers(id) NULL
feature_code VARCHAR(100) NULL
feature_config JSONB NULL
status VARCHAR(50)
created_at
updated_at
```

Benefit type:

```text
VOUCHER
DISCOUNT
GIFT
FREE_SHIPPING
EARLY_ACCESS
FEATURE_ACCESS
```

Unlock type:

```text
TIER_PRIVILEGE
POINT_REDEMPTION
MANUAL_GRANT
```

Feature code MVP:

```text
SAMPLE_MIX_ACCESS
BRAND_ITEM_RECOMMENDATION
PRIORITY_BRAND_CHAT
```

Status:

```text
ACTIVE
INACTIVE
ARCHIVED
```

### benefit_redemptions

```text
id UUID PK
benefit_id UUID FK brand_benefits(id)
brand_id UUID FK brands(id)
user_id UUID FK users(id)
points_spent INT
status VARCHAR(50)
redeemed_at TIMESTAMP
used_at TIMESTAMP NULL
expires_at TIMESTAMP NULL
created_at
updated_at
```

Status:

```text
PENDING
REDEEMED
USED
CANCELLED
EXPIRED
```

## Meaning of feature access

Feature access là rule để brand module trả lời câu hỏi:

```text
User này có quyền dùng feature X với brand Y không?
```

Ví dụ:

```text
SAMPLE_MIX_ACCESS = user được dùng SAMPLE brand_items trong AI outfit recommendation.
BRAND_ITEM_RECOMMENDATION = user được nhận PRODUCT recommendations từ brand.
PRIORITY_BRAND_CHAT = user được ưu tiên chat hoặc tag priority.
```

## Contract bắt buộc

Trong `brand/contract`:

```text
CheckBrandFeatureAccess(userID, brandID, featureCode) -> bool
ListEligibleBrandItemsForStyling(userID, filter) -> []BrandItemStylingDTO
```

`styling` chỉ gọi contract này. Styling không query:

```text
brand_benefits
benefit_redemptions
loyalty_accounts
loyalty_tiers
brand_customers
```

## MVP rules for CheckBrandFeatureAccess

Input:

```text
userID
brandID
featureCode
```

Base checks:

```text
- user status ACTIVE
- brand status ACTIVE
- brand_customer exists and status ACTIVE
```

Then check active benefits:

### TIER_PRIVILEGE

A benefit grants feature if:

```text
brand_benefits.status = ACTIVE
benefit_type = FEATURE_ACCESS
unlock_type = TIER_PRIVILEGE
feature_code = requested feature
user current tier rank >= required tier rank
```

Rank comparison must join `loyalty_tiers`.

### POINT_REDEMPTION

A benefit grants feature if:

```text
brand_benefits.status = ACTIVE
benefit_type = FEATURE_ACCESS
unlock_type = POINT_REDEMPTION
feature_code = requested feature
benefit_redemptions exists for user with status REDEEMED or PENDING if business accepts PENDING
redemption expires_at is null or in future
```

MVP should treat `REDEEMED` as active. Use `PENDING` only if current redemption flow uses pending before fulfillment.

### MANUAL_GRANT

MVP can skip manual grant unless needed. If implemented, use `benefit_redemptions` with points_spent = 0 and status REDEEMED.

## Brand item recommendation rule

MVP simple rule:

```text
PRODUCT brand item:
  user must be ACTIVE brand_customer and include_brand_items = true.
  BRAND_ITEM_RECOMMENDATION benefit can be used later if brand wants gating.

SAMPLE brand item:
  user must have SAMPLE_MIX_ACCESS OR brand item config/status indicates sample is public.
```

If no public-sample config exists yet, default SAMPLE requires `SAMPLE_MIX_ACCESS`.

## Benefit redeem API

```text
POST /api/v1/brands/:brandId/benefits/:benefitId/redeem
```

Flow:

```text
1. User must be ACTIVE.
2. User must be brand_customer ACTIVE.
3. Benefit must belong to brand and status ACTIVE.
4. If unlock_type POINT_REDEMPTION:
   - required_points must be non-null and > 0.
   - lock loyalty_account FOR UPDATE.
   - current_points >= required_points.
   - insert loyalty_point_transactions REDEEM negative.
   - update loyalty_accounts.current_points.
   - create benefit_redemptions.
5. If unlock_type TIER_PRIVILEGE:
   - no points deduction.
   - if user qualifies, either return access directly or create redemption/claim record if user explicitly clicked claim.
```

## Tests

- Gold tier benefit grants feature to Gold user.
- Silver user does not get Gold feature.
- Redeemed feature benefit grants access until expires_at.
- Expired redemption no longer grants access.
- Redeem with insufficient points fails and does not create transaction.
- Redeem is atomic.
- Styling can only access feature through brand contract.

## Acceptance checklist

- [ ] brand_benefits and benefit_redemptions created.
- [ ] Feature codes supported.
- [ ] CheckBrandFeatureAccess implemented in brand module.
- [ ] POINT_REDEMPTION creates REDEEM transaction append-only.
- [ ] No complex rule engine.
