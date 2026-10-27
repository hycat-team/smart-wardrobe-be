package ranking

import (
	"time"

	wardrobeDto "smart-wardrobe-be/services/commerce/internal/modules/wardrobe/application/dto"
)

// Rotation score deltas (033 plan D9, US3/T051). They are soft signals like the
// personal profile weights: an item moves up or down the ranked pool, it is
// never filtered out. FR-055 sets the recently-worn window to at least 7 days,
// FR-057 makes "new item" a creation-time property and FR-059 defines the
// cooldown as "used in the last seven days OR recommended in the last three".
const (
	// recentlyWornPenaltyMax is the penalty of an item used today; it decays
	// linearly to zero at the configured window.
	recentlyWornPenaltyMax = 0.45
	// longUnwornBonus is granted once to every item never used or not used for
	// longer than LongUnwornDays, regardless of style/occasion (FR-056).
	longUnwornBonus = 0.15
	// recentlyRecommendedPenalty applies to an item returned by a recent
	// recommendation that was not worn recently, so the same item does not
	// dominate consecutive recommendations.
	recentlyRecommendedPenalty = 0.12
	// DefaultRecentlyWornDays mirrors rag.recently_worn_penalty_days.
	DefaultRecentlyWornDays = 7
	// DefaultRecentlyRecommendedDays mirrors rag.recently_recommended_penalty_days.
	DefaultRecentlyRecommendedDays = 3
	// DefaultLongUnwornDays mirrors rag.long_unworn_bonus_days.
	DefaultLongUnwornDays = 14
)

// RecencyCooldownTag marks a candidate inside the rotation cooldown window. The
// prompt uses it to prefer an equivalent item outside the window (033 FR-059).
const RecencyCooldownTag = "recency-cooldown"

// RecencyWindow carries the configurable rotation windows of one request.
// The zero value means "use the shipped defaults", so a config that predates
// feature 033 keeps working.
type RecencyWindow struct {
	// RecentlyWornDays is the window after which an item stops being "recently worn".
	RecentlyWornDays int
	// RecentlyRecommendedDays is the window after which an item stops being "recently recommended".
	RecentlyRecommendedDays int
	// LongUnwornDays is both the age that grants the long-unworn bonus and the
	// creation-age threshold that marks an item as new.
	LongUnwornDays int
}

// Normalized returns the window with every non-positive field replaced by its
// shipped default.
func (w RecencyWindow) Normalized() RecencyWindow {
	if w.RecentlyWornDays <= 0 {
		w.RecentlyWornDays = DefaultRecentlyWornDays
	}
	if w.RecentlyRecommendedDays <= 0 {
		w.RecentlyRecommendedDays = DefaultRecentlyRecommendedDays
	}
	if w.LongUnwornDays <= 0 {
		w.LongUnwornDays = DefaultLongUnwornDays
	}
	return w
}

// IsInRecencyCooldown reports whether an item sits inside the rotation cooldown
// window: used within RecentlyWornDays, or recommended within
// RecentlyRecommendedDays (033 FR-059). It is the single definition used by
// scoring and by the warm-up statistics, so the two never drift apart.
func IsInRecencyCooldown(item *wardrobeDto.WardrobeItemContractDTO, recency RecencyWindow, now time.Time) bool {
	if item == nil {
		return false
	}
	window := recency.Normalized()
	if item.LastUsedAt != nil && DaysSince(*item.LastUsedAt, now) < window.RecentlyWornDays {
		return true
	}
	return item.LastRecommendedAt != nil &&
		DaysSince(*item.LastRecommendedAt, now) < window.RecentlyRecommendedDays
}

// isNewItem reports whether the item was created inside the rotation bonus
// window. A zero created_at cannot be classified, so the item is treated as old
// rather than new.
func isNewItem(item *wardrobeDto.WardrobeItemContractDTO, now time.Time, windowDays int) bool {
	if item == nil || item.CreatedAt.IsZero() {
		return false
	}
	return DaysSince(item.CreatedAt, now) <= windowDays
}

// DaysSince returns whole days elapsed since the given instant, clamped at zero
// so a clock skew (or a future timestamp) cannot produce a negative age.
func DaysSince(at, now time.Time) int {
	days := int(now.Sub(at).Hours() / 24)
	if days < 0 {
		return 0
	}
	return days
}
