package search

import (
	"context"
	"sort"
	"strings"

	identitycontract "smart-wardrobe-be/services/account/contract/identity"
	identity_dto "smart-wardrobe-be/services/account/dto/identity"
	"smart-wardrobe-be/services/commerce/internal/modules/community/application/dto"
	uc_interfaces "smart-wardrobe-be/services/commerce/internal/modules/community/application/interface/usecase"
	"smart-wardrobe-be/services/commerce/internal/modules/community/application/mapper"
	"smart-wardrobe-be/services/commerce/internal/modules/community/domain/repositories"
	shared_dto "smart-wardrobe-be/shared/application/dto"
	"smart-wardrobe-be/shared/config"

	"github.com/google/uuid"
)

const (
	resultAll   = "all"
	resultUsers = "users"
	resultPosts = "posts"
)

type SearchUseCase struct {
	cfg       *config.Config
	postUC    uc_interfaces.IPostUseCase
	userBrief identitycontract.IUserBriefContract
	statsRepo repositories.IUserFollowStatsRepository
}

func NewSearchUseCase(
	cfg *config.Config,
	postUC uc_interfaces.IPostUseCase,
	userBrief identitycontract.IUserBriefContract,
	statsRepo repositories.IUserFollowStatsRepository,
) uc_interfaces.ISearchUseCase {
	return &SearchUseCase{
		cfg:       cfg,
		postUC:    postUC,
		userBrief: userBrief,
		statsRepo: statsRepo,
	}
}

func (uc *SearchUseCase) Search(ctx context.Context, query dto.SearchQueryReq, viewerUserID *uuid.UUID) (*dto.SearchRes, error) {
	result := &dto.SearchRes{
		Users: emptyUsers(query),
		Posts: emptyPosts(query),
	}
	keyword := strings.TrimSpace(query.Q)
	if keyword == "" {
		return result, nil
	}

	resultType := strings.ToLower(strings.TrimSpace(query.Type))
	if resultType != resultUsers && resultType != resultPosts {
		resultType = resultAll
	}

	if resultType == resultAll || resultType == resultPosts {
		posts, err := uc.postUC.SearchPosts(ctx, query, viewerUserID)
		if err != nil {
			return nil, err
		}
		result.Posts = *posts
	}
	if resultType == resultAll || resultType == resultUsers {
		users, err := uc.searchUsers(ctx, keyword, query)
		if err != nil {
			return nil, err
		}
		result.Users = *users
	}
	return result, nil
}

type rankedUser struct {
	brief     *identity_dto.UserBriefContractDTO
	followers int64
}

func (uc *SearchUseCase) searchUsers(ctx context.Context, keyword string, query dto.SearchQueryReq) (*shared_dto.PaginationResult[*dto.CommunityUserRes], error) {
	cap := uc.cfg.Community.SearchUserCandidateCap
	if cap <= 0 {
		cap = 500
	}
	briefs, err := uc.userBrief.SearchUserBriefs(ctx, keyword, cap)
	if err != nil {
		return nil, err
	}

	ids := make([]uuid.UUID, 0, len(briefs))
	for _, b := range briefs {
		ids = append(ids, b.ID)
	}
	stats, _ := uc.statsRepo.GetByUserIDs(ctx, ids)

	ranked := make([]rankedUser, 0, len(briefs))
	for _, b := range briefs {
		var followers int64
		if s := stats[b.ID]; s != nil {
			followers = int64(s.FollowerCount)
		}
		ranked = append(ranked, rankedUser{brief: b, followers: followers})
	}
	sort.SliceStable(ranked, func(i, j int) bool {
		if ranked[i].followers != ranked[j].followers {
			return ranked[i].followers > ranked[j].followers
		}
		return ranked[i].brief.Username < ranked[j].brief.Username
	})

	normalized := query.PaginationQuery.Normalize()
	total := int64(len(ranked))
	start := normalized.Offset()
	if start > len(ranked) {
		start = len(ranked)
	}
	end := start + normalized.Limit
	if end > len(ranked) {
		end = len(ranked)
	}

	items := make([]*dto.CommunityUserRes, 0, end-start)
	for _, r := range ranked[start:end] {
		items = append(items, mapper.MapToCommunityUserRes(r.brief))
	}
	return &shared_dto.PaginationResult[*dto.CommunityUserRes]{
		Items:    items,
		Metadata: shared_dto.BuildPaginationMetadata(query.PaginationQuery, total),
	}, nil
}

func emptyUsers(query dto.SearchQueryReq) shared_dto.PaginationResult[*dto.CommunityUserRes] {
	return shared_dto.PaginationResult[*dto.CommunityUserRes]{
		Items:    []*dto.CommunityUserRes{},
		Metadata: shared_dto.BuildPaginationMetadata(query.PaginationQuery, 0),
	}
}

func emptyPosts(query dto.SearchQueryReq) shared_dto.PaginationResult[*dto.PostRes] {
	return shared_dto.PaginationResult[*dto.PostRes]{
		Items:    []*dto.PostRes{},
		Metadata: shared_dto.BuildPaginationMetadata(query.PaginationQuery, 0),
	}
}
