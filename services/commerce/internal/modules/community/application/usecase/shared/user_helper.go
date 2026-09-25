package shared

import (
	"context"
	"strings"

	identitycontract "smart-wardrobe-be/services/account/contract/identity"
	identity_dto "smart-wardrobe-be/services/account/dto/identity"
)

// ResolveUserByUsername looks up a single user brief by username (case-insensitive).
// Returns nil when no user matches.
func ResolveUserByUsername(ctx context.Context, contract identitycontract.IUserBriefContract, username string) (*identity_dto.UserBriefContractDTO, error) {
	username = strings.TrimSpace(username)
	if username == "" || contract == nil {
		return nil, nil
	}
	briefs, err := contract.GetByUsernamesOrEmails(ctx, []string{username})
	if err != nil {
		return nil, err
	}
	if len(briefs) == 0 {
		return nil, nil
	}
	return briefs[0], nil
}
