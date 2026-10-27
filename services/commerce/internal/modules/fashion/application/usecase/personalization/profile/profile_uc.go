// Package profile owns the style profile use case of feature 033 (US1): read,
// partial update and delete of the declared style preferences plus the
// fail-soft style embedding refreshed on save.
package profile

import (
	"context"

	sub_contract "smart-wardrobe-be/services/account/contract/subscription"
	sub_dto "smart-wardrobe-be/services/account/dto/subscription"
	aiexecutor_contract "smart-wardrobe-be/services/ai/contract/aiexecutor"
	"smart-wardrobe-be/services/commerce/internal/modules/fashion/application/dto"
	fashioerrors "smart-wardrobe-be/services/commerce/internal/modules/fashion/application/errors"
	uc_interfaces "smart-wardrobe-be/services/commerce/internal/modules/fashion/application/interface/usecase"
	"smart-wardrobe-be/services/commerce/internal/modules/fashion/application/mapper"
	"smart-wardrobe-be/services/commerce/internal/modules/fashion/domain/entities"
	"smart-wardrobe-be/services/commerce/internal/modules/fashion/domain/repositories"
	wardrobe_contract "smart-wardrobe-be/services/commerce/internal/modules/wardrobe/presentation/contract"
	"smart-wardrobe-be/shared/config"
	"smart-wardrobe-be/shared/pkg/logger"

	"github.com/google/uuid"
)

// StyleProfileUseCase implements IStyleProfileUseCase. Entitlement comes from
// the subscription contract (D2); the style embedding is produced here, on save
// only, through the AI orchestrator contract (D3). Cold-start suggestions read
// the owned wardrobe through IWardrobeContract and the item metadata through the
// fashion repository, so no cross-module DB join is needed (D10).
type StyleProfileUseCase struct {
	cfg              *config.Config
	logger           logger.Interface
	aiService        aiexecutor_contract.IAIOrchestrator
	profileRepo      repositories.IUserStyleProfileRepository
	userSubContract  sub_contract.IUserSubscriptionContract
	wardrobeContract wardrobe_contract.IWardrobeContract
	fashionItemRepo  repositories.IFashionItemRepository
}

func NewStyleProfileUseCase(
	cfg *config.Config,
	l logger.Interface,
	aiService aiexecutor_contract.IAIOrchestrator,
	profileRepo repositories.IUserStyleProfileRepository,
	userSubContract sub_contract.IUserSubscriptionContract,
	wardrobeContract wardrobe_contract.IWardrobeContract,
	fashionItemRepo repositories.IFashionItemRepository,
) uc_interfaces.IStyleProfileUseCase {
	return &StyleProfileUseCase{
		cfg:              cfg,
		logger:           l,
		aiService:        aiService,
		profileRepo:      profileRepo,
		userSubContract:  userSubContract,
		wardrobeContract: wardrobeContract,
		fashionItemRepo:  fashionItemRepo,
	}
}

// GetProfile returns the stored profile or the defaults when the user has never
// saved one. A user who lost entitlement still reads the profile (FR-004); the
// response reports personalizationAvailable=false instead of failing.
func (uc *StyleProfileUseCase) GetProfile(ctx context.Context, userID uuid.UUID) (*dto.StyleProfileRes, error) {
	overview, err := uc.userSubContract.GetUserSubscriptionOverview(ctx, userID)
	if err != nil {
		return nil, err
	}

	profile, err := uc.profileRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	return mapper.MapUserStyleProfileRes(profile, personalizationAvailable(overview), uc.maxStyles()), nil
}

// UpdateProfile applies a partial update of the profile.
//
// Order of operations (T027): kill switch -> entitlement -> load existing row ->
// normalize the submitted lists (trim, drop empty, dedupe case-insensitively,
// per-value length, per-list cap) -> refresh the embedding (fail-soft) -> upsert.
//
// The response is re-read from the repository so it always reports what was
// actually stored, including the enabled flag of a freshly created row.
func (uc *StyleProfileUseCase) UpdateProfile(
	ctx context.Context,
	userID uuid.UUID,
	input dto.UpdateStyleProfileReq,
) (*dto.StyleProfileRes, error) {
	if !uc.cfg.Personalization.Enabled {
		return nil, fashioerrors.ErrPersonalizationDisabled()
	}

	overview, err := uc.userSubContract.GetUserSubscriptionOverview(ctx, userID)
	if err != nil {
		return nil, err
	}
	if !personalizationAvailable(overview) {
		return nil, fashioerrors.ErrPersonalizationNotAvailable()
	}

	maxStyles := uc.maxStyles()

	existing, err := uc.profileRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	styles, excludedStyles, enabled, err := mergeStyleProfileInput(existing, input, maxStyles)
	if err != nil {
		return nil, err
	}

	profile := &entities.UserStyleProfile{
		UserID:         userID,
		Styles:         entities.StyleList(styles),
		ExcludedStyles: entities.StyleList(excludedStyles),
		Enabled:        enabled,
	}
	if existing != nil {
		// user_id is the primary key and created_at is preserved by the
		// repository upsert; carrying it keeps the entity faithful.
		profile.CreatedAt = existing.CreatedAt
	}

	uc.applyEmbedding(ctx, profile, existing, styles, excludedStyles)

	if err := uc.profileRepo.Upsert(ctx, profile); err != nil {
		return nil, err
	}

	stored, err := uc.profileRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if stored == nil {
		// Defensive: the upsert must have produced a row.
		stored = profile
	}

	return mapper.MapUserStyleProfileRes(stored, personalizationAvailable(overview), maxStyles), nil
}

// DeleteProfile removes the profile row. It is idempotent: deleting an absent
// row succeeds.
//
// The user's own enabled flag never blocks the delete (FR-012 lets a user turn
// personalization off without losing data, and deleting that data stays their
// right). Only the operator kill switch and the missing entitlement refuse the
// write.
func (uc *StyleProfileUseCase) DeleteProfile(ctx context.Context, userID uuid.UUID) error {
	if !uc.cfg.Personalization.Enabled {
		return fashioerrors.ErrPersonalizationDisabled()
	}

	overview, err := uc.userSubContract.GetUserSubscriptionOverview(ctx, userID)
	if err != nil {
		return err
	}
	if !personalizationAvailable(overview) {
		return fashioerrors.ErrPersonalizationNotAvailable()
	}

	return uc.profileRepo.Delete(ctx, userID)
}

// SuggestStyles derives the dominant styles of the wardrobe the user already
// owns (cold start, contracts §1.4 / D10).
//
// The call is read-only: nothing is persisted, and the user confirms a
// suggestion by calling PUT /me/style-profile afterwards (FR-046: a suggestion
// is never applied implicitly). The operator kill switch does not block this
// read - contracts §8 keeps profile reads available for operational checks - so
// only the plan entitlement is required.
//
// A user with no styled item gets an empty list rather than an error: the
// "Chưa đủ dữ liệu..." wording belongs to the response envelope, not to the
// payload (contracts §1.4).
func (uc *StyleProfileUseCase) SuggestStyles(
	ctx context.Context,
	userID uuid.UUID,
) (*dto.StyleSuggestionRes, error) {
	overview, err := uc.userSubContract.GetUserSubscriptionOverview(ctx, userID)
	if err != nil {
		return nil, err
	}
	if !personalizationAvailable(overview) {
		return nil, fashioerrors.ErrPersonalizationNotAvailable()
	}

	// The style labels come from fashion items; without the wardrobe and the
	// item repository there is nothing to derive, so respond with an empty
	// sample instead of panicking (defensive: Wire always provides both).
	if uc.wardrobeContract == nil || uc.fashionItemRepo == nil {
		return emptyStyleSuggestionRes(), nil
	}

	// Active items are filtered (InWardrobe, not deleted) and capped by the plan
	// limit inside the wardrobe module; the fashion module never reads wardrobe
	// tables (Constitution §II).
	activeItems, err := uc.wardrobeContract.GetActiveWardrobeItemsByUserID(ctx, userID, overview.MaxWardrobeItems)
	if err != nil {
		return nil, err
	}

	fashionItemIDs := distinctFashionItemIDs(activeItems)
	if len(fashionItemIDs) == 0 {
		return emptyStyleSuggestionRes(), nil
	}

	items, err := uc.fashionItemRepo.GetByIDs(ctx, fashionItemIDs)
	if err != nil {
		return nil, err
	}

	return buildStyleSuggestions(items), nil
}

// personalizationAvailable reads the plan entitlement from the overview. A nil
// overview is treated as "not entitled" rather than a panic.
func personalizationAvailable(overview *sub_dto.UserSubscriptionContractDTO) bool {
	return overview != nil && overview.PersonalizationEnabled
}
