package dto

import "github.com/google/uuid"

type CategoryRes struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
	Slug string    `json:"slug"`
}

type CreateCategoryReq struct {
	Name string `json:"name" binding:"required,max=100" label:"tên danh mục"`
	Slug string `json:"slug" binding:"required,max=100" label:"slug danh mục"`
}

type UpdateCategoryReq struct {
	Name string `json:"name" binding:"required,max=100" label:"tên danh mục"`
	Slug string `json:"slug" binding:"required,max=100" label:"slug danh mục"`
}
