package persistence

import (
	"context"
	"errors"
	"time"

	"smart-wardrobe-be/services/commerce/internal/modules/fashion/domain/entities"
	"smart-wardrobe-be/services/commerce/internal/modules/fashion/domain/repositories"
	shared_db "smart-wardrobe-be/shared/infrastructure/db"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// UserStyleProfileRepository persists fashion.user_style_profiles. The primary
// key is user_id, so it does not embed the generic id-keyed repository.
type UserStyleProfileRepository struct {
	db *gorm.DB
}

func NewUserStyleProfileRepository(db *gorm.DB) repositories.IUserStyleProfileRepository {
	return &UserStyleProfileRepository{db: db}
}

func (r *UserStyleProfileRepository) getDB(ctx context.Context) *gorm.DB {
	if tx := shared_db.GetTx(ctx); tx != nil {
		return tx.WithContext(ctx)
	}
	return r.db.WithContext(ctx)
}

func (r *UserStyleProfileRepository) GetByUserID(ctx context.Context, userID uuid.UUID) (*entities.UserStyleProfile, error) {
	var profile entities.UserStyleProfile
	err := r.getDB(ctx).Where("user_id = ?", userID).First(&profile).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &profile, nil
}

// Upsert inserts the profile or updates its user-editable columns. Learned
// columns are intentionally excluded: a PUT of explicit preferences must never
// wipe the representation derived from saved outfits.
//
// Enabled is written exactly as the caller set it, on both halves: the usecase
// already resolved the effective value (enabled=true for a brand-new profile
// unless the request turned it off, data-model.md §2; an explicit false must
// persist, FR-012). The DDL DEFAULT TRUE therefore only applies to rows created
// outside this repository, and the entity keeps enabled free of a gorm
// `default` tag so GORM writes false instead of omitting it.
func (r *UserStyleProfileRepository) Upsert(ctx context.Context, profile *entities.UserStyleProfile) error {
	if profile == nil {
		return nil
	}
	now := time.Now().UTC()
	if profile.CreatedAt.IsZero() {
		profile.CreatedAt = now
	}
	profile.UpdatedAt = now

	return r.getDB(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "user_id"}},
		DoUpdates: clause.Assignments(map[string]any{
			"styles":          profile.Styles,
			"excluded_styles": profile.ExcludedStyles,
			"enabled":         profile.Enabled,
			"style_embedding": profile.StyleEmbedding,
			"embedding_hash":  profile.EmbeddingHash,
			"embedding_model": profile.EmbeddingModel,
			"updated_at":      profile.UpdatedAt,
		}),
	}).Create(profile).Error
}

func (r *UserStyleProfileRepository) Delete(ctx context.Context, userID uuid.UUID) error {
	return r.getDB(ctx).Where("user_id = ?", userID).Delete(&entities.UserStyleProfile{}).Error
}

// UpdateLearnedWithLock serialises concurrent learning updates of one user. The
// row is read with SELECT ... FOR UPDATE, a default row is created when the user
// has no profile yet, and only learned_* is written back.
func (r *UserStyleProfileRepository) UpdateLearnedWithLock(
	ctx context.Context,
	userID uuid.UUID,
	mutate func(profile *entities.UserStyleProfile) error,
) (*entities.UserStyleProfile, error) {
	if mutate == nil {
		return nil, errors.New("UpdateLearnedWithLock requires a mutate function")
	}

	var updated *entities.UserStyleProfile
	err := r.getDB(ctx).Transaction(func(tx *gorm.DB) error {
		profile, err := lockOrCreateProfile(tx, userID)
		if err != nil {
			return err
		}
		if err := mutate(profile); err != nil {
			return err
		}

		now := time.Now().UTC()
		if profile.LearnedAt == nil {
			profile.LearnedAt = &now
		}
		profile.UpdatedAt = now

		if err := tx.Model(&entities.UserStyleProfile{}).
			Where("user_id = ?", userID).
			Updates(map[string]any{
				"learned_styles":    profile.LearnedStyles,
				"learned_embedding": profile.LearnedEmbedding,
				"learned_at":        profile.LearnedAt,
				"updated_at":        profile.UpdatedAt,
			}).Error; err != nil {
			return err
		}

		updated = profile
		return nil
	})
	if err != nil {
		return nil, err
	}
	return updated, nil
}

// lockOrCreateProfile returns the profile row locked FOR UPDATE, creating it
// with defaults when absent. The re-select after the insert observes a row
// created concurrently by another transaction.
func lockOrCreateProfile(tx *gorm.DB, userID uuid.UUID) (*entities.UserStyleProfile, error) {
	var profile entities.UserStyleProfile
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("user_id = ?", userID).
		First(&profile).Error
	if err == nil {
		return &profile, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	now := time.Now().UTC()
	profile = entities.UserStyleProfile{
		UserID:         userID,
		Styles:         entities.StyleList{},
		ExcludedStyles: entities.StyleList{},
		Enabled:        true,
		LearnedStyles:  entities.StyleWeightMap{},
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err := tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}},
		DoNothing: true,
	}).Create(&profile).Error; err != nil {
		return nil, err
	}

	var locked entities.UserStyleProfile
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("user_id = ?", userID).
		First(&locked).Error; err != nil {
		return nil, err
	}
	return &locked, nil
}
