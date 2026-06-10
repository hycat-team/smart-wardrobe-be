package dto

import "math"

type PaginationQuery struct {
	Page  int `form:"page"`
	Limit int `form:"limit"`
}

func (q PaginationQuery) Normalize() PaginationQuery {
	if q.Page <= 0 {
		q.Page = 1
	}
	if q.Limit <= 0 {
		q.Limit = 20
	}
	return q
}

func (q PaginationQuery) Offset() int {
	normalized := q.Normalize()
	return (normalized.Page - 1) * normalized.Limit
}

type PaginationMetadata struct {
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
	TotalItems int64 `json:"totalItems"`
	TotalPages int   `json:"totalPages"`
}

type PaginationResult[T any] struct {
	Items    []T                `json:"items"`
	Metadata PaginationMetadata `json:"metadata"`
}

func BuildPaginationMetadata(query PaginationQuery, totalItems int64) PaginationMetadata {
	normalized := query.Normalize()
	totalPages := 0
	if normalized.Limit > 0 && totalItems > 0 {
		totalPages = int(math.Ceil(float64(totalItems) / float64(normalized.Limit)))
	}

	return PaginationMetadata{
		Page:       normalized.Page,
		Limit:      normalized.Limit,
		TotalItems: totalItems,
		TotalPages: totalPages,
	}
}
