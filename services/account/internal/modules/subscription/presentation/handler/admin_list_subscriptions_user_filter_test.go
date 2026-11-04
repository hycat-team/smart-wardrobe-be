package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	subscriptiondto "smart-wardrobe-be/services/account/dto/subscription"
	uc_interfaces "smart-wardrobe-be/services/account/internal/modules/subscription/application/interface/usecase"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/constants/usersubscriptionstatus"
	"smart-wardrobe-be/shared/pkg/logger"
	shared_pres "smart-wardrobe-be/shared/presentation"
	shared_middleware "smart-wardrobe-be/shared/presentation/middleware"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ---- 034: the admin subscription list can be narrowed to one account ----
//
// The admin user detail dialog shows the raw subscription row (real start/expiry,
// auto-renew) instead of only the effective plan overview. That needs a `userId`
// filter on this list; before it, the only filters were plan code and status, so
// there was no way to ask for one account's row.
//
// The filter is parsed at the edge: a malformed UUID must be a 400 rather than an
// empty page that reads as "this account has no subscription".

const adminSubscriptionsPath = "/api/v1/admin/subscriptions"

type recordingAdminListUseCase struct {
	uc_interfaces.ISubscriptionUseCase
	called bool
	userID *uuid.UUID
}

func (f *recordingAdminListUseCase) AdminList(
	_ context.Context,
	_ *string,
	userID *uuid.UUID,
	_ *usersubscriptionstatus.UserSubscriptionStatus,
	_, _ int,
) ([]*subscriptiondto.AdminUserSubscriptionItemDTO, int64, error) {
	f.called = true
	f.userID = userID
	return nil, 0, nil
}

func getAdminSubscriptions(t *testing.T, rawQuery string, uc uc_interfaces.ISubscriptionUseCase) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)

	log := logger.New("test", "", "info", false)
	h := NewAdminSubscriptionHandler(log, nil, uc, nil)

	engine := gin.New()
	engine.Use(shared_middleware.GlobalErrorHandler(log, "test"))
	engine.GET(adminSubscriptionsPath, shared_pres.WrapHandler(h.ListSubscriptions))

	req := httptest.NewRequest(http.MethodGet, adminSubscriptionsPath+"?"+rawQuery, nil)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)
	return rec
}

func TestListSubscriptionsRejectsAMalformedUserID(t *testing.T) {
	uc := &recordingAdminListUseCase{}

	rec := getAdminSubscriptions(t, "userId=not-a-uuid", uc)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d (%s)", rec.Code, rec.Body.String())
	}
	if uc.called {
		t.Error("a malformed user id must not reach the use case")
	}
	if !strings.Contains(rec.Body.String(), "Mã người dùng") {
		t.Errorf("the message must name the invalid user id, got %s", rec.Body.String())
	}
}

func TestListSubscriptionsPassesTheUserIDFilterThrough(t *testing.T) {
	uc := &recordingAdminListUseCase{}
	target := uuid.New()

	rec := getAdminSubscriptions(t, "userId="+target.String()+"&limit=1", uc)

	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d (%s)", rec.Code, rec.Body.String())
	}
	if !uc.called {
		t.Fatal("the use case must be queried")
	}
	if uc.userID == nil || *uc.userID != target {
		t.Fatalf("want the parsed user id %s, got %v", target, uc.userID)
	}
}

func TestListSubscriptionsWithoutUserIDLeavesTheFilterEmpty(t *testing.T) {
	uc := &recordingAdminListUseCase{}

	rec := getAdminSubscriptions(t, "planCode=premium-monthly", uc)

	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d (%s)", rec.Code, rec.Body.String())
	}
	if uc.userID != nil {
		t.Errorf("want no user id filter, got %v", uc.userID)
	}
}
