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

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// UserRepository implements identity repository actions using GORM
type UserRepository struct {
	shared_persist.GenericRepository[entities.User, uuid.UUID]
}

// NewUserRepository creates a new UserRepository instance
func NewUserRepository(db *gorm.DB) repositories.IUserRepository {
	relations := []string{"StyleProfile"}
	return &UserRepository{
		GenericRepository: *shared_persist.NewGenericRepository[entities.User, uuid.UUID](db, relations),
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

func (r *UserRepository) GetStyleProfile(ctx context.Context, userID uuid.UUID) (*dto.UserStyleProfileRes, error) {
	var profile entities.UserStyleProfile
	err := r.GetDB(ctx).Where("user_id = ?", userID).First(&profile).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &dto.UserStyleProfileRes{
		UserID:          profile.UserID,
		TasteEmbedding:  profile.TasteEmbedding,
		PreferredColors: profile.PreferredColors,
	}, nil
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

func (r *UserRepository) GetAdminUserStats(ctx context.Context, snapshotDate time.Time) (*dto.UserAdminStatsDTO, error) {
	db := r.GetDB(ctx)

	startOfDay := time.Date(snapshotDate.Year(), snapshotDate.Month(), snapshotDate.Day(), 0, 0, 0, 0, snapshotDate.Location())
	endOfDay := startOfDay.Add(24 * time.Hour)

	var totalUsers int64
	if err := db.Model(&entities.User{}).Where("is_deleted = ?", false).Count(&totalUsers).Error; err != nil {
		return nil, err
	}

	var newUsers int64
	if err := db.Model(&entities.User{}).Where("is_deleted = ? AND created_at >= ? AND created_at < ?", false, startOfDay, endOfDay).Count(&newUsers).Error; err != nil {
		return nil, err
	}

	var dauCount int64
	if err := db.Model(&entities.User{}).Where("is_deleted = ? AND updated_at >= ? AND updated_at < ?", false, startOfDay, endOfDay).Count(&dauCount).Error; err != nil {
		return nil, err
	}

	return &dto.UserAdminStatsDTO{
		SnapshotDate:   snapshotDate,
		TotalUsers:     int(totalUsers),
		NewUsers:       int(newUsers),
		ActiveUsersDau: int(dauCount),
	}, nil
}
