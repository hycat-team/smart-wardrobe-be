package dto

import (
	"smart-wardrobe-be/services/account/internal/modules/identity/domain/constants/userstatus"
	"smart-wardrobe-be/shared/domain/constants/roleslug"
	"smart-wardrobe-be/shared/domain/constants/shared/gender"

	"github.com/google/uuid"
)

type UserBriefContractDTO struct {
	ID        uuid.UUID             `json:"id"`
	Username  string                `json:"username"`
	Email     string                `json:"email,omitempty"`
	FirstName string                `json:"firstName"`
	LastName  string                `json:"lastName,omitempty"`
	Gender    gender.Gender         `json:"gender"`
	Status    userstatus.UserStatus `json:"status"`
	AvatarUrl *string               `json:"avatarUrl,omitempty"`
	// 024 FR-056: the subscription module owns the rule "an administrator must
	// not be a grant target", but it does not own identity.users. Carrying the
	// role on the contract is what lets it enforce that rule without reading
	// another module's table.
	RoleSlug roleslug.RoleSlug `json:"roleSlug,omitempty"`
	// 024 FR-049: a soft-deleted account is reported as deleted instead of
	// disappearing, so the administrator is not told to re-check a correct
	// address.
	IsDeleted bool `json:"isDeleted"`
}
