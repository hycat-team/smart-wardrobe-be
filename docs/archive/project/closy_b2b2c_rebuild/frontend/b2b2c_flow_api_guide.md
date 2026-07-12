# Closy B2B2C Frontend Flow And API Guide

Last updated: 2026-06-29

Use this guide for frontend screens, route flow, and exact backend APIs to call. All authenticated requests must send cookies with `credentials: "include"` or Axios `withCredentials: true`.

## Global Rules

- Customer app uses `/api/v1/brands`, `/api/v1/me`, `/api/v1/ai`, `/api/v1/outfits`, and wardrobe APIs.
- Brand portal uses `/api/v1/brand-portal`.
- Admin uses `/api/v1/admin`.
- Brand portal permission is checked by `brand_members` for the `brandId` in the URL. There is no separate global brand role.
- `users` does not have phone. Phone-based customer flows create/use offline `brand_customers`.
- Cloudinary upload flow is always: ask backend for signature, upload directly to Cloudinary, then save `url + publicId` to backend.

## Customer Web Screens

### 1. Brand Discovery

Screen purpose:

- Show active brands.
- Allow user to open brand profile and join loyalty.

Initial APIs:

- `GET /api/v1/brands`

On brand card click:

- `GET /api/v1/brands/{brandId}`
- `GET /api/v1/brands/{brandId}/items`
- `GET /api/v1/brands/{brandId}/benefits`

Join loyalty action:

- `POST /api/v1/brands/{brandId}/join-loyalty`
- Then refresh `GET /api/v1/me/brand-loyalties/{brandId}`.

Main UI states:

- Not joined: show join button.
- Joined: show point balance, tier, nearest expiring points, benefits.
- Brand inactive/forbidden: show unavailable state.

### 2. My Brand Loyalties

Screen purpose:

- Show all brands the user has joined.
- Show current points, tier, total spend, and nearest expiring points.

Initial API:

- `GET /api/v1/me/brand-loyalties`

Card detail click:

- `GET /api/v1/me/brand-loyalties/{brandId}`
- `GET /api/v1/me/brand-loyalties/{brandId}/transactions`
- `GET /api/v1/brands/{brandId}/benefits`

Fields to show:

- `currentPoints`
- `lifetimePoints`
- `totalSpend`
- `currentTier`
- `nearestExpiringPointLot.remainingPoints`
- `nearestExpiringPointLot.expiresAt`

Important behavior:

- If `nearestExpiringPointLot` is `null`, hide expiry warning.
- Transaction history is the ledger view; do not reconstruct balance from lots.

### 3. Benefit Detail And Redemption

Screen purpose:

- Show benefit details and allow redeem where applicable.

Initial APIs:

- `GET /api/v1/brands/{brandId}/benefits/{benefitId}`
- `GET /api/v1/me/brand-loyalties/{brandId}`

Redeem action:

- `POST /api/v1/brands/{brandId}/benefits/{benefitId}/redeem`

After redeem refresh:

- `GET /api/v1/me/brand-loyalties/{brandId}`
- `GET /api/v1/me/benefit-redemptions`
- `GET /api/v1/me/brand-loyalties/{brandId}/transactions`

Important behavior:

- `POINT_REDEMPTION` consumes points.
- `TIER_PRIVILEGE` may grant access by tier without spending points.
- `SAMPLE_MIX_ACCESS` is checked by backend when AI/style flows use brand sample items.

### 4. Brand Items And Digital Sample Feedback

Screen purpose:

- Show brand products/samples to user.
- Let user view item detail and submit feedback.

List/detail APIs:

- `GET /api/v1/brands/{brandId}/items`
- `GET /api/v1/brands/{brandId}/items/{itemId}`

Feedback action:

- `POST /api/v1/brands/{brandId}/items/{itemId}/feedbacks`

Feedback body:

```json
{
  "outfitId": "uuid-or-null",
  "voteType": "UP",
  "rating": 5,
  "feedbackText": "Good fit"
}
```

Important behavior:

- Public item detail only returns active brand items.
- Feedback should include `outfitId` if the sample was evaluated after an AI outfit was saved.

### 5. AI Outfit Recommendation With Brand Items

Screen purpose:

- Let user request AI outfit recommendations using wardrobe items plus eligible brand items.

Primary API:

- `POST /api/v1/ai/outfit-recommendations`

Expected request shape:

```json
{
  "styleTarget": "casual",
  "occasion": "coffee",
  "includeBrandItems": true
}
```

After recommendation:

- If user saves outfit, call the existing outfit save API.
- If saved outfit contains a brand sample, allow feedback with `POST /api/v1/brands/{brandId}/items/{itemId}/feedbacks`.

Important behavior:

- Backend filters brand sample eligibility using `SAMPLE_MIX_ACCESS`.
- Product brand items can be used when active and eligible.

### 6. Brand Customer Chat

Screen purpose:

- Let customer chat with brand support.

Initial API:

- `GET /api/v1/brands/{brandId}/conversation`

Send message:

- `POST /api/v1/brands/{brandId}/conversation/messages`

Body:

```json
{
  "message": "I need help with sizing"
}
```

Important behavior:

- MVP uses polling, not realtime.
- Sending a message can implicitly reopen the conversation.

### 7. Offline Claim

Screen purpose:

- Optional MVP flow for a user to link an offline customer profile.

Action API:

- `POST /api/v1/brands/claim`

Body:

```json
{
  "claimToken": "token-from-brand-staff"
}
```

After success:

- Refresh `GET /api/v1/me/brand-loyalties`.

Important behavior:

- This is accepted backend technical debt and should not be presented as strong identity verification.

## Brand Portal Screens

### 1. Portal Brand Switcher

Screen purpose:

- Let brand staff select a brand they can manage.

Initial API:

- `GET /api/v1/brand-portal/me/brands`

After selecting brand:

- Store selected `brandId` in route/state.
- All portal APIs must include this `brandId`.

### 2. Brand Profile And Logo

Screen purpose:

- View brand profile.
- Upload/update brand logo.

Initial API:

- `GET /api/v1/brand-portal/brands/{brandId}`

Logo upload flow:

1. `GET /api/v1/brand-portal/brands/logo-upload-signature`
2. Upload file directly to Cloudinary using returned `signature`, `timestamp`, `apiKey`, `folder`.
3. Save Cloudinary result with `PATCH /api/v1/brand-portal/brands/{brandId}/logo`.

Patch body:

```json
{
  "logoUrl": "https://res.cloudinary.com/.../logo.png",
  "logoPublicId": "brands/logos/.../logo"
}
```

Important behavior:

- Create brand request can also send `logoUrl` and `logoPublicId`.
- Always store both values after upload.

### 3. Brand Members

Screen purpose:

- Manage staff.

Initial API:

- `GET /api/v1/brand-portal/brands/{brandId}/members`

Add/update member:

- `POST /api/v1/brand-portal/brands/{brandId}/members`

Body:

```json
{
  "userId": "uuid",
  "role": "MANAGER"
}
```

Important behavior:

- Only owner/manager should see member management actions.

### 4. Customers And Offline Purchase

Screen purpose:

- CRM list/detail.
- Create offline customer.
- Add loyalty points.

Initial APIs:

- `GET /api/v1/brand-portal/brands/{brandId}/customers`
- `GET /api/v1/brand-portal/brands/{brandId}/customers/{customerId}`

Create offline customer:

- `POST /api/v1/brand-portal/brands/{brandId}/customers/offline-purchase`

Body:

```json
{
  "customerName": "Nguyen Van A",
  "phoneE164": "+84901234567",
  "externalCustomerCode": "ERP-12345"
}
```

Add points for real user:

- `POST /api/v1/brand-portal/brands/{brandId}/loyalty/points`

```json
{
  "userId": "uuid",
  "purchaseAmount": 500000,
  "transactionType": "EARN",
  "reason": "In-store purchase",
  "idempotencyKey": "uuid"
}
```

Add points for offline phone customer:

- `POST /api/v1/brand-portal/brands/{brandId}/loyalty/points`

```json
{
  "phone": "+84901234567",
  "customerName": "Nguyen Van A",
  "purchaseAmount": 500000,
  "transactionType": "EARN",
  "reason": "In-store purchase",
  "idempotencyKey": "uuid"
}
```

After add points:

- Refresh customer list/detail.
- If the loyalty account id is known, call `GET /api/v1/brand-portal/brands/{brandId}/loyalty/accounts/{accountId}/transactions`.

Important behavior:

- Always generate a new UUID for `idempotencyKey` per purchase event.
- Do not use phone to search real `users`; backend cannot do that in MVP.

### 5. Loyalty Program And Tiers

Screen purpose:

- Show loyalty rules and tiers to staff.

Initial APIs:

- `GET /api/v1/brand-portal/brands/{brandId}/loyalty/program`
- `GET /api/v1/brand-portal/brands/{brandId}/loyalty/tiers`

Important behavior:

- MVP exposes read APIs.
- Program/tier management screens are not required unless later added.

### 6. Benefits Management

Screen purpose:

- Create/list/activate/archive benefits.

Initial API:

- `GET /api/v1/brand-portal/brands/{brandId}/benefits`

Create:

- `POST /api/v1/brand-portal/brands/{brandId}/benefits`

Status update:

- `PATCH /api/v1/brand-portal/brands/{brandId}/benefits/{benefitId}/status`

Important behavior:

- `FEATURE_ACCESS` + `TIER_PRIVILEGE` can unlock features such as `SAMPLE_MIX_ACCESS`.
- For `POINT_REDEMPTION`, set `requiredPoints`.
- For `TIER_PRIVILEGE`, set `requiredTierId`.

### 7. Brand Items Management

Screen purpose:

- Upload/create/edit/publish/archive brand products and digital samples.

Initial APIs:

- `GET /api/v1/brand-portal/brands/{brandId}/items`
- `GET /api/v1/brand-portal/brands/{brandId}/items/{itemId}`

Image upload flow:

1. `GET /api/v1/brand-portal/brands/{brandId}/items/upload-signature`
2. Upload file directly to Cloudinary.
3. Create item with returned `imageUrl` and `imagePublicId`.

Create:

- `POST /api/v1/brand-portal/brands/{brandId}/items`

Edit:

- `PUT /api/v1/brand-portal/brands/{brandId}/items/{itemId}`

Publish/archive:

- `PATCH /api/v1/brand-portal/brands/{brandId}/items/{itemId}/status`

Status body:

```json
{
  "status": "ACTIVE"
}
```

Feedback analytics:

- `GET /api/v1/brand-portal/brands/{brandId}/items/{itemId}/feedbacks`

Important behavior:

- Staff detail returns items in any status if the staff has brand access.
- Public user item APIs only expose active items.

### 8. Brand Chat Inbox

Screen purpose:

- Staff inbox and conversation messages.

Initial APIs:

- `GET /api/v1/brand-portal/brands/{brandId}/conversations`
- `GET /api/v1/brand-portal/brands/{brandId}/conversations/{conversationId}/messages`

Send staff reply:

- `POST /api/v1/brand-portal/brands/{brandId}/conversations/{conversationId}/messages`

Body:

```json
{
  "message": "Thanks, we will check this for you."
}
```

Important behavior:

- MVP uses polling.
- No unread/mark-read backend API yet.

## Admin Screens

### Brand Approval

Screen purpose:

- Admin creates/approves/suspends brands.

APIs:

- `POST /api/v1/admin/brands`
- `PATCH /api/v1/admin/brands/{brandId}/status`

Status body:

```json
{
  "status": "ACTIVE"
}
```

## Recommended Frontend Route Map

- `/brands`: brand discovery.
- `/brands/:brandId`: public brand profile.
- `/brands/:brandId/items/:itemId`: public item detail.
- `/brands/:brandId/benefits/:benefitId`: benefit detail.
- `/me/loyalty`: all joined brand loyalties.
- `/me/loyalty/:brandId`: loyalty detail and transaction history.
- `/me/benefits`: redeemed benefits.
- `/portal/brands`: brand switcher.
- `/portal/brands/:brandId`: brand overview.
- `/portal/brands/:brandId/profile`: profile and logo.
- `/portal/brands/:brandId/members`: member management.
- `/portal/brands/:brandId/customers`: customer CRM.
- `/portal/brands/:brandId/customers/:customerId`: customer detail and point history.
- `/portal/brands/:brandId/loyalty`: program and tiers.
- `/portal/brands/:brandId/benefits`: benefit management.
- `/portal/brands/:brandId/items`: item management.
- `/portal/brands/:brandId/items/:itemId`: item detail/edit/feedback.
- `/portal/brands/:brandId/chat`: support inbox.
