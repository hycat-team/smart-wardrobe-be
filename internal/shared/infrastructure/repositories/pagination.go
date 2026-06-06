package repositories

import (
	"math"

	"gorm.io/gorm"
	shared_dto "smart-wardrobe-be/internal/shared/application/dto"
)

func NormalizePagination(query shared_dto.PaginationQuery) shared_dto.PaginationQuery {
	if query.Page <= 0 {
		query.Page = 1
	}
	if query.Limit <= 0 {
		query.Limit = 20
	}

	return query
}

func Offset(query shared_dto.PaginationQuery) int {
	query = NormalizePagination(query)
	return (query.Page - 1) * query.Limit
}

func ApplyPagination(db *gorm.DB, query shared_dto.PaginationQuery) *gorm.DB {
	query = NormalizePagination(query)
	return db.Offset(Offset(query)).Limit(query.Limit)
}

func BuildPaginationMetadata(query shared_dto.PaginationQuery, totalItems int64) shared_dto.PaginationMetadata {
	query = NormalizePagination(query)
	totalPages := 0
	if query.Limit > 0 && totalItems > 0 {
		totalPages = int(math.Ceil(float64(totalItems) / float64(query.Limit)))
	}

	return shared_dto.PaginationMetadata{
		Page:       query.Page,
		Limit:      query.Limit,
		TotalItems: totalItems,
		TotalPages: totalPages,
	}
}
