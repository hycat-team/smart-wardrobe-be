package usecase

import (
	"context"
)

type IPostHotnessUseCase interface {
	RefreshHotness(ctx context.Context) error
}
