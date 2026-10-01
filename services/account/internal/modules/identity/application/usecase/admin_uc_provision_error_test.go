package usecase_test

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"

	"smart-wardrobe-be/services/account/dto/identity"
	identityerrors "smart-wardrobe-be/services/account/internal/modules/identity/application/errors"
	subscriptionerrors "smart-wardrobe-be/services/account/internal/modules/subscription/application/errors"
	"smart-wardrobe-be/shared/application/constants/apperror"

	"github.com/google/uuid"
)

// ---- 024 T197: a provisioning failure leaves as an identity-module error ----
//
// The account and its default entitlement are one transaction (024 T196,
// FR-013), so `uow.Execute` is where the default plan is provisioned. Two of the
// three account-creation doors already translated what comes out of it:
// register_uc.go:194 returns `ErrAccountCreationFailed()` and
// google_auth_uc.go:251 returns it beside `webErrServer`. The administrator door
// returned the error untouched, and provisioning answers with the subscription
// module's own factories - `ErrDefaultPlanConfigNotFound` and
// `ErrUserSubscriptionNotFound`, both `apperror.NewNotFound`.
//
// The result was `POST /admin/users` answering HTTP 404 with a message about
// the plan catalogue, for an operation that neither looks up nor deletes
// anything. The caller has no way to read that as "creating an account failed".
//
// 024 T209 removed the other half of that fix's cost. The factory kept the
// ownership and the status, but it reached for the cause's own text to keep the
// operator informed, and that text is internal - English, driver wording,
// whatever the dependency felt like saying - and ended up in a client response.
// These tests therefore pin two separate things: the answer is this module's, and
// it names the KIND of failure without ever quoting the cause.

// The message a broken plan catalogue produces. This is the subscription module's
// OWN factory rather than a hand-written copy of it: `ErrAccountCreationFailedWithCause`
// classifies the cause by matching it against that factory (024 T209), so a
// reproduction with the same wording would be relying on the two happening to
// agree, and would silently stop matching the day either sentence is edited.
var provisioningFailure = subscriptionerrors.ErrDefaultPlanConfigNotFound()

func TestCreateUserByAdminReturnsAnIdentityErrorWhenProvisioningFails(t *testing.T) {
	uow := newEagerIdentityUow()
	repo := newEagerIdentityRepo(uow)
	spy := &eagerProvisioningSpy{uow: uow, err: provisioningFailure}

	uc := newEagerUserUseCase(repo, spy, uow)
	res, err := uc.CreateUserByAdmin(context.Background(), uuid.New(), dto.CreateAdminUserReq{
		Username:  "duck.nguyen",
		Email:     "duck.nguyen@example.com",
		Password:  "Str0ng-Passw0rd!",
		FirstName: "Duck",
	})
	if err == nil {
		t.Fatal("must return an error when provisioning fails")
	}
	if res != nil {
		t.Error("must not return an account when the transaction rolled back")
	}

	var appErr *apperror.Error
	if !errors.As(err, &appErr) {
		t.Fatalf("the answer must be an apperror, got %T: %v", err, err)
	}

	// The subscription factory says 404. Returning it unchanged is what made an
	// account-creation failure look like a missing resource.
	if appErr.Status == http.StatusNotFound {
		t.Errorf("must not surface the dependency's 404, got %d (%s)", appErr.Status, appErr.Message)
	}
	if appErr.Status != http.StatusInternalServerError {
		t.Errorf("want the identity module's 500, got %d (%s)", appErr.Status, appErr.Message)
	}

	// It must be recognisable as THIS module's error, so a caller (or a test)
	// can match on it the way the other two doors allow.
	if !errors.Is(err, identityerrors.ErrAccountCreationFailedWithCause(provisioningFailure)) {
		t.Errorf("must be the identity module's own factory, got %v", err)
	}

	// And it must still say WHY. "Lỗi khi khởi tạo tài khoản mới" alone leaves an
	// administrator with nothing to act on; the reason is that the default plan
	// configuration is missing. 024 T209 kept that while dropping the raw text:
	// the sentence is fixed and names the KIND of failure, and the wording of the
	// dependency it replaces never reaches the client.
	if !strings.Contains(appErr.Message, "gói hội viên mặc định") {
		t.Errorf("the message must name the real reason, got %q", appErr.Message)
	}

	// The cause stays available for the logs, which is where the raw text belongs.
	if !errors.Is(appErr.Unwrap(), provisioningFailure) {
		t.Errorf("the original error must remain as the cause, got %v", appErr.Unwrap())
	}
}

// The reason is not a fixed sentence: a different provisioning failure has to come
// out with ITS kind named, or the fix would only help one failure mode.
func TestCreateUserByAdminKeepsTheReasonOfAnyProvisioningFailure(t *testing.T) {
	uow := newEagerIdentityUow()
	repo := newEagerIdentityRepo(uow)
	publishFailed := errors.New("outbox insert failed")
	spy := &eagerProvisioningSpy{uow: uow, err: publishFailed}

	uc := newEagerUserUseCase(repo, spy, uow)
	_, err := uc.CreateUserByAdmin(context.Background(), uuid.New(), dto.CreateAdminUserReq{
		Username:  "duck.nguyen",
		Email:     "duck.nguyen@example.com",
		Password:  "Str0ng-Passw0rd!",
		FirstName: "Duck",
	})
	if err == nil {
		t.Fatal("must return an error when provisioning fails")
	}

	var appErr *apperror.Error
	if !errors.As(err, &appErr) || appErr.Status != http.StatusInternalServerError {
		t.Fatalf("an unrecognised cause must still be an internal identity error, got %v", err)
	}

	// 024 T209: this assertion used to require the opposite - the message had to
	// CONTAIN `publishFailed.Error()`. That is what put a raw, internal, English
	// driver string in the body of `POST /admin/users`, against
	// apperror/error.go:15-17. The reason is still reported, as a fixed sentence
	// about the KIND of failure, and the cause is still available to the log.
	if !strings.Contains(appErr.Message, "gói hội viên mặc định") {
		t.Errorf("the message must still name the stage that failed, got %q", appErr.Message)
	}
	if strings.Contains(appErr.Message, publishFailed.Error()) {
		t.Errorf("the message must not carry the raw error text %q, got %q", publishFailed.Error(), appErr.Message)
	}
	if !errors.Is(appErr.Unwrap(), publishFailed) {
		t.Errorf("the raw cause must remain reachable for the log, got %v", appErr.Unwrap())
	}
}
