package wardrobestatus

type WardrobeItemStatus int16

const (
	InWardrobe WardrobeItemStatus = 0
	Selling    WardrobeItemStatus = 1
	Sold       WardrobeItemStatus = 2
	Processing WardrobeItemStatus = 3 // AI processing in background
	Failed     WardrobeItemStatus = 4 // AI processing failed
)
