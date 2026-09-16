package types

import (
	"context"
	"smart-wardrobe-be/services/commerce/internal/modules/fashion/application/dto"
)

// RecommendationQueryRewriter định nghĩa giao diện (contract) cho các bộ viết lại truy vấn (query rewriter) trong kiến trúc RAG.
type RecommendationQueryRewriter interface {
	// Rewrite viết lại hoặc làm phong phú thêm ý định của người dùng thành cấu trúc truy vấn nâng cao [RecommendationRetrievalQuery].
	Rewrite(ctx context.Context, intent dto.ParsedIntent) (RecommendationRetrievalQuery, error)
}
