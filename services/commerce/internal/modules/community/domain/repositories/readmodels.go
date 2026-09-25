package repositories

import (
	"time"

	"smart-wardrobe-be/services/commerce/internal/modules/community/domain/constants/feedsort"
	"smart-wardrobe-be/services/commerce/internal/modules/community/domain/constants/feedtype"
	"smart-wardrobe-be/services/commerce/internal/modules/community/domain/constants/poststatus"
	"smart-wardrobe-be/services/commerce/internal/modules/community/domain/constants/posttype"
	"smart-wardrobe-be/services/commerce/internal/modules/community/domain/entities"

	"github.com/google/uuid"
)

// PostFilter drives feed/detail queries. FollowingOf is set when Type is
// following; AuthorUserID filters posts of a single author.
type PostFilter struct {
	ViewerUserID  *uuid.UUID
	Type          feedtype.FeedType
	Sort          feedsort.FeedSort
	PostType      *posttype.PostType
	AuthorUserID  *uuid.UUID
	FollowingOf   *uuid.UUID
	HotWindowDays int
	HotGravity    float64
	LikeWeight    int
	CommentWeight int
	Page          int
	Limit         int
}

type PostListResult struct {
	Items      []*entities.Post
	TotalCount int64
}

type CommentListResult struct {
	Items      []*entities.PostComment
	TotalCount int64
}

// FollowEdge is one directed follow relationship with its creation time.
type FollowEdge struct {
	UserID     uuid.UUID
	FollowedAt time.Time
}

// SearchPostsFilter drives the content search query (prefix-limited match).
type SearchPostsFilter struct {
	Keyword            string
	PostType           *posttype.PostType
	ContentPrefixChars int
	HotWindowDays      int
	HotGravity         float64
	LikeWeight         int
	CommentWeight      int
	Page               int
	Limit              int
}

// AdminPostFilter drives the moderation post list.
type AdminPostFilter struct {
	Keyword            string
	Status             *poststatus.PostStatus
	ContentPrefixChars int
	Page               int
	Limit              int
}

// AdminCommentFilter drives the moderation comment list.
type AdminCommentFilter struct {
	Keyword string
	Page    int
	Limit   int
}
