package wardrobestatus

type WardrobeItemStatus int16

const (
	InWardrobe WardrobeItemStatus = 0
	Selling    WardrobeItemStatus = 1
	Sold       WardrobeItemStatus = 2
)
