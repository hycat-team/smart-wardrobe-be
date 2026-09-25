package search

import (
	"context"
	"testing"

	identitycontract "smart-wardrobe-be/services/account/contract/identity"
	identity_dto "smart-wardrobe-be/services/account/dto/identity"
	"smart-wardrobe-be/services/commerce/internal/modules/community/application/dto"
	uc_interfaces "smart-wardrobe-be/services/commerce/internal/modules/community/application/interface/usecase"
	"smart-wardrobe-be/services/commerce/internal/modules/community/domain/entities"
	"smart-wardrobe-be/services/commerce/internal/modules/community/domain/repositories"
	"smart-wardrobe-be/shared/config"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakePostUC struct {
	uc_interfaces.IPostUseCase
}

func (f *fakePostUC) SearchPosts(context.Context, dto.SearchQueryReq, *uuid.UUID) (*dto.PostListRes, error) {
	return &dto.PostListRes{Items: []*dto.PostRes{{PublicID: "p1"}}}, nil
}

type fakeBrief struct {
	identitycontract.IUserBriefContract
	users []*identity_dto.UserBriefContractDTO
}

func (f *fakeBrief) SearchUserBriefs(context.Context, string, int) ([]*identity_dto.UserBriefContractDTO, error) {
	return f.users, nil
}

type fakeStats struct {
	repositories.IUserFollowStatsRepository
	byID map[uuid.UUID]*entities.UserFollowStats
}

func (f *fakeStats) GetByUserIDs(context.Context, []uuid.UUID) (map[uuid.UUID]*entities.UserFollowStats, error) {
	return f.byID, nil
}

func newSearchUC(users []*identity_dto.UserBriefContractDTO, stats map[uuid.UUID]*entities.UserFollowStats) *SearchUseCase {
	cfg := &config.Config{}
	cfg.Community.SearchUserCandidateCap = 500
	return &SearchUseCase{
		cfg:       cfg,
		postUC:    &fakePostUC{},
		userBrief: &fakeBrief{users: users},
		statsRepo: &fakeStats{byID: stats},
	}
}

func TestSearch_EmptyQuery(t *testing.T) {
	uc := newSearchUC(nil, nil)
	res, err := uc.Search(context.Background(), dto.SearchQueryReq{Q: "   "}, nil)
	require.NoError(t, err)
	assert.Empty(t, res.Users.Items)
	assert.Empty(t, res.Posts.Items)
}

func TestSearch_UsersSortedByFollowers(t *testing.T) {
	low := &identity_dto.UserBriefContractDTO{ID: uuid.New(), Username: "b_low"}
	high := &identity_dto.UserBriefContractDTO{ID: uuid.New(), Username: "a_high"}
	stats := map[uuid.UUID]*entities.UserFollowStats{
		low.ID:  {UserID: low.ID, FollowerCount: 1},
		high.ID: {UserID: high.ID, FollowerCount: 10},
	}
	uc := newSearchUC([]*identity_dto.UserBriefContractDTO{low, high}, stats)

	res, err := uc.Search(context.Background(), dto.SearchQueryReq{Q: "a", Type: "users"}, nil)
	require.NoError(t, err)
	require.Len(t, res.Users.Items, 2)
	assert.Equal(t, "a_high", res.Users.Items[0].Username)
	assert.Empty(t, res.Posts.Items)
}

func TestSearch_PostsOnly(t *testing.T) {
	uc := newSearchUC(nil, nil)
	res, err := uc.Search(context.Background(), dto.SearchQueryReq{Q: "outfit", Type: "posts"}, nil)
	require.NoError(t, err)
	assert.Len(t, res.Posts.Items, 1)
	assert.Empty(t, res.Users.Items)
}
