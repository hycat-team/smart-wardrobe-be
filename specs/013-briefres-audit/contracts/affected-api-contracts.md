# API Contracts bị ảnh hưởng — Feature 013-briefres-audit

**Feature**: 013-briefres-audit | **Ngày**: 2026-08-22

> Tổng hợp các endpoint có response thay đổi khi implement BriefRes. Đây là nguồn dữ liệu cho frontend-note.md (tạo ở CUỐI giai đoạn implement + verify).
> Policy: breaking change — bỏ field cũ, thay BriefRes.

## Brand module

| Endpoint                                             | Context      | Response hiện tại                                         | Response đề xuất                        |
| ---------------------------------------------------- | ------------ | --------------------------------------------------------- | --------------------------------------- |
| GET /api/v1/brands                                   | Public       | BrandRes (13f)                                            | BrandBriefRes                           |
| GET /api/v1/brands/:brandId                          | Public       | BrandRes (13f)                                            | BrandBriefRes                           |
| GET /api/v1/brands/:brandId/items                    | Public       | BrandItemRes + FashionItemRes (30f)                       | BrandItemBriefRes + FashionItemBriefRes |
| GET /api/v1/brands/:brandId/items/samples            | User         | BrandItemRes + FashionItemRes                             | BrandItemBriefRes + FashionItemBriefRes |
| GET /api/v1/brand-items/:itemId                      | Public       | BrandItemRes + FashionItemRes                             | BrandItemRes + FashionItemBriefRes      |
| GET /api/v1/brands/:brandId/benefits                 | Public       | BrandBenefitRes (14f)                                     | BrandBenefitBriefRes + brand            |
| GET /api/v1/brand-benefits/:benefitId                | User         | BrandBenefitRes                                           | + brand brief                           |
| POST /api/v1/brand-benefits/:benefitId/redeem        | User         | BenefitRedemptionRes (thiếu benefit)                      | + benefit \*BenefitBriefRes             |
| GET /api/v1/me/benefit-redemptions                   | User         | BenefitRedemptionRes                                      | + benefit \*BenefitBriefRes             |
| POST /api/v1/brands/:brandId/join-loyalty            | User         | BrandCustomerRes (phoneE164)                              | BrandCustomerBriefRes                   |
| POST /api/v1/brands/claim                            | User         | BrandCustomerRes                                          | BrandCustomerBriefRes                   |
| GET /api/v1/brands/:brandId/conversation             | User         | BrandConversationDetailRes                                | BrandConversationUserRes + msgs         |
| POST /api/v1/brands/:brandId/conversation/read       | User         | BrandConversationRes                                      | BrandConversationUserRes                |
| GET /api/v1/me/brand-loyalties                       | User         | BrandLoyaltyRes                                           | bỏ FK nội bộ                            |
| GET /api/v1/me/brand-loyalties/:brandId              | User         | BrandLoyaltyRes                                           | bỏ FK nội bộ                            |
| GET /api/v1/me/brand-loyalties/:brandId/transactions | User         | LoyaltyPointTransactionDetailRes (16f, rò idempotencyKey) | LoyaltyPointTransactionBriefRes         |
| GET /api/v1/me/brand-loyalties/:brandId/lots         | User         | LoyaltyPointLotRes                                        | bỏ earnTransactionId                    |
| GET /api/v1/brands/:brandId/loyalty/tiers/:tierId    | Public       | LoyaltyTierDetailRes + BrandBenefitRes                    | bỏ internal + BrandBenefitBriefRes      |
| GET /brand-portal/.../items/:itemId/feedbacks        | Portal staff | DigitalSampleResponseRes (userId)                         | + user \*UserBriefRes                   |

## Identity module

| Endpoint                                 | Context      | Response hiện tại | Response đề xuất  |
| ---------------------------------------- | ------------ | ----------------- | ----------------- |
| PUT /api/v1/me/avatar                    | User         | UserRes (full)    | UserBriefRes      |
| (contract) IUserContract display lookups | cross-module | UserRes (26 leaf) | UserBriefRes (7f) |

## Wardrobe module

| Endpoint                                        | Context | Response hiện tại                      | Response đề xuất                 |
| ----------------------------------------------- | ------- | -------------------------------------- | -------------------------------- |
| GET /api/v1/me/wardrobe-items                   | User    | WardrobeItemRes (12+16f)               | WardrobeItemBriefRes             |
| POST /api/v1/wardrobe-items/{id}/clone          | User    | []WardrobeItemRes                      | []WardrobeItemBriefRes           |
| POST /api/v1/wardrobe-items/catalog-init        | User    | []WardrobeItemRes                      | []WardrobeItemBriefRes           |
| POST /api/v1/wardrobe-items/batch-upload        | User    | []WardrobeItemRes                      | []WardrobeItemBriefRes + taskId  |
| PATCH /api/v1/wardrobe-items/{id}               | User    | WardrobeItemRes                        | bỏ userId/isDeleted              |
| POST /api/v1/wardrobe-items/{id}/retry-analysis | User    | WardrobeItemRes                        | bỏ userId/isDeleted              |
| GET /api/v1/wardrobe-items/{id}                 | User    | WardrobeItemRes                        | bỏ HSL/timestamps                |
| GET /api/v1/system-catalog/wardrobe-items       | Public  | SearchWardrobeItemRes + FashionItemRes | + FashionItemBriefRes            |
| GET /api/v1/me/outfits                          | User    | PaginationResult[OutfitRes]            | PaginationResult[OutfitBriefRes] |
| POST/PUT/GET /api/v1/outfits[/{id}]             | User    | OutfitRes                              | bỏ userId                        |

## Fashion module

| Endpoint                               | Context | Response hiện tại              | Response đề xuất                    |
| -------------------------------------- | ------- | ------------------------------ | ----------------------------------- |
| POST /api/v1/ai/outfit-recommendations | User    | RecommendedFashionItemRes (8f) | RecommendedFashionItemBriefRes (6f) |
| SSE batch upload completed             | User    | data = full WardrobeItemRes    | data = brief item                   |
| GET /api/v1/ai/chat/sessions           | User    | ChatSessionRes                 | ChatSessionBriefRes (nếu chốt)      |

## Subscription module

| Endpoint                                 | Context | Response hiện tại                 | Response đề xuất                          |
| ---------------------------------------- | ------- | --------------------------------- | ----------------------------------------- |
| GET /api/v1/subscriptions/me             | User    | UserSubscriptionOverviewDTO (14f) | 11f (bỏ fallback\*)                       |
| GET /api/v1/subscriptions/me/daily-quota | User    | UserSubscriptionDTO (17f)         | 12f (bỏ fallback\* + maxWardrobe/Outfits) |
| GET /api/v1/subscriptions/plans          | Public  | []SubscriptionPlanDTO (12f)       | []SubscriptionPlanBriefRes (10f)          |
| GET /api/v1/subscriptions/me/wallet      | User    | WalletDTO                         | optional trim userID                      |

## Dashboard module

| Endpoint                                                           | Context | Response hiện tại                     | Response đề xuất     |
| ------------------------------------------------------------------ | ------- | ------------------------------------- | -------------------- |
| GET /api/v1/brand/dashboard/operations/digital-sample-lab/{itemId} | Brand   | DigitalSampleLabAnalyticsRes (ItemID) | + item brief         |
| GET /api/v1/brand/dashboard/drilldown/sample-lab-feedbacks         | Brand   | DrillDownSampleFeedbackDTO (UserID)   | + user brief         |
| GET /api/v1/brand/dashboard/kpi-cards                              | Brand   | BrandKPICardsRes (10f)                | bỏ BrandID/UpdatedAt |

## aiexecutor module

| Endpoint                    | Context | Response hiện tại                 | Response đề xuất       |
| --------------------------- | ------- | --------------------------------- | ---------------------- |
| GET /api/v1/admin/ai/grants | Admin   | AIPolicyGrantDTO (AICostPolicyID) | + AICostPolicyBriefRes |

## KHÔNG thay đổi

- Toàn bộ admin endpoints (brand admin, identity admin, subscription admin, dashboard admin, aiexecutor admin, admin module audit-logs)
- Portal endpoints (/brand-portal/\*)
- Module notification (service-only, không có API)
- User dashboard endpoints
