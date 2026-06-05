package usecase

import (
	"context"
	"time"

	"smart-wardrobe-be/internal/modules/subscription/domain/repositories"
	"smart-wardrobe-be/internal/shared/application/constants/errorcode"
	"smart-wardrobe-be/internal/shared/domain/constants/currency"
	"smart-wardrobe-be/internal/shared/domain/constants/walletstatementtype"
	"smart-wardrobe-be/internal/shared/domain/entities"
	sharedmoney "smart-wardrobe-be/internal/shared/domain/money"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// processWalletTransaction is an internal workflow to encapsulate the logic
// for safely modifying a user's wallet balance and recording the transaction statement.
// It must be executed within a UnitOfWork transaction context (txCtx).
// A negative amount indicates a deduction, while a positive amount indicates a top-up.
func processWalletTransaction(
	txCtx context.Context,
	walletRepo repositories.IUserWalletRepository,
	statementRepo repositories.IWalletStatementRepository,
	userID uuid.UUID,
	amount decimal.Decimal,
	transactionType walletstatementtype.WalletStatementType,
	description string,
	referenceID *uuid.UUID,
	now time.Time,
) error {
	// Zero-ledger statement skip optimization
	if amount.IsZero() {
		return nil
	}

	wallet, err := walletRepo.GetByUserIDWithLock(txCtx, userID)
	if err != nil {
		return errorcode.NewInternalError("Lỗi khi truy vấn thông tin số dư ví")
	}

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

	// For deductions, ensure sufficient balance
	if amount.IsNegative() && wallet.Balance.LessThan(amount.Neg()) {
		return errorcode.NewBadRequest("Số dư tài khoản nội bộ không đủ để thực hiện giao dịch")
	}

	prevBalance := wallet.Balance
	wallet.Balance = wallet.Balance.Add(amount)
	wallet.UpdatedAt = now

	if isNewWallet {
		if err := walletRepo.Create(txCtx, wallet); err != nil {
			return errorcode.NewInternalError("Lỗi khi khởi tạo ví mới")
		}
	} else {
		if err := walletRepo.Update(txCtx, wallet); err != nil {
			return errorcode.NewInternalError("Lỗi khi cập nhật số dư ví tài khoản")
		}
	}

	statement := &entities.WalletStatement{
		UserID:          userID,
		Amount:          amount,
		TransactionType: transactionType,
		PreviousBalance: prevBalance,
		NewBalance:      wallet.Balance,
		ReferenceID:     referenceID,
		Description:     description,
	}

	if err := statementRepo.Create(txCtx, statement); err != nil {
		return errorcode.NewInternalError("Lỗi khi lưu lịch sử biến động số dư ví")
	}

	return nil
}
