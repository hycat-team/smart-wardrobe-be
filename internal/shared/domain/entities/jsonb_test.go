package entities

import (
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
