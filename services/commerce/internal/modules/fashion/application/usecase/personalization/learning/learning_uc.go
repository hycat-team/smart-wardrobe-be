package learning

import (
	"context"
	"time"

	"smart-wardrobe-be/services/commerce/internal/modules/fashion/application/dto"
	uc_interfaces "smart-wardrobe-be/services/commerce/internal/modules/fashion/application/interface/usecase"
	"smart-wardrobe-be/services/commerce/internal/modules/fashion/application/usecase/personalization/shared"
	"smart-wardrobe-be/services/commerce/internal/modules/fashion/domain/entities"
	"smart-wardrobe-be/services/commerce/internal/modules/fashion/domain/repositories"
	"smart-wardrobe-be/shared/config"
	shared_repos "smart-wardrobe-be/shared/domain/repositories"
	"smart-wardrobe-be/shared/pkg/logger"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// StyleLearningUseCase implements IStyleLearningUseCase (feature 033, D7): one
// signal in, one decayed update of the derived representation out. Explicit
// preferences are never touched - the repository only writes learned_* columns.
type StyleLearningUseCase struct {
	cfg             *config.Config
	logger          logger.Interface
	signalRepo      repositories.IStyleSignalRepository
	profileRepo     repositories.IUserStyleProfileRepository
	fashionItemRepo repositories.IFashionItemRepository
	uow             shared_repos.IUnitOfWork
}

func NewStyleLearningUseCase(
	cfg *config.Config,
	l logger.Interface,
	signalRepo repositories.IStyleSignalRepository,
	profileRepo repositories.IUserStyleProfileRepository,
	fashionItemRepo repositories.IFashionItemRepository,
	uow shared_repos.IUnitOfWork,
) uc_interfaces.IStyleLearningUseCase {
	return &StyleLearningUseCase{
		cfg:             cfg,
		logger:          l,
		signalRepo:      signalRepo,
		profileRepo:     profileRepo,
		fashionItemRepo: fashionItemRepo,
		uow:             uow,
	}
}

// ApplySignal records the signal and updates the learned profile (see
// IStyleLearningUseCase).
//
// Order of operations:
//  1. kill switch -> ack and skip without writing anything (contracts §8); it
//     is checked first so a disabled pipeline swallows even a malformed message
//     instead of dead-lettering it;
//  2. build/validate the signal (a malformed payload never reaches the DB);
//  3. insert the signal with ON CONFLICT (id) DO NOTHING; a duplicate means the
//     event was already learned and the call returns without decaying twice
//     (FR-037);
//  4. load the item metadata and apply the decayed update to learned_styles /
//     learned_embedding.
//
// Steps 3 and 4 run in one transaction: a failure in the profile update rolls
// the signal back, so the retried message can still learn it.
func (uc *StyleLearningUseCase) ApplySignal(ctx context.Context, input dto.StyleSignalInput) error {
	if !uc.cfg.Personalization.Enabled {
		uc.logger.Debug("Style learning skipped",
			zap.String("reason", "personalization_disabled"),
			zap.String("signalId", input.SignalID.String()),
		)
		return nil
	}

	signal, sourceWeight, err := buildStyleSignal(input)
	if err != nil {
		return err
	}

	return uc.executeInUnitOfWork(ctx, func(txCtx context.Context) error {
		inserted, err := uc.signalRepo.InsertIfAbsent(txCtx, signal)
		if err != nil {
			return err
		}
		if !inserted {
			uc.logger.Debug("Style signal already learned",
				zap.String("reason", "duplicate_signal"),
				zap.String("signalId", signal.ID.String()),
			)
			return nil
		}

		observations, embeddings, err := uc.loadObservations(txCtx, signal.FashionItemIDs)
		if err != nil {
			return err
		}

		_, err = uc.profileRepo.UpdateLearnedWithLock(txCtx, signal.UserID, func(profile *entities.UserStyleProfile) error {
			decay := uc.learningDecay()
			profile.LearnedStyles = applyStyleSignal(profile.LearnedStyles, observations, sourceWeight, decay)
			// No embedding yet means the items are still being analysed: leave
			// the learned vector untouched instead of decaying it away.
			if len(embeddings) > 0 {
				mean := shared.MeanVectors(embeddings)
				if merged := shared.DecayBlendVector(profile.LearnedEmbedding, mean, decay); len(merged) > 0 {
					profile.LearnedEmbedding = entities.Vector(merged)
				}
			}
			learnedAt := time.Now().UTC()
			profile.LearnedAt = &learnedAt
			return nil
		})
		return err
	})
}

// loadObservations reads the style/embedding metadata of the signalled items.
// A missing item (deleted after the outfit was saved) simply contributes
// nothing, which is why a lookup failure is returned but an empty result is not.
func (uc *StyleLearningUseCase) loadObservations(
	ctx context.Context,
	fashionItemIDs entities.UUIDList,
) (map[string]float64, [][]float32, error) {
	if uc.fashionItemRepo == nil || len(fashionItemIDs) == 0 {
		return nil, nil, nil
	}

	items, err := uc.fashionItemRepo.GetByIDs(ctx, []uuid.UUID(fashionItemIDs))
	if err != nil {
		return nil, nil, err
	}
	observations, embeddings := collectStyleObservations(items)
	return observations, embeddings, nil
}

// executeInUnitOfWork runs fn in a transaction when the use case was wired with
// a unit of work. The repositories read the transaction from the context, so the
// signal row and the profile update commit atomically.
func (uc *StyleLearningUseCase) executeInUnitOfWork(ctx context.Context, fn func(context.Context) error) error {
	if uc.uow == nil {
		return fn(ctx)
	}
	return uc.uow.Execute(ctx, fn)
}

// learningDecay reads personalization.learning_decay with the documented
// fallback. Values above 1 are clamped so the learned weights can never grow
// without bound through a typo in configuration.
func (uc *StyleLearningUseCase) learningDecay() float64 {
	decay := uc.cfg.Personalization.LearningDecay
	switch {
	case decay <= 0:
		return defaultLearningDecay
	case decay > 1:
		return 1
	default:
		return decay
	}
}
