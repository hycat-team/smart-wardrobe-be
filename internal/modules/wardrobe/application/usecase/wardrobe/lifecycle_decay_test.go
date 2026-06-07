package wardrobe

import (
	"math"
	"testing"
	"time"
)

func TestCalculateLifecycleDecayFactor(t *testing.T) {
	now := time.Date(2026, 6, 7, 0, 0, 0, 0, time.UTC)
	createdAt := now.AddDate(0, 0, -30)

	if got := CalculateLifecycleDecayFactor(nil, createdAt, now); got != 1.0 {
		t.Fatalf("expected decay factor 1.0 before grace period, got %v", got)
	}

	lastUsedAt := now.AddDate(0, 0, -180)
	if got := CalculateLifecycleDecayFactor(&lastUsedAt, createdAt, now); got != 1.0 {
		t.Fatalf("expected decay factor 1.0 at grace period boundary, got %v", got)
	}

	oldLastUsedAt := now.AddDate(0, 0, -210)
	got := CalculateLifecycleDecayFactor(&oldLastUsedAt, createdAt, now)
	want := math.Exp(-garmentDecayLambda * 30)
	if math.Abs(got-want) > 0.000001 {
		t.Fatalf("expected decay factor %v, got %v", want, got)
	}
}
