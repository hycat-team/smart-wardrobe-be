package mapper

import (
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
