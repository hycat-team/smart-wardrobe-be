package brand

import (
	"context"
	"errors"
	"net/http"
	"testing"

	identity_contract "smart-wardrobe-be/services/account/contract/identity"
	identity_dto "smart-wardrobe-be/services/account/dto/identity"
	"smart-wardrobe-be/services/commerce/internal/modules/brand/domain/constants/brandmemberstatus"
	"smart-wardrobe-be/services/commerce/internal/modules/brand/domain/entities"
	"smart-wardrobe-be/services/commerce/internal/modules/brand/domain/repositories"
	"smart-wardrobe-be/shared/application/constants/apperror"
	"smart-wardrobe-be/shared/domain/constants/brandmemberrole"

	"github.com/google/uuid"
)

type memberTestMemberRepo struct {
	repositories.IBrandMemberRepository
	members map[uuid.UUID]*entities.BrandMember
	deleted []uuid.UUID
}

func (m *memberTestMemberRepo) GetByID(ctx context.Context, id uuid.UUID) (*entities.BrandMember, error) {
	return m.members[id], nil
}

func (m *memberTestMemberRepo) GetByBrandID(ctx context.Context, brandID uuid.UUID) ([]*entities.BrandMember, error) {
	var list []*entities.BrandMember
	for _, member := range m.members {
		if member.BrandID == brandID {
			list = append(list, member)
		}
	}
	return list, nil
}
func (m *memberTestMemberRepo) Update(ctx context.Context, member *entities.BrandMember) error {
	m.members[member.ID] = member
	return nil
}

func (m *memberTestMemberRepo) Delete(ctx context.Context, id uuid.UUID) error {
	delete(m.members, id)
	m.deleted = append(m.deleted, id)
	return nil
}

type memberTestUserBrief struct {
	identity_contract.IUserBriefContract
	user *identity_dto.UserBriefContractDTO
}

func (m *memberTestUserBrief) GetByID(ctx context.Context, id uuid.UUID) (*identity_dto.UserBriefContractDTO, error) {
	return m.user, nil
}

func newMemberTestUseCase(member *entities.BrandMember) (*BrandUseCase, *memberTestMemberRepo) {
	repo := &memberTestMemberRepo{members: map[uuid.UUID]*entities.BrandMember{member.ID: member}}
	uc := &BrandUseCase{
		memberRepo: repo,
		userBriefContract: &memberTestUserBrief{user: &identity_dto.UserBriefContractDTO{
			ID:        member.UserID,
			Email:     "staff@example.com",
			FirstName: "Staff",
			LastName:  "Member",
		}},
	}
	return uc, repo
}

func memberWithRole(role brandmemberrole.BrandMemberRole, brandID uuid.UUID) *entities.BrandMember {
	member := &entities.BrandMember{
		BrandID: brandID,
		UserID:  uuid.New(),
		Role:    role,
		Status:  brandmemberstatus.Active,
	}
	member.ID = uuid.New()
	return member
}

func assertMemberErrorStatus(t *testing.T, err error, status int) {
	t.Helper()
	var appErr *apperror.Error
	if !errors.As(err, &appErr) {
		t.Fatalf("expected *apperror.Error, got %v", err)
	}
	if appErr.Status != status {
		t.Fatalf("expected status %d, got %d", status, appErr.Status)
	}
}

func TestUpdateBrandMemberStatus_Staff(t *testing.T) {
	brandID := uuid.New()
	member := memberWithRole(brandmemberrole.Staff, brandID)
	uc, _ := newMemberTestUseCase(member)

	res, err := uc.UpdateBrandMemberStatus(context.Background(), uuid.New(), brandID, member.ID, brandmemberstatus.Disabled)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if member.Status != brandmemberstatus.Disabled {
		t.Fatalf("expected disabled status, got %s", member.Status)
	}
	if res.UserEmail != "staff@example.com" || res.UserFullName != "Staff Member" {
		t.Fatalf("expected enriched user, got %+v", res)
	}
}

func TestUpdateBrandMemberStatus_RejectsOwner(t *testing.T) {
	brandID := uuid.New()
	member := memberWithRole(brandmemberrole.Owner, brandID)
	uc, _ := newMemberTestUseCase(member)

	_, err := uc.UpdateBrandMemberStatus(context.Background(), uuid.New(), brandID, member.ID, brandmemberstatus.Disabled)
	assertMemberErrorStatus(t, err, http.StatusForbidden)
}

func TestUpdateBrandMemberStatus_RejectsOtherBrand(t *testing.T) {
	member := memberWithRole(brandmemberrole.Staff, uuid.New())
	uc, _ := newMemberTestUseCase(member)

	_, err := uc.UpdateBrandMemberStatus(context.Background(), uuid.New(), uuid.New(), member.ID, brandmemberstatus.Disabled)
	assertMemberErrorStatus(t, err, http.StatusNotFound)
}

func TestRemoveBrandMember_Staff(t *testing.T) {
	brandID := uuid.New()
	member := memberWithRole(brandmemberrole.Staff, brandID)
	uc, repo := newMemberTestUseCase(member)

	if err := uc.RemoveBrandMember(context.Background(), uuid.New(), brandID, member.ID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(repo.deleted) != 1 || repo.deleted[0] != member.ID {
		t.Fatalf("expected member deletion, got %+v", repo.deleted)
	}
}

func TestRemoveBrandMember_RejectsOwner(t *testing.T) {
	brandID := uuid.New()
	member := memberWithRole(brandmemberrole.Owner, brandID)
	uc, repo := newMemberTestUseCase(member)

	err := uc.RemoveBrandMember(context.Background(), uuid.New(), brandID, member.ID)
	assertMemberErrorStatus(t, err, http.StatusForbidden)
	if len(repo.deleted) != 0 {
		t.Fatal("expected owner not to be deleted")
	}
}
