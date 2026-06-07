package dto

import (
	shared_dto "smart-wardrobe-be/internal/shared/application/dto"
	"smart-wardrobe-be/internal/shared/domain/entities"

	"github.com/google/uuid"
)

type FeedQuery struct {
	Sort     string
	Page     int
	Limit    int
	Username *string
	PostType *string
}

type FeedResult struct {
	Items    []*FeedPostRecord
	Metadata shared_dto.PaginationMetadata
}

type FeedPostRecord struct {
	Post               *entities.Post
	GlobalHotnessScore float64
}

type PostScoreMetric struct {
	PostID        uuid.UUID
	LikeCount     int
	CommentCount  int
	CreatedAtUnix int64
}
