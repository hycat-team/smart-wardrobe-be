package mapper

import (
	identity_dto "smart-wardrobe-be/services/account/dto/identity"
	"smart-wardrobe-be/services/account/dto/subscription"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/entities"
	sharedmoney "smart-wardrobe-be/shared/domain/money"
)

// DepositTransactionToRes maps a deposit transaction entity into the admin
// `DepositTransactionRes` DTO.
//
// It is the ONE mapping behind both the admin list and the admin detail
// endpoint (R5/FR-007): the detail used to answer with the raw entity, so its
// keys (`ID`, `UserID`, `OrderCode`) and its numeric `status` differed from the
// list's camelCase DTO. A shared mapper makes the two shapes identical by
// construction instead of by two code paths staying in step.
//
// The `User` and `TargetPlan` labels are LEFT NIL here. The list must not pay
// for cross-module lookups, and since those fields carry no `omitempty` the list
// item still publishes both keys as null — the same key set the detail answers.
// `DepositTransactionDetailToRes` below is the only place that fills them.
func DepositTransactionToRes(tx *entities.DepositTransaction) *dto.DepositTransactionRes {
	if tx == nil {
		return nil
	}
	return &dto.DepositTransactionRes{
		ID:                   tx.ID,
		UserID:               tx.UserID,
		OrderCode:            tx.OrderCode,
		Amount:               sharedmoney.ToFloat(tx.Amount),
		Currency:             string(tx.Currency),
		Status:               DepositStatusToString(tx.Status),
		TransactionType:      string(tx.TransactionType),
		TargetPlanID:         tx.SubscriptionPlanID,
		PaymentUrl:           tx.PaymentUrl,
		Reference:            tx.SuccessfulProviderReference,
		PaymentLinkId:        tx.PaymentLinkID,
		FailureReason:        tx.FailureReason,
		CancelledAt:          tx.CancelledAt,
		NextReconciliationAt: tx.NextReconciliationAt,
		CreatedAt:            tx.CreatedAt,
		UpdatedAt:            tx.UpdatedAt,
	}
}

// DepositTransactionDetailToRes maps the detail endpoint's enriched read.
//
// It starts from `DepositTransactionToRes` rather than re-listing the base
// fields: two field lists would drift, and the whole point of the shared mapper
// is that the detail is the list item plus labels, never a second shape. Every
// argument after the entity is optional — nil means "no label", which is a
// normal outcome (soft-deleted buyer, removed plan, wallet top-up).
func DepositTransactionDetailToRes(
	tx *entities.DepositTransaction,
	user *identity_dto.UserBriefContractDTO,
	plan *entities.SubscriptionPlan,
) *dto.DepositTransactionRes {
	res := DepositTransactionToRes(tx)
	if res == nil {
		return nil
	}
	res.User = MapTransactionUserBrief(user)
	res.TargetPlan = MapToSubscriptionPlanBrief(plan)
	return res
}

// MapTransactionUserBrief projects the identity contract payload onto the six
// fields this endpoint publishes.
//
// The projection is a WHITELIST, not a copy: the contract DTO carries
// `roleSlug`, `isDeleted`, `createdByAdmin`, `gender`, `status` and `createdAt`,
// none of which the reconciliation screen renders. Keeping the mapping here
// (instead of importing identity's own `MapUserBriefToRes`) is what keeps the
// subscription module from depending on another module's mapper: modules may
// share `application/dto`, not each other's internal mapping layer.
//
// Empty strings become nil pointers so the JSON says "not set" instead of
// shipping `""` for an account that never filled in a last name.
func MapTransactionUserBrief(user *identity_dto.UserBriefContractDTO) *dto.TransactionUserBriefRes {
	if user == nil {
		return nil
	}
	return &dto.TransactionUserBriefRes{
		ID:        user.ID,
		Username:  user.Username,
		Email:     optionalString(user.Email),
		FirstName: optionalString(user.FirstName),
		LastName:  optionalString(user.LastName),
		AvatarUrl: user.AvatarUrl,
	}
}

func optionalString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}
