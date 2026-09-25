package follow

import (
	"context"
	"sort"
	"strings"
	"time"

	identitycontract "smart-wardrobe-be/services/account/contract/identity"
	identity_dto "smart-wardrobe-be/services/account/dto/identity"
	"smart-wardrobe-be/services/commerce/internal/modules/community/application/dto"
	communityerrors "smart-wardrobe-be/services/commerce/internal/modules/community/application/errors"
	uc_interfaces "smart-wardrobe-be/services/commerce/internal/modules/community/application/interface/usecase"
	"smart-wardrobe-be/services/commerce/internal/modules/community/application/mapper"
	usecase_shared "smart-wardrobe-be/services/commerce/internal/modules/community/application/usecase/shared"
	"smart-wardrobe-be/services/commerce/internal/modules/community/domain/constants/followtype"
	"smart-wardrobe-be/services/commerce/internal/modules/community/domain/entities"
	"smart-wardrobe-be/services/commerce/internal/modules/community/domain/repositories"
	shared_dto "smart-wardrobe-be/shared/application/dto"
	shared_caching "smart-wardrobe-be/shared/application/interface/caching"
	"smart-wardrobe-be/shared/config"
	shared_repos "smart-wardrobe-be/shared/domain/repositories"

	"github.com/google/uuid"
)

type FollowUseCase struct {
	cfg         *config.Config
	uow         shared_repos.IUnitOfWork
	followRepo  repositories.IFollowRepository
	statsRepo   repositories.IUserFollowStatsRepository
	userBrief   identitycontract.IUserBriefContract
	rateLimiter shared_caching.IRateLimiter
}

func NewFollowUseCase(
	cfg *config.Config,
	uow shared_repos.IUnitOfWork,
	followRepo repositories.IFollowRepository,
	statsRepo repositories.IUserFollowStatsRepository,
	userBrief identitycontract.IUserBriefContract,
	rateLimiter shared_caching.IRateLimiter,
) uc_interfaces.IFollowUseCase {
	return &FollowUseCase{
		cfg:         cfg,
		uow:         uow,
		followRepo:  followRepo,
		statsRepo:   statsRepo,
		userBrief:   userBrief,
		rateLimiter: rateLimiter,
	}
}

func (uc *FollowUseCase) ToggleFollow(ctx context.Context, userID uuid.UUID, username string, isFollowing bool) error {
	target, err := usecase_shared.ResolveUserByUsername(ctx, uc.userBrief, username)
	if err != nil {
		return err
	}
	if target == nil {
		return communityerrors.ErrUserNotFound()
	}
	if target.ID == userID {
		return communityerrors.ErrSelfFollow()
	}
	if err := uc.checkRateLimit(ctx, userID); err != nil {
		return err
	}

	exists, err := uc.followRepo.Exists(ctx, userID, target.ID)
	if err != nil {
		return err
	}
	if isFollowing == exists {
		return nil
	}

	return uc.uow.Execute(ctx, func(txCtx context.Context) error {
		if isFollowing {
			if err := uc.followRepo.Create(txCtx, &entities.Follow{FollowerID: userID, FolloweeID: target.ID}); err != nil {
				return err
			}
			if err := uc.statsRepo.Increment(txCtx, target.ID, 1, 0); err != nil {
				return err
			}
			return uc.statsRepo.Increment(txCtx, userID, 0, 1)
		}
		if err := uc.followRepo.Delete(txCtx, userID, target.ID); err != nil {
			return err
		}
		if err := uc.statsRepo.Increment(txCtx, target.ID, -1, 0); err != nil {
			return err
		}
		return uc.statsRepo.Increment(txCtx, userID, 0, -1)
	})
}

type resolvedEdge struct {
	userID   uuid.UUID
	relation string
	at       int64
}

func (uc *FollowUseCase) ListFollows(ctx context.Context, username string, _ *uuid.UUID, query dto.GetFollowsQueryReq) (*dto.FollowUserListRes, error) {
	target, err := usecase_shared.ResolveUserByUsername(ctx, uc.userBrief, username)
	if err != nil {
		return nil, err
	}
	if target == nil {
		return nil, communityerrors.ErrUserNotFound()
	}

	ftype := followtype.FollowType(strings.ToLower(strings.TrimSpace(query.Type)))
	if query.Type != "" && !ftype.IsValid() {
		return nil, communityerrors.ErrInvalidFollowType()
	}

	edges, err := uc.collectEdges(ctx, target.ID, ftype)
	if err != nil {
		return nil, err
	}

	keyword := strings.ToLower(strings.TrimSpace(query.Q))
	briefs := uc.fetchBriefs(ctx, edges)
	filtered := make([]resolvedEdge, 0, len(edges))
	for _, edge := range edges {
		brief, ok := briefs[edge.userID]
		if !ok {
			continue
		}
		if keyword != "" && !matchesKeyword(brief, keyword) {
			continue
		}
		filtered = append(filtered, edge)
	}

	sort.SliceStable(filtered, func(i, j int) bool { return filtered[i].at > filtered[j].at })

	normalized := query.PaginationQuery.Normalize()
	total := int64(len(filtered))
	start := normalized.Offset()
	if start > len(filtered) {
		start = len(filtered)
	}
	end := start + normalized.Limit
	if end > len(filtered) {
		end = len(filtered)
	}

	items := make([]*dto.FollowUserRes, 0, end-start)
	for _, edge := range filtered[start:end] {
		items = append(items, mapper.MapToFollowUserRes(briefs[edge.userID], edge.relation, time.Unix(edge.at, 0)))
	}

	return &dto.FollowUserListRes{
		Items:    items,
		Metadata: shared_dto.BuildPaginationMetadata(query.PaginationQuery, total),
	}, nil
}

func (uc *FollowUseCase) collectEdges(ctx context.Context, targetID uuid.UUID, ftype followtype.FollowType) ([]resolvedEdge, error) {
	switch ftype {
	case followtype.Following:
		return uc.edgesOf(ctx, targetID, followtype.RelationFollowing, true)
	case followtype.Followers:
		return uc.edgesOf(ctx, targetID, followtype.RelationFollower, false)
	default:
		following, err := uc.edgesOf(ctx, targetID, followtype.RelationFollowing, true)
		if err != nil {
			return nil, err
		}
		followers, err := uc.edgesOf(ctx, targetID, followtype.RelationFollower, false)
		if err != nil {
			return nil, err
		}
		return append(following, followers...), nil
	}
}

func (uc *FollowUseCase) edgesOf(ctx context.Context, targetID uuid.UUID, relation string, following bool) ([]resolvedEdge, error) {
	var (
		raw []repositories.FollowEdge
		err error
	)
	if following {
		raw, err = uc.followRepo.ListFollowing(ctx, targetID)
	} else {
		raw, err = uc.followRepo.ListFollowers(ctx, targetID)
	}
	if err != nil {
		return nil, err
	}
	out := make([]resolvedEdge, 0, len(raw))
	for _, e := range raw {
		out = append(out, resolvedEdge{userID: e.UserID, relation: relation, at: e.FollowedAt.Unix()})
	}
	return out, nil
}

func (uc *FollowUseCase) fetchBriefs(ctx context.Context, edges []resolvedEdge) map[uuid.UUID]*identity_dto.UserBriefContractDTO {
	result := make(map[uuid.UUID]*identity_dto.UserBriefContractDTO, len(edges))
	if len(edges) == 0 || uc.userBrief == nil {
		return result
	}
	ids := make([]uuid.UUID, 0, len(edges))
	for _, e := range edges {
		ids = append(ids, e.userID)
	}
	briefs, err := uc.userBrief.GetByIDs(ctx, ids)
	if err != nil {
		return result
	}
	for _, b := range briefs {
		result[b.ID] = b
	}
	return result
}

func (uc *FollowUseCase) checkRateLimit(ctx context.Context, userID uuid.UUID) error {
	limit := uc.cfg.Community.FollowRateLimitPerHour
	if uc.rateLimiter == nil || limit <= 0 {
		return nil
	}
	allowed, err := uc.rateLimiter.Allow(ctx, "community:follow:"+userID.String(), limit, 3600)
	if err != nil || !allowed {
		return communityerrors.ErrRateLimited()
	}
	return nil
}

func matchesKeyword(user *identity_dto.UserBriefContractDTO, keyword string) bool {
	if user == nil {
		return false
	}
	return strings.Contains(strings.ToLower(user.Username), keyword) ||
		strings.Contains(strings.ToLower(user.FirstName), keyword) ||
		strings.Contains(strings.ToLower(user.LastName), keyword)
}
