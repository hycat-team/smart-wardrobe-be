package entities

import (
	"database/sql/driver"
	"encoding/json"
	"testing"
)

func TestBodyProfileUnmarshalLegacyShape(t *testing.T) {
	payload := []byte(`{
		"height": 165,
		"weight": 52,
		"bodyType": "pear",
		"estimatedBodyShape": "rectangle"
	}`)

	var profile bodyProfile
	if err := json.Unmarshal(payload, &profile); err != nil {
		t.Fatalf("expected legacy body profile to unmarshal: %v", err)
	}

	if profile.HeightCM != 165 {
		t.Fatalf("expected height 165, got %v", profile.HeightCM)
	}
	if profile.WeightKG != 52 {
		t.Fatalf("expected weight 52, got %v", profile.WeightKG)
	}
	if profile.BodyShape != "pear" {
		t.Fatalf("expected body shape pear, got %q", profile.BodyShape)
	}
	if profile.InferredByAI == nil || profile.InferredByAI.BodyShape != "rectangle" {
		t.Fatalf("expected inferred body shape rectangle, got %+v", profile.InferredByAI)
	}
}

// Value() must return string (not []byte): pgx simple_protocol encodes []byte
// as bytea, which a jsonb column rejects (SQLSTATE 22P02 / 42601).
func TestJSONBValuesAreStrings(t *testing.T) {
	cases := map[string]driver.Valuer{
		"JSONDocument":    JSONDocument(`{"a":1}`),
		"bodyProfile":     bodyProfile{BodyShape: "pear"},
		"preferredColors": preferredColors{"red"},
	}
	for name, v := range cases {
		got, err := v.Value()
		if err != nil {
			t.Fatalf("%s.Value(): %v", name, err)
		}
		s, ok := got.(string)
		if !ok {
			t.Fatalf("%s.Value() = %T, want string", name, got)
		}
		if !json.Valid([]byte(s)) {
			t.Fatalf("%s.Value() = %q, not valid JSON", name, s)
		}
	}
}

func TestJSONBScansAcceptStringAndBytes(t *testing.T) {
	var doc JSONDocument
	if err := doc.Scan(`{"a":1}`); err != nil || len(doc) == 0 {
		t.Fatalf("JSONDocument.Scan(string) = %v, doc=%q", err, doc)
	}
	if err := doc.Scan([]byte(`{"b":2}`)); err != nil || len(doc) == 0 {
		t.Fatalf("JSONDocument.Scan([]byte) = %v, doc=%q", err, doc)
	}

	var profile bodyProfile
	if err := profile.Scan(`{"body_shape":"pear"}`); err != nil || profile.BodyShape != "pear" {
		t.Fatalf("bodyProfile.Scan(string) = %v, profile=%+v", err, profile)
	}
	if err := profile.Scan([]byte(`{"body_shape":"apple"}`)); err != nil || profile.BodyShape != "apple" {
		t.Fatalf("bodyProfile.Scan([]byte) = %v, profile=%+v", err, profile)
	}

	var colors preferredColors
	if err := colors.Scan(`["red"]`); err != nil || len(colors) != 1 || colors[0] != "red" {
		t.Fatalf("preferredColors.Scan(string) = %v, colors=%v", err, colors)
	}
	if err := colors.Scan([]byte(`["blue"]`)); err != nil || len(colors) != 1 || colors[0] != "blue" {
		t.Fatalf("preferredColors.Scan([]byte) = %v, colors=%v", err, colors)
	}
}
