package shared

import shared_dto "smart-wardrobe-be/internal/shared/application/dto"

// BuildPageBoundedMetadata keeps pagination metadata aligned with the items returned in the current page.
func BuildPageBoundedMetadata(query shared_dto.PaginationQuery, itemCount int) shared_dto.PaginationMetadata {
	normalized := query.Normalize()
	totalPages := 0
	if itemCount > 0 {
		totalPages = 1
	}

	return shared_dto.PaginationMetadata{
		Page:       normalized.Page,
		Limit:      normalized.Limit,
		TotalItems: int64(itemCount),
		TotalPages: totalPages,
	}
}
