package entities

import "testing"

// Value() must return string (not []byte): pgx simple_protocol (FR-046) encodes
// []byte as bytea, which the jsonb audit payload column rejects.
func TestJSONDocumentValueIsString(t *testing.T) {
	got, err := JSONDocument{"k": "v"}.Value()
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := got.(string); !ok {
		t.Fatalf("Value() = %T, want string", got)
	}

	nilVal, err := JSONDocument(nil).Value()
	if err != nil || nilVal != "{}" {
		t.Fatalf("nil Value() = %v, %v; want \"{}\"", nilVal, err)
	}
}

func TestJSONDocumentScanString(t *testing.T) {
	var doc JSONDocument
	if err := doc.Scan(`{"a":1}`); err != nil {
		t.Fatalf("Scan(string): %v", err)
	}
	if doc["a"] == nil {
		t.Fatalf("expected key a, got %+v", doc)
	}
}
