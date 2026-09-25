package admin

import (
	"context"

	identitycontract "smart-wardrobe-be/services/account/contract/identity"
	admin_dto "smart-wardrobe-be/services/account/dto/admin"
	identity_dto "smart-wardrobe-be/services/account/dto/identity"
	"smart-wardrobe-be/services/commerce/internal/modules/community/application/dto"
	communityerrors "smart-wardrobe-be/services/commerce/internal/modules/community/application/errors"
	uc_interfaces "smart-wardrobe-be/services/commerce/internal/modules/community/application/interface/usecase"
	"smart-wardrobe-be/services/commerce/internal/modules/community/application/mapper"
	"smart-wardrobe-be/services/commerce/internal/modules/community/domain/constants/hiddenreason"
	"smart-wardrobe-be/services/commerce/internal/modules/community/domain/constants/poststatus"
	"smart-wardrobe-be/services/commerce/internal/modules/community/domain/entities"
	"smart-wardrobe-be/services/commerce/internal/modules/community/domain/repositories"
	"smart-wardrobe-be/shared/application/constants/eventconstants"
	shared_dto "smart-wardrobe-be/shared/application/dto"
	shared_event "smart-wardrobe-be/shared/application/event"
	"smart-wardrobe-be/shared/config"
	"smart-wardrobe-be/shared/pkg/logger"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

const (
	targetTypePost    = "community_post"
	targetTypeComment = "community_comment"
)

type AdminUseCase struct {
	cfg            *config.Config
	logger         logger.Interface
	postRepo       repositories.IPostRepository
	commentRepo    repositories.IPostCommentRepository
	userBrief      identitycontract.IUserBriefContract
	eventPublisher shared_event.IEventPublisher
}

func NewAdminUseCase(
	cfg *config.Config,
	log logger.Interface,
	postRepo repositories.IPostRepository,
	commentRepo repositories.IPostCommentRepository,
	userBrief identitycontract.IUserBriefContract,
	eventPublisher shared_event.IEventPublisher,
) uc_interfaces.IAdminUseCase {
	return &AdminUseCase{
		cfg:            cfg,
		logger:         log,
		postRepo:       postRepo,
		commentRepo:    commentRepo,
		userBrief:      userBrief,
		eventPublisher: eventPublisher,
	}
}

func (uc *AdminUseCase) ListPosts(ctx context.Context, query dto.GetAdminPostsQueryReq) (*dto.AdminPostListRes, error) {
	filter := repositories.AdminPostFilter{
		Keyword:            query.Q,
		ContentPrefixChars: uc.cfg.Community.SearchContentPrefixChars,
		Page:               query.Page,
		Limit:              query.Limit,
	}
	if query.Status != "" {
		status := poststatus.PostStatus(query.Status)
		if status.IsValid() {
			filter.Status = &status
		}
	}
	result, err := uc.postRepo.AdminList(ctx, filter)
	if err != nil {
		return nil, err
	}
	authors := uc.fetchAuthors(ctx, collectPostAuthorIDs(result.Items))
	items := make([]*dto.AdminPostRes, 0, len(result.Items))
	for _, post := range result.Items {
		items = append(items, mapper.MapToAdminPostRes(post, authors[post.UserID]))
	}
	return &dto.AdminPostListRes{
		Items:    items,
		Metadata: shared_dto.BuildPaginationMetadata(query.PaginationQuery, result.TotalCount),
	}, nil
}

func collectPostAuthorIDs(posts []*entities.Post) []uuid.UUID {
	ids := make([]uuid.UUID, 0, len(posts))
	for _, post := range posts {
		ids = append(ids, post.UserID)
	}
	return ids
}

func (uc *AdminUseCase) fetchAuthors(ctx context.Context, userIDs []uuid.UUID) map[uuid.UUID]*identity_dto.UserBriefContractDTO {
	result := make(map[uuid.UUID]*identity_dto.UserBriefContractDTO, len(userIDs))
	if len(userIDs) == 0 || uc.userBrief == nil {
		return result
	}
	briefs, err := uc.userBrief.GetByIDs(ctx, dedupeUUIDs(userIDs))
	if err != nil {
		uc.logger.Warn("failed to enrich admin moderation authors", zap.Error(err))
		return result
	}
	for _, brief := range briefs {
		result[brief.ID] = brief
	}
	return result
}

func dedupeUUIDs(ids []uuid.UUID) []uuid.UUID {
	seen := make(map[uuid.UUID]struct{}, len(ids))
	out := make([]uuid.UUID, 0, len(ids))
	for _, id := range ids {
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

func (uc *AdminUseCase) HidePost(ctx context.Context, adminID, postID uuid.UUID) error {
	post, err := uc.postRepo.GetByID(ctx, postID)
	if err != nil {
		return err
	}
	if post == nil {
		return communityerrors.ErrPostNotFound()
	}
	reason := hiddenreason.Admin
	if err := uc.postRepo.UpdateModerationStatus(ctx, postID, poststatus.Hidden, &adminID, &reason); err != nil {
		return err
	}
	uc.audit(ctx, adminID, "community_post_hide", targetTypePost, postID)
	return nil
}

func (uc *AdminUseCase) RestorePost(ctx context.Context, adminID, postID uuid.UUID) error {
	post, err := uc.postRepo.GetByID(ctx, postID)
	if err != nil {
		return err
	}
	if post == nil {
		return communityerrors.ErrPostNotFound()
	}
	if post.Status == poststatus.Deleted && post.ModeratedBy == nil {
		return communityerrors.ErrCannotRestoreAuthorDeleted()
	}
	if err := uc.postRepo.UpdateModerationStatus(ctx, postID, poststatus.Published, nil, nil); err != nil {
		return err
	}
	uc.audit(ctx, adminID, "community_post_restore", targetTypePost, postID)
	return nil
}

func (uc *AdminUseCase) DeletePost(ctx context.Context, adminID, postID uuid.UUID) error {
	post, err := uc.postRepo.GetByID(ctx, postID)
	if err != nil {
		return err
	}
	if post == nil {
		return communityerrors.ErrPostNotFound()
	}
	if err := uc.postRepo.UpdateModerationStatus(ctx, postID, poststatus.Deleted, &adminID, nil); err != nil {
		return err
	}
	uc.audit(ctx, adminID, "community_post_delete", targetTypePost, postID)
	return nil
}

func (uc *AdminUseCase) ListComments(ctx context.Context, query dto.GetAdminCommentsQueryReq) (*dto.AdminCommentListRes, error) {
	result, err := uc.commentRepo.AdminList(ctx, repositories.AdminCommentFilter{
		Keyword: query.Q,
		Page:    query.Page,
		Limit:   query.Limit,
	})
	if err != nil {
		return nil, err
	}
	items := make([]*dto.AdminCommentRes, 0, len(result.Items))
	userIDs := make([]uuid.UUID, 0, len(result.Items))
	for _, comment := range result.Items {
		userIDs = append(userIDs, comment.UserID)
	}
	authors := uc.fetchAuthors(ctx, userIDs)
	for _, comment := range result.Items {
		items = append(items, mapper.MapToAdminCommentRes(comment, authors[comment.UserID]))
	}
	return &dto.AdminCommentListRes{
		Items:    items,
		Metadata: shared_dto.BuildPaginationMetadata(query.PaginationQuery, result.TotalCount),
	}, nil
}

func (uc *AdminUseCase) HideComment(ctx context.Context, adminID, commentID uuid.UUID) error {
	return uc.setCommentStatus(ctx, adminID, commentID, poststatus.Hidden, "community_comment_hide")
}

func (uc *AdminUseCase) DeleteComment(ctx context.Context, adminID, commentID uuid.UUID) error {
	return uc.setCommentStatus(ctx, adminID, commentID, poststatus.Deleted, "community_comment_delete")
}

func (uc *AdminUseCase) RestoreComment(ctx context.Context, adminID, commentID uuid.UUID) error {
	comment, err := uc.commentRepo.GetByID(ctx, commentID)
	if err != nil {
		return err
	}
	if comment == nil {
		return communityerrors.ErrCommentNotFound()
	}
	if comment.Status == poststatus.Deleted && comment.ModeratedBy == nil {
		return communityerrors.ErrCannotRestoreAuthorDeleted()
	}
	comment.Status = poststatus.Published
	comment.ModeratedBy = nil
	if err := uc.commentRepo.Update(ctx, comment); err != nil {
		return err
	}
	uc.cascadeCommentStatus(ctx, comment, poststatus.Published, nil)
	uc.audit(ctx, adminID, "community_comment_restore", targetTypeComment, commentID)
	return nil
}

func (uc *AdminUseCase) setCommentStatus(ctx context.Context, adminID, commentID uuid.UUID, status poststatus.PostStatus, action string) error {
	comment, err := uc.commentRepo.GetByID(ctx, commentID)
	if err != nil {
		return err
	}
	if comment == nil {
		return communityerrors.ErrCommentNotFound()
	}
	comment.Status = status
	comment.ModeratedBy = &adminID
	if err := uc.commentRepo.Update(ctx, comment); err != nil {
		return err
	}
	uc.cascadeCommentStatus(ctx, comment, status, &adminID)
	uc.audit(ctx, adminID, action, targetTypeComment, commentID)
	return nil
}

// cascadeCommentStatus applies the same status to the replies of a root comment.
func (uc *AdminUseCase) cascadeCommentStatus(ctx context.Context, comment *entities.PostComment, status poststatus.PostStatus, moderatedBy *uuid.UUID) {
	if comment.ParentCommentID != nil {
		return
	}
	if _, err := uc.commentRepo.UpdateStatusByRoot(ctx, comment.ID, status, moderatedBy); err != nil {
		uc.logger.Warn("failed to cascade comment status", zap.String("comment_id", comment.ID.String()), zap.Error(err))
	}
}

func (uc *AdminUseCase) audit(ctx context.Context, adminID uuid.UUID, action, targetType string, targetID uuid.UUID) {
	if uc.eventPublisher == nil {
		return
	}
	payload := admin_dto.RecordAuditLogInput{
		ActorUserID: adminID,
		Action:      action,
		TargetType:  targetType,
		TargetID:    &targetID,
	}
	if err := uc.eventPublisher.Publish(ctx, eventconstants.TopicAuditLog, payload); err != nil {
		uc.logger.Warn("failed to publish community audit log", zap.String("action", action), zap.Error(err))
	}
}
