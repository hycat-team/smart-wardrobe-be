package mapper

import (
	identity_dto "smart-wardrobe-be/services/account/dto/identity"
	"smart-wardrobe-be/services/commerce/internal/modules/community/application/dto"
	"smart-wardrobe-be/services/commerce/internal/modules/community/domain/entities"
	wardrobe_dto "smart-wardrobe-be/services/commerce/internal/modules/wardrobe/application/dto"
)

// MapToPostRes builds the API response, enriching with author brief, outfit
// brief and viewer-specific flags computed by the use case.
func MapToPostRes(
	post *entities.Post,
	author *identity_dto.UserBriefContractDTO,
	outfit *wardrobe_dto.OutfitContractDTO,
	isLiked bool,
	isFollowingAuthor bool,
) *dto.PostRes {
	if post == nil {
		return nil
	}

	res := &dto.PostRes{
		ID:                post.ID,
		PublicID:          post.PublicID,
		User:              mapToCommunityUser(author),
		PostType:          post.PostType,
		Status:            post.Status,
		Title:             post.Title,
		Content:           post.Content,
		LikeCount:         post.LikeCount,
		CommentCount:      post.CommentCount,
		IsLiked:           isLiked,
		IsFollowingAuthor: isFollowingAuthor,
		SharePath:         "/community/posts/" + post.PublicID,
		CreatedAt:         post.CreatedAt,
		UpdatedAt:         post.UpdatedAt,
	}

	if outfit != nil {
		res.Outfit = &dto.OutfitBriefRes{
			ID:            outfit.ID,
			Name:          outfit.Name,
			CoverImageURL: outfit.CoverImageURL,
		}
	}

	if len(post.Media) > 0 {
		media := make([]*dto.PostMediaRes, 0, len(post.Media))
		for _, m := range post.Media {
			media = append(media, &dto.PostMediaRes{
				ID:        m.ID,
				MediaType: string(m.MediaType),
				MediaURL:  m.MediaURL,
				PublicID:  m.PublicID,
				SortOrder: m.SortOrder,
			})
		}
		res.Media = media
	}

	return res
}

// MapToOutfitBrief builds the outfit projection attached to an outfit post.
func MapToOutfitBrief(outfit *wardrobe_dto.OutfitContractDTO) *dto.OutfitBriefRes {
	if outfit == nil {
		return nil
	}
	return &dto.OutfitBriefRes{
		ID:            outfit.ID,
		Name:          outfit.Name,
		CoverImageURL: outfit.CoverImageURL,
	}
}
