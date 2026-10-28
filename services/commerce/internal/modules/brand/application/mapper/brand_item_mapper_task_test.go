package mapper

import (
	"testing"

	"smart-wardrobe-be/services/commerce/internal/modules/brand/domain/constants/branditem/branditemstatus"
	"smart-wardrobe-be/services/commerce/internal/modules/brand/domain/entities"

	"github.com/google/uuid"
)

func TestMapToBrandItemRes_TaskIDOnlyForDraft(t *testing.T) {
	taskID := uuid.New()

	draft := &entities.BrandItem{Status: branditemstatus.Draft, TaskID: &taskID}
	if res := MapToBrandItemRes(draft, nil); res.TaskID == nil {
		t.Fatal("expected task id on draft item")
	}

	for _, status := range []branditemstatus.BrandItemStatus{
		branditemstatus.Active,
		branditemstatus.Failed,
		branditemstatus.Archived,
	} {
		item := &entities.BrandItem{Status: status, TaskID: &taskID}
		if res := MapToBrandItemRes(item, nil); res.TaskID != nil {
			t.Fatalf("expected task id hidden for status %s", status)
		}
	}
}
