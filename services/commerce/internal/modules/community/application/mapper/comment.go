package mapper

import (
	identity_dto "smart-wardrobe-be/services/account/dto/identity"
	"smart-wardrobe-be/services/commerce/internal/modules/community/application/dto"
	"smart-wardrobe-be/services/commerce/internal/modules/community/domain/constants/poststatus"
	"smart-wardrobe-be/services/commerce/internal/modules/community/domain/entities"
)

// MapToCommentRes maps a comment. A soft-deleted root that still has replies is
// rendered as a placeholder (empty content, isDeleted=true).
func MapToCommentRes(comment *entities.PostComment, author *identity_dto.UserBriefContractDTO, replyCount int64) *dto.CommentRes {
	if comment == nil {
		return nil
	}

	res := &dto.CommentRes{
		ID:              comment.ID,
		User:            mapToCommunityUser(author),
		ParentCommentID: comment.ParentCommentID,
		ReplyCount:      replyCount,
		IsDeleted:       comment.Status == poststatus.Deleted,
		CreatedAt:       comment.CreatedAt,
	}

	if !res.IsDeleted {
		res.Content = comment.Content
	}

	return res
}

// MapToPostLikeUserRes maps a liker to the canonical user shape.
func MapToPostLikeUserRes(user *identity_dto.UserBriefContractDTO) *dto.CommunityUserRes {
	return mapToCommunityUser(user)
}
