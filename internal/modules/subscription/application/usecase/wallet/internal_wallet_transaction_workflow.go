package wallet

import (
	"context"
	"time"

	"smart-wardrobe-be/internal/modules/subscription/domain/repositories"
	"smart-wardrobe-be/internal/shared/application/constants/apperror"
	"smart-wardrobe-be/internal/shared/domain/constants/currency"
	"smart-wardrobe-be/internal/shared/domain/constants/walletstatementtype"
	"smart-wardrobe-be/internal/shared/domain/entities"
	sharedmoney "smart-wardrobe-be/internal/shared/domain/money"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type WalletStatementMetadata struct {
	SourcePlanCode             *string
	SourceTierRank             *int
	ActiveTierRankAtCompletion *int
	RenewalAttemptKey          *string
}

// ProcessWalletTransaction is an internal workflow to encapsulate the logic
// for safely modifying a user's wallet balance and recording the transaction statement.
// It must be executed within a UnitOfWork transaction context (txCtx) to guarantee ACID properties.
// A negative amount indicates a deduction, while a positive amount indicates a top-up.
func ProcessWalletTransaction(
	txCtx context.Context,
	walletRepo repositories.IUserWalletRepository,
	statementRepo repositories.IWalletStatementRepository,
	userID uuid.UUID,
	amount decimal.Decimal,
	transactionType walletstatementtype.WalletStatementType,
	description string,
	referenceID *uuid.UUID,
	metadata WalletStatementMetadata,
	now time.Time,
) error {
	// Validate metadata invariants based on transactionType
	switch transactionType {
	case walletstatementtype.Topup, walletstatementtype.LowerTierPaymentCredit, walletstatementtype.SameLifetimePaymentCredit:
		if metadata.SourcePlanCode != nil || metadata.SourceTierRank != nil || metadata.ActiveTierRankAtCompletion != nil || metadata.RenewalAttemptKey != nil {
			return apperror.NewBadRequest("Giao dịch Topup hoặc Credit không được chứa thông tin gói cước hoặc gia hạn.")
		}
	case walletstatementtype.SubscriptionPurchase:
		if metadata.SourcePlanCode == nil || *metadata.SourcePlanCode == "" ||
			metadata.SourceTierRank == nil ||
			metadata.ActiveTierRankAtCompletion == nil ||
			metadata.RenewalAttemptKey != nil {
			return apperror.NewBadRequest("Giao dịch mua gói cước thiếu thông tin hoặc chứa thông tin không hợp lệ.")
		}
	case walletstatementtype.SubscriptionRenewal:
		if metadata.SourcePlanCode == nil || *metadata.SourcePlanCode == "" ||
			metadata.SourceTierRank == nil ||
			metadata.ActiveTierRankAtCompletion == nil ||
			metadata.RenewalAttemptKey == nil || *metadata.RenewalAttemptKey == "" {
			return apperror.NewBadRequest("Giao dịch gia hạn gói cước thiếu thông tin hoặc chứa thông tin không hợp lệ.")
		}
	}

	// If the transaction amount is zero, skip database writes entirely as a ledger optimization.
	if amount.IsZero() {
		return nil
	}

	// Retrieve the user's wallet using a database row lock (FOR UPDATE)
	// to prevent race conditions during concurrent deposits or purchases.
	wallet, err := walletRepo.GetByUserIDWithLock(txCtx, userID)
	if err != nil {
		return apperror.NewInternalError("Không thể truy vấn thông tin số dư ví.")
	}

	// Initialize a new wallet if the user does not have one yet.
	var isNewWallet bool
	if wallet == nil {
		isNewWallet = true
		wallet = &entities.UserWallet{
			UserID:    userID,
			Balance:   sharedmoney.Zero,
			Currency:  currency.VND,
			CreatedAt: now,
			UpdatedAt: now,
		}
	}

	// Validate sufficiency of funds for deductions (where amount is negative).
	// If the balance is less than the absolute deduction amount, return a bad request error.
	if amount.IsNegative() && wallet.Balance.LessThan(amount.Neg()) {
		return apperror.NewBadRequest("Số dư ví không đủ để thực hiện giao dịch.")
	}

	// Adjust the wallet balance and update the timestamp.
	prevBalance := wallet.Balance
	wallet.Balance = wallet.Balance.Add(amount)
	wallet.UpdatedAt = now

	// Persist the updated wallet balance.
	// Create a new wallet record if it didn't exist, otherwise update the existing one.
	if isNewWallet {
		if err := walletRepo.Create(txCtx, wallet); err != nil {
			return apperror.NewInternalError("Không thể khởi tạo ví mới.")
		}
	} else {
		if err := walletRepo.Update(txCtx, wallet); err != nil {
			return apperror.NewInternalError("Không thể cập nhật số dư ví.")
		}
	}

	// Create a wallet statement to audit the balance change (double-entry bookkeeping).
	statement := &entities.WalletStatement{
		UserID:                     userID,
		Amount:                     amount,
		TransactionType:            transactionType,
		PreviousBalance:            prevBalance,
		NewBalance:                 wallet.Balance,
		ReferenceID:                referenceID,
		SourcePlanCode:             metadata.SourcePlanCode,
		SourceTierRank:             metadata.SourceTierRank,
		ActiveTierRankAtCompletion: metadata.ActiveTierRankAtCompletion,
		RenewalAttemptKey:          metadata.RenewalAttemptKey,
		Description:                description,
	}

	// Persist the statement record.
	if err := statementRepo.Create(txCtx, statement); err != nil {
		return apperror.NewInternalError("Không thể lưu lịch sử giao dịch ví.")
	}

	return nil
}
