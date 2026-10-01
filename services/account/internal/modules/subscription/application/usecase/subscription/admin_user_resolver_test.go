package subscription

import (
	"context"
	"strings"
	"testing"
	"unicode/utf8"

	identity_contract "smart-wardrobe-be/services/account/contract/identity"
	identity_dto "smart-wardrobe-be/services/account/dto/identity"
	"smart-wardrobe-be/services/account/internal/modules/identity/domain/constants/userstatus"
	"smart-wardrobe-be/shared/domain/constants/roleslug"
	"smart-wardrobe-be/shared/domain/constants/shared/gender"

	"github.com/google/uuid"
)

// fakeIdentityBriefContract answers the two lookups the resolver makes, and
// records how many round trips it took.
//
// The call counter is the point of the fake: "one lookup for the whole list"
// (FR-050) is a behavioural requirement, not an optimisation, and a test that
// only checked the resolved values would pass even if the implementation
// looped over the identifiers.
type fakeIdentityBriefContract struct {
	users       []*identity_dto.UserBriefContractDTO
	byID        map[uuid.UUID]*identity_dto.UserBriefContractDTO
	lookupCalls int
	byIDCalls   int
	lastLookup  []string
	// 024 T143: the ids of the most recent batched lookup.
	lastByIDs []uuid.UUID
}

func (f *fakeIdentityBriefContract) GetByID(ctx context.Context, userID uuid.UUID) (*identity_dto.UserBriefContractDTO, error) {
	return f.byID[userID], nil
}

func (f *fakeIdentityBriefContract) GetByIDs(ctx context.Context, ids []uuid.UUID) ([]*identity_dto.UserBriefContractDTO, error) {
	f.byIDCalls++
	// 024 T143: recorded so a test can assert the whole UUID list goes in ONE
	// call rather than one per id.
	f.lastByIDs = ids
	out := make([]*identity_dto.UserBriefContractDTO, 0, len(ids))
	for _, id := range ids {
		if user, ok := f.byID[id]; ok {
			out = append(out, user)
		}
	}
	return out, nil
}

// GetByUsernamesOrEmails mirrors the real repository: it EXCLUDES soft-deleted
// accounts (FR-049). Before this the fake returned them, which is why a test could
// assert `user_soft_deleted` while production could never emit it.
func (f *fakeIdentityBriefContract) GetByUsernamesOrEmails(ctx context.Context, identifiers []string) ([]*identity_dto.UserBriefContractDTO, error) {
	f.lookupCalls++
	f.lastLookup = identifiers
	return f.matchByUsernameOrEmail(identifiers, false), nil
}

// GetByUsernamesOrEmailsIncludingDeleted is the admin grant path's lookup
// (024 T136): it keeps the deleted rows so the caller can tell them apart.
func (f *fakeIdentityBriefContract) GetByUsernamesOrEmailsIncludingDeleted(ctx context.Context, identifiers []string) ([]*identity_dto.UserBriefContractDTO, error) {
	f.lookupCalls++
	f.lastLookup = identifiers
	return f.matchByUsernameOrEmail(identifiers, true), nil
}

func (f *fakeIdentityBriefContract) matchByUsernameOrEmail(identifiers []string, includeDeleted bool) []*identity_dto.UserBriefContractDTO {
	out := make([]*identity_dto.UserBriefContractDTO, 0, len(identifiers))
	for _, raw := range identifiers {
		key := normalizeIdentifier(raw)
		for _, user := range f.users {
			if user.IsDeleted && !includeDeleted {
				continue
			}
			if normalizeIdentifier(user.Username) == key || normalizeIdentifier(user.Email) == key {
				out = append(out, user)
			}
		}
	}
	return out
}

func (f *fakeIdentityBriefContract) SearchUserBriefs(ctx context.Context, keyword string, limit int) ([]*identity_dto.UserBriefContractDTO, error) {
	return nil, nil
}

var _ identity_contract.IUserBriefContract = (*fakeIdentityBriefContract)(nil)
var _ identity_contract.AdminGrantLookup = (*fakeIdentityBriefContract)(nil)

func grantTestUser(username, email string, role roleslug.RoleSlug, deleted bool) *identity_dto.UserBriefContractDTO {
	return &identity_dto.UserBriefContractDTO{
		ID:        uuid.New(),
		Username:  username,
		Email:     email,
		FirstName: "Test",
		Gender:    gender.Gender(1),
		Status:    userstatus.Active,
		RoleSlug:  role,
		IsDeleted: deleted,
	}
}

// FR-045/FR-046: the system detects the identifier form. The administrator does
// not declare a type, and the same list may mix all three forms.
func TestAdminUserResolverDetectsIdentifierForm(t *testing.T) {
	byUUID := grantTestUser("byuuid", "byuuid@example.com", roleslug.User, false)
	byUsername := grantTestUser("MixedCaseName", "mixed@example.com", roleslug.User, false)
	byEmail := grantTestUser("plainuser", "Plain@Example.COM", roleslug.User, false)

	contract := &fakeIdentityBriefContract{
		users: []*identity_dto.UserBriefContractDTO{byUUID, byUsername, byEmail},
		byID:  map[uuid.UUID]*identity_dto.UserBriefContractDTO{byUUID.ID: byUUID},
	}
	resolver := NewAdminUserResolver(contract)

	resolved, err := resolver.Resolve(context.Background(), []string{
		byUUID.ID.String(),
		"mixedcasename",
		"plain@example.com",
	})
	if err != nil {
		t.Fatalf("Resolve lỗi: %v", err)
	}

	if got := resolved[normalizeIdentifier(byUUID.ID.String())]; got == nil || got.UserID != byUUID.ID || got.Kind != KindUUID {
		t.Errorf("UUID phải được phân giải đúng, có = %+v", got)
	}
	if got := resolved["mixedcasename"]; got == nil || got.UserID != byUsername.ID || got.Kind != KindUsername {
		t.Errorf("tên đăng nhập phải được phân giải, có = %+v", got)
	}
	// FR-047: matching ignores case, so an uppercase address resolves.
	if got := resolved["plain@example.com"]; got == nil || got.UserID != byEmail.ID || got.Kind != KindEmail {
		t.Errorf("email không phân biệt hoa thường phải phân giải được, có = %+v", got)
	}
}

// FR-050: the whole list is resolved in ONE lookup, not one per identifier. A
// batch of 500 that looped would mean 500 round trips to the identity module
// inside a single request.
func TestAdminUserResolverResolvesWholeListInOneLookup(t *testing.T) {
	var users []*identity_dto.UserBriefContractDTO
	identifiers := make([]string, 0, 20)
	for i := 0; i < 20; i++ {
		user := grantTestUser("user"+string(rune('a'+i)), "user"+string(rune('a'+i))+"@example.com", roleslug.User, false)
		users = append(users, user)
		identifiers = append(identifiers, user.Username)
	}

	contract := &fakeIdentityBriefContract{users: users, byID: map[uuid.UUID]*identity_dto.UserBriefContractDTO{}}
	resolver := NewAdminUserResolver(contract)

	if _, err := resolver.Resolve(context.Background(), identifiers); err != nil {
		t.Fatalf("Resolve lỗi: %v", err)
	}
	if contract.lookupCalls != 1 {
		t.Errorf("phải phân giải cả danh sách trong 1 lần tra cứu, thực tế = %d", contract.lookupCalls)
	}
	if contract.byIDCalls != 0 {
		t.Errorf("không có UUID nào thì không được gọi tra cứu theo ID, thực tế = %d lần", contract.byIDCalls)
	}
}

// FR-048: one identifier matching two accounts is REFUSED, never resolved to
// whichever row came back first. Granting a paid plan to the wrong account is
// not something the administrator can discover afterwards.
func TestAdminUserResolverRefusesAmbiguousIdentifier(t *testing.T) {
	// A username that is also somebody else's email. The schema has no unique
	// index covering both columns, so it is theoretically possible.
	first := grantTestUser("shared", "first@example.com", roleslug.User, false)
	second := grantTestUser("other", "shared", roleslug.User, false)

	contract := &fakeIdentityBriefContract{
		users: []*identity_dto.UserBriefContractDTO{first, second},
		byID:  map[uuid.UUID]*identity_dto.UserBriefContractDTO{},
	}
	resolver := NewAdminUserResolver(contract)

	resolved, err := resolver.Resolve(context.Background(), []string{"shared"})
	if err != nil {
		t.Fatalf("Resolve lỗi: %v", err)
	}

	entry := resolved["shared"]
	if entry == nil {
		t.Fatal("mã định danh mơ hồ phải xuất hiện trong kết quả để báo lỗi")
	}
	if !IsAmbiguous(entry) {
		t.Errorf("mã định danh khớp nhiều tài khoản phải được đánh dấu mơ hồ, có = %+v", entry)
	}
	if entry.UserID != uuid.Nil {
		t.Errorf("không được chọn bừa một tài khoản khi mơ hồ, có = %s", entry.UserID)
	}
}

// FR-057: an identifier matching nothing is "not found", and it must be
// DISTINGUISHABLE from the ambiguous case — one is fixed by correcting a typo,
// the other by being more specific.
func TestAdminUserResolverDistinguishesNotFoundFromAmbiguous(t *testing.T) {
	contract := &fakeIdentityBriefContract{users: nil, byID: map[uuid.UUID]*identity_dto.UserBriefContractDTO{}}
	resolver := NewAdminUserResolver(contract)

	resolved, err := resolver.Resolve(context.Background(), []string{"nobody-here"})
	if err != nil {
		t.Fatalf("Resolve lỗi: %v", err)
	}
	if _, present := resolved["nobody-here"]; present {
		t.Error("mã định danh không khớp tài khoản nào không được gán tài khoản")
	}
}

// FR-049: a soft-deleted account is RETURNED with its flag, so the caller can
// say "this account was deleted" instead of "check the spelling".
func TestAdminUserResolverReturnsSoftDeletedUser(t *testing.T) {
	deleted := grantTestUser("gone", "gone@example.com", roleslug.User, true)
	contract := &fakeIdentityBriefContract{
		users: []*identity_dto.UserBriefContractDTO{deleted},
		byID:  map[uuid.UUID]*identity_dto.UserBriefContractDTO{},
	}
	resolver := NewAdminUserResolver(contract)

	resolved, err := resolver.Resolve(context.Background(), []string{"gone@example.com"})
	if err != nil {
		t.Fatalf("Resolve lỗi: %v", err)
	}
	entry := resolved["gone@example.com"]
	if entry == nil || entry.User == nil {
		t.Fatal("tài khoản xoá mềm phải được trả về kèm cờ, không bị lọc mất")
	}
	if !entry.User.IsDeleted {
		t.Error("cờ IsDeleted phải được mang tới nơi gọi")
	}
}

// FR-056: the role travels on the contract, because the subscription module does
// not own identity.users and could not otherwise tell an admin from a customer.
func TestAdminUserResolverExposesRoleForAdminRefusal(t *testing.T) {
	admin := grantTestUser("root", "root@example.com", roleslug.Admin, false)
	contract := &fakeIdentityBriefContract{
		users: []*identity_dto.UserBriefContractDTO{admin},
		byID:  map[uuid.UUID]*identity_dto.UserBriefContractDTO{},
	}
	resolver := NewAdminUserResolver(contract)

	resolved, err := resolver.Resolve(context.Background(), []string{"root"})
	if err != nil {
		t.Fatalf("Resolve lỗi: %v", err)
	}
	if !IsAdministratorTarget(resolved["root"].User) {
		t.Error("tài khoản quản trị viên phải bị nhận ra là mục tiêu không được cấp gói")
	}
}

// A blank or whitespace identifier never reaches the database, and a repeated
// identifier is only resolved once.
func TestAdminUserResolverSkipsBlanksAndDuplicates(t *testing.T) {
	user := grantTestUser("solo", "solo@example.com", roleslug.User, false)
	contract := &fakeIdentityBriefContract{
		users: []*identity_dto.UserBriefContractDTO{user},
		byID:  map[uuid.UUID]*identity_dto.UserBriefContractDTO{},
	}
	resolver := NewAdminUserResolver(contract)

	resolved, err := resolver.Resolve(context.Background(), []string{"", "   ", "solo", "SOLO"})
	if err != nil {
		t.Fatalf("Resolve lỗi: %v", err)
	}
	if len(resolved) != 1 {
		t.Errorf("chỉ một mã định danh hợp lệ được giữ lại, thực tế = %d", len(resolved))
	}
	for _, blank := range []string{"", "   "} {
		if _, present := resolved[normalizeIdentifier(blank)]; present && normalizeIdentifier(blank) != "" {
			t.Errorf("mã định danh trống không được phân giải: %q", blank)
		}
	}
}

// FR-062: an identifier over 255 characters must be refused as a FORMAT problem,
// never looked up.
//
// The "never looked up" half is the whole point. A 50,000-character string
// passed to `LOWER(username) = $1` cannot use a b-tree index, so it degrades to
// a full table scan - and it would do that for every other row in the same
// request too.
func TestAdminUserResolverRejectsOverlongIdentifierWithoutQuerying(t *testing.T) {
	user := grantTestUser("normal", "normal@example.com", roleslug.User, false)
	contract := &fakeIdentityBriefContract{
		users: []*identity_dto.UserBriefContractDTO{user},
		byID:  map[uuid.UUID]*identity_dto.UserBriefContractDTO{},
	}
	resolver := NewAdminUserResolver(contract)

	overlong := strings.Repeat("a", MaxIdentifierLength+1)
	resolved, err := resolver.Resolve(context.Background(), []string{overlong, "normal@example.com"})
	if err != nil {
		t.Fatalf("Resolve lỗi: %v", err)
	}

	// The valid identifier in the SAME request still resolved, which is the
	// point of refusing per item rather than failing the request.
	if got := resolved["normal@example.com"]; got == nil || got.User == nil {
		t.Error("mã định danh hợp lệ phải vẫn được phân giải")
	}

	// Exactly ONE lookup: the over-long identifier never reached the database.
	if contract.lookupCalls != 1 {
		t.Errorf("mã định danh quá dài không được gửi xuống cơ sở dữ liệu, có = %d lần tra cứu", contract.lookupCalls)
	}
	for _, queried := range contract.lastLookup {
		if ExceedsIdentifierLength(queried) {
			t.Errorf("đã truy vấn một chuỗi quá dài (%d ký tự)", utf8.RuneCountInString(queried))
		}
	}
}

// The limit is a CHARACTER count. A Vietnamese name of 200 runes is over 255
// BYTES and well under 255 characters, and must still be accepted - otherwise
// the guard would reject real accounts while letting ASCII abuse through.
func TestMaxIdentifierLengthIsCountedInRunesNotBytes(t *testing.T) {
	vietnamese := strings.Repeat("á", 200) // 200 runes, 400 bytes
	if len(vietnamese) <= MaxIdentifierLength {
		t.Fatalf("bộ kiểm thử cần chuỗi nhiều byte hơn giới hạn, có = %d byte", len(vietnamese))
	}
	if ExceedsIdentifierLength(vietnamese) {
		t.Error("giới hạn phải tính theo ký tự, không phải theo byte")
	}
	// 256 ASCII characters IS over the limit.
	if !ExceedsIdentifierLength(strings.Repeat("a", MaxIdentifierLength+1)) {
		t.Error("256 ký tự ASCII phải bị từ chối")
	}
	// Exactly 255 is allowed - the limit is inclusive.
	if ExceedsIdentifierLength(strings.Repeat("a", MaxIdentifierLength)) {
		t.Error("255 ký tự phải được chấp nhận, giới hạn là bao hàm")
	}
}
