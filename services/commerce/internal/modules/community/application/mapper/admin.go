package mapper

import (
	identity_dto "smart-wardrobe-be/services/account/dto/identity"
	"smart-wardrobe-be/services/commerce/internal/modules/community/application/dto"
	"smart-wardrobe-be/services/commerce/internal/modules/community/domain/entities"
)

// MapToAdminPostRes maps a post to the moderation view (no viewer enrichment,
// author enriched when provided).
func MapToAdminPostRes(post *entities.Post, author *identity_dto.UserBriefContractDTO) *dto.AdminPostRes {
	if post == nil {
		return nil
	}
	base := MapToPostRes(post, author, nil, false, false)
	res := &dto.AdminPostRes{
		PostRes:     *base,
		ModeratedBy: post.ModeratedBy,
	}
	if post.HiddenReason != nil {
		reason := string(*post.HiddenReason)
		res.HiddenReason = &reason
	}
	return res
}

// MapToAdminCommentRes maps a comment to the moderation view.
func MapToAdminCommentRes(comment *entities.PostComment, author *identity_dto.UserBriefContractDTO) *dto.AdminCommentRes {
	if comment == nil {
		return nil
	}
	base := MapToCommentRes(comment, author, 0)
	return &dto.AdminCommentRes{
		CommentRes: *base,
		PostID:     comment.PostID,
		Status:     string(comment.Status),
	}
}
