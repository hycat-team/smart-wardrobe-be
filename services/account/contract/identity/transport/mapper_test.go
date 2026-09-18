package transport_test

import (
	"testing"
	"time"

	identitytransport "smart-wardrobe-be/services/account/contract/identity/transport"
	identitydto "smart-wardrobe-be/services/account/dto/identity"
	"smart-wardrobe-be/services/account/internal/modules/identity/domain/constants/userstatus"
	"smart-wardrobe-be/shared/domain/constants/shared/gender"

	"github.com/google/uuid"
)

func TestUserBriefRoundTrip(t *testing.T) {
	avatar := "https://cdn.example.com/a.png"
	original := &identitydto.UserBriefContractDTO{
		ID:        uuid.New(),
		Username:  "alice",
		Email:     "alice@example.com",
		FirstName: "Alice",
		LastName:  "Nguyen",
		Gender:    gender.Female,
		Status:    userstatus.Active,
		AvatarUrl: &avatar,
	}

	wire := identitytransport.ContractDTOToUserBrief(original)
	back, err := identitytransport.UserBriefToDTO(wire)
	if err != nil {
		t.Fatalf("from proto: %v", err)
	}

	if back.ID != original.ID || back.Username != original.Username || back.Email != original.Email {
		t.Fatalf("identity fields mismatch: %+v", back)
	}
	if back.Gender != original.Gender || back.Status != original.Status {
		t.Fatalf("enum fields mismatch: gender=%v status=%v", back.Gender, back.Status)
	}
	if back.AvatarUrl == nil || *back.AvatarUrl != avatar {
		t.Fatalf("avatar mismatch: %+v", back.AvatarUrl)
	}
}

func TestUserAdminStatsRoundTrip(t *testing.T) {
	original := &identitydto.UserAdminStatsDTO{
		SnapshotDate:   time.Date(2026, 9, 17, 0, 0, 0, 0, time.UTC),
		TotalUsers:     120,
		NewUsers:       7,
		ActiveUsersDau: 42,
	}

	wire := identitytransport.UserAdminStatsToProto(original)
	back := identitytransport.UserAdminStatsToDTO(wire)

	if back.TotalUsers != original.TotalUsers || back.NewUsers != original.NewUsers || back.ActiveUsersDau != original.ActiveUsersDau {
		t.Fatalf("stats mismatch: %+v", back)
	}
	if !back.SnapshotDate.Equal(original.SnapshotDate) {
		t.Fatalf("snapshot date mismatch: %v", back.SnapshotDate)
	}
}
