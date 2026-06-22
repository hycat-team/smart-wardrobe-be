package mapper

import (
	community_dto "smart-wardrobe-be/internal/modules/community/application/dto"
	identity_dto "smart-wardrobe-be/internal/modules/identity/application/dto"
	wardrobe_dto "smart-wardrobe-be/internal/modules/wardrobe/application/dto"
	"smart-wardrobe-be/internal/shared/domain/entities"
)

func MapPostItem(item *entities.PostItem) *community_dto.PostItemRes {
	if item == nil {
		return nil
	}
	return &community_dto.PostItemRes{
		ID:            item.ID,
		Item:          MapWardrobeItem(item.WardrobeItem),
		Price:         item.Price,
		ItemCondition: item.ItemCondition,
		Status:        item.Status,
		BuyerUserID:   item.BuyerUserID,
		TransferState: item.TransferState,
		SoldAt:        item.SoldAt,
		DeclinedAt:    item.DeclinedAt,
	}
}

func MapPost(post *entities.Post, items []*entities.PostItem, media []*entities.PostMedia, isLiked bool, globalHotnessScore float64, finalFeedScore float64) *community_dto.PostRes {
	if post == nil {
		return nil
	}

	postItems := make([]*community_dto.PostItemRes, 0, len(items))
	for _, item := range items {
		postItems = append(postItems, MapPostItem(item))
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
		IsDeleted:          post.IsDeleted,
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

func MapUserDisplayFields(user *entities.User) (string, string, *string) {
	if user == nil {
		return "", "", nil
	}

	firstName := ""
	lastName := ""
	if user.FirstName != nil {
		firstName = *user.FirstName
	}
	if user.LastName != nil {
		lastName = *user.LastName
	}

	return firstName, lastName, user.AvatarUrl
}

func MapCommentUserRes(comment *entities.Comment) (string, string, string, *string) {
	if comment == nil || comment.User == nil {
		return "", "", "", nil
	}

	firstName, lastName, avatarURL := MapUserDisplayFields(comment.User)
	return comment.User.Username, firstName, lastName, avatarURL
}

func MapLikeUserRes(user *entities.User) *community_dto.PostLikeUserRes {
	if user == nil {
		return nil
	}

	firstName, lastName, avatarURL := MapUserDisplayFields(user)
	return &community_dto.PostLikeUserRes{
		ID:        user.ID,
		Username:  user.Username,
		FirstName: firstName,
		LastName:  lastName,
		AvatarURL: avatarURL,
	}
}

func MapCommentRes(comment *entities.Comment) *community_dto.CommentRes {
	if comment == nil {
		return nil
	}

	username, firstName, lastName, avatarURL := MapCommentUserRes(comment)
	return &community_dto.CommentRes{
		ID:              comment.ID,
		UserID:          comment.UserID,
		Username:        username,
		FirstName:       firstName,
		LastName:        lastName,
		AvatarURL:       avatarURL,
		Content:         comment.Content,
		ParentCommentID: comment.ParentCommentID,
		CreatedAt:       comment.CreatedAt,
	}
}

func MapTransferBuyerSummary(user *identity_dto.UserRes) *community_dto.TransferBuyerSummaryRes {
	return community_dto.NewTransferBuyerSummary(user)
}
