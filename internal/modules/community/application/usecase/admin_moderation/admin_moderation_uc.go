package admin_moderation

import (
	"context"
	"math"

	"smart-wardrobe-be/internal/modules/community/application/dto"
	communityerrors "smart-wardrobe-be/internal/modules/community/application/errors"
	uc_interfaces "smart-wardrobe-be/internal/modules/community/application/interface/usecase"
	"smart-wardrobe-be/internal/modules/community/application/mapper"
	postusecase "smart-wardrobe-be/internal/modules/community/application/usecase/post"
	"smart-wardrobe-be/internal/modules/community/domain/repositories"
	wardrobe_contract "smart-wardrobe-be/internal/modules/wardrobe/contract"
	shared_dto "smart-wardrobe-be/internal/shared/application/dto"
	"smart-wardrobe-be/internal/shared/domain/constants/postitemstatus"
	"smart-wardrobe-be/internal/shared/domain/entities"
	shared_repos "smart-wardrobe-be/internal/shared/domain/repositories"
	"smart-wardrobe-be/pkg/logger"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type AdminCommunityModerationUseCase struct {
	postRepo      repositories.IPostRepository
	postItemRepo  repositories.IPostItemRepository
	postMediaRepo repositories.IPostMediaRepository
	commentRepo   repositories.ICommentRepository
	wardrobeCtr   wardrobe_contract.IWardrobeContract
	uow           shared_repos.IUnitOfWork
	logger        logger.Interface
}

func NewAdminCommunityModerationUseCase(
	log logger.Interface,
	postRepo repositories.IPostRepository,
	postItemRepo repositories.IPostItemRepository,
	postMediaRepo repositories.IPostMediaRepository,
	commentRepo repositories.ICommentRepository,
	wardrobeCtr wardrobe_contract.IWardrobeContract,
	uow shared_repos.IUnitOfWork,
) uc_interfaces.IAdminCommunityModerationUseCase {
	return &AdminCommunityModerationUseCase{
		postRepo:      postRepo,
		postItemRepo:  postItemRepo,
		postMediaRepo: postMediaRepo,
		commentRepo:   commentRepo,
		wardrobeCtr:   wardrobeCtr,
		uow:           uow,
		logger:        log,
	}
}

func (uc *AdminCommunityModerationUseCase) AdminDeletePost(ctx context.Context, adminUserID uuid.UUID, postPublicID string) error {
	post, err := uc.postRepo.GetByPublicID(ctx, postPublicID)
	if err != nil {
		return err
	}
	if post == nil {
		return communityerrors.ErrPostNotFound
	}

	postItems, err := uc.postItemRepo.GetByPostID(ctx, post.ID)
	if err != nil {
		return err
	}

	deletePost := func(txCtx context.Context) error {
		if err := uc.postRepo.SoftDelete(txCtx, post.ID); err != nil {
			return err
		}

		for _, itemID := range postusecase.UniqueItemIDs(postItems) {
			if err := postusecase.SyncWardrobeStatusByItem(txCtx, uc.postItemRepo, uc.wardrobeCtr, itemID); err != nil {
				return err
			}
		}

		uc.logger.Info("[CommunityModeration] Admin deleted post",
			zap.String("admin_user_id", adminUserID.String()),
			zap.String("action", "delete_post"),
			zap.String("target_type", "post"),
			zap.String("target_id", postPublicID),
		)
		return nil
	}

	return uc.uow.Execute(ctx, deletePost)
}

func (uc *AdminCommunityModerationUseCase) AdminHidePostItem(ctx context.Context, adminUserID uuid.UUID, postItemID uuid.UUID) error {
	postItem, err := uc.postItemRepo.GetByID(ctx, postItemID)
	if err != nil {
		return err
	}
	if postItem == nil {
		return communityerrors.ErrPostItemNotFound
	}

	hidePostItem := func(txCtx context.Context) error {
		postItem.Status = postitemstatus.Hidden
		if err := uc.postItemRepo.Update(txCtx, postItem); err != nil {
			return err
		}
		if err := postusecase.SyncWardrobeStatusByItem(txCtx, uc.postItemRepo, uc.wardrobeCtr, postItem.ItemID); err != nil {
			return err
		}
		if err := postusecase.SyncPostTotalPrice(txCtx, uc.postRepo, uc.postItemRepo, postItem.PostID); err != nil {
			return err
		}

		uc.logger.Info("[CommunityModeration] Admin hid post item",
			zap.String("admin_user_id", adminUserID.String()),
			zap.String("action", "hide_post_item"),
			zap.String("target_type", "post_item"),
			zap.String("target_id", postItemID.String()),
		)
		return nil
	}

	return uc.uow.Execute(ctx, hidePostItem)
}

func (uc *AdminCommunityModerationUseCase) AdminDeletePostItem(ctx context.Context, adminUserID uuid.UUID, postItemID uuid.UUID) error {
	postItem, err := uc.postItemRepo.GetByID(ctx, postItemID)
	if err != nil {
		return err
	}
	if postItem == nil {
		return communityerrors.ErrPostItemNotFound
	}

	post, err := uc.postRepo.GetByID(ctx, postItem.PostID)
	if err != nil {
		return err
	}
	if post == nil {
		return communityerrors.ErrPostNotFound
	}

	deletePost := func(txCtx context.Context) error {
		postItems, err := uc.postItemRepo.GetByPostID(txCtx, post.ID)
		if err != nil {
			return err
		}

		if err := uc.postRepo.SoftDelete(txCtx, post.ID); err != nil {
			return err
		}

		for _, itemID := range postusecase.UniqueItemIDs(postItems) {
			if err := postusecase.SyncWardrobeStatusByItem(txCtx, uc.postItemRepo, uc.wardrobeCtr, itemID); err != nil {
				return err
			}
		}

		uc.logger.Info("[CommunityModeration] Admin deleted post by post item moderation",
			zap.String("admin_user_id", adminUserID.String()),
			zap.String("action", "delete_post_item"),
			zap.String("target_type", "post_item"),
			zap.String("target_id", postItemID.String()),
			zap.String("post_id", post.PublicID),
		)
		return nil
	}

	return uc.uow.Execute(ctx, deletePost)
}

func (uc *AdminCommunityModerationUseCase) AdminDeleteComment(ctx context.Context, adminUserID uuid.UUID, commentID uuid.UUID) error {
	deleteComment := func(txCtx context.Context) error {
		comment, err := uc.commentRepo.GetByID(txCtx, commentID)
		if err != nil {
			return err
		}
		if comment == nil {
			return communityerrors.ErrCommentNotFound
		}

		post, err := uc.postRepo.GetByID(txCtx, comment.PostID)
		if err != nil {
			return err
		}
		if post == nil {
			return communityerrors.ErrPostNotFound
		}

		if err := uc.commentRepo.SoftDelete(txCtx, commentID); err != nil {
			return err
		}

		decrement := 1
		if comment.ParentCommentID == nil {
			replies, err := uc.commentRepo.GetRepliesByParentID(txCtx, post.ID, commentID)
			if err != nil {
				return err
			}
			decrement += len(replies)
			if err := uc.commentRepo.SoftDeleteByParentID(txCtx, commentID); err != nil {
				return err
			}
		}

		post.CommentCount -= decrement
		if post.CommentCount < 0 {
			post.CommentCount = 0
		}
		if err := uc.postRepo.Update(txCtx, post); err != nil {
			return err
		}
		if err := uc.postRepo.MarkHotnessDirty(txCtx, post.ID); err != nil {
			return err
		}

		uc.logger.Info("[CommunityModeration] Admin deleted comment",
			zap.String("admin_user_id", adminUserID.String()),
			zap.String("action", "delete_comment"),
			zap.String("target_type", "comment"),
			zap.String("target_id", commentID.String()),
		)
		return nil
	}

	return uc.uow.Execute(ctx, deleteComment)
}

func (uc *AdminCommunityModerationUseCase) GetPostsForAdmin(ctx context.Context, query dto.AdminGetPostsQueryReq) (*dto.AdminPostListRes, error) {
	filter := repositories.AdminPostFilter{
		PostType:  query.PostType,
		IsDeleted: query.IsDeleted,
		Query:     query.Query,
		Page:      query.Page,
		Limit:     query.Limit,
	}

	result, err := uc.postRepo.GetPostsForAdmin(ctx, filter)
	if err != nil {
		return nil, err
	}

	resPosts := make([]*dto.PostRes, len(result.Posts))
	postIDs := make([]uuid.UUID, 0, len(result.Posts))
	for _, post := range result.Posts {
		postIDs = append(postIDs, post.ID)
	}

	postItems, err := uc.postItemRepo.GetByPostIDs(ctx, uniqueUUIDs(postIDs))
	if err != nil {
		return nil, err
	}
	postItemsByPostID := make(map[uuid.UUID][]*entities.PostItem)
	for _, item := range postItems {
		if item == nil {
			continue
		}
		postItemsByPostID[item.PostID] = append(postItemsByPostID[item.PostID], item)
	}

	postMedia, err := uc.postMediaRepo.GetByPostIDs(ctx, uniqueUUIDs(postIDs))
	if err != nil {
		return nil, err
	}
	postMediaByPostID := make(map[uuid.UUID][]*entities.PostMedia)
	for _, item := range postMedia {
		if item == nil {
			continue
		}
		postMediaByPostID[item.PostID] = append(postMediaByPostID[item.PostID], item)
	}

	for idx, post := range result.Posts {
		resPosts[idx] = mapper.MapPost(post, postItemsByPostID[post.ID], postMediaByPostID[post.ID], false, 0, 0)
	}

	limit := query.Limit
	if limit <= 0 {
		limit = 20
	}
	page := query.Page
	if page <= 0 {
		page = 1
	}

	totalPages := 0
	if limit > 0 && result.TotalCount > 0 {
		totalPages = int(math.Ceil(float64(result.TotalCount) / float64(limit)))
	}

	return &shared_dto.PaginationResult[*dto.PostRes]{
		Items: resPosts,
		Metadata: shared_dto.PaginationMetadata{
			Page:       page,
			Limit:      limit,
			TotalItems: result.TotalCount,
			TotalPages: totalPages,
		},
	}, nil
}

func (uc *AdminCommunityModerationUseCase) GetPostItemsForAdmin(ctx context.Context, query dto.AdminGetPostItemsQueryReq) (*dto.AdminPostItemListRes, error) {
	filter := repositories.AdminPostItemFilter{
		Status:        query.Status,
		TransferState: query.TransferState,
		Page:          query.Page,
		Limit:         query.Limit,
	}

	result, err := uc.postItemRepo.GetPostItemsForAdmin(ctx, filter)
	if err != nil {
		return nil, err
	}

	resItems := make([]*dto.PostItemRes, len(result.PostItems))
	for idx, item := range result.PostItems {
		resItems[idx] = mapper.MapPostItem(item)
	}

	limit := query.Limit
	if limit <= 0 {
		limit = 20
	}
	page := query.Page
	if page <= 0 {
		page = 1
	}

	totalPages := 0
	if limit > 0 && result.TotalCount > 0 {
		totalPages = int(math.Ceil(float64(result.TotalCount) / float64(limit)))
	}

	return &shared_dto.PaginationResult[*dto.PostItemRes]{
		Items: resItems,
		Metadata: shared_dto.PaginationMetadata{
			Page:       page,
			Limit:      limit,
			TotalItems: result.TotalCount,
			TotalPages: totalPages,
		},
	}, nil
}

func (uc *AdminCommunityModerationUseCase) AdminRestorePost(ctx context.Context, adminUserID uuid.UUID, postPublicID string) error {
	post, err := uc.postRepo.GetByPublicIDIncludingDeleted(ctx, postPublicID)
	if err != nil {
		return err
	}
	if post == nil {
		return communityerrors.ErrPostNotFound
	}

	postItems, err := uc.postItemRepo.GetByPostID(ctx, post.ID)
	if err != nil {
		return err
	}

	restorePost := func(txCtx context.Context) error {
		if err := uc.postRepo.Restore(txCtx, post.ID); err != nil {
			return err
		}

		for _, itemID := range postusecase.UniqueItemIDs(postItems) {
			if err := postusecase.SyncWardrobeStatusByItem(txCtx, uc.postItemRepo, uc.wardrobeCtr, itemID); err != nil {
				return err
			}
		}

		uc.logger.Info("[CommunityModeration] Admin restored post",
			zap.String("admin_user_id", adminUserID.String()),
			zap.String("action", "restore_post"),
			zap.String("target_type", "post"),
			zap.String("target_id", postPublicID),
		)
		return nil
	}

	return uc.uow.Execute(ctx, restorePost)
}

func (uc *AdminCommunityModerationUseCase) AdminRestoreComment(ctx context.Context, adminUserID uuid.UUID, commentID uuid.UUID) error {
	restoreComment := func(txCtx context.Context) error {
		comment, err := uc.commentRepo.GetByIDIncludingDeleted(txCtx, commentID)
		if err != nil {
			return err
		}
		if comment == nil {
			return communityerrors.ErrCommentNotFound
		}
		if !comment.IsDeleted {
			return nil
		}

		post, err := uc.postRepo.GetByID(txCtx, comment.PostID)
		if err != nil {
			return err
		}
		if post == nil {
			return communityerrors.ErrPostNotFound
		}

		if comment.ParentCommentID != nil {
			parentComment, err := uc.commentRepo.GetByIDIncludingDeleted(txCtx, *comment.ParentCommentID)
			if err != nil {
				return err
			}
			if parentComment == nil || parentComment.IsDeleted {
				return communityerrors.ErrCommentReplyTargetInvalid
			}
		}

		if err := uc.commentRepo.Restore(txCtx, commentID); err != nil {
			return err
		}

		increment := 1
		if comment.ParentCommentID == nil {
			replies, err := uc.commentRepo.GetRepliesByParentIDIncludingDeleted(txCtx, post.ID, commentID)
			if err != nil {
				return err
			}
			for _, reply := range replies {
				if reply == nil || !reply.IsDeleted {
					continue
				}
				if err := uc.commentRepo.Restore(txCtx, reply.ID); err != nil {
					return err
				}
				increment++
			}
		}

		post.CommentCount += increment
		if err := uc.postRepo.Update(txCtx, post); err != nil {
			return err
		}
		if err := uc.postRepo.MarkHotnessDirty(txCtx, post.ID); err != nil {
			return err
		}

		uc.logger.Info("[CommunityModeration] Admin restored comment",
			zap.String("admin_user_id", adminUserID.String()),
			zap.String("action", "restore_comment"),
			zap.String("target_type", "comment"),
			zap.String("target_id", commentID.String()),
		)
		return nil
	}

	return uc.uow.Execute(ctx, restoreComment)
}

var _ uc_interfaces.IAdminCommunityModerationUseCase = (*AdminCommunityModerationUseCase)(nil)

func uniqueUUIDs(ids []uuid.UUID) []uuid.UUID {
	if len(ids) == 0 {
		return nil
	}

	seen := make(map[uuid.UUID]struct{}, len(ids))
	result := make([]uuid.UUID, 0, len(ids))
	for _, id := range ids {
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		result = append(result, id)
	}

	return result
}
