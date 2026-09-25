package persistence

import (
	"context"

	shared_db "smart-wardrobe-be/shared/infrastructure/db"

	"gorm.io/gorm"
)

// getDB returns the transaction bound to ctx when present (inside a UnitOfWork),
// otherwise the base connection. It mirrors GenericRepository.GetDB for repos
// that do not embed the generic implementation.
func getDB(ctx context.Context, dbConn *gorm.DB) *gorm.DB {
	if tx := shared_db.GetTx(ctx); tx != nil {
		return tx.WithContext(ctx)
	}
	return dbConn.WithContext(ctx)
}
