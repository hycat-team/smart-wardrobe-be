package usecase

import (
	"context"

	"smart-wardrobe-be/internal/modules/community/application/dto"
	uc_interfaces "smart-wardrobe-be/internal/modules/community/application/interface/usecase"
	"smart-wardrobe-be/internal/modules/community/application/mapper"
	"smart-wardrobe-be/internal/modules/community/domain/repositories"
	wardrobe_contract "smart-wardrobe-be/internal/modules/wardrobe/contract"
	"smart-wardrobe-be/internal/shared/application/constants/errorcode"
	"smart-wardrobe-be/internal/shared/domain/constants/itemcondition"
	"smart-wardrobe-be/internal/shared/domain/constants/postitemstatus"
	"smart-wardrobe-be/internal/shared/domain/constants/posttype"
	"smart-wardrobe-be/internal/shared/domain/entities"
	shared_repos "smart-wardrobe-be/internal/shared/domain/repositories"

	"github.com/google/uuid"
)

type PostUseCase struct {
	postRepo      repositories.IPostRepository
	postItemRepo  repositories.IPostItemRepository
	postMediaRepo repositories.IPostMediaRepository
	commentRepo   repositories.ICommentRepository
	wardrobeCtr   wardrobe_contract.IWardrobeContract
	uow           shared_repos.IUnitOfWork
}

func NewPostUseCase(
	postRepo repositories.IPostRepository,
	postItemRepo repositories.IPostItemRepository,
	postMediaRepo repositories.IPostMediaRepository,
	commentRepo repositories.ICommentRepository,
	wardrobeCtr wardrobe_contract.IWardrobeContract,
	uow shared_repos.IUnitOfWork,
) uc_interfaces.IPostUseCase {
	return &PostUseCase{
		postRepo:      postRepo,
		postItemRepo:  postItemRepo,
		postMediaRepo: postMediaRepo,
		commentRepo:   commentRepo,
		wardrobeCtr:   wardrobeCtr,
		uow:           uow,
	}
}

func (uc *PostUseCase) CreatePost(ctx context.Context, userID uuid.UUID, input dto.CreatePostReq) (*dto.PostRes, error) {
	if err := uc.wardrobeCtr.VerifyItemsForPost(ctx, userID, input.ItemIDs); err != nil {
		return nil, err
	}

	post := &entities.Post{
		UserID:   userID,
		PostType: posttype.PostType(input.PostType),
		Title:    input.Title,
		Content:  input.Content,
	}
	if input.ContactInfo != nil {
		post.ContactInfo = input.ContactInfo
	}
	if input.TotalPrice != nil {
		post.TotalPrice = *input.TotalPrice
	}

	createPost := func(txCtx context.Context) error {
		if err := uc.postRepo.Create(txCtx, post); err != nil {
			return err
		}

		postItems := make([]*entities.PostItem, 0, len(input.ItemIDs))
		for _, itemID := range input.ItemIDs {
			postItems = append(postItems, &entities.PostItem{
				PostID:        post.ID,
				ItemID:        itemID,
				Price:         post.TotalPrice,
				ItemCondition: itemcondition.Standard,
				Status:        postitemstatus.Available,
			})
		}
		for _, item := range postItems {
			if err := uc.postItemRepo.Create(txCtx, item); err != nil {
				return err
			}
		}

		mediaItems := make([]*entities.PostMedia, 0, len(input.Media))
		for _, item := range input.Media {
			mediaItems = append(mediaItems, &entities.PostMedia{
				PostID:    post.ID,
				MediaType: item.MediaType,
				MediaURL:  item.MediaURL,
				PublicID:  item.PublicID,
				SortOrder: item.SortOrder,
			})
		}
		if len(mediaItems) > 0 {
			if err := uc.postMediaRepo.BulkCreate(txCtx, mediaItems); err != nil {
				return err
			}
		}

		return nil
	}

	if err := uc.uow.Execute(ctx, createPost); err != nil {
		return nil, err
	}

	return uc.GetPostDetail(ctx, post.ID)
}

func (uc *PostUseCase) GetFeed(ctx context.Context) ([]*dto.PostRes, error) {
	posts, err := uc.postRepo.GetFeed(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]*dto.PostRes, len(posts))
	for i, item := range posts {
		result[i] = mapper.MapPost(item, nil, nil, nil)
	}
	return result, nil
}

func (uc *PostUseCase) GetPostDetail(ctx context.Context, postID uuid.UUID) (*dto.PostRes, error) {
	post, items, media, err := uc.postRepo.GetDetail(ctx, postID)
	if err != nil {
		return nil, err
	}
	if post == nil {
		return nil, errorcode.NewNotFound("Không tìm thấy bài đăng.")
	}

	comments, err := uc.commentRepo.GetByPostID(ctx, postID)
	if err != nil {
		return nil, err
	}

	return mapper.MapPost(post, items, media, comments), nil
}

func (uc *PostUseCase) DeletePost(ctx context.Context, userID uuid.UUID, postID uuid.UUID) error {
	post, err := uc.postRepo.GetByID(ctx, postID)
	if err != nil {
		return err
	}
	if post == nil || post.UserID != userID {
		return errorcode.NewNotFound("Không tìm thấy bài đăng.")
	}
	return uc.postRepo.Delete(ctx, postID)
}

func (uc *PostUseCase) RemovePostItems(ctx context.Context, userID uuid.UUID, postID uuid.UUID, postItemIDs []uuid.UUID) error {
	post, err := uc.postRepo.GetByID(ctx, postID)
	if err != nil {
		return err
	}
	if post == nil || post.UserID != userID {
		return errorcode.NewNotFound("Không tìm thấy bài đăng.")
	}
	return uc.postItemRepo.DeleteByPostAndIDs(ctx, postID, postItemIDs)
}

var _ uc_interfaces.IPostUseCase = (*PostUseCase)(nil)
