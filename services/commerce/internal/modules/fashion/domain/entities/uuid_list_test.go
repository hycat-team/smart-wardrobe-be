package entities

import (
	"testing"

	"github.com/google/uuid"
)

// Value must produce a Postgres array literal as a string (not []byte): a
// []byte driver value is encoded as bytea under the simple protocol and a
// uuid[] column rejects it.
func TestUUIDListValue(t *testing.T) {
	first := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	second := uuid.MustParse("22222222-2222-2222-2222-222222222222")

	cases := []struct {
		name string
		list UUIDList
		want any
	}{
		{name: "nil is SQL NULL", list: nil, want: nil},
		{name: "empty", list: UUIDList{}, want: "{}"},
		{name: "one", list: UUIDList{first}, want: "{11111111-1111-1111-1111-111111111111}"},
		{
			name: "many",
			list: UUIDList{first, second},
			want: "{11111111-1111-1111-1111-111111111111,22222222-2222-2222-2222-222222222222}",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := tc.list.Value()
			if err != nil {
				t.Fatalf("Value() error: %v", err)
			}
			if got != tc.want {
				t.Fatalf("Value() = %#v, want %#v", got, tc.want)
			}
			if _, ok := got.([]byte); ok {
				t.Fatal("Value() returned []byte; it must return a string")
			}
		})
	}
}

func TestUUIDListScan(t *testing.T) {
	first := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	second := uuid.MustParse("22222222-2222-2222-2222-222222222222")

	t.Run("null", func(t *testing.T) {
		var got UUIDList
		if err := got.Scan(nil); err != nil {
			t.Fatalf("Scan(nil) error: %v", err)
		}
		if got != nil {
			t.Fatalf("Scan(nil) = %#v, want nil", got)
		}
	})

	t.Run("empty", func(t *testing.T) {
		var got UUIDList
		if err := got.Scan("{}"); err != nil {
			t.Fatalf("Scan(\"{}\") error: %v", err)
		}
		if got == nil || len(got) != 0 {
			t.Fatalf("Scan(\"{}\") = %#v, want an empty non-nil list", got)
		}
	})

	t.Run("string", func(t *testing.T) {
		var got UUIDList
		if err := got.Scan("{11111111-1111-1111-1111-111111111111}"); err != nil {
			t.Fatalf("Scan(string) error: %v", err)
		}
		if len(got) != 1 || got[0] != first {
			t.Fatalf("Scan(string) = %#v, want [%s]", got, first)
		}
	})

	t.Run("bytes", func(t *testing.T) {
		var got UUIDList
		if err := got.Scan([]byte("{11111111-1111-1111-1111-111111111111,22222222-2222-2222-2222-222222222222}")); err != nil {
			t.Fatalf("Scan([]byte) error: %v", err)
		}
		if len(got) != 2 || got[0] != first || got[1] != second {
			t.Fatalf("Scan([]byte) = %#v, want [%s %s]", got, first, second)
		}
	})

	t.Run("malformed", func(t *testing.T) {
		cases := []any{
			"11111111-1111-1111-1111-111111111111",
			"{not-a-uuid}",
			"{11111111-1111-1111-1111-111111111111,oops}",
			42,
		}
		for _, value := range cases {
			var got UUIDList
			if err := got.Scan(value); err == nil {
				t.Fatalf("Scan(%#v) accepted a malformed value: %#v", value, got)
			}
		}
	})
}

func TestUUIDListValueScanRoundTrip(t *testing.T) {
	original := UUIDList{uuid.New(), uuid.New(), uuid.New()}
	value, err := original.Value()
	if err != nil {
		t.Fatalf("Value() error: %v", err)
	}

	var restored UUIDList
	if err := restored.Scan(value); err != nil {
		t.Fatalf("Scan() error: %v", err)
	}
	if len(restored) != len(original) {
		t.Fatalf("round trip length = %d, want %d", len(restored), len(original))
	}
	for i := range original {
		if restored[i] != original[i] {
			t.Fatalf("round trip element %d = %s, want %s", i, restored[i], original[i])
		}
	}
}
