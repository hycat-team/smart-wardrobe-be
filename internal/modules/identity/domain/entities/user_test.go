package entities

import (
	"testing"
)

func TestUserGetEffectiveBodyProfileFallsBackToInference(t *testing.T) {
	user := &User{
		BodyProfile: &BodyProfile{
			HeightCM:       170,
			WeightKG:       60,
			VerifiedByUser: false,
			InferredByAI: &InferredBodyProfile{
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
