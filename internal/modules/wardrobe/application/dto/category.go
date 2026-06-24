package dto

import "github.com/google/uuid"

type CategoryRes struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Slug      string    `json:"slug"`
	SortOrder int       `json:"sortOrder"`
}

type CreateCategoryReq struct {
	Name      string `json:"name" binding:"required,max=100" label:"tên danh mục"`
	Slug      string `json:"slug" binding:"required,max=100" label:"slug danh mục"`
	SortOrder *int   `json:"sortOrder,omitempty" binding:"omitempty,min=0" label:"thứ tự hiển thị"`
}

type UpdateCategoryReq struct {
	Name      string `json:"name" binding:"required,max=100" label:"tên danh mục"`
	Slug      string `json:"slug" binding:"required,max=100" label:"slug danh mục"`
	SortOrder *int   `json:"sortOrder,omitempty" binding:"omitempty,min=0" label:"thứ tự hiển thị"`
}
