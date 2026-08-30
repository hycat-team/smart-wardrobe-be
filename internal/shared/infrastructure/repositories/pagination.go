package repositories

import (
	"gorm.io/gorm"
	shared_dto "smart-wardrobe-be/internal/shared/application/dto"
)

func ApplyPagination(db *gorm.DB, query shared_dto.PaginationQuery) *gorm.DB {
	normalized := query.Normalize()
	return db.Offset(normalized.Offset()).Limit(normalized.Limit)
}
