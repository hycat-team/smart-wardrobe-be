package repositories

import (
	"testing"
)

// Value() must return string (not []byte): pgx simple_protocol (FR-046) encodes
// []byte as bytea, which a jsonb column rejects (SQLSTATE 22P02 / 42601).
func TestJSONFieldValueIsString(t *testing.T) {
	got, err := JSONField(`{"a":1}`).Value()
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := got.(string); !ok {
		t.Fatalf("Value() = %T, want string", got)
	}

	empty, err := JSONField(nil).Value()
	if err != nil || empty != nil {
		t.Fatalf("empty Value() = %v, %v; want nil, nil", empty, err)
	}
}

func TestJSONFieldScan(t *testing.T) {
	var j JSONField
	if err := j.Scan(`{"a":1}`); err != nil {
		t.Fatalf("Scan(string): %v", err)
	}
	if err := j.Scan([]byte(`{"b":2}`)); err != nil {
		t.Fatalf("Scan([]byte): %v", err)
	}
	if err := j.Scan(123); err == nil {
		t.Fatal("Scan(int) should error")
	}
}
