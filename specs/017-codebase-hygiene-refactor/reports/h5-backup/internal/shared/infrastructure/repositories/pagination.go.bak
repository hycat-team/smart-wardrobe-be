package repositories

import (
	"gorm.io/gorm"
	shared_dto "smart-wardrobe-be/internal/shared/application/dto"
)

func NormalizePagination(query shared_dto.PaginationQuery) shared_dto.PaginationQuery {
	return query.Normalize()
}

func Offset(query shared_dto.PaginationQuery) int {
	return query.Offset()
}

func ApplyPagination(db *gorm.DB, query shared_dto.PaginationQuery) *gorm.DB {
	normalized := query.Normalize()
	return db.Offset(normalized.Offset()).Limit(normalized.Limit)
}

func BuildPaginationMetadata(query shared_dto.PaginationQuery, totalItems int64) shared_dto.PaginationMetadata {
	return shared_dto.BuildPaginationMetadata(query, totalItems)
}
