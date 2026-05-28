package usecase

import (
	"context"
	"time"

	"smart-wardrobe-be/internal/modules/subscription/domain/repositories"
	"smart-wardrobe-be/internal/shared/application/constants/errorcode"
	"smart-wardrobe-be/internal/shared/domain/entities"

	"github.com/google/uuid"
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
	amount float64,
	transactionType string,
	description string,
	referenceID *uuid.UUID,
	now time.Time,
) error {
	// Zero-ledger statement skip optimization
	if amount == 0.00 {
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
			Balance:   0,
			Currency:  "VND",
			CreatedAt: now,
			UpdatedAt: now,
		}
	}

	// For deductions, ensure sufficient balance
	if amount < 0 && wallet.Balance < -amount {
		return errorcode.NewBadRequest("Số dư tài khoản nội bộ không đủ để thực hiện giao dịch")
	}

	prevBalance := wallet.Balance
	wallet.Balance += amount
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
