# Module Boundaries

## identity

Sở hữu:

```text
users
refresh_tokens
phone/email verification state
login/register/recovery/profile identity logic
```

Không sở hữu:

```text
brands
brand_members
loyalty
wardrobe
AI quota
```

Cung cấp cho module khác:

```text
GetUserByID
FindUserByPhone
CreateBrandCreatedUnverifiedUser
MarkPhoneVerified
```

Tên contract có thể khác theo codebase hiện tại, nhưng behavior phải tương đương.

## wardrobe

Sở hữu:

```text
categories
fashion_items
wardrobe_items
outfits
outfit_items
user_style_profiles
digital_sample_responses nếu đặt cùng wardrobe do liên quan outfit/sample interaction
```

Không tự quyết định:

```text
brand membership
loyalty tier
brand feature access
brand item eligibility
AI quota
```

Wardrobe contract cần có:

```text
CreateFashionItem(input) -> FashionItemDTO
GetFashionItem(id) -> FashionItemDTO
ListUserWardrobeItemsForStyling(userID, filter) -> []WardrobeItemStylingDTO
GetUserStyleProfile(userID) -> StyleProfileDTO
SaveOutfitWithItems(input) -> OutfitDTO
```

## styling

Sở hữu:

```text
AI outfit recommendation orchestration
AI chat orchestration
prompt building
RAG/retrieval/reranking orchestration
quota reservation/finalization orchestration
```

Không được:

```text
query trực tiếp brand tables
query trực tiếp loyalty tables
query raw wardrobe DB nếu đã có wardrobe contract
create brand item eligibility rule riêng
create GenerateSampleTrialStyling usecase
```

Styling input chỉ thêm:

```text
include_brand_items boolean
```

Không thêm:

```text
required_brand_item_id
```

## brand

Sở hữu:

```text
brands
brand_members
brand_customers
loyalty_programs
loyalty_tiers
loyalty_accounts
loyalty_point_transactions
brand_benefits
benefit_redemptions
brand_conversations
brand_conversation_messages
brand_items eligibility rules
```

Brand contract cần có:

```text
CheckBrandMemberRole(userID, brandID) -> role/error
ListEligibleBrandItemsForStyling(userID, filter) -> []BrandItemStylingDTO
CheckBrandFeatureAccess(userID, brandID, featureCode) -> bool
GrantOrAdjustLoyaltyPoints(input) -> LoyaltyTransactionDTO
GetBrandCustomerLoyalty(userID, brandID) -> LoyaltyAccountDTO
```

## subscription

Sở hữu:

```text
subscription plans
user subscriptions
AI quota/cost policies
AI usage ledger
payment/provider events
```

Styling dùng subscription contract để reserve/finalize/refund quota. Brand loyalty không được dùng subscription ledger để trừ/cộng điểm loyalty.
