# Backend Technical Debt - Closy B2B2C MVP

Last updated: 2026-06-29

This document tracks accepted backend technical debt for the B2B2C MVP. P0/P1 frontend-facing APIs have been implemented; items below are deliberately deferred unless demo or production readiness requires them.

## Accepted MVP Debt

### 1. Offline Customer Claim/Link Hardening

Current state:

- Brand staff can create an offline `brand_customer` by phone/external code.
- Brand staff can create a claim token for that offline customer.
- A real user can claim/link that offline customer using the token.
- Tokens are hashed, expire, and are marked consumed.

Debt:

- No revoke claim token API.
- No list/status API for issued claim tokens.
- No one-active-token-per-customer rule.
- No rate limit dedicated to claim attempts.
- No secondary verification such as receipt code, OTP, phone, or email.
- Claim audit is minimal and does not fully model who generated/rotated/revoked a token.

Frontend expectation:

- Treat claim/link as an MVP optional flow.
- Do not present it as a strong identity verification flow.

### 2. No Phone Lookup For Real Users

Current state:

- `users` does not store phone number.
- Brand staff cannot find a real Closy user by phone.
- Phone in loyalty APIs is only for offline `brand_customers` scoped by `brand_id + phone_hash`.

Debt:

- No verified phone field on `users`.
- No user lookup by phone.
- No public customer/member QR lookup contract.

Frontend expectation:

- To add points to an existing online user, use `userId`.
- To add points by phone, treat the customer as offline until claim/link happens.

### 3. Brand Chat Realtime And Read State

Current state:

- Brand chat uses HTTP list/send APIs.
- The user and staff UI can poll messages.

Debt:

- No WebSocket/SSE realtime for brand chat.
- No unread count.
- No mark-read API.
- No close/reopen API exposed to frontend.
- Conversation reopen currently happens implicitly when a new message is sent.

Frontend expectation:

- Use polling for MVP.
- Derive basic freshness from `lastMessageAt`.

### 4. Loyalty Lots Visibility

Current state:

- `loyalty_point_lots` is the source for redeemable and expiring point balance.
- User loyalty detail exposes only one `nearestExpiringPointLot`.

Debt:

- No endpoint to list all lots.
- No staff-facing lot audit view.
- No per-lot redemption breakdown in API responses.

Frontend expectation:

- Show current point balance from loyalty account.
- Show only the nearest expiring lot if present.
- Use transaction history for ledger display.

### 5. Brand Subscription And B2B Billing

Current state:

- Brand subscription/B2B billing is outside MVP implementation.

Debt:

- No brand subscription plans.
- No brand billing lifecycle.
- No brand portal billing screens/API.

Frontend expectation:

- Do not build billing screens for MVP.

### 6. Campaign Module

Current state:

- Campaigns are out of scope.

Debt:

- No campaign schema.
- No campaign benefit targeting.
- No campaign analytics.

Frontend expectation:

- Do not build campaign screens for MVP.

## Implemented API Gap Closure

Implemented on 2026-06-29:

- Brand logo now supports `logoUrl` and `logoPublicId`.
- Brand logo upload signature.
- Brand item upload signature.
- User loyalty list/detail/transactions.
- Portal brand switcher API.
- Public brand detail API.
- Portal customer detail API.
- Portal loyalty program/tier/account transaction read APIs.
- User and staff brand item detail APIs.
- Brand item status patch API.
- User benefit detail and redemption list APIs.
