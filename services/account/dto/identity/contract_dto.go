package dto

import (
	"time"

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
	// 025 T019: the campaign module owns the rule "an account created by an
	// administrator is not a campaign target", but it does not own
	// identity.users. Carrying the flag on the contract is what lets it enforce
	// that rule from the catch-up sweep, which reads the database and never sees
	// the creation event.
	//
	// It is deliberately NOT derived from IsDeleted or from email verification:
	// the table records nothing about who created the row, and an administrator
	// may create an account that is already verified. One fact is not evidence
	// about the other (research.md R4).
	CreatedByAdmin bool `json:"createdByAdmin"`
	// 025 T019: the campaign priority rule compares this against the frozen
	// watermark (FR-031). Carried so the campaign module never reads
	// identity.users.
	CreatedAt time.Time `json:"createdAt"`
}
