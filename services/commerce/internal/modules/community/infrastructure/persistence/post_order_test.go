package persistence

import (
	"strings"
	"testing"

	"smart-wardrobe-be/services/commerce/internal/modules/community/domain/constants/feedsort"
	"smart-wardrobe-be/services/commerce/internal/modules/community/domain/repositories"

	"github.com/stretchr/testify/assert"
)

func TestPostOrderClause(t *testing.T) {
	hot := postOrderClause(repositories.PostFilter{Sort: feedsort.Hot, LikeWeight: 1, CommentWeight: 2, HotGravity: 1.5})
	assert.Contains(t, hot, "POWER(")
	assert.Contains(t, strings.ToLower(hot), "like_count")
	assert.Contains(t, strings.ToLower(hot), "comment_count")

	latest := postOrderClause(repositories.PostFilter{Sort: feedsort.Latest})
	assert.Contains(t, latest, "created_at DESC")
	assert.NotContains(t, latest, "POWER(")
}
