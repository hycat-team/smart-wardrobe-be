package entities

import "github.com/google/uuid"

func (w *WardrobeItem) FashionCategoryID() *uuid.UUID {
	if w == nil || w.FashionItem == nil {
		return nil
	}
	return w.FashionItem.CategoryID
}

func (w *WardrobeItem) FashionCategory() *Category {
	if w == nil || w.FashionItem == nil {
		return nil
	}
	return w.FashionItem.Category
}

func (w *WardrobeItem) FashionImageUrl() string {
	if w == nil || w.FashionItem == nil {
		return ""
	}
	return w.FashionItem.ImageUrl
}

func (w *WardrobeItem) FashionImagePublicID() string {
	if w == nil || w.FashionItem == nil {
		return ""
	}
	return w.FashionItem.ImagePublicID
}

func (w *WardrobeItem) FashionEmbedding() Vector {
	if w == nil || w.FashionItem == nil {
		return nil
	}
	return w.FashionItem.Embedding
}
