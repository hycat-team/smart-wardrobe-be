package campaign

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	sub_dto "smart-wardrobe-be/services/account/dto/subscription"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/entities"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/repositories"
	"smart-wardrobe-be/shared/application/constants/apperror"
	"smart-wardrobe-be/shared/domain/constants/adminaudit"

	"github.com/google/uuid"
)

// 026 T036 (FR-052..FR-055, FR-060, FR-061, FR-063) — `CampaignRegistryUseCase.Create`.
//
// WHAT A UNIT TEST CAN PROVE HERE AND WHAT IT CANNOT. Everything this file asserts is
// a DECISION: which error the operator is told, whether the row was written, and
// whether exactly one audit command was issued inside the transaction. It cannot prove
// that the row reached a disk, that the outbox is the transactional one, or that a
// unique violation arrives as `ErrCampaignCodeExists` — those are the database's, and
// they are covered by the repository integration test and by quickstart V1.
//
// THE ROLLBACK FAKE IS THE POINT OF ONE TEST HERE. FR-061 says a failed write must
// leave no audit record, and the natural way to break that is to move the audit call
// after the transaction. A fake that merely counts calls cannot see the difference, so
// `snapshottingUow` below models what a transaction actually does to the outbox: an
// insert that is rolled back leaves nothing.

// ---- fixtures ----

// registryStub is a registry that records what the use case asked of it.
//
// It implements the WHOLE interface rather than only `Create`, because an interface
// with unimplemented methods does not satisfy `ICampaignRepository` and the file would
// not compile — which is the useful property: adding a repository method in a later
// phase breaks this stub loudly instead of silently not exercising it.
type registryStub struct {
	created []*entities.Campaign
	// createErr is returned by `Create`. Tests set it to
	// `repositories.ErrCampaignCodeExists` for the 409 case and to a plain error for
	// the infrastructure case.
	createErr error
	// rows is the in-memory registry the READ and CLOSE paths see. It was added with
	// the close tests (026 T045), which need a campaign that already exists — a stub
	// that only answered `Create` could express "open a campaign" and nothing else,
	// and a second stub type implementing the same seven methods would be the copy
	// this file's own header warns against.
	//
	// `Create` appends to BOTH slices: `created` is what the create tests read, and
	// `rows` is what a later operation on the same campaign would find — exactly as
	// the real table behaves, so a test cannot pass against a stub where the write
	// and the read are disconnected.
	rows map[string]*entities.Campaign
	// closeErr is returned by `Close` for the infrastructure case: a database that
	// refuses the statement, which must be reported as a 500 and not as a 412.
	closeErr error
	// closeVersions records every `expectedVersion` the use case handed to `Close`, so
	// a test can assert that the version it read is the one compared (FR-055) rather
	// than only that the call happened.
	closeVersions []int
}

func (s *registryStub) Create(_ context.Context, campaign *entities.Campaign) error {
	if s.createErr != nil {
		return s.createErr
	}
	copied := *campaign
	s.created = append(s.created, &copied)
	if s.rows == nil {
		s.rows = map[string]*entities.Campaign{}
	}
	s.rows[campaign.Code] = &copied
	return nil
}

func (s *registryStub) stored() []*entities.Campaign { return s.created }

// seedRow puts a campaign into the stub as if it had been created earlier, which is
// the state every close test starts from: a campaign that EXISTS and is open.
func (s *registryStub) seedRow(campaign *entities.Campaign) {
	if s.rows == nil {
		s.rows = map[string]*entities.Campaign{}
	}
	copied := *campaign
	s.rows[campaign.Code] = &copied
}

// row returns the stored campaign under a code, or nil.
func (s *registryStub) row(campaignCode string) *entities.Campaign {
	stored, ok := s.rows[campaignCode]
	if !ok {
		return nil
	}
	copied := *stored
	return &copied
}

func (s *registryStub) ListAll(context.Context) ([]*entities.Campaign, error) {
	return nil, nil
}

func (s *registryStub) ListOpen(context.Context) ([]*entities.Campaign, error) {
	return nil, nil
}

// GetByCode answers from `rows`, and returns nil rather than an error for an absent
// code — the same `nil, nil` the real repository uses, which is what lets the use
// case tell 404 from 500.
func (s *registryStub) GetByCode(_ context.Context, campaignCode string) (*entities.Campaign, error) {
	return s.row(campaignCode), nil
}

func (s *registryStub) GetByCodeForShare(ctx context.Context, campaignCode string) (*entities.Campaign, error) {
	return s.GetByCode(ctx, campaignCode)
}

func (s *registryStub) Update(context.Context, *entities.Campaign, int) (bool, error) {
	return false, nil
}

// Close reproduces the repository's conditional statement: one row changes only when
// the code exists AND carries the expected version, and zero rows otherwise.
//
// The version is bumped HERE rather than by the use case, because that is what the
// statement does — a stub that let the caller write the increment would hide a use
// case that forgot to pass the right version.
func (s *registryStub) Close(_ context.Context, campaignCode string, expectedVersion int) (bool, error) {
	s.closeVersions = append(s.closeVersions, expectedVersion)
	if s.closeErr != nil {
		return false, s.closeErr
	}
	stored, ok := s.rows[campaignCode]
	if !ok || stored.Version != expectedVersion {
		return false, nil
	}
	copied := *stored
	closedAt := baseInstant()
	copied.ClosedAt = &closedAt
	copied.Version = stored.Version + 1
	s.rows[campaignCode] = &copied
	return true, nil
}

func (s *registryStub) ExistsClaims(context.Context, string) (bool, error) { return false, nil }

var _ repositories.ICampaignRepository = (*registryStub)(nil)

// snapshottingUow models COMMIT and ROLLBACK over the audit outbox.
//
// The audit contract appends to a slice the moment it is called, so a use case that
// wrote its audit record OUTSIDE the transaction would still leave a record behind
// after a rollback — and FR-061 would be broken with every assertion here still green.
// This uow therefore truncates the recorded commands back to the count they had on
// entry whenever the closure returns an error, which is what a rolled-back INSERT
// inside the transaction does in production.
//
// It also COUNTS the entries, so "exactly one transaction" (constitution §V) is
// asserted rather than assumed.
type snapshottingUow struct {
	audit *fakeAuditContract
	// entries is how many times `Execute` has been called. A second call is the
	// nesting mistake 025 R3 forbids.
	entries int
}

func (u *snapshottingUow) Execute(ctx context.Context, fn func(ctx context.Context) error) error {
	u.entries++
	before := len(u.audit.all())
	if err := fn(ctx); err != nil {
		u.audit.truncateTo(before)
		return err
	}
	return nil
}

// registryFixture is the wired registry use case over the stub registry, the shared
// audit fake and the snapshotting unit of work.
type registryFixture struct {
	uc    *CampaignRegistryUseCase
	repo  *registryStub
	audit *fakeAuditContract
	uow   *snapshottingUow
}

func newRegistryFixture() *registryFixture {
	f := &registryFixture{
		repo:  &registryStub{},
		audit: &fakeAuditContract{},
	}
	f.uow = &snapshottingUow{audit: f.audit}
	uc := NewCampaignRegistryUseCase(testConfig(), f.repo, f.audit, f.uow, &recordingLogger{}).(*CampaignRegistryUseCase)
	uc.now = func() time.Time { return baseInstant() }
	f.uc = uc
	return f
}

// validCreateReq is a request that passes every rule, so a test can break exactly one
// thing and know the failure it is reading.
func validCreateReq() *sub_dto.CampaignCreateReq {
	planSlug := "premium-monthly"
	quota := 100
	reserve := 10
	endsAt := baseInstant().AddDate(0, 1, 0).Format(time.RFC3339)
	return &sub_dto.CampaignCreateReq{
		Code:     "tet-2027",
		PlanSlug: &planSlug,
		Quota:    &quota,
		Reserve:  &reserve,
		StartsAt: baseInstant().Format(time.RFC3339),
		EndsAt:   &endsAt,
	}
}

func intPtr(v int) *int       { return &v }
func strPtr(v string) *string { return &v }

// httpStatusOf is the HTTP code the global error handler would answer with.
//
// It reads the application error's own `Status` rather than calling `apperror.From`,
// because `From` substitutes a generic 500 for anything it does not recognise — which
// would make a test that accidentally returned a bare `errors.New` pass an assertion
// about 500 without ever having produced an application error.
func httpStatusOf(t *testing.T, err error) int {
	t.Helper()
	var appErr *apperror.Error
	if !errors.As(err, &appErr) {
		t.Fatalf("error %v is not an *apperror.Error, so it carries no HTTP code", err)
	}
	return appErr.Status
}

// ---- the happy path, and FR-060's "exactly one" ----

// TestCreateWritesOneCampaignAndExactlyOneAuditRecord is T036 / FR-052 / FR-060.
//
// "EXACTLY ONE" IS THE WHOLE REQUIREMENT, in both directions. Zero audit records would
// leave a campaign nobody can be shown to have opened; two would leave an operator
// unable to answer which one describes the change. So the count is asserted, not merely
// the presence — and the FR-063 fields are asserted alongside it, because a record that
// exists but cannot be found by the per-campaign read endpoint is no better than none.
func TestCreateWritesOneCampaignAndExactlyOneAuditRecord(t *testing.T) {
	f := newRegistryFixture()
	actor := uuid.New()
	reqIP := "203.0.113.7"

	summary, err := f.uc.Create(context.Background(), actor, validCreateReq(), &reqIP)
	if err != nil {
		t.Fatalf("a valid create must succeed, got %v", err)
	}

	stored := f.repo.stored()
	if len(stored) != 1 {
		t.Fatalf("expected exactly one campaign row, got %d", len(stored))
	}
	row := stored[0]
	if row.Code != "tet-2027" || row.PlanSlug != "premium-monthly" || row.Quota != 100 || row.Reserve != 10 {
		t.Errorf("the stored row lost the request: %+v", row)
	}
	if !row.StartsAt.Equal(baseInstant()) {
		t.Errorf("starts_at = %s, want the requested %s", row.StartsAt, baseInstant())
	}
	if row.EndsAt == nil || !row.EndsAt.After(row.StartsAt) {
		t.Errorf("ends_at must be after starts_at, got %v", row.EndsAt)
	}
	// FR-055: the client stores this version and sends it back as the If-Match of its
	// next write, so it has to be on the response rather than only in the column
	// default.
	if row.Version != 1 {
		t.Errorf("a newly created campaign must start at version 1, got %d", row.Version)
	}

	// Exactly one transaction. A second one would open a second connection, and the row
	// written in the first would be invisible to the audit insert in the second.
	if f.uow.entries != 1 {
		t.Errorf("FR-060: Create entered %d transactions; constitution §V allows exactly one per operation", f.uow.entries)
	}

	records := f.audit.all()
	if len(records) != 1 {
		t.Fatalf("FR-060: expected exactly one audit command, got %d", len(records))
	}
	record := records[0]
	if record.Action != string(adminaudit.CampaignCreate) {
		t.Errorf("action = %q, want %q", record.Action, adminaudit.CampaignCreate)
	}
	if record.TargetType != string(adminaudit.Campaign) {
		t.Errorf("FR-063: targetType = %q, want %q", record.TargetType, adminaudit.Campaign)
	}
	// FR-063: `target_id` is NULL for every campaign record, because the existing audit
	// filter takes a UUID while a campaign is keyed by a text code.
	if record.TargetID != nil {
		t.Errorf("FR-063: targetId must be nil for a campaign record, got %v", *record.TargetID)
	}
	// FR-063: the code travels in the payload, and that is the only thing the
	// per-campaign read endpoint filters on.
	if got := record.Payload["campaignCode"]; got != "tet-2027" {
		t.Errorf("FR-063: payload.campaignCode = %v, want \"tet-2027\"", got)
	}
	if record.actor != actor {
		t.Errorf("the audit actor = %s, want the administrator %s", record.actor, actor)
	}
	if record.Payload == nil || record.Payload["startsAt"] == nil {
		t.Errorf("the audit payload must carry the campaign's window, got %#v", record.Payload)
	}

	// The response is the READ shape, with the numbers a client renders before it has
	// ever refreshed the page.
	if summary == nil {
		t.Fatal("expected a summary in the create response")
	}
	if summary.Code != row.Code || summary.Quota != row.Quota || summary.HardCap != 110 {
		t.Errorf("the create response does not describe the campaign it created: %+v", summary)
	}
	if summary.GrantedMainCount != 0 || summary.GrantedCompensationCount != 0 || summary.GrantedTotalCount != 0 {
		t.Errorf("a campaign nobody was granted from must report zero everywhere: %+v", summary)
	}
	if summary.Status != sub_dto.CampaignStatusRunning {
		t.Errorf("status = %q; starts_at is in the past, so a fresh campaign must read as running",
			summary.Status)
	}
	if summary.RemainingMain != 100 || summary.RemainingCompensation != 10 {
		t.Errorf("remaining = %d/%d, want the untouched quota/reserve 100/10",
			summary.RemainingMain, summary.RemainingCompensation)
	}
}

// TestCreateStatusReflectsAStartsAtInTheFuture is the other half of the same rule.
//
// quickstart V1 opens a campaign whose `starts_at` is in the PAST, and this one opens
// one for next month. Both are valid (spec.md US1), and they must not be reported the
// same way — an operator who schedules a launch and is told "running" will not check
// again.
func TestCreateStatusReflectsAStartsAtInTheFuture(t *testing.T) {
	f := newRegistryFixture()
	req := validCreateReq()
	future := baseInstant().AddDate(0, 1, 0)
	req.StartsAt = future.Format(time.RFC3339)
	req.EndsAt = nil

	summary, err := f.uc.Create(context.Background(), uuid.New(), req, nil)
	if err != nil {
		t.Fatalf("a future campaign must be accepted, got %v", err)
	}
	if summary.Status != sub_dto.CampaignStatusNotStarted {
		t.Errorf("status = %q, want %q for a campaign that has not opened yet",
			summary.Status, sub_dto.CampaignStatusNotStarted)
	}
	if summary.EndsAt != nil {
		t.Errorf("an omitted endsAt means an open-ended campaign, not the zero time: got %v", summary.EndsAt)
	}
}

// ---- FR-054: 409 ----

// TestCreateRefusesADuplicateCodeWith409 is US1 acceptance scenario 3.
//
// The refusal comes from the CONSTRAINT, not from a pre-read, and the fake models it by
// returning the sentinel the repository's `Create` translates a unique violation into.
// The assertion that matters most is the second one: no second row was written, because
// "409" alone would also be satisfied by a use case that inserted first and apologised
// afterwards.
func TestCreateRefusesADuplicateCodeWith409(t *testing.T) {
	f := newRegistryFixture()
	f.repo.createErr = repositories.ErrCampaignCodeExists

	_, err := f.uc.Create(context.Background(), uuid.New(), validCreateReq(), nil)
	if err == nil {
		t.Fatal("a duplicate code must be refused, got success")
	}
	if status := httpStatusOf(t, err); status != 409 {
		t.Errorf("FR-054: a duplicate code must be 409, got %d (%v)", status, err)
	}
	if len(f.repo.stored()) != 0 {
		t.Errorf("US1.3: no second campaign may exist after a 409, got %d rows", len(f.repo.stored()))
	}
	if len(f.audit.all()) != 0 {
		t.Errorf("FR-061: a refused create must leave no audit record, got %d", len(f.audit.all()))
	}
}

// TestCreateReportsADatabaseFailureAsAnInfrastructureError is the OTHER half of the
// duplicate case, and it is the half that is easy to get wrong.
//
// `ErrCampaignCodeExists` and "the database is down" both arrive from the same
// `Create` call. Reporting the second as 409 would tell an operator their campaign code
// is taken when it is not, and they would go looking for a name collision that does not
// exist — while the real fault, an unreachable database, went unreported.
func TestCreateReportsADatabaseFailureAsAnInfrastructureError(t *testing.T) {
	f := newRegistryFixture()
	f.repo.createErr = context.DeadlineExceeded

	_, err := f.uc.Create(context.Background(), uuid.New(), validCreateReq(), nil)
	if err == nil {
		t.Fatal("a failing insert must fail the operation")
	}
	if status := httpStatusOf(t, err); status == 409 {
		t.Errorf("a database fault must not be reported as 409; the operator would hunt for a code collision that does not exist (got %v)", err)
	}
	if status := httpStatusOf(t, err); status != 500 {
		t.Errorf("expected 500 for an infrastructure failure, got %d (%v)", status, err)
	}
}

// ---- FR-054: 400 versus 422 ----

// TestCreateDistinguishesBrokenShapeFromWrongMeaning is FR-054's table, and the whole
// reason it exists is that a client has to tell the two apart.
//
// A 400 means "this is not a campaign I can read" and a 422 means "this is a campaign
// and the numbers in it are wrong". An operator acts on them differently — one opens
// the request, the other edits the form — so a use case that answered 422 for a
// misspelt code, or 400 for a zero quota, would be correct about both facts and wrong
// about both actions.
func TestCreateDistinguishesBrokenShapeFromWrongMeaning(t *testing.T) {
	blankPlanSlug := ""
	negativeReserve := -1
	endsAtSameInstant := baseInstant()

	cases := []struct {
		name   string
		mutate func(*sub_dto.CampaignCreateReq)
		want   int
		// mentions is a fragment the message must contain, so an operator is told
		// WHICH field to fix rather than only that something is wrong.
		mentions string
	}{
		{
			name:     "code with a space is a broken shape",
			mutate:   func(r *sub_dto.CampaignCreateReq) { r.Code = "Launch 2026" },
			want:     400,
			mentions: "Mã chiến dịch",
		},
		{
			name:     "code with uppercase is a broken shape",
			mutate:   func(r *sub_dto.CampaignCreateReq) { r.Code = "Tet-2027" },
			want:     400,
			mentions: "Mã chiến dịch",
		},
		{
			name:     "a two-character code is a broken shape",
			mutate:   func(r *sub_dto.CampaignCreateReq) { r.Code = "ab" },
			want:     400,
			mentions: "Mã chiến dịch",
		},
		{
			name:     "a zone-less startsAt is a broken shape",
			mutate:   func(r *sub_dto.CampaignCreateReq) { r.StartsAt = "2026-10-02 00:00:00" },
			want:     400,
			mentions: fieldStartsAt,
		},
		{
			name:     "a date without a time is a broken shape",
			mutate:   func(r *sub_dto.CampaignCreateReq) { r.StartsAt = "2026-10-02" },
			want:     400,
			mentions: fieldStartsAt,
		},
		{
			name: "a zone-less endsAt is a broken shape",
			mutate: func(r *sub_dto.CampaignCreateReq) {
				r.EndsAt = strPtr("2026-11-02 00:00:00")
			},
			want:     400,
			mentions: fieldEndsAt,
		},
		{
			name: "an empty-string endsAt is a malformed timestamp, not an open-ended campaign",
			// The distinction the pointer on the DTO exists for: `"endsAt": null` is an
			// open-ended campaign and `"endsAt": ""` is a timestamp that is not one.
			mutate:   func(r *sub_dto.CampaignCreateReq) { r.EndsAt = strPtr("") },
			want:     400,
			mentions: fieldEndsAt,
		},
		{
			name:     "a blank planSlug is wrong meaning",
			mutate:   func(r *sub_dto.CampaignCreateReq) { r.PlanSlug = &blankPlanSlug },
			want:     422,
			mentions: fieldPlanSlug,
		},
		{
			name:     "a whitespace-only planSlug is wrong meaning",
			mutate:   func(r *sub_dto.CampaignCreateReq) { r.PlanSlug = strPtr("   ") },
			want:     422,
			mentions: fieldPlanSlug,
		},
		{
			name:     "a zero quota is wrong meaning",
			mutate:   func(r *sub_dto.CampaignCreateReq) { r.Quota = intPtr(0) },
			want:     422,
			mentions: fieldQuota,
		},
		{
			name:     "a negative quota is wrong meaning",
			mutate:   func(r *sub_dto.CampaignCreateReq) { r.Quota = intPtr(-5) },
			want:     422,
			mentions: fieldQuota,
		},
		{
			name:     "a negative reserve is wrong meaning",
			mutate:   func(r *sub_dto.CampaignCreateReq) { r.Reserve = &negativeReserve },
			want:     422,
			mentions: fieldReserve,
		},
		{
			name: "an endsAt before startsAt is wrong meaning",
			mutate: func(r *sub_dto.CampaignCreateReq) {
				earlier := baseInstant().Add(-24 * time.Hour)
				r.EndsAt = strPtr(earlier.Format(time.RFC3339))
			},
			want:     422,
			mentions: fieldEndsAt,
		},
		{
			name: "an endsAt EQUAL to startsAt is wrong meaning",
			// The same instant is refused as well as an earlier one: a campaign whose
			// window closes the instant it opens grants nobody, which is the meaningless
			// state a zero quota describes and the reason the CHECK says `>`.
			mutate: func(r *sub_dto.CampaignCreateReq) {
				r.EndsAt = strPtr(endsAtSameInstant.Format(time.RFC3339))
			},
			want:     422,
			mentions: fieldEndsAt,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newRegistryFixture()
			req := validCreateReq()
			tc.mutate(req)

			_, err := f.uc.Create(context.Background(), uuid.New(), req, nil)
			if err == nil {
				t.Fatalf("expected a refusal with %d, got success", tc.want)
			}
			if status := httpStatusOf(t, err); status != tc.want {
				t.Errorf("FR-054: expected %d, got %d (%v)", tc.want, status, err)
			}
			if !strings.Contains(err.Error(), tc.mentions) {
				t.Errorf("the message must name the offending field (%q), got %q", tc.mentions, err.Error())
			}
			// A rejected request writes nothing at all — no row, no audit record — so a
			// typo cannot leave a trace that says somebody tried.
			if len(f.repo.stored()) != 0 {
				t.Errorf("a refused create must not write a row, got %d", len(f.repo.stored()))
			}
			if len(f.audit.all()) != 0 {
				t.Errorf("a refused create must not write an audit record, got %d", len(f.audit.all()))
			}
		})
	}
}

// TestCreateAcceptsAnOpenEndedCampaign is the negative control for the `endsAt` rules
// above: absent must mean "no end", not "the zero time".
func TestCreateAcceptsAnOpenEndedCampaign(t *testing.T) {
	f := newRegistryFixture()
	req := validCreateReq()
	req.EndsAt = nil
	req.Reserve = nil

	summary, err := f.uc.Create(context.Background(), uuid.New(), req, nil)
	if err != nil {
		t.Fatalf("an open-ended campaign must be accepted, got %v", err)
	}
	if len(f.repo.stored()) != 1 || f.repo.stored()[0].EndsAt != nil {
		t.Fatalf("ends_at must stay NULL, got %v", f.repo.stored()[0].EndsAt)
	}
	// FR-012: an absent reserve is 0, which is the column default — the campaign's hard
	// ceiling is exactly its quota.
	if summary.Reserve != 0 || summary.HardCap != 100 {
		t.Errorf("an omitted reserve must be 0, got reserve=%d hardCap=%d", summary.Reserve, summary.HardCap)
	}
}

// ---- FR-061: the rollback ----

// TestCreateRollsBackAndLeavesNoAuditRecordWhenTheAuditFails is FR-061.
//
// `IAuditLogContract.Record` publishes through the transactional OUTBOX, so its failure
// must abort the whole operation: a campaign nobody can be shown to have opened is
// exactly what FR-062 forbids, and the alternative — writing the row and logging the
// audit failure afterwards — produces a registry whose contents and whose history
// disagree.
//
// THE FAKE RECORDS THE COMMAND AND *THEN* FAILS (see `fakeAuditContract`), which is what
// makes this a test of the rollback rather than of a stub that never wrote anything. The
// assertion is on BOTH sides of the transaction: checking only that an error came back
// would also pass for a use case that recorded the audit after the commit and then
// reported the failure.
func TestCreateRollsBackAndLeavesNoAuditRecordWhenTheAuditFails(t *testing.T) {
	f := newRegistryFixture()
	f.audit.fail = true

	_, err := f.uc.Create(context.Background(), uuid.New(), validCreateReq(), nil)
	if err == nil {
		t.Fatal("a failed audit insert must fail the create")
	}
	if status := httpStatusOf(t, err); status != 500 {
		t.Errorf("expected 500 for an audit failure, got %d (%v)", status, err)
	}
	// The command really WAS issued, so "nothing is left" is the rollback's doing and not
	// the fake's refusal.
	if f.audit.issuedCount() != 1 {
		t.Fatalf("the fixture must have issued exactly one audit command, got %d - "+
			"without it this test cannot tell a rollback from a stub that never records",
			f.audit.issuedCount())
	}
	if len(f.audit.all()) != 0 {
		t.Errorf("FR-061: a failed operation must leave NO audit record, got %d", len(f.audit.all()))
	}
	if f.uow.entries != 1 {
		t.Errorf("expected the operation to have entered one transaction, got %d", f.uow.entries)
	}

	// The counterfactual that gives all of that meaning: the same fixture with a working
	// audit contract DOES leave exactly one record. Without it, "no records" is
	// satisfied by a fake that never records anything.
	healthy := newRegistryFixture()
	if _, err := healthy.uc.Create(context.Background(), uuid.New(), validCreateReq(), nil); err != nil {
		t.Fatalf("the control create failed: %v", err)
	}
	if len(healthy.audit.all()) != 1 {
		t.Errorf("the control must leave exactly one audit record, got %d", len(healthy.audit.all()))
	}
}

// TestCreateIsRejectedBeforeAnyTransactionWhenTheShapeIsWrong is the ordering FR-054's
// table implies and the transactional guarantee depends on.
//
// A malformed request must not open a transaction at all. If it did, every typo would
// take a connection from the pool and take a lock it does not need — and the audit
// trail would gain outbox rows for requests that never described a change.
func TestCreateIsRejectedBeforeAnyTransactionWhenTheShapeIsWrong(t *testing.T) {
	f := newRegistryFixture()
	req := validCreateReq()
	req.Code = "Launch 2026"

	if _, err := f.uc.Create(context.Background(), uuid.New(), req, nil); err == nil {
		t.Fatal("expected a refusal")
	}
	if f.uow.entries != 0 {
		t.Errorf("a rejected request must not open a transaction, got %d", f.uow.entries)
	}
}
