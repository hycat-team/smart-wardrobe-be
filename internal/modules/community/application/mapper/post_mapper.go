package mapper

import (
	community_dto "smart-wardrobe-be/internal/modules/community/application/dto"
	wardrobe_dto "smart-wardrobe-be/internal/modules/wardrobe/application/dto"
	"smart-wardrobe-be/internal/shared/domain/entities"
)

func MapPost(post *entities.Post, items []*entities.PostItem, media []*entities.PostMedia, isLiked bool, globalHotnessScore float64, finalFeedScore float64) *community_dto.PostRes {
	if post == nil {
		return nil
	}

	postItems := make([]*community_dto.PostItemRes, 0, len(items))
	for _, item := range items {
		postItems = append(postItems, &community_dto.PostItemRes{
			ID:            item.ID,
			Item:          MapWardrobeItem(item.WardrobeItem),
			Price:         item.Price,
			ItemCondition: item.ItemCondition,
			Status:        item.Status,
			BuyerUserID:   item.BuyerUserID,
			TransferState: item.TransferState,
			SoldAt:        item.SoldAt,
			DeclinedAt:    item.DeclinedAt,
		})
	}

	postMedia := make([]*community_dto.PostMediaRes, 0, len(media))
	for _, item := range media {
		postMedia = append(postMedia, &community_dto.PostMediaRes{
			ID:        item.ID,
			MediaType: item.MediaType,
			MediaURL:  item.MediaURL,
			PublicID:  item.PublicID,
			SortOrder: item.SortOrder,
		})
	}

	firstName := ""
	lastName := ""
	var avatarURL *string
	username := ""
	if post.User != nil {
		username = post.User.Username
		if post.User.FirstName != nil {
			firstName = *post.User.FirstName
		}
		if post.User.LastName != nil {
			lastName = *post.User.LastName
		}
		avatarURL = post.User.AvatarUrl
	}

	return &community_dto.PostRes{
		ID:                 post.ID,
		PublicID:           post.PublicID,
		UserID:             post.UserID,
		Username:           username,
		FirstName:          firstName,
		LastName:           lastName,
		AvatarURL:          avatarURL,
		PostType:           post.PostType,
		Title:              post.Title,
		Content:            post.Content,
		ContactInfo:        post.ContactInfo,
		TotalPrice:         post.TotalPrice,
		LikeCount:          post.LikeCount,
		CommentCount:       post.CommentCount,
		IsLiked:            isLiked,
		GlobalHotnessScore: globalHotnessScore,
		FinalFeedScore:     finalFeedScore,
		SharePath:          "/posts/" + post.PublicID,
		Items:              postItems,
		Media:              postMedia,
		CreatedAt:          post.CreatedAt,
		UpdatedAt:          post.UpdatedAt,
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

	var category *wardrobe_dto.CategoryRes
	if item.Category != nil {
		category = &wardrobe_dto.CategoryRes{
			ID:   item.Category.ID,
			Name: item.Category.Name,
			Slug: item.Category.Slug,
		}
	}

	return &wardrobe_dto.WardrobeItemRes{
		ID:            item.ID,
		UserID:        item.UserID,
		Category:      category,
		ImageUrl:      item.ImageUrl,
		ImagePublicID: item.ImagePublicID,
		Color:         color,
		Style:         style,
		Material:      material,
		Pattern:       pattern,
		Fit:           fit,
		Seasonality:   seasonality,
		Price:         item.Price,
		Status:        item.Status,
		IsLocked:      false,
		CreatedAt:     item.CreatedAt,
	}
}
