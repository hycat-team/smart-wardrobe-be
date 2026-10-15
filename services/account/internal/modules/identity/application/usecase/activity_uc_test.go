package usecase_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"
	"time"

	dto "smart-wardrobe-be/services/account/dto/identity"
	usecase_interfaces "smart-wardrobe-be/services/account/internal/modules/identity/application/interface/usecase"
	usecase "smart-wardrobe-be/services/account/internal/modules/identity/application/usecase"
	"smart-wardrobe-be/services/account/internal/modules/identity/domain/constants/useractivitytype"
	identityrepositories "smart-wardrobe-be/services/account/internal/modules/identity/domain/repositories"
	"smart-wardrobe-be/shared/application/constants/apperror"
	shared_dto "smart-wardrobe-be/shared/application/dto"
	"smart-wardrobe-be/shared/config"
	"smart-wardrobe-be/shared/pkg/utils/timeutils"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 029 T008: the admin activity list use case (FR-001, FR-002, FR-005).
//
// The rules that must not be skippable by a non-HTTP caller live HERE, beside
// the query they constrain (the campaign query use case made the same choice):
// a malformed userId, an activity type outside {login, refresh}, a malformed or
// inverted date range — all typed by hand — are the use case's to refuse. The
// handler only binds the query string and hands it over.

// stubActivityRepo is the activity repository view the use case needs. Anything
// else panics through the embedded nil interface, which fails the test rather
// than silently doing nothing.
type stubActivityRepo struct {
	identityrepositories.IUserActivityRepository
	rows  []identityrepositories.AdminActivityEventRow
	total int64
	err   error

	calls      int
	lastFilter identityrepositories.AdminActivityEventFilter
	lastPage   int
	lastLimit  int
}

func (s *stubActivityRepo) ListAdmin(
	_ context.Context,
	filter identityrepositories.AdminActivityEventFilter,
	page, limit int,
) ([]identityrepositories.AdminActivityEventRow, int64, error) {
	s.calls++
	s.lastFilter = filter
	s.lastPage = page
	s.lastLimit = limit
	return s.rows, s.total, s.err
}

func newActivityUseCase(repo *stubActivityRepo) usecase_interfaces.IUserUseCase {
	return usecase.NewUserUseCase(nil, repo, nil, nil, nil, nil, nil, nil, nil, &config.Config{}, nil, nil)
}

func requireActivityBadRequest(t *testing.T, err error) *apperror.Error {
	t.Helper()
	if err == nil {
		t.Fatal("expected a 400 validation error, got nil")
	}
	var appErr *apperror.Error
	if !errors.As(err, &appErr) {
		t.Fatalf("expected *apperror.Error, got %T: %v", err, err)
	}
	if appErr.Status != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d (%s)", appErr.Status, appErr.Message)
	}
	return appErr
}

func TestListActivityEventsRejectsMalformedUserID(t *testing.T) {
	repo := &stubActivityRepo{}
	uc := newActivityUseCase(repo)

	_, err := uc.ListActivityEvents(context.Background(), dto.GetActivityEventsQueryReq{UserID: "abc"})

	appErr := requireActivityBadRequest(t, err)
	assert.Contains(t, appErr.Message, "userId", "the message must name the offending parameter")
	assert.Zero(t, repo.calls, "a refused filter must not reach the repository")
}

func TestListActivityEventsRejectsUnknownActivityType(t *testing.T) {
	repo := &stubActivityRepo{}
	uc := newActivityUseCase(repo)

	for _, invalid := range []string{"logout", "LOGIN", "refreshed", "signup"} {
		_, err := uc.ListActivityEvents(context.Background(), dto.GetActivityEventsQueryReq{ActivityType: invalid})
		appErr := requireActivityBadRequest(t, err)
		assert.Contains(t, appErr.Message, "activityType", "invalid %q must be named in the message", invalid)
	}
	assert.Zero(t, repo.calls, "a refused filter must not reach the repository")

	// The two values of the existing constant set are the ones accepted.
	for _, valid := range []string{string(useractivitytype.Login), string(useractivitytype.Refresh)} {
		repo := &stubActivityRepo{}
		uc := newActivityUseCase(repo)
		_, err := uc.ListActivityEvents(context.Background(), dto.GetActivityEventsQueryReq{ActivityType: valid})
		require.NoError(t, err, "%q must be accepted", valid)
		assert.Equal(t, 1, repo.calls)
	}
}

func TestListActivityEventsRejectsBadDateRanges(t *testing.T) {
	for name, query := range map[string]dto.GetActivityEventsQueryReq{
		"malformed fromDate": {FromDate: "15-10-2026"},
		"malformed toDate":   {ToDate: "2026/10/15"},
		"impossible day":     {FromDate: "2026-02-30"},
		"inverted range":     {FromDate: "2026-10-16", ToDate: "2026-10-15"},
		"adjacent inversion": {FromDate: "2026-10-16", ToDate: "2026-10-15"},
	} {
		t.Run(name, func(t *testing.T) {
			repo := &stubActivityRepo{}
			uc := newActivityUseCase(repo)

			_, err := uc.ListActivityEvents(context.Background(), query)

			requireActivityBadRequest(t, err)
			assert.Zero(t, repo.calls, "a refused period must not reach the repository")
		})
	}
}

func TestListActivityEventsNormalizesPaginationAndPassesFilter(t *testing.T) {
	repo := &stubActivityRepo{total: 3}
	uc := newActivityUseCase(repo)

	userID := uuid.New()
	res, err := uc.ListActivityEvents(context.Background(), dto.GetActivityEventsQueryReq{
		PaginationQuery: shared_dto.PaginationQuery{Page: 0, Limit: 0},
		UserID:          userID.String(),
		ActivityType:    string(useractivitytype.Login),
		FromDate:        "2026-10-15",
		ToDate:          "2026-10-15",
	})
	require.NoError(t, err)

	assert.Equal(t, 1, repo.lastPage, "an omitted page normalizes to 1")
	assert.Equal(t, 20, repo.lastLimit, "an omitted limit normalizes to 20")
	require.NotNil(t, repo.lastFilter.UserID)
	assert.Equal(t, userID, *repo.lastFilter.UserID)
	require.NotNil(t, repo.lastFilter.ActivityType)
	assert.Equal(t, useractivitytype.Login, *repo.lastFilter.ActivityType)
	assert.True(t, repo.lastFilter.From.Equal(time.Date(2026, time.October, 15, 0, 0, 0, 0, timeutils.VnLoc())),
		"fromDate must become the start of the VN day, got %s", repo.lastFilter.From)
	assert.True(t, repo.lastFilter.To.Equal(time.Date(2026, time.October, 16, 0, 0, 0, 0, timeutils.VnLoc())),
		"toDate must become 00:00 of the next VN day, got %s", repo.lastFilter.To)

	assert.Equal(t, 1, res.Metadata.Page)
	assert.Equal(t, 20, res.Metadata.Limit)
	assert.Equal(t, int64(3), res.Metadata.TotalItems)

	t.Run("the limit is capped at 100", func(t *testing.T) {
		repo := &stubActivityRepo{}
		uc := newActivityUseCase(repo)
		_, err := uc.ListActivityEvents(context.Background(), dto.GetActivityEventsQueryReq{
			PaginationQuery: shared_dto.PaginationQuery{Page: 2, Limit: 500},
		})
		require.NoError(t, err)
		assert.Equal(t, 2, repo.lastPage)
		assert.Equal(t, 100, repo.lastLimit)
	})

	t.Run("an omitted side leaves the range open", func(t *testing.T) {
		repo := &stubActivityRepo{}
		uc := newActivityUseCase(repo)
		_, err := uc.ListActivityEvents(context.Background(), dto.GetActivityEventsQueryReq{
			FromDate: "2026-10-15",
		})
		require.NoError(t, err)
		assert.False(t, repo.lastFilter.From.IsZero())
		assert.True(t, repo.lastFilter.To.IsZero(), "a missing toDate must stay zero so the repository drops the condition")
		assert.Nil(t, repo.lastFilter.UserID)
		assert.Nil(t, repo.lastFilter.ActivityType)
	})
}

func TestListActivityEventsMapsRowsAndOmitsNullNames(t *testing.T) {
	occurred := time.Date(2026, 10, 15, 8, 31, 2, 0, timeutils.VnLoc())
	firstID := uuid.New()
	secondID := uuid.New()
	repo := &stubActivityRepo{
		total: 2,
		rows: []identityrepositories.AdminActivityEventRow{
			{
				ID:           firstID,
				UserID:       uuid.New(),
				ActivityType: string(useractivitytype.Login),
				OccurredAt:   occurred,
				CreatedAt:    occurred,
				Email:        "named@example.test",
				FirstName:    ptr("An"),
				LastName:     ptr("Nguyễn"),
			},
			{
				ID:           secondID,
				UserID:       uuid.New(),
				ActivityType: string(useractivitytype.Refresh),
				OccurredAt:   occurred.Add(-time.Hour),
				CreatedAt:    occurred.Add(-time.Hour),
				Email:        "anonymous@example.test",
				// FirstName/LastName nil: the account never set them.
			},
		},
	}
	uc := newActivityUseCase(repo)

	res, err := uc.ListActivityEvents(context.Background(), dto.GetActivityEventsQueryReq{})
	require.NoError(t, err)
	require.Len(t, res.Items, 2)

	first := res.Items[0]
	assert.Equal(t, firstID, first.ID)
	assert.Equal(t, "named@example.test", first.UserEmail)
	require.NotNil(t, first.UserFirstName)
	require.NotNil(t, first.UserLastName)
	assert.Equal(t, "An", *first.UserFirstName)
	assert.Equal(t, "Nguyễn", *first.UserLastName)

	second := res.Items[1]
	assert.Equal(t, secondID, second.ID)
	assert.Nil(t, second.UserFirstName)
	assert.Nil(t, second.UserLastName)

	raw, err := json.Marshal(second)
	require.NoError(t, err)
	var parsed map[string]any
	require.NoError(t, json.Unmarshal(raw, &parsed))
	_, hasFirst := parsed["userFirstName"]
	_, hasLast := parsed["userLastName"]
	assert.False(t, hasFirst, "a NULL first name must be absent from the JSON, got %s", raw)
	assert.False(t, hasLast, "a NULL last name must be absent from the JSON, got %s", raw)
	for _, key := range []string{"id", "userId", "activityType", "occurredAt", "createdAt", "userEmail"} {
		assert.Contains(t, parsed, key, "the JSON must carry %q", key)
	}
}

func ptr(v string) *string { return &v }
