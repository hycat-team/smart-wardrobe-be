# Phase 05a - Brand Core: brands, brand_members, brand_customers

## Mục tiêu

Tạo nền tảng Brand Portal: brand là tổ chức, staff là users được phân quyền qua `brand_members`, customer/member của brand nằm trong `brand_customers`.

## Không làm trong phase này

```text
- Không làm campaign.
- Không làm brand subscription/B2B billing.
- Không làm brand_orders.
- Không làm loyalty points usecase ở file này.
- Không cho brand xem wardrobe user.
```

## Schema target

### brands

```text
id UUID PK
slug VARCHAR(100) UNIQUE
name VARCHAR(255)
description TEXT NULL
logo_url VARCHAR(500) NULL
status VARCHAR(50)
created_at
updated_at
```

Status:

```text
ACTIVE
SUSPENDED
ARCHIVED
```

### brand_members

```text
id UUID PK
brand_id UUID FK brands(id)
user_id UUID FK users(id)
role VARCHAR(50)
status VARCHAR(50)
created_at
updated_at
unique(brand_id, user_id)
```

Role:

```text
OWNER
MANAGER
SUPPORT_STAFF
MARKETER
```

Status:

```text
ACTIVE
INVITED
DISABLED
```

### brand_customers

```text
id UUID PK
brand_id UUID FK brands(id)
user_id UUID FK users(id)
customer_name VARCHAR(255) NULL
external_customer_code VARCHAR(100) NULL
joined_source VARCHAR(50)
status VARCHAR(50)
joined_at TIMESTAMP
created_by_member_id UUID NULL
created_at
updated_at
unique(brand_id, user_id)
```

Joined source:

```text
SELF_JOIN
OFFLINE_PURCHASE
STAFF_CREATED
```

Status:

```text
ACTIVE
BLOCKED
LEFT
```

`created_by_member_id` nên trỏ `brand_members.id` nếu tiện. Nếu codebase dễ hơn thì trỏ `users.id`, nhưng phải đặt tên rõ như `created_by_user_id`.

## Permission rules

Brand Portal request phải:

```text
- auth user exists
- brand exists and status ACTIVE
- brand_members row exists for auth user + brand
- brand_members.status = ACTIVE
- role đủ quyền cho action
```

MVP role permissions:

```text
OWNER: all actions
MANAGER: manage customers, loyalty, items, benefits, chat
SUPPORT_STAFF: view customers, chat, maybe add points if allowed by business
MARKETER: manage brand_items/benefits read-only customers; no manual point adjustment unless allowed
```

Nếu chưa chắc permission granular, implement helper central:

```text
RequireBrandRole(userID, brandID, allowedRoles...)
```

Không scatter role checks trong từng handler.

## APIs tối thiểu

```text
POST /api/v1/brand-portal/brands
GET /api/v1/brand-portal/brands/:brandId
POST /api/v1/brand-portal/brands/:brandId/members
GET /api/v1/brand-portal/brands/:brandId/members
GET /api/v1/brand-portal/brands/:brandId/customers
GET /api/v1/brands
POST /api/v1/brands/:brandId/join-loyalty
```

`POST /brand-portal/brands`:

```text
- tạo brand
- tạo brand_members OWNER cho auth user
```

`POST /brands/:brandId/join-loyalty`:

```text
- chỉ user ACTIVE
- tạo brand_customers nếu chưa có
- joined_source SELF_JOIN
- tạo loyalty_account nếu phase 05b đã có
```

Nếu Phase 05b chưa merge, endpoint join loyalty có thể chỉ tạo brand_customer và TODO create loyalty_account sau, nhưng final phase 05 phải hoàn chỉnh.

## Domain/service structure

Khuyến nghị trong module `brand`:

```text
brand/domain/entities/brand.go
brand/domain/entities/brand_member.go
brand/domain/entities/brand_customer.go
brand/domain/repositories/...
brand/usecase/brand_core_usecase.go
brand/delivery/http/...
brand/contract/...
```

Fit vào structure thật của repo, không tạo song song nếu repo có convention khác.

## Tests

- Create brand tạo owner member.
- User không phải member không vào được brand portal.
- Disabled member không vào được.
- Suspended brand không cho portal actions.
- User ACTIVE join loyalty tạo brand_customer.
- Duplicate brand_customer không tạo duplicate.

## Acceptance checklist

- [ ] Có brands/brand_members/brand_customers tables.
- [ ] Brand staff login bằng users, không có brand account riêng.
- [ ] Brand portal access check tập trung.
- [ ] Customer membership unique theo brand/user.
- [ ] Campaign và brand subscription chưa được tạo.
