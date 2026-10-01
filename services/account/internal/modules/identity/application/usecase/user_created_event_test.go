package usecase_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"smart-wardrobe-be/services/account/dto/identity"
	"smart-wardrobe-be/services/account/internal/modules/identity/domain/constants/authprovider"
	"smart-wardrobe-be/services/account/internal/modules/identity/domain/entities"
	"smart-wardrobe-be/shared/application/constants/eventconstants"
	shared_event "smart-wardrobe-be/shared/application/event"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// 025 T029b — three conditions the COMPILER cannot check.
//
// Everything else about `identity.user.created` is verified by the type system or
// by the existing 024 tests. These three are not:
//
//   - FR-064: the message carries a correlation id that ties the queue row to the
//     worker's log, on all three creation doors. Nothing in a signature forces a
//     correlation id to exist.
//   - FR-085: the identity module does not read the `campaigns` configuration.
//     Nothing stops a future line from doing it.
//   - FR-042: a failed event write fails the account creation. A publisher that
//     swallowed its error would compile perfectly and lose every message.
//
// They live in this file because all three creation doors are already constructed
// here, so none of them needs a new fixture.

// --- FR-064: the correlation id survives from publisher to worker log --------

// TestUserCreatedCarriesCorrelationIdOnEveryDoor asserts the published record has
// a non-empty correlation id for each of the three doors, and that the id is the
// one a worker's log would print.
//
// The worker side of that identity is `event.Message.CorrelationId()` plus
// `event.CorrelationIdFromContext`, and the outbox worker copies the record's
// headers onto the AMQP message. Reading it back through those exact functions is
// what makes this a test of the real contract rather than of the spy: if the
// header key ever changes, this goes red with the worker.
func TestUserCreatedCarriesCorrelationIdOnEveryDoor(t *testing.T) {
	doors := map[string]func(publisher shared_event.IEventPublisher) (recordedPublish, error){
		"email sign-up": func(publisher shared_event.IEventPublisher) (recordedPublish, error) {
			return runRegisterDoor(t, publisher)
		},
		"google first login": func(publisher shared_event.IEventPublisher) (recordedPublish, error) {
			return runGoogleDoor(t, publisher)
		},
		"administrator": func(publisher shared_event.IEventPublisher) (recordedPublish, error) {
			return runAdminDoor(t, publisher)
		},
	}

	for name, run := range doors {
		t.Run(name, func(t *testing.T) {
			publisher := newRecordingPublisher()
			record, err := run(publisher)
			if err != nil {
				t.Fatalf("the creation door failed: %v", err)
			}

			if record.correlationId == "" {
				t.Fatal("FR-064: the message must carry a correlation id; an empty one cannot be traced back to its publisher")
			}

			// Read it back the way the consumer does. This is the half of FR-064
			// that makes the id OBSERVABLE from the worker side, not merely present.
			delivered := shared_event.Message{
				Body:    nil,
				Headers: map[string]any{"correlationId": record.correlationId},
			}
			if got := delivered.CorrelationId(); got != record.correlationId {
				t.Errorf("FR-064: the worker's log would print %q, not the id the publisher wrote %q", got, record.correlationId)
			}
			if got := shared_event.CorrelationIdFromContext(shared_event.ContextWithCorrelationId(context.Background(), delivered)); got != record.correlationId {
				t.Errorf("FR-064: the handler context carries %q, want %q", got, record.correlationId)
			}

			// Two publishes must not share an id, or the trace points at the wrong
			// account.
			if name != "" {
				second := newRecordingPublisher()
				if _, err := run(second); err != nil {
					t.Fatalf("the second creation failed: %v", err)
				}
				if second.all()[0].correlationId == record.correlationId {
					t.Error("FR-064: two publishes share a correlation id, so a trace cannot distinguish them")
				}
			}
		})
	}
}

// TestUserCreatedPublishesInsideTheCreationTransaction is the structural half of
// FR-042: the publish must be part of the transaction that writes the account, on
// every door. Publishing afterwards would leave a window in which the account
// commits without its message.
func TestUserCreatedPublishesInsideTheCreationTransaction(t *testing.T) {
	uow := newEagerIdentityUow()
	repo := newEagerIdentityRepo(uow)
	spy := &eagerProvisioningSpy{uow: uow}
	publisher := newRecordingPublisher()

	uc, otpSvc := newEagerRegisterUseCaseWithPublisher(repo, spy, uow, publisher)
	otpSvc.data = eagerRegisterPayload
	if _, err := uc.ConfirmRegisterOtp(context.Background(), dto.ConfirmRegisterOtpReq{
		Email:   "duck.nguyen@example.com",
		OtpCode: "123456",
	}); err != nil {
		t.Fatalf("ConfirmRegisterOtp failed: %v", err)
	}

	records := publisher.all()
	if len(records) != 1 {
		t.Fatalf("expected exactly one published message, got %d", len(records))
	}
	if !records[0].inTx {
		t.Error("the publish must happen inside the transaction that writes the account, not after it commits")
	}
}

// --- FR-085: the identity module never reads the campaign configuration ------

// TestIdentityModuleDoesNotReadCampaignConfig scans the identity module's source
// for any reference to the `campaigns` configuration key.
//
// FR-085 makes the account-creation module a pure PUBLISHER: it must not know
// whether a campaign is running, must not branch on one, and must not look up a
// plan to decide what to publish. A single reference would make account creation
// depend on campaign configuration, and adding a second consumer of the event
// would then require editing this module — which is exactly what FR-090 forbids.
//
// A text scan, because nothing else can see the question. The compiler cannot, and
// neither can a unit test: the dependency this rule forbids is a read of a
// configuration key, which is invisible to both types.
func TestIdentityModuleDoesNotReadCampaignConfig(t *testing.T) {
	root := filepath.Join("..", "..", "..", "..", "modules", "identity")
	if _, err := os.Stat(root); err != nil {
		t.Fatalf("cannot locate the identity module at %s: %v", root, err)
	}

	forbidden := []string{"cfg.Campaigns", "cfg.Campaign", "CampaignConfig", "campaigns"}
	var hits []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}
		// Test files are excluded: FR-085 is a rule about PRODUCTION code, and this
		// very test names the forbidden identifiers in order to search for them.
		// Including them would make the scan fail on itself, which would train
		// everyone to ignore it.
		if strings.HasSuffix(path, "_test.go") {
			return nil
		}
		content, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		for _, needle := range forbidden {
			if strings.Contains(string(content), needle) {
				rel, _ := filepath.Rel(root, path)
				hits = append(hits, rel+": "+needle)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("scanning the identity module failed: %v", err)
	}

	if len(hits) > 0 {
		t.Errorf("FR-085: modules/identity must not read the campaign configuration - publisher-first means it knows nothing about campaigns.\nFound:\n  %s",
			strings.Join(hits, "\n  "))
	}
}

// --- FR-042: a failed event write fails the account creation -----------------

// TestAccountCreationRollsBackWhenTheEventCannotBeWritten is the anti-drain test.
//
// FR-042 chooses to fail the registration rather than lose the message, because an
// account that exists but whose creation was never announced is invisible: no
// campaign grant, no welcome email, no record that anything was expected. The
// message, by contrast, is durable and retried.
//
// The assertion is stronger than "an error came back": it checks that NO account
// row survived. A publisher that logged-and-continued would return nil, and an
// implementation that returned the error but outside the transaction would leave
// the account committed — both of which look identical from the return value alone.
func TestAccountCreationRollsBackWhenTheEventCannotBeWritten(t *testing.T) {
	uow := newEagerIdentityUow()
	repo := newEagerIdentityRepo(uow)
	spy := &eagerProvisioningSpy{uow: uow}
	publisher := newRecordingPublisher()
	publisher.failWith(errors.New("outbox insert failed"))

	uc, otpSvc := newEagerRegisterUseCaseWithPublisher(repo, spy, uow, publisher)
	otpSvc.data = eagerRegisterPayload

	ok, err := uc.ConfirmRegisterOtp(context.Background(), dto.ConfirmRegisterOtpReq{
		Email:   "duck.nguyen@example.com",
		OtpCode: "123456",
	})
	if err == nil {
		t.Fatal("FR-042: a registration whose message could not be written must report an error")
	}
	if ok {
		t.Error("must not report success when the message was lost")
	}
	if len(uow.accounts) != 0 {
		t.Errorf("FR-042: no account may survive - %d committed. An account nobody was told about is the failure this rule exists to prevent",
			len(uow.accounts))
	}
	if uow.commits != 0 {
		t.Errorf("nothing may commit, commits = %d", uow.commits)
	}
	if uow.rollbacks != 1 {
		t.Errorf("the transaction must roll back, rollbacks = %d", uow.rollbacks)
	}
}

// --- the three doors, each returning its one published record ---------------

func runRegisterDoor(t *testing.T, publisher shared_event.IEventPublisher) (recordedPublish, error) {
	t.Helper()
	uow := newEagerIdentityUow()
	repo := newEagerIdentityRepo(uow)
	spy := &eagerProvisioningSpy{uow: uow}
	uc, otpSvc := newEagerRegisterUseCaseWithPublisher(repo, spy, uow, publisher)
	otpSvc.data = eagerRegisterPayload

	if _, err := uc.ConfirmRegisterOtp(context.Background(), dto.ConfirmRegisterOtpReq{
		Email:   "duck.nguyen@example.com",
		OtpCode: "123456",
	}); err != nil {
		return recordedPublish{}, err
	}
	return singleRecord(t, publisher)
}

func runGoogleDoor(t *testing.T, publisher shared_event.IEventPublisher) (recordedPublish, error) {
	t.Helper()
	uow := newEagerIdentityUow()
	repo := newEagerIdentityRepo(uow)
	spy := &eagerProvisioningSpy{uow: uow}
	uc := newEagerGoogleUCWithPublisher(repo, spy, uow, identityClaim(), publisher)

	if _, err := uc.LoginWithIDToken(context.Background(), dto.GoogleLoginReq{IDToken: "token"}); err != nil {
		return recordedPublish{}, err
	}
	return singleRecord(t, publisher)
}

func runAdminDoor(t *testing.T, publisher shared_event.IEventPublisher) (recordedPublish, error) {
	t.Helper()
	uow := newEagerIdentityUow()
	repo := newEagerIdentityRepo(uow)
	spy := &eagerProvisioningSpy{uow: uow}
	uc := newEagerUserUseCaseWithPublisher(repo, spy, uow, publisher)

	if _, err := uc.CreateUserByAdmin(context.Background(), uuid.New(), dto.CreateAdminUserReq{
		Username:  "duck.nguyen",
		Email:     "duck.nguyen@example.com",
		Password:  "Str0ng-Passw0rd!",
		FirstName: "Duck",
	}); err != nil {
		return recordedPublish{}, err
	}
	return singleRecord(t, publisher)
}

func singleRecord(t *testing.T, publisher shared_event.IEventPublisher) (recordedPublish, error) {
	t.Helper()
	recorder, ok := publisher.(*recordingPublisher)
	if !ok {
		t.Fatalf("expected the recording publisher, got %T", publisher)
	}
	records := recorder.all()
	if len(records) != 1 {
		t.Fatalf("expected exactly one published message, got %d", len(records))
	}
	if records[0].topic != eventconstants.TopicUserCreated {
		t.Errorf("published on %q, want %q", records[0].topic, eventconstants.TopicUserCreated)
	}
	return records[0], nil
}

// --- the payload contract, and the flag that must agree across two sources ---

// TestCreatedByAdminFlagAgreesWithTheStoredColumn is the anti-drift test for the
// one boolean two independent paths derive the same fact from.
//
// `created_by_admin` in the database and `createdByAdmin` in the message are
// written at two different places. If they ever disagree, one of the two
// exclusion paths silently stops working: the event path refuses the
// administrator account, or the catch-up sweep does — never both, and nothing
// reports it. That is the exact failure FR-091 exists to prevent, and it is
// invisible from either side alone.
func TestCreatedByAdminFlagAgreesWithTheStoredColumn(t *testing.T) {
	cases := []struct {
		name        string
		run         func(t *testing.T, publisher shared_event.IEventPublisher) (recordedPublish, error)
		wantInEvent bool
	}{
		{"email sign-up", runRegisterDoor, false},
		{"google first login", runGoogleDoor, false},
		{"administrator", runAdminDoor, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			publisher := newRecordingPublisher()
			record, err := tc.run(t, publisher)
			if err != nil {
				t.Fatalf("the creation door failed: %v", err)
			}

			if record.payload.CreatedByAdmin != tc.wantInEvent {
				t.Errorf("the message says createdByAdmin=%v, want %v", record.payload.CreatedByAdmin, tc.wantInEvent)
			}
			// The two self-serve doors must ALSO leave the stored column at its
			// default. If one of them started setting it, the event and the sweep
			// would begin to disagree.
			if !tc.wantInEvent {
				assert.False(t, storedCreatedByAdminFor(t, tc.name),
					"a self-serve door must not set created_by_admin")
			}
		})
	}
}

// storedCreatedByAdminFor reports what the creation door wrote to the column.
//
// It reads the entity the fake repository staged rather than the message, so the
// two sources being compared really are two: one is the row, the other is the
// event. Reading the message twice would assert that a value equals itself.
func storedCreatedByAdminFor(t *testing.T, door string) bool {
	t.Helper()
	uow := newEagerIdentityUow()
	repo := newEagerIdentityRepo(uow)
	spy := &eagerProvisioningSpy{uow: uow}
	publisher := newRecordingPublisher()

	switch door {
	case "email sign-up":
		uc, otpSvc := newEagerRegisterUseCaseWithPublisher(repo, spy, uow, publisher)
		otpSvc.data = eagerRegisterPayload
		if _, err := uc.ConfirmRegisterOtp(context.Background(), dto.ConfirmRegisterOtpReq{
			Email:   "duck.nguyen@example.com",
			OtpCode: "123456",
		}); err != nil {
			t.Fatalf("ConfirmRegisterOtp failed: %v", err)
		}
	case "google first login":
		uc := newEagerGoogleUCWithPublisher(repo, spy, uow, identityClaim(), publisher)
		if _, err := uc.LoginWithIDToken(context.Background(), dto.GoogleLoginReq{IDToken: "token"}); err != nil {
			t.Fatalf("LoginWithIDToken failed: %v", err)
		}
	default:
		uc := newEagerUserUseCaseWithPublisher(repo, spy, uow, publisher)
		if _, err := uc.CreateUserByAdmin(context.Background(), uuid.New(), dto.CreateAdminUserReq{
			Username:  "duck.nguyen",
			Email:     "duck.nguyen@example.com",
			Password:  "Str0ng-Passw0rd!",
			FirstName: "Duck",
		}); err != nil {
			t.Fatalf("CreateUserByAdmin failed: %v", err)
		}
	}

	for _, row := range uow.accounts {
		if row.Username == "<entitlement>" {
			continue
		}
		return row.CreatedByAdmin
	}
	t.Fatal("no account row was committed")
	return false
}

// TestOnlyTheAdministratorDoorWritesCreatedByAdminTrue is the text scan behind
// FR-091's "exactly one write site".
//
// The compiler cannot see this: setting a struct field to true is ordinary code,
// and both self-serve doors would compile perfectly while quietly eating
// customers' promotional quota. Nor can a unit test that only exercises the doors
// it knows about — it cannot tell an absent writer from a writer it did not call.
func TestOnlyTheAdministratorDoorWritesCreatedByAdminTrue(t *testing.T) {
	root := filepath.Join("..", "..", "..", "..", "modules", "identity")
	var hits []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}
		content, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		text := string(content)
		if !strings.Contains(text, "CreatedByAdmin: true") && !strings.Contains(text, "CreatedByAdmin=true") {
			return nil
		}
		rel, _ := filepath.Rel(root, path)
		// Test files assert on the flag; only production writes count.
		if strings.HasSuffix(path, "_test.go") {
			return nil
		}
		if strings.HasSuffix(rel, "admin_uc.go") {
			return nil
		}
		hits = append(hits, rel)
		return nil
	})
	if err != nil {
		t.Fatalf("scanning the identity module failed: %v", err)
	}
	if len(hits) > 0 {
		t.Errorf("FR-091: only the administrator door may set CreatedByAdmin true; also written in: %s",
			strings.Join(hits, ", "))
	}
}

// TestUserCreatedPayloadCarriesNoPersonalData is FR-087 as a reflection check.
//
// The payload is a struct, so "no personally identifiable data" is a claim about
// its FIELDS, and reflection is the only thing that can test it. A future field
// named `Email` would compile, would pass every behavioural test, and would put
// customer data into a broker that persists and fans messages out.
func TestUserCreatedPayloadCarriesNoPersonalData(t *testing.T) {
	forbidden := []string{
		"email", "username", "firstname", "lastname", "fullname",
		"password", "passwordhash", "phone", "address", "dateofbirth",
		"dob", "avatar", "gender", "token",
	}
	allowed := map[string]bool{
		"userid": true, "provider": true, "registeredat": true, "createdbyadmin": true,
	}

	typ := reflectTypeOfPayload()
	if typ == nil {
		t.Fatal("expected the payload type to exist")
	}
	if typ.NumField() != 4 {
		t.Errorf("FR-087: the payload must carry exactly 4 fields, found %d - every extra field is a chance to leak something", typ.NumField())
	}
	for i := 0; i < typ.NumField(); i++ {
		name := strings.ToLower(typ.Field(i).Name)
		if allowed[name] {
			continue
		}
		for _, needle := range forbidden {
			if strings.Contains(name, needle) {
				t.Errorf("FR-087: field %q looks like personally identifiable data; the payload carries only the account id, the source, the creation instant and the administrator flag",
					typ.Field(i).Name)
				break
			}
		}
		if !allowed[name] {
			t.Errorf("FR-087: unexpected field %q on the identity.user.created payload", typ.Field(i).Name)
		}
	}
}

// TestUserCreatedPayloadIsTheDeclaredShape pins the JSON contract.
//
// `camelCase` labels are what a receiver written in another language binds to, so
// renaming one silently breaks every consumer — including the commerce and AI
// services, which are compiled from the same contract.
func TestUserCreatedPayloadIsTheDeclaredShape(t *testing.T) {
	typ := reflectTypeOfPayload()
	if typ == nil {
		t.Fatal("expected the payload type to exist")
	}
	want := map[string]string{
		"UserID":         "userId",
		"Provider":       "provider",
		"RegisteredAt":   "registeredAt",
		"CreatedByAdmin": "createdByAdmin",
	}
	for i := 0; i < typ.NumField(); i++ {
		tag := typ.Field(i).Tag.Get("json")
		expected, ok := want[typ.Field(i).Name]
		if !ok {
			t.Errorf("unexpected field %q", typ.Field(i).Name)
			continue
		}
		if tag != expected {
			t.Errorf("field %q has json tag %q, want %q", typ.Field(i).Name, tag, expected)
		}
	}
}

// TestUserCreatedProviderMatchesTheColumn pins `provider` to the stored value.
//
// The message says which sign-up door produced the account, and the sweep can only
// trust it if it is the value the row carries — not a re-derived literal, which
// could drift the moment a provider is added.
func TestUserCreatedProviderMatchesTheColumn(t *testing.T) {
	uow := newEagerIdentityUow()
	repo := newEagerIdentityRepo(uow)
	spy := &eagerProvisioningSpy{uow: uow}
	publisher := newRecordingPublisher()

	uc, otpSvc := newEagerRegisterUseCaseWithPublisher(repo, spy, uow, publisher)
	otpSvc.data = eagerRegisterPayload
	if _, err := uc.ConfirmRegisterOtp(context.Background(), dto.ConfirmRegisterOtpReq{
		Email:   "duck.nguyen@example.com",
		OtpCode: "123456",
	}); err != nil {
		t.Fatalf("ConfirmRegisterOtp failed: %v", err)
	}

	record, err := singleRecord(t, publisher)
	if err != nil {
		t.Fatal(err)
	}
	if record.payload.Provider != string(authprovider.Local) {
		t.Errorf("provider = %q, want %q", record.payload.Provider, authprovider.Local)
	}

	// registeredAt must be the STORED creation instant, because FR-031 compares it
	// against the frozen watermark. A second clock reading taken at publish time
	// would differ from the column and move the priority watermark.
	var stored *entities.User
	for _, row := range uow.accounts {
		if row.Username != "<entitlement>" {
			stored = row
		}
	}
	if stored == nil {
		t.Fatal("no account row was committed")
	}
	if stored.CreatedAt.IsZero() {
		t.Fatal("the fake repository must stamp CreatedAt, as GORM does from the column default")
	}
	if !record.payload.RegisteredAt.Equal(stored.CreatedAt) {
		t.Errorf("registeredAt = %v but users.created_at = %v; FR-031 compares the stored instant against the watermark",
			record.payload.RegisteredAt, stored.CreatedAt)
	}
}

var _ = dto.UserCreatedEventPayload{}
