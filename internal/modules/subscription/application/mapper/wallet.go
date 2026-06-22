package mapper

import (
	"smart-wardrobe-be/internal/modules/subscription/application/dto"
	"smart-wardrobe-be/internal/shared/domain/entities"
	sharedmoney "smart-wardrobe-be/internal/shared/domain/money"
)

// MapToWalletDTO maps a UserWallet entity into a WalletDTO.
func MapToWalletDTO(wallet *entities.UserWallet) *dto.WalletDTO {
	if wallet == nil {
		return nil
	}
	return &dto.WalletDTO{
		UserID:    wallet.UserID,
		Balance:   sharedmoney.ToFloat(wallet.Balance),
		Currency:  wallet.Currency,
		UpdatedAt: wallet.UpdatedAt,
	}
}

// MapToWalletStatementDTO maps a WalletStatement entity into a WalletStatementDTO.
func MapToWalletStatementDTO(s *entities.WalletStatement) *dto.WalletStatementDTO {
	if s == nil {
		return nil
	}
	return &dto.WalletStatementDTO{
		ID:              s.ID,
		UserID:          s.UserID,
		Amount:          sharedmoney.ToFloat(s.Amount),
		TransactionType: s.TransactionType,
		PreviousBalance: sharedmoney.ToFloat(s.PreviousBalance),
		NewBalance:      sharedmoney.ToFloat(s.NewBalance),
		Description:     s.Description,
		CreatedAt:       s.CreatedAt,
	}
}

// MapToWalletStatementDTOList maps a slice of WalletStatement entities into a slice of WalletStatementDTOs.
func MapToWalletStatementDTOList(statements []*entities.WalletStatement) []*dto.WalletStatementDTO {
	result := make([]*dto.WalletStatementDTO, len(statements))
	for idx, s := range statements {
		result[idx] = MapToWalletStatementDTO(s)
	}
	return result
}
