package interaction

import (
	"context"
	"strings"
	"unicode/utf8"

	identity_dto "smart-wardrobe-be/services/account/dto/identity"
	"smart-wardrobe-be/services/commerce/internal/modules/community/application/dto"
	communityerrors "smart-wardrobe-be/services/commerce/internal/modules/community/application/errors"
	"smart-wardrobe-be/services/commerce/internal/modules/community/application/mapper"
	"smart-wardrobe-be/services/commerce/internal/modules/community/domain/constants/poststatus"
	"smart-wardrobe-be/services/commerce/internal/modules/community/domain/entities"

	"github.com/google/uuid"
)

func (uc *InteractionUseCase) AddComment(ctx context.Context, userID uuid.UUID, publicID string, input dto.AddCommentReq) (*dto.CommentRes, error) {
	if err := uc.checkRateLimit(ctx, "comment", userID, uc.cfg.Community.CommentRateLimitPerHour); err != nil {
		return nil, err
	}
	post, err := uc.getPublishedPost(ctx, publicID)
	if err != nil {
		return nil, err
	}

	content := strings.TrimSpace(input.Content)
	if utf8.RuneCountInString(content) > uc.cfg.Community.MaxCommentLength {
		return nil, communityerrors.ErrCommentTooLong(uc.cfg.Community.MaxCommentLength)
	}

	parentID, rootID, err := uc.resolveCommentParent(ctx, post.ID, input.ParentCommentID)
	if err != nil {
		return nil, err
	}

	comment := &entities.PostComment{
		PostID:          post.ID,
		UserID:          userID,
		ParentCommentID: parentID,
		RootCommentID:   rootID,
		Content:         content,
		Status:          poststatus.Published,
	}
	comment.ID = uuid.New()

	if err := uc.uow.Execute(ctx, func(txCtx context.Context) error {
		if err := uc.commentRepo.Create(txCtx, comment); err != nil {
			return err
		}
		return uc.postRepo.IncrementCommentCount(txCtx, post.ID, 1)
	}); err != nil {
		return nil, err
	}

	return mapper.MapToCommentRes(comment, uc.fetchAuthor(ctx, userID), 0), nil
}

func (uc *InteractionUseCase) UpdateComment(ctx context.Context, userID uuid.UUID, publicID string, commentID uuid.UUID, content string) (*dto.CommentRes, error) {
	post, err := uc.getPublishedPost(ctx, publicID)
	if err != nil {
		return nil, err
	}
	comment, err := uc.getActiveComment(ctx, post.ID, commentID)
	if err != nil {
		return nil, err
	}
	if comment.UserID != userID {
		return nil, communityerrors.ErrNotCommentOwner()
	}

	content = strings.TrimSpace(content)
	if utf8.RuneCountInString(content) > uc.cfg.Community.MaxCommentLength {
		return nil, communityerrors.ErrCommentTooLong(uc.cfg.Community.MaxCommentLength)
	}
	comment.Content = content
	if err := uc.commentRepo.Update(ctx, comment); err != nil {
		return nil, err
	}
	return mapper.MapToCommentRes(comment, uc.fetchAuthor(ctx, userID), 0), nil
}

func (uc *InteractionUseCase) DeleteComment(ctx context.Context, userID uuid.UUID, publicID string, commentID uuid.UUID) error {
	post, err := uc.getPublishedPost(ctx, publicID)
	if err != nil {
		return err
	}
	comment, err := uc.getActiveComment(ctx, post.ID, commentID)
	if err != nil {
		return err
	}
	if comment.UserID != userID {
		return communityerrors.ErrNotCommentOwner()
	}

	comment.Status = poststatus.Deleted
	return uc.uow.Execute(ctx, func(txCtx context.Context) error {
		if err := uc.commentRepo.Update(txCtx, comment); err != nil {
			return err
		}
		return uc.postRepo.IncrementCommentCount(txCtx, post.ID, -1)
	})
}

func (uc *InteractionUseCase) getActiveComment(ctx context.Context, postID, commentID uuid.UUID) (*entities.PostComment, error) {
	comment, err := uc.commentRepo.GetByID(ctx, commentID)
	if err != nil {
		return nil, err
	}
	if comment == nil || comment.PostID != postID || comment.Status != poststatus.Published {
		return nil, communityerrors.ErrCommentNotFound()
	}
	return comment, nil
}

// resolveCommentParent enforces the 2-level rule: a reply to a level-1 comment
// is attached to its root, and root_comment_id always points at the root.
func (uc *InteractionUseCase) resolveCommentParent(ctx context.Context, postID uuid.UUID, parentCommentID *uuid.UUID) (*uuid.UUID, *uuid.UUID, error) {
	if parentCommentID == nil {
		return nil, nil, nil
	}
	parent, err := uc.commentRepo.GetByID(ctx, *parentCommentID)
	if err != nil {
		return nil, nil, err
	}
	if parent == nil || parent.PostID != postID || parent.Status != poststatus.Published {
		return nil, nil, communityerrors.ErrParentCommentInvalid()
	}
	rootID := parent.ID
	if parent.ParentCommentID != nil {
		if parent.RootCommentID != nil {
			rootID = *parent.RootCommentID
		} else {
			rootID = *parent.ParentCommentID
		}
	}
	return &rootID, &rootID, nil
}

func (uc *InteractionUseCase) fetchAuthor(ctx context.Context, userID uuid.UUID) *identity_dto.UserBriefContractDTO {
	if uc.userBrief == nil {
		return nil
	}
	briefs, err := uc.userBrief.GetByIDs(ctx, []uuid.UUID{userID})
	if err != nil || len(briefs) == 0 {
		return nil
	}
	return briefs[0]
}

func (uc *InteractionUseCase) fetchAuthors(ctx context.Context, userIDs []uuid.UUID) map[uuid.UUID]*identity_dto.UserBriefContractDTO {
	result := make(map[uuid.UUID]*identity_dto.UserBriefContractDTO, len(userIDs))
	if len(userIDs) == 0 || uc.userBrief == nil {
		return result
	}
	briefs, err := uc.userBrief.GetByIDs(ctx, dedupeUUIDs(userIDs))
	if err != nil {
		return result
	}
	for _, b := range briefs {
		result[b.ID] = b
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
