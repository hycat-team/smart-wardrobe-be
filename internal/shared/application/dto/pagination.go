package dto

type PaginationQuery struct {
	Page  int `form:"page"`
	Limit int `form:"limit"`
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
