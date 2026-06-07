package wardrobe

import (
	"math"
	"time"
)

const (
	garmentDecayGracePeriodDays = 180.0
	garmentDecayLambda          = 0.01
)

func CalculateLifecycleDecayFactor(lastUsedAt *time.Time, createdAt time.Time, now time.Time) float64 {
	referenceTime := createdAt
	if lastUsedAt != nil {
		referenceTime = *lastUsedAt
	}

	elapsedDays := now.Sub(referenceTime).Hours() / 24
	if elapsedDays <= garmentDecayGracePeriodDays {
		return 1.0
	}

	return math.Exp(-garmentDecayLambda * (elapsedDays - garmentDecayGracePeriodDays))
}
