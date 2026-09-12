package ordercode

import (
	"testing"
	"time"
)

func TestGenerateStaysBelowMaxSafeInteger(t *testing.T) {
	for i := 0; i < 1000; i++ {
		code := Generate()
		if code <= 0 {
			t.Fatalf("expected positive order code, got %d", code)
		}
		if code > MaxSafeOrderCode {
			t.Fatalf("order code %d exceeds JS MAX_SAFE_INTEGER %d", code, MaxSafeOrderCode)
		}
	}
}

func TestGenerateIsWellAboveLegacySerialRange(t *testing.T) {
	code := Generate()
	if code < 1_000_000_000_000 {
		t.Fatalf("expected time-ordered code far above legacy serial range, got %d", code)
	}
}

func TestGenerateUniquenessComesFromTimeOrdering(t *testing.T) {
	base := time.Date(2026, 9, 12, 0, 0, 0, 0, time.UTC)
	seen := make(map[int64]struct{}, 2000)
	prev := int64(0)
	for i := 0; i < 2000; i++ {
		code := GenerateAt(base.Add(time.Duration(i)*time.Millisecond), int64(i))
		if code <= prev {
			t.Fatalf("expected strictly increasing codes across milliseconds, got %d after %d", code, prev)
		}
		if _, dup := seen[code]; dup {
			t.Fatalf("duplicate order code %d", code)
		}
		seen[code] = struct{}{}
		prev = code
	}
}

func TestRapidGenerationsStayInRange(t *testing.T) {
	for i := 0; i < 2000; i++ {
		code := Generate()
		if code < 1_000_000_000_000 || code > MaxSafeOrderCode {
			t.Fatalf("order code %d out of range", code)
		}
	}
}

func TestGenerateAtIsDeterministic(t *testing.T) {
	now := time.Date(2026, 9, 12, 0, 0, 0, 0, time.UTC)
	want := now.UnixMilli()*1000 + 7
	if got := GenerateAt(now, 7); got != want {
		t.Fatalf("expected %d, got %d", want, got)
	}
	if got := GenerateAt(now, 1007); got != want {
		t.Fatalf("expected suffix modulo 1000 to wrap, want %d got %d", want, got)
	}
}
