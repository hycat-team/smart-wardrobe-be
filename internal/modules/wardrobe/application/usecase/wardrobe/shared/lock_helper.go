package shared

import (
	"smart-wardrobe-be/internal/shared/domain/entities"

	"github.com/google/uuid"
)

// BuildLockedMap creates a map of wardrobe item IDs to their locked status (true if locked)
// based on their creation order (newest first) and subscription limits.
func BuildLockedMap(items []*entities.WardrobeItem, maxItems int) map[uuid.UUID]bool {
	lockedMap := make(map[uuid.UUID]bool)
	for idx, item := range items {
		if idx >= maxItems {
			lockedMap[item.ID] = true
		}
	}
	return lockedMap
}

// IsItemLocked checks whether a specific wardrobe item is locked based on its position
// in the sorted wardrobe items list and the subscription limit.
func IsItemLocked(items []*entities.WardrobeItem, itemID uuid.UUID, maxItems int) bool {
	for idx, item := range items {
		if item.ID == itemID {
			return idx >= maxItems
		}
	}
	return false
}

// FilterActiveItems filters out locked items and returns only active (unlocked) items
// according to the subscription plan limit.
func FilterActiveItems(items []*entities.WardrobeItem, maxItems int) []*entities.WardrobeItem {
	var activeItems []*entities.WardrobeItem
	for idx, item := range items {
		if idx < maxItems {
			activeItems = append(activeItems, item)
		}
	}
	return activeItems
}
