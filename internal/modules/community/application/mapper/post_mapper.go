package mapper

import (
	"smart-wardrobe-be/internal/modules/community/application/dto"
	wardrobe_dto "smart-wardrobe-be/internal/modules/wardrobe/application/dto"
	"smart-wardrobe-be/internal/shared/domain/entities"
)

func MapPost(post *entities.Post, items []*entities.PostItem, media []*entities.PostMedia, comments []*entities.Comment) *dto.PostRes {
	if post == nil {
		return nil
	}

	postItems := make([]*dto.PostItemRes, 0, len(items))
	for _, item := range items {
		postItems = append(postItems, &dto.PostItemRes{
			ID:            item.ID,
			Item:          MapWardrobeItem(item.WardrobeItem),
			Price:         item.Price,
			ItemCondition: int16(item.ItemCondition),
			Status:        int16(item.Status),
			BuyerUserID:   item.BuyerUserID,
			TransferState: int16(item.TransferState),
			SoldAt:        item.SoldAt,
		})
	}

	postMedia := make([]*dto.PostMediaRes, 0, len(media))
	for _, item := range media {
		postMedia = append(postMedia, &dto.PostMediaRes{
			ID:        item.ID,
			MediaType: item.MediaType,
			MediaURL:  item.MediaURL,
			PublicID:  item.PublicID,
			SortOrder: item.SortOrder,
		})
	}

	postComments := make([]*dto.CommentRes, 0, len(comments))
	for _, item := range comments {
		postComments = append(postComments, &dto.CommentRes{
			ID:        item.ID,
			UserID:    item.UserID,
			Content:   item.Content,
			CreatedAt: item.CreatedAt,
		})
	}

	return &dto.PostRes{
		ID:           post.ID,
		UserID:       post.UserID,
		PostType:     string(post.PostType),
		Title:        post.Title,
		Content:      post.Content,
		ContactInfo:  post.ContactInfo,
		TotalPrice:   post.TotalPrice,
		LikeCount:    post.LikeCount,
		CommentCount: post.CommentCount,
		Items:        postItems,
		Media:        postMedia,
		Comments:     postComments,
		CreatedAt:    post.CreatedAt,
		UpdatedAt:    post.UpdatedAt,
	}
}

func MapWardrobeItem(item *entities.WardrobeItem) *wardrobe_dto.WardrobeItemRes {
	if item == nil {
		return nil
	}

	color := ""
	style := ""
	material := ""
	pattern := ""
	fit := ""
	seasonality := ""
	if item.Color != nil {
		color = *item.Color
	}
	if item.Style != nil {
		style = *item.Style
	}
	if item.Material != nil {
		material = *item.Material
	}
	if item.Pattern != nil {
		pattern = *item.Pattern
	}
	if item.Fit != nil {
		fit = *item.Fit
	}
	if item.Seasonality != nil {
		seasonality = *item.Seasonality
	}

	return &wardrobe_dto.WardrobeItemRes{
		ID:            item.ID,
		UserID:        item.UserID,
		Category:      nil,
		ImageUrl:      item.ImageUrl,
		ImagePublicID: item.ImagePublicID,
		Color:         color,
		Style:         style,
		Material:      material,
		Pattern:       pattern,
		Fit:           fit,
		Seasonality:   seasonality,
		Status:        item.Status,
		IsLocked:      false,
		CreatedAt:     item.CreatedAt,
	}
}
