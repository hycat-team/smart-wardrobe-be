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

func TestUserGetEffectiveBodyProfileFallsBackToInference(t *testing.T) {
	user := &User{
		BodyProfile: &bodyProfile{
			HeightCM:       170,
			WeightKG:       60,
			VerifiedByUser: false,
			InferredByAI: &inferredBodyProfile{
				BodyShape: "hourglass",
			},
		},
	}

	effective := user.GetEffectiveBodyProfile()
	if effective == nil {
		t.Fatal("expected effective body profile")
	}
	if effective.BodyShape != "hourglass" {
		t.Fatalf("expected inferred body shape fallback, got %q", effective.BodyShape)
	}
}
