package persistence

import (
	"context"
	"testing"

	"smart-wardrobe-be/services/commerce/internal/modules/fashion/domain/entities"
)

// 033 T022 - the repository half of fashion.user_style_profiles.
//
// The harness (openPersonalizationTestDB, the cross-package advisory lock and
// the identity fixture) lives in personalization_schema_test.go, which owns the
// DDL assertions; the repository tests are kept in their own file so neither
// grows past the project's file-length guideline.

// TestIntegrationUserStyleProfileRepositoryInsertAndRead covers the base read
// path: a user without a profile reads as nil (not an error), and an upsert
// round-trips the declared preferences.
func TestIntegrationUserStyleProfileRepositoryInsertAndRead(t *testing.T) {
	dbConn := openPersonalizationTestDB(t)
	repo := NewUserStyleProfileRepository(dbConn)
	ctx := context.Background()

	userID := createPersonalizationTestUser(t, dbConn)

	if profile, err := repo.GetByUserID(ctx, userID); err != nil {
		t.Fatalf("read a missing profile: %v", err)
	} else if profile != nil {
		t.Fatalf("GetByUserID returned a row for a user without a profile: %+v", profile)
	}

	if err := repo.Upsert(ctx, &entities.UserStyleProfile{
		UserID:         userID,
		Styles:         entities.StyleList{"minimal", "casual"},
		ExcludedStyles: entities.StyleList{"sporty"},
		Enabled:        true,
		LearnedStyles:  entities.StyleWeightMap{},
	}); err != nil {
		t.Fatalf("insert the profile: %v", err)
	}
	created, err := repo.GetByUserID(ctx, userID)
	if err != nil || created == nil {
		t.Fatalf("read the profile back: profile=%v err=%v", created, err)
	}
	if len(created.Styles) != 2 || created.ExcludedStyles[0] != "sporty" || !created.Enabled {
		t.Fatalf("stored profile does not match the input: %+v", created)
	}
}

// TestIntegrationUserStyleProfileRepositoryLearnedUpdateCreatesRow covers the
// cold path of D7: the first signal may arrive before the user ever opened the
// profile screen, so the locked update has to create the row with defaults.
func TestIntegrationUserStyleProfileRepositoryLearnedUpdateCreatesRow(t *testing.T) {
	dbConn := openPersonalizationTestDB(t)
	repo := NewUserStyleProfileRepository(dbConn)
	ctx := context.Background()

	userID := createPersonalizationTestUser(t, dbConn)

	learned, err := repo.UpdateLearnedWithLock(ctx, userID, func(profile *entities.UserStyleProfile) error {
		profile.LearnedStyles = entities.StyleWeightMap{"minimal": 0.5}
		return nil
	})
	if err != nil {
		t.Fatalf("UpdateLearnedWithLock on a user without a profile: %v", err)
	}
	if learned == nil || !learned.Enabled {
		t.Fatalf("the lazily created profile must be enabled with defaults: %+v", learned)
	}
	if learned.LearnedAt == nil {
		t.Fatal("UpdateLearnedWithLock did not stamp learned_at on the created row")
	}
	if len(learned.Styles) != 0 || len(learned.ExcludedStyles) != 0 {
		t.Fatalf("the created row must start with empty declared preferences: %+v", learned)
	}
}

// TestIntegrationUserStyleProfileRepositoryUpsertKeepsLearnedState covers the
// separation D7 requires: a PUT of the explicit preferences updates styles /
// excluded / enabled and leaves learned_* (and created_at) untouched, while the
// locked learned update leaves the explicit preferences untouched.
func TestIntegrationUserStyleProfileRepositoryUpsertKeepsLearnedState(t *testing.T) {
	dbConn := openPersonalizationTestDB(t)
	repo := NewUserStyleProfileRepository(dbConn)
	ctx := context.Background()

	userID := createPersonalizationTestUser(t, dbConn)
	if err := repo.Upsert(ctx, &entities.UserStyleProfile{
		UserID:         userID,
		Styles:         entities.StyleList{"minimal", "casual"},
		ExcludedStyles: entities.StyleList{"sporty"},
		Enabled:        true,
		LearnedStyles:  entities.StyleWeightMap{},
	}); err != nil {
		t.Fatalf("insert the profile: %v", err)
	}
	created, err := repo.GetByUserID(ctx, userID)
	if err != nil || created == nil {
		t.Fatalf("read the profile back: profile=%v err=%v", created, err)
	}

	learned, err := repo.UpdateLearnedWithLock(ctx, userID, func(profile *entities.UserStyleProfile) error {
		profile.LearnedStyles = entities.StyleWeightMap{"minimal": 1.0, "casual": 0.4}
		return nil
	})
	if err != nil {
		t.Fatalf("UpdateLearnedWithLock: %v", err)
	}
	if learned.LearnedAt == nil {
		t.Fatal("UpdateLearnedWithLock did not stamp learned_at")
	}
	if len(learned.Styles) != 2 {
		t.Fatalf("the locked update lost the explicit styles: %+v", learned.Styles)
	}

	createdAt := created.CreatedAt
	if err := repo.Upsert(ctx, &entities.UserStyleProfile{
		UserID:         userID,
		Styles:         entities.StyleList{"formal"},
		ExcludedStyles: entities.StyleList{},
		Enabled:        false,
	}); err != nil {
		t.Fatalf("upsert over an existing profile: %v", err)
	}
	updated, err := repo.GetByUserID(ctx, userID)
	if err != nil || updated == nil {
		t.Fatalf("read the profile after the second upsert: profile=%v err=%v", updated, err)
	}
	if len(updated.Styles) != 1 || updated.Styles[0] != "formal" {
		t.Fatalf("the upsert did not apply the new styles: %+v", updated)
	}
	// The user must be able to turn the profile off without deleting it (FR-012).
	if updated.Enabled {
		t.Fatalf("the upsert did not persist enabled=false: %+v", updated)
	}
	if _, ok := updated.LearnedStyles["minimal"]; !ok {
		t.Fatalf("the upsert wiped the learned styles: %+v", updated.LearnedStyles)
	}
	if updated.LearnedAt == nil {
		t.Fatal("the upsert wiped learned_at")
	}
	if !updated.CreatedAt.Equal(createdAt) {
		t.Fatalf("the upsert rewrote created_at: got %s want %s", updated.CreatedAt, createdAt)
	}

	if err := repo.Delete(ctx, userID); err != nil {
		t.Fatalf("delete the profile: %v", err)
	}
	if profile, err := repo.GetByUserID(ctx, userID); err != nil {
		t.Fatalf("read the profile after delete: %v", err)
	} else if profile != nil {
		t.Fatalf("the profile survived Delete: %+v", profile)
	}
}
