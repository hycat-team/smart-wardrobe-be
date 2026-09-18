// Package transport maps identity wire types (proto) to the owner's contract
// DTOs. It lives in the owner service but outside internal/ so the gRPC server
// (internal) and remote clients (contracts/.../client) reuse one mapping and
// the domain enum stays encapsulated (Constitution II/IV).
package transport

import (
	identityv1 "smart-wardrobe-be/contracts/identity/v1"
	identitydto "smart-wardrobe-be/services/account/dto/identity"
	"smart-wardrobe-be/services/account/internal/modules/identity/domain/constants/userstatus"
	"smart-wardrobe-be/shared/application/constants/apperror"
	"smart-wardrobe-be/shared/domain/constants/shared/gender"

	"github.com/google/uuid"
)

// UserBriefToDTO converts one wire user brief into the contract DTO.
func UserBriefToDTO(pb *identityv1.UserBrief) (*identitydto.UserBriefContractDTO, error) {
	if pb == nil {
		return nil, nil
	}
	id, err := uuid.Parse(pb.GetId())
	if err != nil {
		wrapErr := apperror.NewInternalError(apperror.GenericInternalMessage)
		wrapErr.Cause = err
		return nil, wrapErr
	}

	out := &identitydto.UserBriefContractDTO{
		ID:        id,
		Username:  pb.GetUsername(),
		Email:     pb.GetEmail(),
		FirstName: pb.GetFirstName(),
		LastName:  pb.GetLastName(),
		Gender:    gender.Gender(pb.GetGender()),
		Status:    userstatus.UserStatus(pb.GetStatus()),
	}
	if avatar := pb.GetAvatarUrl(); avatar != "" {
		out.AvatarUrl = &avatar
	}
	return out, nil
}

// UserBriefsToDTOs converts a wire user brief list into contract DTOs.
func UserBriefsToDTOs(pb []*identityv1.UserBrief) ([]*identitydto.UserBriefContractDTO, error) {
	out := make([]*identitydto.UserBriefContractDTO, 0, len(pb))
	for _, item := range pb {
		d, err := UserBriefToDTO(item)
		if err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, nil
}

// UserAdminStatsToDTO converts wire identity stats into the contract DTO.
func UserAdminStatsToDTO(pb *identityv1.UserAdminStats) *identitydto.UserAdminStatsDTO {
	if pb == nil {
		return nil
	}
	out := &identitydto.UserAdminStatsDTO{
		TotalUsers:     int(pb.GetTotalUsers()),
		NewUsers:       int(pb.GetNewUsers()),
		ActiveUsersDau: int(pb.GetActiveUsersDau()),
	}
	if ts := pb.GetSnapshotDate(); ts != nil {
		out.SnapshotDate = ts.AsTime()
	}
	return out
}

// UserIDsFromProto parses a wire user-id list, skipping malformed entries.
func UserIDsFromProto(ids []string) []uuid.UUID {
	out := make([]uuid.UUID, 0, len(ids))
	for _, raw := range ids {
		id, err := uuid.Parse(raw)
		if err != nil {
			continue
		}
		out = append(out, id)
	}
	return out
}
