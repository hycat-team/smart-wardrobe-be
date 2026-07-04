package entities

import (
	shared_entities "smart-wardrobe-be/internal/shared/domain/entities"
)

type Category struct {
	shared_entities.AuditableEntity
	Name      string `gorm:"type:varchar(100);uniqueIndex;not null"`
	Slug      string `gorm:"type:varchar(100);uniqueIndex;not null"`
	SortOrder int    `gorm:"column:sort_order;type:int;not null;default:0"`
}
