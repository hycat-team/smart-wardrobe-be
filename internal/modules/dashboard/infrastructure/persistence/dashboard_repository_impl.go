package persistence

import (
	"context"

	"smart-wardrobe-be/internal/modules/dashboard/domain/repositories"

	"gorm.io/gorm"
)

type contextKey string

const ctxTxKey contextKey = "dashboard_tx"

type DashboardRepository struct {
	db *gorm.DB
}

func NewDashboardRepository(db *gorm.DB) repositories.IDashboardRepository {
	return &DashboardRepository{db: db}
}

func (r *DashboardRepository) getDB(ctx context.Context) *gorm.DB {
	if ctx == nil {
		return r.db
	}
	if tx, ok := ctx.Value(ctxTxKey).(*gorm.DB); ok && tx != nil {
		return tx.WithContext(ctx)
	}
	return r.db.WithContext(ctx)
}
