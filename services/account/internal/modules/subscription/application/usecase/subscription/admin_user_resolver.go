package subscription

import (
	"context"
	"strings"
	"unicode/utf8"

	identity_contract "smart-wardrobe-be/services/account/contract/identity"
	identity_dto "smart-wardrobe-be/services/account/dto/identity"
	"smart-wardrobe-be/services/account/internal/modules/subscription/application/mapper"
	"smart-wardrobe-be/shared/domain/constants/roleslug"

	"github.com/google/uuid"
)

// ResolvedUser is the outcome of resolving one administrator-supplied
// identifier.
type ResolvedUser struct {
	UserID uuid.UUID
	User   *identity_dto.UserBriefContractDTO
	// Kind records which form the administrator supplied. It is detected, never
	// declared: the contract is one endpoint and the administrator must not have
	// to choose a type (FR-046).
	Kind IdentifierKind
}

// IdentifierKind is the detected form of an identifier.
type IdentifierKind string

const (
	KindUUID     IdentifierKind = "uuid"
	KindUsername IdentifierKind = "username"
	KindEmail    IdentifierKind = "email"
)

// AdminUserResolver turns the identifiers an administrator types into accounts.
//
// It resolves the WHOLE list in one call rather than one call per item. A batch
// of 500 would otherwise be 500 round trips to the identity module — and with a
// gRPC adapter, 500 network hops inside a request that has a 60-second budget.
//
// Ambiguity is refused rather than resolved (FR-048). If one string matches two
// accounts, picking either one could hand a paid plan to the wrong person, and
// the administrator has no way to find out afterwards.
type AdminUserResolver struct {
	// 024 T136: the NARROW grant view, not `IUserBriefContract`. Only this flow
	// needs to see soft-deleted accounts, and the full contract is served over
	// gRPC where they have no business travelling.
	userBriefContract identity_contract.AdminGrantLookup
}

func NewAdminUserResolver(userBriefContract identity_contract.AdminGrantLookup) *AdminUserResolver {
	return &AdminUserResolver{userBriefContract: userBriefContract}
}

// Resolve maps every identifier to an account, keyed by the lowercased
// identifier the caller supplied.
//
// The map is keyed on the lowercased form because username and email matching is
// case-insensitive (FR-047), so "KHACHHANG@EXAMPLE.COM" and
// "khachhang@example.com" are the same account and must land on the same entry.
func (r *AdminUserResolver) Resolve(ctx context.Context, identifiers []string) (map[string]*ResolvedUser, error) {
	result := make(map[string]*ResolvedUser, len(identifiers))
	if len(identifiers) == 0 {
		return result, nil
	}

	// UUIDs are resolved locally; only username/email candidates go over the
	// contract. Sending a UUID to a username/email lookup would match nothing
	// and would make the administrator see "not found" for a perfectly valid id.
	lookup := make([]string, 0, len(identifiers))
	uuidByIdentifier := make(map[string]uuid.UUID, len(identifiers))
	for _, raw := range identifiers {
		key := normalizeIdentifier(raw)
		if key == "" {
			continue
		}
		// FR-062: over-long identifiers are dropped HERE, before any database
		// call, so they never reach an index scan. They are simply absent from
		// the result map, which makes the caller report them as not found - and
		// the reason code says the format was wrong, not that the account is
		// unknown.
		if ExceedsIdentifierLength(key) {
			continue
		}
		if _, seen := result[key]; seen {
			continue
		}
		if parsed, err := uuid.Parse(key); err == nil {
			uuidByIdentifier[key] = parsed
			continue
		}
		lookup = append(lookup, key)
	}

	if len(lookup) > 0 {
		users, err := r.userBriefContract.GetByUsernamesOrEmailsIncludingDeleted(ctx, lookup)
		if err != nil {
			return nil, err
		}
		// One account can arrive more than once: the list may repeat an
		// identifier, and a row matches under both its username and its email.
		// Those repeats are ONE account, not several. Counting them separately
		// would report every repeated identifier as ambiguous and refuse to grant
		// a perfectly clear account.
		unique := make([]*identity_dto.UserBriefContractDTO, 0, len(users))
		seenUsers := make(map[uuid.UUID]struct{}, len(users))
		for _, user := range users {
			if user == nil {
				continue
			}
			if _, dup := seenUsers[user.ID]; dup {
				continue
			}
			seenUsers[user.ID] = struct{}{}
			unique = append(unique, user)
		}
		// The reverse index is what makes ambiguity DETECTABLE. Reading the rows
		// directly would silently resolve a match to whichever row the database
		// happened to return first.
		matchesByIdentifier := make(map[string][]*identity_dto.UserBriefContractDTO, len(unique))
		for _, user := range unique {
			matchesByIdentifier[normalizeIdentifier(user.Username)] = append(matchesByIdentifier[normalizeIdentifier(user.Username)], user)
			if user.Email != "" {
				matchesByIdentifier[normalizeIdentifier(user.Email)] = append(matchesByIdentifier[normalizeIdentifier(user.Email)], user)
			}
		}
		for _, key := range lookup {
			matches := matchesByIdentifier[key]
			switch len(matches) {
			case 1:
				result[key] = &ResolvedUser{UserID: matches[0].ID, User: matches[0], Kind: kindOf(key)}
			case 0:
				// Absent from the map, which the caller reports as user_not_found.
			default:
				// Two or more DIFFERENT accounts match. Recorded without an
				// account so the caller reports identifier_ambiguous rather than
				// picking one.
				result[key] = &ResolvedUser{Kind: kindOf(key)}
			}
		}
	}

	if len(uuidByIdentifier) > 0 {
		// 024 T143: ONE call for every UUID in the list.
		//
		// This used to loop and call `GetByIDs` with a single id per iteration,
		// which made a 500-UUID batch 500 sequential round trips - exactly what the
		// type's own doc comment above promises never happens, and a real risk
		// against FR-076's 60-second budget for 500 items.
		ids := make([]uuid.UUID, 0, len(uuidByIdentifier))
		for _, id := range uuidByIdentifier {
			ids = append(ids, id)
		}
		users, err := r.userBriefContract.GetByIDs(ctx, ids)
		if err != nil {
			return nil, err
		}
		byID := make(map[uuid.UUID]*identity_dto.UserBriefContractDTO, len(users))
		for _, user := range users {
			if user != nil {
				byID[user.ID] = user
			}
		}
		for key, id := range uuidByIdentifier {
			if user := byID[id]; user != nil {
				result[key] = &ResolvedUser{UserID: user.ID, User: user, Kind: KindUUID}
				continue
			}
			// A UUID that resolves to nothing is still recorded, so the caller can
			// report it as not found rather than as unresolvable.
			result[key] = &ResolvedUser{UserID: id, Kind: KindUUID}
		}
	}

	return result, nil
}

// MaxIdentifierLength bounds an administrator-supplied identifier (024 FR-062).
// Aliased from the mapper package, which owns the constant so the limit and the
// message quoting it cannot drift apart.
const MaxIdentifierLength = mapper.MaxIdentifierLength

// IsAmbiguous reports whether an entry exists but carries no account, which is
// how a multi-account match is represented. Distinguishing it from "not found"
// matters: one is fixed by making the identifier more specific, the other by
// correcting a typo.
func IsAmbiguous(resolved *ResolvedUser) bool {
	return resolved != nil && resolved.User == nil
}

// ExceedsIdentifierLength reports whether an identifier is too long to look up.
//
// Measured in RUNES, not bytes. FR-062's limit is expressed in characters, and
// counting bytes would reject a 100-character Vietnamese name at roughly 200
// bytes - which is a perfectly valid account the administrator could not grant.
//
// The reason this must be checked before the lookup: a 50,000-character
// identifier handed to `LOWER(username) = $1` cannot use a b-tree index at all
// and degrades to a full scan of the users table, so one over-long string would
// slow every other row's lookup in the same request.
func ExceedsIdentifierLength(raw string) bool {
	return utf8.RuneCountInString(raw) > mapper.MaxIdentifierLength
}

// ExceedsPlanSlugLength reports whether a plan slug is too long to look up.
//
// Same guard as the identifier, for the same reason, on a narrower column: the
// slug goes to `WHERE slug = $1` and the column is 100 characters, so anything
// longer cannot use the index and turns one lookup into a table scan.
func ExceedsPlanSlugLength(raw string) bool {
	return utf8.RuneCountInString(raw) > mapper.MaxPlanSlugLength
}

func normalizeIdentifier(raw string) string {
	return strings.ToLower(strings.TrimSpace(raw))
}

func kindOf(key string) IdentifierKind {
	if strings.Contains(key, "@") {
		return KindEmail
	}
	return KindUsername
}

// IsAdministratorTarget reports whether the account is a system administrator,
// who must never be a grant target (FR-056). Granting a paid plan to an
// administrator account would be granting it to staff and charging the company.
func IsAdministratorTarget(user *identity_dto.UserBriefContractDTO) bool {
	return user != nil && user.RoleSlug == roleslug.Admin
}
