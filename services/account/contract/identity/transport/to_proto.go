package transport

import (
	identityv1 "smart-wardrobe-be/contracts/identity/v1"
	identitydto "smart-wardrobe-be/services/account/dto/identity"

	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// ContractDTOToUserBrief converts the contract DTO into its wire form.
func ContractDTOToUserBrief(d *identitydto.UserBriefContractDTO) *identityv1.UserBrief {
	if d == nil {
		return nil
	}
	out := &identityv1.UserBrief{
		Id:        d.ID.String(),
		Username:  d.Username,
		Email:     d.Email,
		FirstName: d.FirstName,
		LastName:  d.LastName,
		Gender:    int32(d.Gender),
		Status:    int32(d.Status),
	}
	if d.AvatarUrl != nil {
		out.AvatarUrl = *d.AvatarUrl
	}
	return out
}

// ContractDTOsToUserBriefs converts contract DTOs into wire user briefs.
func ContractDTOsToUserBriefs(ds []*identitydto.UserBriefContractDTO) []*identityv1.UserBrief {
	out := make([]*identityv1.UserBrief, 0, len(ds))
	for _, d := range ds {
		out = append(out, ContractDTOToUserBrief(d))
	}
	return out
}

// UserAdminStatsToProto converts the contract DTO into its wire form.
func UserAdminStatsToProto(d *identitydto.UserAdminStatsDTO) *identityv1.UserAdminStats {
	if d == nil {
		return nil
	}
	return &identityv1.UserAdminStats{
		SnapshotDate:   timestamppb.New(d.SnapshotDate),
		TotalUsers:     int32(d.TotalUsers),
		NewUsers:       int32(d.NewUsers),
		ActiveUsersDau: int32(d.ActiveUsersDau),
	}
}

// UserIDsToProtoStrings renders user ids for the wire.
func UserIDsToProtoStrings(ids []uuid.UUID) []string {
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		out = append(out, id.String())
	}
	return out
}
