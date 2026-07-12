# Phase 05c - Unified Loyalty Points Usecase

## Mục tiêu

Tạo một API thống nhất để brand staff cộng/trừ điểm hoặc ghi nhận purchase offline cho customer bằng `user_id` hoặc số điện thoại. Nếu chỉ có số điện thoại, hệ thống tạo hoặc dùng lại `brand_customers` offline, không tạo `users`.

## Không làm trong phase này

```text
- Không tạo endpoint offline-purchases riêng.
- Không tạo brand_orders.
- Không sửa transaction cũ.
- Không dùng remaining_points.
- Không cho brand staff tạo ACTIVE user trực tiếp.
- Không tạo `users UNVERIFIED` từ phone offline.
- Không dùng SMS/phone OTP cho offline loyalty.
```

## API thống nhất

```text
POST /api/v1/brand-portal/brands/:brandId/loyalty/points
```

Request body gợi ý:

```json
{
  "user_id": "nullable UUID",
  "phone": "nullable string",
  "customer_name": "nullable string",
  "purchase_amount": 500000,
  "points_delta": null,
  "transaction_type": "EARN",
  "reason": "Offline purchase",
  "reference_type": "MANUAL_PURCHASE",
  "reference_id": "nullable UUID",
  "idempotency_key": "nullable string"
}
```

Input rules:

```text
- Phải có ít nhất một trong các thông tin định danh customer: user_id, phone, external_customer_code.
- Nếu có user_id, tìm hoặc tạo `brand_customer` linked với user đó.
- Nếu không có user_id nhưng có phone, normalize phone, tính `phone_hash`, tìm hoặc tạo `brand_customer` offline/unlinked.
- Không tạo `users` từ phone offline.
- Phải có purchase_amount hoặc points_delta.
- Nếu purchase_amount có và points_delta null: tính điểm theo loyalty_programs.amount_per_point.
- Nếu points_delta có: dùng cho manual ADJUST hoặc manual EARN/REFUND tùy transaction_type.
- purchase_amount không âm.
- points_delta không được bằng 0.
- transaction_type cho portal endpoint chỉ nên cho phép EARN, ADJUST, REFUND.
- REDEEM đi qua benefit redeem usecase.
- EXPIRE đi qua expiry job/usecase, không cho staff gọi trực tiếp qua endpoint này.
```

## Permission rules

Brand staff phải pass:

```text
brands.status = ACTIVE
brand_members.status = ACTIVE
role in OWNER, MANAGER
```

Nếu muốn SUPPORT_STAFF được cộng điểm, phải được business duyệt riêng. MVP mặc định không cho.

## Usecase flow

Tất cả phải chạy trong DB transaction.

```text
1. Check brand member permission.
2. Resolve brand customer:
   - nếu user_id: load user.
   - nếu user_id hợp lệ: find/create brand_customer với user_id.
   - nếu phone: normalize phone, hash phone, find/create brand_customer với user_id NULL.
3. Nếu có user và user status SUSPENDED/DELETED: reject.
4. Find or create brand_customer:
   - joined_source OFFLINE_PURCHASE nếu có purchase_amount.
   - joined_source SELF_JOIN nếu user tự join qua app.
   - Không dùng STAFF_CREATED.
   - status ACTIVE.
   - customer_name update nếu currently empty và input có.
5. Find or create loyalty_account.
6. Load active loyalty_program của brand.
7. Calculate points_delta nếu purchase_amount provided.
8. Determine expires_at nếu transaction type EARN and loyalty_program.point_expiry_days not null.
9. Lock loyalty_account row FOR UPDATE.
10. Check idempotency_key/reference duplicate.
11. Calculate new balance.
12. Validate current_points không âm nếu points_delta âm.
13. Update loyalty_accounts:
    - current_points += points_delta
    - lifetime_points += max(points_delta, 0) for EARN/positive ADJUST? See rule below.
    - total_spend += purchase_amount only for purchase-based EARN/REFUND rules.
    - current_tier_id recalculated by total_spend.
14. Insert loyalty_point_transactions append-only.
15. Commit.
```

`loyalty_account` phải gắn `brand_customer_id`. `user_id` trong `loyalty_account` nullable và được set khi brand customer đã linked Closy account.

## Points calculation

If:

```text
purchase_amount = 500000
amount_per_point = 10000
rounding_mode = FLOOR
```

Then:

```text
earned_points = floor(500000 / 10000) = 50
```

Rounding modes:

```text
FLOOR: floor(amount / amount_per_point)
ROUND: round(amount / amount_per_point)
CEIL: ceil(amount / amount_per_point)
```

If amount is less than amount_per_point and FLOOR, earned points can be 0. If earned points = 0 but purchase_amount > 0, still update `total_spend` and tier, but insert transaction only if there are points or if audit wants zero transaction. MVP should avoid zero-point transaction unless current ledger pattern allows.

## Lifetime points rule

Default MVP:

```text
- EARN positive: lifetime_points += points_delta
- ADJUST positive: do not increase lifetime_points unless reason explicitly says bonus points should count lifetime
- REFUND: business-specific; MVP can use points_delta positive to return points but not increase lifetime_points
- REDEEM/EXPIRE: never decrease lifetime_points
```

Keep this in domain comments/tests.

## Total spend rule

```text
- purchase_amount in EARN increases total_spend.
- Redeem points does not affect total_spend.
- Manual ADJUST points does not affect total_spend unless purchase_amount is provided.
- Refund purchase can decrease total_spend only if business requires. MVP can defer purchase refund because no brand_orders.
```

## Tier recalculation

After total_spend changes:

```sql
SELECT id
FROM loyalty_tiers
WHERE brand_id = $brandID
  AND min_total_spend <= $totalSpend
ORDER BY rank DESC
LIMIT 1;
```

If no tier, `current_tier_id = NULL`.

## Idempotency behavior

If same `idempotency_key` already exists for same brand:

```text
- Return existing transaction/result.
- Do not create another transaction.
- Do not update balance again.
```

If `reference_type + reference_id` is used as idempotency alternative, behavior tương tự.

## Response body

Return:

```json
{
  "transaction_id": "uuid",
  "brand_id": "uuid",
  "brand_customer_id": "uuid",
  "user_id": "nullable uuid",
  "customer_status": "ACTIVE",
  "points_delta": 50,
  "balance_after": 120,
  "total_spend": 1500000,
  "current_tier": {
    "id": "uuid",
    "name": "Silver"
  }
}
```

## Tests

- Staff enters phone not found -> creates offline brand_customer with `user_id NULL`, loyalty_account, EARN transaction.
- Staff enters phone already linked -> uses existing brand_customer/user.
- Staff enters user_id -> no phone required.
- Duplicate idempotency_key -> no double points.
- purchase_amount calculates points correctly for FLOOR/ROUND/CEIL.
- current_points cannot go negative.
- total_spend increases and tier updates.
- Redeem/expire not available via this endpoint.
- Non-member staff cannot grant points.
- Phone offline flow does not create `users`.

## Acceptance checklist

- [ ] One unified points API.
- [ ] Accepts user_id or phone.
- [ ] Phone offline creates/uses `brand_customers`, not `users`.
- [ ] `joined_source` uses SELF_JOIN or OFFLINE_PURCHASE, not STAFF_CREATED.
- [ ] Atomic DB transaction.
- [ ] Idempotent behavior.
- [ ] Append-only transaction insert.
- [ ] No remaining_points.
