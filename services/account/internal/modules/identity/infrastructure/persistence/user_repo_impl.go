package persistence

import (
	"context"
	"errors"
	"strings"
	"time"

	"smart-wardrobe-be/services/account/dto/identity"
	"smart-wardrobe-be/services/account/internal/modules/identity/domain/constants/userstatus"
	"smart-wardrobe-be/services/account/internal/modules/identity/domain/entities"
	"smart-wardrobe-be/services/account/internal/modules/identity/domain/repositories"
	shared_dto "smart-wardrobe-be/shared/application/dto"
	shared_persist "smart-wardrobe-be/shared/infrastructure/repositories"
	"smart-wardrobe-be/shared/pkg/utils/timeutils"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// UserRepository implements identity repository actions using GORM
type UserRepository struct {
	shared_persist.GenericRepository[entities.User, uuid.UUID]
	activityRepo repositories.IUserActivityRepository
}

// NewUserRepository creates a new UserRepository instance. activityRepo
// supplies the admin dashboard DAU; a nil value disables it (tests that only
// exercise other user queries pass nil).
func NewUserRepository(db *gorm.DB, activityRepo repositories.IUserActivityRepository) repositories.IUserRepository {
	relations := []string{}
	return &UserRepository{
		GenericRepository: *shared_persist.NewGenericRepository[entities.User, uuid.UUID](db, relations),
		activityRepo:      activityRepo,
	}
}

// GetByEmail searches for a user matching the provided email address
func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*entities.User, error) {
	var user entities.User
	err := r.GetDB(ctx).Where("LOWER(email) = LOWER(?) AND is_deleted = ?", email, false).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

// GetByProviderID finds a user by their external identity provider id (e.g. Google sub).
func (r *UserRepository) GetByProviderID(ctx context.Context, provider string, providerID string) (*entities.User, error) {
	var user entities.User
	err := r.GetDB(ctx).Where("provider = ? AND provider_id = ? AND is_deleted = ?", provider, providerID, false).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) GetByIDs(ctx context.Context, userIDs []uuid.UUID) ([]*entities.User, error) {
	if len(userIDs) == 0 {
		return nil, nil
	}

	var users []*entities.User
	err := r.GetQueryWithPreload(ctx).
		Where("id IN ? AND is_deleted = ?", userIDs, false).
		Find(&users).Error
	if err != nil {
		return nil, err
	}
	return users, nil
}

// IsEmailExists checks if email already exists in system
func (r *UserRepository) IsEmailExists(ctx context.Context, email string) (bool, error) {
	var count int64
	err := r.GetDB(ctx).Model(&entities.User{}).Where("LOWER(email) = LOWER(?) AND is_deleted = ?", email, false).Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// IsUsernameExists checks if username already exists in system
func (r *UserRepository) IsUsernameExists(ctx context.Context, username string) (bool, error) {
	var count int64
	err := r.GetDB(ctx).Model(&entities.User{}).Where("LOWER(username) = LOWER(?) AND is_deleted = ?", username, false).Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// GetByUsername searches for a user by username.
func (r *UserRepository) GetByUsername(ctx context.Context, username string) (*entities.User, error) {
	var user entities.User
	err := r.GetDB(ctx).Where("LOWER(username) = LOWER(?) AND is_deleted = ?", username, false).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

// GetByUsernameOrEmail searches for a user by email or username
func (r *UserRepository) GetByUsernameOrEmail(ctx context.Context, loginName string) (*entities.User, error) {
	var user entities.User
	err := r.GetDB(ctx).Where("(LOWER(username) = LOWER(?) OR LOWER(email) = LOWER(?)) AND is_deleted = ?", loginName, loginName, false).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) GetByUsernamesOrEmails(ctx context.Context, identifiers []string) ([]*entities.User, error) {
	if len(identifiers) == 0 {
		return []*entities.User{}, nil
	}
	lowers := make([]string, 0, len(identifiers))
	for _, iden := range identifiers {
		trimmed := strings.TrimSpace(iden)
		if trimmed != "" {
			lowers = append(lowers, strings.ToLower(trimmed))
		}
	}
	if len(lowers) == 0 {
		return []*entities.User{}, nil
	}

	var users []*entities.User
	err := r.GetQueryWithPreload(ctx).
		Where("(LOWER(username) IN ? OR LOWER(email) IN ?) AND is_deleted = ?", lowers, lowers, false).
		Find(&users).Error
	return users, err
}

// GetByUsernamesOrEmailsIncludingDeleted is the ONLY lookup that returns
// soft-deleted accounts, and it exists for one caller: the administrator grant
// flow, which has to tell "this account was deleted" apart from "no account
// matches" (FR-049, and the `user_soft_deleted` reason in
// contracts/admin-grant-api.md).
//
// 024 T136. It is deliberately NOT a flag on GetByUsernamesOrEmails. That method
// is shared with commerce (brand members, community follows), which must keep
// excluding deleted accounts - an option would put that decision on every caller
// and one forgotten `false` would silently let a deleted account be added to a
// brand.
//
// The `is_deleted` column is read from the row itself, so the caller sees the
// flag rather than having to infer the state from the row's absence.
func (r *UserRepository) GetByUsernamesOrEmailsIncludingDeleted(ctx context.Context, identifiers []string) ([]*entities.User, error) {
	if len(identifiers) == 0 {
		return []*entities.User{}, nil
	}
	lowers := make([]string, 0, len(identifiers))
	for _, iden := range identifiers {
		trimmed := strings.TrimSpace(iden)
		if trimmed != "" {
			lowers = append(lowers, strings.ToLower(trimmed))
		}
	}
	if len(lowers) == 0 {
		return []*entities.User{}, nil
	}

	var users []*entities.User
	err := r.GetQueryWithPreload(ctx).
		Where("(LOWER(username) IN ? OR LOWER(email) IN ?)", lowers, lowers).
		Find(&users).Error
	return users, err
}

func (r *UserRepository) SearchByKeyword(ctx context.Context, keyword string, limit int) ([]*entities.User, error) {
	kw := strings.ToLower(strings.TrimSpace(keyword))
	if kw == "" {
		return []*entities.User{}, nil
	}
	if limit <= 0 {
		limit = 500
	}
	pattern := "%" + escapeLikePattern(kw) + "%"

	var users []*entities.User
	err := r.GetQueryWithPreload(ctx).
		Where("is_deleted = ?", false).
		Where("(LOWER(username) LIKE ? ESCAPE '\\' OR LOWER(first_name) LIKE ? ESCAPE '\\' OR LOWER(last_name) LIKE ? ESCAPE '\\')",
			pattern, pattern, pattern).
		Order("created_at DESC").
		Limit(limit).
		Find(&users).Error
	if err != nil {
		return nil, err
	}
	return users, nil
}

func escapeLikePattern(s string) string {
	return strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(s)
}

// normalizeCampaignAccountLimit applies the project's shared list defaults to a
// caller-supplied batch size.
//
// It delegates to `shared_dto.PaginationQuery.Normalize` rather than carrying
// local constants, so the default (20) and the ceiling (100) are the same numbers
// every other list in the project uses and cannot drift apart from them.
func normalizeCampaignAccountLimit(limit int) int {
	return shared_dto.PaginationQuery{Limit: limit}.Normalize().Limit
}

// ListCreatedInWindow returns one batch of non-deleted accounts created inside
// [filter.From, filter.To), oldest first (025 T022).
func (r *UserRepository) ListCreatedInWindow(ctx context.Context, filter repositories.CampaignAccountFilter) (*repositories.CampaignAccountPage, error) {
	// Soft-deleted accounts are excluded here rather than filtered by the caller:
	// a deleted account must never receive new entitlements, and the sweep walks
	// this list unattended.
	window := r.GetDB(ctx).Model(&entities.User{}).
		Where("is_deleted = ?", false).
		Where("created_at >= ? AND created_at < ?", filter.From, filter.To)

	// Count the WHOLE window BEFORE the cursor is applied, and before Find.
	//
	// It answers "how many accounts are in this window at all", which is what tells
	// the sweep how many batches are left. Counting the cursor-narrowed set would
	// just return the batch size and make the number useless for that purpose —
	// and `TotalCount` is documented as the window total, not the batch size.
	var windowTotal int64
	if err := window.Count(&windowTotal).Error; err != nil {
		return nil, err
	}

	batch := window
	// The cursor is the COMPOSITE key (created_at, id), not the timestamp alone.
	// `created_at` has millisecond resolution, so accounts registered in the same
	// test run routinely share an instant; paging by the timestamp alone would
	// either skip the remainder of that instant or return it forever. Ordering by
	// BOTH columns and filtering on the same two-column predicate is what makes
	// the batch boundaries stable.
	if !filter.CursorTime.IsZero() {
		batch = batch.Where(
			"(created_at > ?) OR (created_at = ? AND id > ?)",
			filter.CursorTime, filter.CursorTime, filter.CursorID,
		)
	}

	var users []*entities.User
	err := batch.
		Order("created_at ASC, id ASC").
		Limit(normalizeCampaignAccountLimit(filter.Limit)).
		Find(&users).Error
	if err != nil {
		return nil, err
	}

	return &repositories.CampaignAccountPage{
		Items:      users,
		TotalCount: windowTotal,
	}, nil
}

// ListAllActiveUserIDs returns the ids of every non-deleted user, unpaginated.
// The caller is the admin dashboard reader, which needs the whole population;
// GetUsersForAdmin is page-sized (default 20) and would truncate it.
func (r *UserRepository) ListAllActiveUserIDs(ctx context.Context) ([]uuid.UUID, error) {
	var ids []uuid.UUID
	err := r.GetDB(ctx).
		Model(&entities.User{}).
		Where("is_deleted = ?", false).
		Pluck("id", &ids).Error
	if err != nil {
		return nil, err
	}
	return ids, nil
}

func (r *UserRepository) GetUsersForAdmin(ctx context.Context, filter repositories.UserFilter) (*repositories.UserListResult, error) {
	db := r.GetDB(ctx).Model(&entities.User{}).Where("users.is_deleted = ?", false)

	if filter.RoleSlug != nil && *filter.RoleSlug != "" {
		db = db.Where("users.role_slug = ?", *filter.RoleSlug)
	}

	if filter.IsActive != nil {
		status := userstatus.Active
		if !*filter.IsActive {
			status = userstatus.Inactive
		}
		db = db.Where("users.status = ?", status)
	}

	if filter.Query != nil && *filter.Query != "" {
		queryStr := "%" + strings.ToLower(*filter.Query) + "%"
		db = db.Where(
			"LOWER(users.username) LIKE ? OR LOWER(users.email) LIKE ? OR LOWER(COALESCE(users.first_name, '')) LIKE ? OR LOWER(COALESCE(users.last_name, '')) LIKE ?",
			queryStr, queryStr, queryStr, queryStr,
		)
	}

	var totalCount int64
	if err := db.Count(&totalCount).Error; err != nil {
		return nil, err
	}

	var users []*entities.User
	paginationQuery := shared_dto.PaginationQuery{
		Page:  filter.Page,
		Limit: filter.Limit,
	}
	db = shared_persist.ApplyPagination(db, paginationQuery)

	err := db.Order("users.created_at DESC").Find(&users).Error
	if err != nil {
		return nil, err
	}

	return &repositories.UserListResult{
		Users:      users,
		TotalCount: totalCount,
	}, nil
}

// GetAdminUserStats returns the admin dashboard user counts for the Vietnam
// calendar day containing snapshotDate. ActiveUsersDau counts distinct users
// with a real activity event (login/refresh) in that day, not users.updated_at:
// an updated_at move says nothing about activity.
func (r *UserRepository) GetAdminUserStats(ctx context.Context, snapshotDate time.Time) (*dto.UserAdminStatsDTO, error) {
	db := r.GetDB(ctx)

	// Daily stats cut on the Vietnam calendar day.
	startOfDay, endOfDay := timeutils.VNDayRange(snapshotDate)

	var totalUsers int64
	if err := db.Model(&entities.User{}).Where("is_deleted = ?", false).Count(&totalUsers).Error; err != nil {
		return nil, err
	}

	var newUsers int64
	if err := db.Model(&entities.User{}).Where("is_deleted = ? AND created_at >= ? AND created_at < ?", false, startOfDay, endOfDay).Count(&newUsers).Error; err != nil {
		return nil, err
	}

	var dauCount int64
	if r.activityRepo != nil {
		count, err := r.activityRepo.CountDistinctUsers(ctx, startOfDay, endOfDay)
		if err != nil {
			return nil, err
		}
		dauCount = count
	}

	return &dto.UserAdminStatsDTO{
		SnapshotDate:   snapshotDate,
		TotalUsers:     int(totalUsers),
		NewUsers:       int(newUsers),
		ActiveUsersDau: int(dauCount),
	}, nil
}
