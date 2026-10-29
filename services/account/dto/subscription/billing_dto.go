package dto

import (
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/constants/depositstatus"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/constants/plankind"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/constants/walletstatementtype"
	shared_dto "smart-wardrobe-be/shared/application/dto"
	"smart-wardrobe-be/shared/domain/constants/shared/currency"
	"time"

	"github.com/google/uuid"
)

type GetWalletStatementsQueryReq struct {
	shared_dto.PaginationQuery
}

// UserSubscriptionOverviewRes is the HTTP API response for GET /api/v1/subscriptions/me.
// Field set mirrors UserSubscriptionContractDTO (the inter-module payload) but the two
// types evolve independently per the DTO Convention (design-pattern.md).
type UserSubscriptionOverviewRes struct {
	PlanID             uuid.UUID  `json:"planId"`
	PlanName           string     `json:"planName"`
	PlanSlug           string     `json:"planSlug"`
	ExpiresAt          *time.Time `json:"expiresAt"`
	IsAutoRenewEnabled bool       `json:"isAutoRenewEnabled"`
	MaxWardrobeItems   int        `json:"maxWardrobeItems"`
	MaxOutfits         int        `json:"maxOutfits"`
	AiOutfitDailyQuota int        `json:"aiOutfitDailyQuota"`
	AiChatDailyQuota   int        `json:"aiChatDailyQuota"`
}

type WalletDTO struct {
	UserID    uuid.UUID         `json:"userID"`
	Balance   float64           `json:"balance"`
	Currency  currency.Currency `json:"currency"`
	UpdatedAt time.Time         `json:"updatedAt"`
}

type WalletStatementDTO struct {
	ID              uuid.UUID                               `json:"id"`
	UserID          uuid.UUID                               `json:"userID"`
	Amount          float64                                 `json:"amount"`
	TransactionType walletstatementtype.WalletStatementType `json:"transactionType"`
	PreviousBalance float64                                 `json:"previousBalance"`
	NewBalance      float64                                 `json:"newBalance"`
	Description     string                                  `json:"description"`
	CreatedAt       time.Time                               `json:"createdAt"`
}

type WalletTopUpReq struct {
	Amount    float64 `json:"amount" binding:"required,gt=0" label:"số tiền nạp"`
	ReturnUrl string  `json:"returnUrl"`
	CancelUrl string  `json:"cancelUrl"`
}

type DirectPurchaseReq struct {
	PlanSlug  string `json:"planSlug" binding:"required" label:"gói hội viên"`
	ReturnUrl string `json:"returnUrl"`
	CancelUrl string `json:"cancelUrl"`
}

type PaymentLinkDTO struct {
	PaymentUrl           string                      `json:"paymentUrl"`
	OrderCode            int64                       `json:"orderCode"`
	PaymentStatus        depositstatus.DepositStatus `json:"paymentStatus"`
	ExpiresAt            *time.Time                  `json:"expiresAt,omitempty"`
	NextReconciliationAt *time.Time                  `json:"nextReconciliationAt,omitempty"`
}

type PayOSWebhookData struct {
	OrderCode           int64  `json:"orderCode"`
	Amount              int    `json:"amount"`
	Description         string `json:"description"`
	AccountNumber       string `json:"accountNumber"`
	Reference           string `json:"reference"`
	TransactionDateTime string `json:"transactionDateTime"`
	PaymentLinkId       string `json:"paymentLinkId"`
	Code                string `json:"code"`
	Desc                string `json:"desc"`
}

type PayOSWebhookReq struct {
	Code      string           `json:"code"`
	Desc      string           `json:"desc"`
	Data      PayOSWebhookData `json:"data"`
	Signature string           `json:"signature"`
}

type SubscriptionPlanDTO struct {
	ID                 uuid.UUID         `json:"id"`
	Slug               string            `json:"slug"`
	Name               string            `json:"name"`
	Price              float64           `json:"price"`
	MaxWardrobeItems   int               `json:"maxWardrobeItems"`
	MaxOutfits         int               `json:"maxOutfits"`
	AiOutfitDailyQuota int               `json:"aiOutfitDailyQuota"`
	AiChatDailyQuota   int               `json:"aiChatDailyQuota"`
	DurationDays       *int              `json:"durationDays,omitempty"`
	PlanKind           plankind.PlanKind `json:"planKind"`
	TierRank           int               `json:"tierRank"`
	IsActive           bool              `json:"isActive"`
}

type SubscriptionPlanBriefRes struct {
	ID                 uuid.UUID         `json:"id"`
	Slug               string            `json:"slug"`
	Name               string            `json:"name"`
	Price              float64           `json:"price"`
	MaxWardrobeItems   int               `json:"maxWardrobeItems"`
	MaxOutfits         int               `json:"maxOutfits"`
	AiOutfitDailyQuota int               `json:"aiOutfitDailyQuota"`
	AiChatDailyQuota   int               `json:"aiChatDailyQuota"`
	DurationDays       *int              `json:"durationDays,omitempty"`
	PlanKind           plankind.PlanKind `json:"planKind"`
	TierRank           int               `json:"tierRank"`
}

type CreateSubscriptionPlanReq struct {
	Slug               string            `json:"slug" binding:"required" label:"mã định danh gói"`
	Name               string            `json:"name" binding:"required" label:"tên gói"`
	Price              float64           `json:"price" binding:"gte=0" label:"giá gói"`
	MaxWardrobeItems   int               `json:"maxWardrobeItems" binding:"gte=0" label:"số đồ tối đa"`
	MaxOutfits         int               `json:"maxOutfits" binding:"gte=0" label:"số trang phục tối đa"`
	AiOutfitDailyQuota int               `json:"aiOutfitDailyQuota" binding:"gte=0" label:"hạn mức AI outfit mỗi ngày"`
	AiChatDailyQuota   int               `json:"aiChatDailyQuota" binding:"gte=0" label:"hạn mức AI chat mỗi ngày"`
	DurationDays       *int              `json:"durationDays" label:"thời hạn gói (ngày)"`
	PlanKind           plankind.PlanKind `json:"planKind" label:"loại gói"`
	TierRank           int               `json:"tierRank" label:"cấp bậc gói"`
	IsActive           *bool             `json:"isActive" label:"trạng thái hoạt động"`
}

type UpdateSubscriptionPlanReq struct {
	Name               *string  `json:"name" label:"tên gói"`
	Price              *float64 `json:"price" label:"giá gói"`
	MaxWardrobeItems   *int     `json:"maxWardrobeItems" label:"số đồ tối đa"`
	MaxOutfits         *int     `json:"maxOutfits" label:"số trang phục tối đa"`
	AiOutfitDailyQuota *int     `json:"aiOutfitDailyQuota" label:"hạn mức AI outfit mỗi ngày"`
	AiChatDailyQuota   *int     `json:"aiChatDailyQuota" label:"hạn mức AI chat mỗi ngày"`
	DurationDays       *int     `json:"durationDays" label:"thời hạn gói (ngày)"`
	TierRank           *int     `json:"tierRank" label:"cấp bậc gói"`
	IsActive           *bool    `json:"isActive" label:"trạng thái hoạt động"`
}

type SetPlanActiveReq struct {
	IsActive bool `json:"isActive" label:"trạng thái hoạt động"`
}

type AdminDepositTransactionListRes = shared_dto.PaginationResult[*DepositTransactionRes]

// TransactionUserBriefRes is the display-only projection of the account behind a
// deposit transaction.
//
// WHY A LOCAL TYPE AND NOT `identity_dto.UserBriefRes`. The subscription module
// must not consume another module's presentation shape: the identity contract
// hands over `UserBriefContractDTO`, which carries facts this screen has no
// business publishing (`roleSlug`, `isDeleted`, `createdByAdmin`, `gender`,
// `status`, `createdAt`). Declaring the six fields that the reconciliation screen
// actually renders keeps the wire format decided HERE, by the module that owns
// the response, and it is the same choice `CampaignClaimRes.Username` makes: the
// consumer names the field it needs instead of importing a mapper chain.
type TransactionUserBriefRes struct {
	ID        uuid.UUID `json:"id"`
	Username  string    `json:"username"`
	Email     *string   `json:"email"`
	FirstName *string   `json:"firstName"`
	LastName  *string   `json:"lastName"`
	AvatarUrl *string   `json:"avatarUrl"`
}

type DepositTransactionRes struct {
	ID        uuid.UUID `json:"id"`
	UserID    uuid.UUID `json:"userId"`
	OrderCode int64     `json:"orderCode"`
	Amount    float64   `json:"amount"`
	Currency  string    `json:"currency"`
	// Status is the admin API's string form (11 values, FR-008, data-model §2):
	// `pending`, `success`, `failed_legacy`, ... A value outside that set is
	// reported as `unknown` (see mapper.DepositStatusToString).
	//
	// It is a string ONLY here. `PaymentLinkDTO.PaymentStatus` — the user-facing
	// DTO — deliberately keeps the numeric type (R4), which is why the type
	// itself carries no MarshalJSON.
	Status               string     `json:"status"`
	TransactionType      string     `json:"transactionType"`
	TargetPlanID         *uuid.UUID `json:"targetPlanId,omitempty"`
	PaymentUrl           *string    `json:"paymentUrl,omitempty"`
	Reference            *string    `json:"reference,omitempty"`
	PaymentLinkId        *string    `json:"paymentLinkId,omitempty"`
	FailureReason        *string    `json:"failureReason,omitempty"`
	CancelledAt          *time.Time `json:"cancelledAt,omitempty"`
	NextReconciliationAt *time.Time `json:"nextReconciliationAt,omitempty"`
	CreatedAt            time.Time  `json:"createdAt"`
	UpdatedAt            time.Time  `json:"updatedAt"`
	// User and TargetPlan are POPULATED BY THE DETAIL ENDPOINT ONLY; the list
	// answers them as null (029 FR-007 and the list's "no cross-module call"
	// budget). They deliberately carry NO `omitempty`: the two endpoints must
	// expose the SAME key set, and a null key that disappears would make the
	// detail a different shape from a list item — which is exactly what
	// `TestAdminBillingDetailReturnsTheSameCamelCaseShapeAsTheList` pins.
	User       *TransactionUserBriefRes  `json:"user"`
	TargetPlan *SubscriptionPlanBriefRes `json:"targetPlan"`
}

// PurchaseWithWalletReq dùng cho endpoint mua gói bằng ví nội bộ
type PurchaseWithWalletReq struct {
	PlanSlug string `json:"planSlug" binding:"required" label:"gói hội viên"`
}

// SetAutoRenewReq represents the request to update auto renew status
type SetAutoRenewReq struct {
	Enabled *bool `json:"enabled" binding:"required" label:"trạng thái tự động gia hạn"`
}

// ---------------------------------------------------------------------------
// Administrator plan grant (024 FR-060..FR-065, contracts/admin-grant-api.md)
//
// ONE ENDPOINT ONLY. Granting to a single account is a one-item list, using the
// same request, the same response and the same rules. Two endpoints would be two
// implementations of one set of rules, and they would drift apart.
// ---------------------------------------------------------------------------

// AdminGrantBatchItem is one requested grant.
//
// DELIBERATELY NO `binding` TAGS ON THESE FIELDS (FR-064). Gin's `binding`
// validator runs across the WHOLE request, so a single empty item would reject
// the entire list. FR-064 requires a bad item to be reported on its own without
// affecting the other 499, which means per-item validation has to live in the
// application layer - the only layer where a per-item result exists. Only the
// collection itself carries a `binding` constraint, because a missing or
// oversized collection cannot be processed item by item.
type AdminGrantBatchItem struct {
	UserIdentifier string `json:"userIdentifier" label:"định danh tài khoản"`
	PlanSlug       string `json:"planSlug" label:"mã gói"`
}

// AdminGrantBatchReq is the request body.
//
// 024 T167: `Items` carries NO `binding` constraint, deliberately. It used to carry
// `required,min=1,max=500`, which did two things wrong:
//
//   - It ran before the handler, so the handler's own checks for an empty and an
//     oversized list were unreachable dead code, and the administrator got the
//     generic validator text instead of the two messages the API contract specifies.
//   - `max` on a slice falls through to the message for a STRING, so 501 items came
//     back as "danh sách tài khoản không được vượt quá 500 **ký tự**". That is the
//     wrong unit outright: the limit is on the number of items, not on text length,
//     and an administrator reading it would look for a string-length problem.
//
// The limits themselves are unchanged and still enforced — by the handler, which
// rejects the WHOLE request (FR-062/FR-063), and again by the use case, so they hold
// even if the handler is bypassed. Do not re-add a tag here without checking both.
type AdminGrantBatchReq struct {
	Items []AdminGrantBatchItem `json:"items" label:"danh sách tài khoản"`
}

// Per-item status. Each item carries exactly one of these three values.

const (
	AdminGrantStatusSuccess = "success"
	AdminGrantStatusSkipped = "skipped"
	AdminGrantStatusError   = "error"
)

// Reason codes. These are part of the API contract: the web console groups and
// filters on them, so they are stable strings and must not be reworded.
const (
	AdminGrantReasonIdentifierBlank     = "identifier_blank"
	AdminGrantReasonUserNotFound        = "user_not_found"
	AdminGrantReasonIdentifierAmbiguous = "identifier_ambiguous"
	// AdminGrantReasonIdentifierTooLong is reported when the identifier exceeds
	// 255 characters (FR-062). It is deliberately NOT "user_not_found": the
	// identifier may well be correct, it is simply too long to look up, and
	// telling the administrator to check the spelling would send them looking
	// for a typo that is not there.
	AdminGrantReasonIdentifierTooLong = "identifier_too_long"
	AdminGrantReasonPlanNotFound      = "plan_not_found"
	// AdminGrantReasonPlanSlugTooLong is reported when the plan slug exceeds the
	// width of subscription.subscription_plans.slug. FR-062 bounds IDENTIFIERS
	// only, so this is an added guard rather than part of that requirement - the
	// same index-degradation rationale applies, on a narrower column.
	AdminGrantReasonPlanSlugTooLong    = "plan_slug_too_long"
	AdminGrantReasonPlanInactive       = "plan_inactive"
	AdminGrantReasonTargetIsAdmin      = "target_is_admin"
	AdminGrantReasonUserSoftDeleted    = "user_soft_deleted"
	AdminGrantReasonDuplicateInRequest = "duplicate_in_request"
	AdminGrantReasonDuplicateRequest   = "duplicate_request"
	AdminGrantReasonConcurrentModif    = "concurrent_modification"
	// 024 T129: an item's own transaction failed for a reason that is not a
	// version conflict - a dead dependency, a refused audit write, a broken
	// invariant. It used to be reported as `concurrent_modification`, which named
	// a cause the operator cannot act on and told them to retry.
	AdminGrantReasonGrantFailed = "grant_failed"
	// 024 T143: the list ran out of its FR-076 time budget. Distinct from
	// grant_failed, which means one item failed on its own merits: here the
	// request itself was too slow, and the fix is to resubmit, not to fix a line.
	AdminGrantReasonBatchTimeout = "batch_timeout"
)

// AdminGrantBatchResult is one item's outcome.
//
// Index and UserIdentifier always echo exactly what the caller sent, in the order
// they sent it (FR-065). ResolvedUserId appears only on success, so the
// administrator knows which account a successful line refers to without a second
// lookup (SC-016).
type AdminGrantBatchResult struct {
	Index          int        `json:"index"`
	UserIdentifier string     `json:"userIdentifier"`
	ResolvedUserID *uuid.UUID `json:"resolvedUserId,omitempty"`
	PlanSlug       string     `json:"planSlug"`
	Status         string     `json:"status"`
	PlanName       string     `json:"planName,omitempty"`
	ExpiresAt      *time.Time `json:"expiresAt,omitempty"`
	Reason         string     `json:"reason,omitempty"`
	Message        string     `json:"message,omitempty"`
}

// AdminGrantBatchRes is the response body.
type AdminGrantBatchRes struct {
	Total     int                     `json:"total"`
	Succeeded int                     `json:"succeeded"`
	Skipped   int                     `json:"skipped"`
	Failed    int                     `json:"failed"`
	Results   []AdminGrantBatchResult `json:"results"`
}

// ---------------------------------------------------------------------------
// AI entitlement restore tool (024 FR-088, FR-089)
//
// This is a REPAIR, not a grant: it changes nobody's plan, it only re-publishes
// the signal so the AI service rebuilds an entitlement that should have existed.
// ---------------------------------------------------------------------------

// RestorePolicyGrantsReq asks for a rescan and a re-publish.
//
// DryRun DEFAULTS TO true. This is a bulk write into customer data, so an
// operator has to see the result BEFORE applying it, not after. A wrong default
// here is a dangerous default.
type RestorePolicyGrantsReq struct {
	DryRun *bool `json:"dryRun,omitempty" label:"chế độ chạy thử"`
}

// Per-account status within a restore run.
const (
	RestoreStatusSuccess = "success"
	RestoreStatusSkipped = "skipped"
	RestoreStatusError   = "error"
)

// Reason codes of a restore run.
const (
	// RestoreReasonPlanNotInCatalog: the plan named by the subscription record is
	// no longer in the catalogue, so there is nothing to rebuild the entitlement from.
	RestoreReasonPlanNotInCatalog = "plan_not_in_catalog"
	// RestoreReasonSubscriptionExpired: the plan has expired. A restore run
	// deliberately leaves these alone - publishing for an expired account would
	// create an "active" entitlement whose end date is in the past, and nothing
	// downstream would notice. Such an account belongs to the renewal and
	// downgrade flows.
	RestoreReasonSubscriptionExpired = "subscription_expired"
	// RestoreReasonPublishFailed: the event could not be published for this account.
	RestoreReasonPublishFailed = "publish_failed"
	// RestoreReasonAlreadyRestored: an earlier run already restored this account.
	// The event key is derived from the subscription's version - which a restore
	// does not change - so a repeated run is skipped. That is what makes re-running
	// safe.
	RestoreReasonAlreadyRestored = "already_restored"
)

// RestoreResult is one account's outcome.
type RestoreResult struct {
	UserID   string `json:"userId"`
	PlanSlug string `json:"planSlug"`
	Status   string `json:"status"`
	Reason   string `json:"reason,omitempty"`
	Message  string `json:"message,omitempty"`
}

// RestorePolicyGrantsRes is the response body.
type RestorePolicyGrantsRes struct {
	DryRun   bool            `json:"dryRun"`
	Scanned  int             `json:"scanned"`
	Restored int             `json:"restored"`
	Skipped  int             `json:"skipped"`
	Failed   int             `json:"failed"`
	Results  []RestoreResult `json:"results"`
}
