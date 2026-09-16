package entities

type Category struct {
	AuditableEntity
	Name      string `gorm:"type:varchar(100);uniqueIndex;not null"`
	Slug      string `gorm:"type:varchar(100);uniqueIndex;not null"`
	SortOrder int    `gorm:"column:sort_order;type:int;not null;default:0"`
}
