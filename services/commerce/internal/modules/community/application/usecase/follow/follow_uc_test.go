package follow

import (
	"context"
	"testing"

	identitycontract "smart-wardrobe-be/services/account/contract/identity"
	identity_dto "smart-wardrobe-be/services/account/dto/identity"
	"smart-wardrobe-be/services/commerce/internal/modules/community/domain/entities"
	"smart-wardrobe-be/services/commerce/internal/modules/community/domain/repositories"
	"smart-wardrobe-be/shared/config"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeUserBrief struct {
	identitycontract.IUserBriefContract
	users map[string]*identity_dto.UserBriefContractDTO
}

func (f *fakeUserBrief) GetByUsernamesOrEmails(_ context.Context, ids []string) ([]*identity_dto.UserBriefContractDTO, error) {
	out := make([]*identity_dto.UserBriefContractDTO, 0, len(ids))
	for _, id := range ids {
		if u, ok := f.users[id]; ok {
			out = append(out, u)
		}
	}
	return out, nil
}

type fakeFollowRepo struct {
	repositories.IFollowRepository
	exists  bool
	created *entities.Follow
	deleted bool
}

func (f *fakeFollowRepo) Exists(_ context.Context, _, _ uuid.UUID) (bool, error) {
	return f.exists, nil
}

func (f *fakeFollowRepo) Create(_ context.Context, follow *entities.Follow) error {
	f.created = follow
	f.exists = true
	return nil
}

func (f *fakeFollowRepo) Delete(_ context.Context, _, _ uuid.UUID) error {
	f.deleted = true
	f.exists = false
	return nil
}

type fakeStatsRepo struct {
	repositories.IUserFollowStatsRepository
	deltas []int
}

func (f *fakeStatsRepo) Increment(_ context.Context, _ uuid.UUID, followerDelta, followingDelta int) error {
	f.deltas = append(f.deltas, followerDelta, followingDelta)
	return nil
}

type fakeUoW struct{}

func (fakeUoW) Execute(ctx context.Context, fn func(context.Context) error) error { return fn(ctx) }

func newTestUseCase(users map[string]*identity_dto.UserBriefContractDTO, followRepo *fakeFollowRepo, statsRepo *fakeStatsRepo) *FollowUseCase {
	return &FollowUseCase{
		cfg:        &config.Config{},
		uow:        fakeUoW{},
		followRepo: followRepo,
		statsRepo:  statsRepo,
		userBrief:  &fakeUserBrief{users: users},
	}
}

func TestToggleFollow(t *testing.T) {
	ctx := context.Background()
	alice := uuid.New()
	bobID := uuid.New()
	bob := &identity_dto.UserBriefContractDTO{ID: bobID, Username: "bob"}
	users := map[string]*identity_dto.UserBriefContractDTO{"bob": bob}

	t.Run("rejects self follow", func(t *testing.T) {
		self := &identity_dto.UserBriefContractDTO{ID: alice, Username: "alice"}
		uc := newTestUseCase(map[string]*identity_dto.UserBriefContractDTO{"alice": self}, &fakeFollowRepo{}, &fakeStatsRepo{})
		err := uc.ToggleFollow(ctx, alice, "alice", true)
		require.Error(t, err)
	})

	t.Run("rejects unknown user", func(t *testing.T) {
		uc := newTestUseCase(users, &fakeFollowRepo{}, &fakeStatsRepo{})
		err := uc.ToggleFollow(ctx, alice, "ghost", true)
		require.Error(t, err)
	})

	t.Run("follow creates edge and updates counters", func(t *testing.T) {
		repo := &fakeFollowRepo{exists: false}
		stats := &fakeStatsRepo{}
		uc := newTestUseCase(users, repo, stats)
		require.NoError(t, uc.ToggleFollow(ctx, alice, "bob", true))
		require.NotNil(t, repo.created)
		assert.Equal(t, alice, repo.created.FollowerID)
		assert.Equal(t, bobID, repo.created.FolloweeID)
		assert.Equal(t, []int{1, 0, 0, 1}, stats.deltas)
	})

	t.Run("idempotent when already following", func(t *testing.T) {
		repo := &fakeFollowRepo{exists: true}
		stats := &fakeStatsRepo{}
		uc := newTestUseCase(users, repo, stats)
		require.NoError(t, uc.ToggleFollow(ctx, alice, "bob", true))
		assert.Nil(t, repo.created)
		assert.Empty(t, stats.deltas)
	})

	t.Run("unfollow removes edge", func(t *testing.T) {
		repo := &fakeFollowRepo{exists: true}
		stats := &fakeStatsRepo{}
		uc := newTestUseCase(users, repo, stats)
		require.NoError(t, uc.ToggleFollow(ctx, alice, "bob", false))
		assert.True(t, repo.deleted)
		assert.Equal(t, []int{-1, 0, 0, -1}, stats.deltas)
	})
}

func TestMatchesKeyword(t *testing.T) {
	user := &identity_dto.UserBriefContractDTO{Username: "Alice", FirstName: "Alice", LastName: "Nguyen"}
	assert.True(t, matchesKeyword(user, "ali"))
	assert.True(t, matchesKeyword(user, "ngu"))
	assert.False(t, matchesKeyword(user, "bob"))
}
