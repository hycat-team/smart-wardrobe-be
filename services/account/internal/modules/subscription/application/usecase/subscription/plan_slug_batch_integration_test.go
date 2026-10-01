package subscription

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/entities"
	"smart-wardrobe-be/services/account/internal/modules/subscription/infrastructure/persistence"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// slugQueryBatchSize exercises the batch lookup against a table big enough that a
// sequential implementation would be visibly slower, while staying quick.
const slugQueryBatchSize = 50

// TestIntegrationGetBySlugsAnyStatusResolvesTheWholeSetInOneQuery covers the SQL
// that 024 T150 added, which no unit test can reach: `slug IN (...)` against real
// Postgres.
//
// The unit test proves the grant path ASKS for one query. Only this proves the
// query returns what the caller expects - notably that it does not filter on
// `is_active`, which is what keeps `plan_inactive` distinguishable from
// `plan_not_found`.
func TestIntegrationGetBySlugsAnyStatusResolvesTheWholeSetInOneQuery(t *testing.T) {
	dbConn := openSubscriptionTestDB(t)
	cleanupSubscriptionTables(t, dbConn)

	ctx := context.Background()
	planRepo := persistence.NewSubscriptionPlanRepository(dbConn)

	active := make([]string, 0, slugQueryBatchSize)
	deactivated := make([]string, 0, 10)
	for i := 0; i < slugQueryBatchSize; i++ {
		slug := fmt.Sprintf("batch-active-%02d", i)
		active = append(active, slug)
		if err := planRepo.Create(ctx, &entities.SubscriptionPlan{
			Slug: slug, Name: slug, IsActive: true, Price: decimal.Zero,
		}); err != nil {
			t.Fatalf("seed plan %s: %v", slug, err)
		}
	}
	for i := 0; i < 10; i++ {
		slug := fmt.Sprintf("batch-off-%02d", i)
		deactivated = append(deactivated, slug)
		plan := &entities.SubscriptionPlan{
			Slug: slug, Name: slug, IsActive: true, Price: decimal.Zero,
		}
		if err := planRepo.Create(ctx, plan); err != nil {
			t.Fatalf("seed deactivated plan %s: %v", slug, err)
		}
		// `SubscriptionPlan.IsActive` carries `gorm:"...;default:true"`, and GORM
		// drops a field from the INSERT when its value is the type's zero value and
		// the column declares a default. `false` is the zero value for bool, so
		// asking for an inactive plan at construction time silently produced an
		// ACTIVE one - which is precisely the state this test needs to be absent.
		//
		// Setting it afterwards is the only way to actually express "inactive".
		if err := dbConn.Model(&entities.SubscriptionPlan{}).
			Where("slug = ?", slug).
			Update("is_active", false).Error; err != nil {
			t.Fatalf("deactivate plan %s: %v", slug, err)
		}
	}

	// A slug that does not exist at all, to pin the "absent" contract.
	requested := make([]string, 0, len(active)+len(deactivated)+1)
	requested = append(requested, active...)
	requested = append(requested, deactivated...)
	requested = append(requested, "batch-does-not-exist")

	found, err := planRepo.GetBySlugsAnyStatus(ctx, requested)
	if err != nil {
		t.Fatalf("GetBySlugsAnyStatus: %v", err)
	}

	if len(found) != len(active)+len(deactivated) {
		t.Errorf("expected %d plans, got %d", len(active)+len(deactivated), len(found))
	}

	// The behaviour T135 depends on: a deactivated plan must come BACK. If this
	// lookup filtered on is_active the caller could no longer tell "you deactivated
	// this plan" from "you mistyped the slug".
	for _, slug := range deactivated {
		plan, ok := found[slug]
		if !ok {
			t.Errorf("plan đã vô hiệu hóa %s phải vẫn được trả về", slug)
			continue
		}
		if plan.IsActive {
			t.Errorf("plan %s bị trả về với IsActive=true", slug)
		}
	}

	for _, slug := range active {
		plan, ok := found[slug]
		if !ok {
			t.Errorf("plan đang hoạt động %s phải được trả về", slug)
			continue
		}
		if plan.ID == uuid.Nil {
			t.Errorf("plan %s trả về mà không có ID", slug)
		}
	}

	// A missing slug is ABSENT rather than present-and-nil, because that is the
	// distinction the plan_not_found reason code depends on.
	if _, present := found["batch-does-not-exist"]; present {
		t.Error("mã gọi không tồn tại phải vắng hẳn khỏi map, không được có mặt với giá trị nil")
	}

	// An empty set must not produce `IN (NULL)` or an empty-slug match.
	empty, err := planRepo.GetBySlugsAnyStatus(ctx, nil)
	if err != nil {
		t.Fatalf("GetBySlugsAnyStatus(nil): %v", err)
	}
	if len(empty) != 0 {
		t.Errorf("danh sách rỗng phải trả về map rỗng, thực tế = %d mục", len(empty))
	}
}

// The batch must not silently ignore case. Slugs arrive lowercased from the
// grant path, but the column is not declared case-insensitive, so a mixed-case
// request is expected to find nothing rather than to match by accident.
func TestIntegrationGetBySlugsAnyStatusIsCaseSensitiveAgainstTheColumn(t *testing.T) {
	dbConn := openSubscriptionTestDB(t)
	cleanupSubscriptionTables(t, dbConn)

	ctx := context.Background()
	planRepo := persistence.NewSubscriptionPlanRepository(dbConn)
	if err := planRepo.Create(ctx, &entities.SubscriptionPlan{
		Slug: "case-plan", Name: "Case Plan", IsActive: true, Price: decimal.Zero,
	}); err != nil {
		t.Fatalf("seed plan: %v", err)
	}

	exact, err := planRepo.GetBySlugsAnyStatus(ctx, []string{"case-plan"})
	if err != nil {
		t.Fatalf("exact lookup: %v", err)
	}
	if len(exact) != 1 {
		t.Fatalf("tra cứu đúng ký tự phải thấy 1 plan, thực tế = %d", len(exact))
	}

	mixed, err := planRepo.GetBySlugsAnyStatus(ctx, []string{"CASE-PLAN"})
	if err != nil {
		t.Fatalf("mixed-case lookup: %v", err)
	}
	if len(mixed) != 0 {
		t.Errorf("cột slug phân biệt hoa thường, tra cứu sai hoa thường phải không ra kết quả, thực tế = %d", len(mixed))
	}
}

// A slug longer than the column is refused before it can reach SQL as a value that
// cannot use the index. This mirrors the guard in the grant path and checks the
// repository does not paper over it.
func TestIntegrationGetBySlugsAnyStatusIgnoresUnindexableSlugWithoutErroring(t *testing.T) {
	dbConn := openSubscriptionTestDB(t)
	cleanupSubscriptionTables(t, dbConn)

	ctx := context.Background()
	planRepo := persistence.NewSubscriptionPlanRepository(dbConn)
	if err := planRepo.Create(ctx, &entities.SubscriptionPlan{
		Slug: "findable", Name: "Findable", IsActive: true, Price: decimal.Zero,
	}); err != nil {
		t.Fatalf("seed plan: %v", err)
	}

	found, err := planRepo.GetBySlugsAnyStatus(ctx, []string{strings.Repeat("z", 300), "findable"})
	if err != nil {
		t.Fatalf("a single oversized slug must not fail the whole query: %v", err)
	}
	if len(found) != 1 {
		t.Errorf("mã gọi quá dài không được chặn phần còn lại, thực tế = %d mục", len(found))
	}
}
