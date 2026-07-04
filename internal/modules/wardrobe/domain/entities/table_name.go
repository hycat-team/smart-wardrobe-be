package entities

func (WardrobeItem) TableName() string {
	return "wardrobe_items"
}

func (Outfit) TableName() string {
	return "outfits"
}

func (OutfitItem) TableName() string {
	return "outfit_items"
}
