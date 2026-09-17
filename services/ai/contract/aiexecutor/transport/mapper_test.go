package transport_test

import (
	"testing"
	"time"

	aiv1 "smart-wardrobe-be/contracts/ai/v1"
	aitransport "smart-wardrobe-be/services/ai/contract/aiexecutor/transport"
	aidto "smart-wardrobe-be/services/ai/dto/aiexecutor"

	"github.com/google/uuid"
)

func TestOptionsJSONRoundTrip(t *testing.T) {
	original := aidto.OrchestratorOptions{
		MaxOutputTokens:  1024,
		Temperature:      0.7,
		ResponseMIMEType: "application/json",
		ResponseSchema:   map[string]any{"type": "object"},
		UserID:           uuid.New(),
		Operation:        "chat",
	}

	wire, err := aitransport.OptionsToProto(original)
	if err != nil {
		t.Fatalf("to proto: %v", err)
	}
	back, err := aitransport.OptionsFromProto(wire)
	if err != nil {
		t.Fatalf("from proto: %v", err)
	}

	if back.MaxOutputTokens != original.MaxOutputTokens || back.Temperature != original.Temperature {
		t.Fatalf("options mismatch: %+v", back)
	}
	if back.UserID != original.UserID || back.Operation != original.Operation {
		t.Fatalf("scope mismatch: %+v", back)
	}
	schema, ok := back.ResponseSchema.(map[string]any)
	if !ok || schema["type"] != "object" {
		t.Fatalf("schema mismatch: %+v", back.ResponseSchema)
	}
}

func TestAIStatsRoundTrip(t *testing.T) {
	original := &aidto.AIUsageAdminStatsDTO{
		SnapshotDate:         time.Date(2026, 9, 17, 0, 0, 0, 0, time.UTC),
		AIRequestsCount:      1000,
		AIErrorCount:         12,
		AIPaidTokens:         500000,
		AIActualCostMicroVND: 250000,
	}

	wire := aitransport.StatsFromDTO(original)
	back := aitransport.AdminStatsToDTO(wire)

	if back.AIRequestsCount != original.AIRequestsCount || back.AIPaidTokens != original.AIPaidTokens {
		t.Fatalf("stats mismatch: %+v", back)
	}
	if !back.SnapshotDate.Equal(original.SnapshotDate) {
		t.Fatalf("snapshot mismatch: %+v", back.SnapshotDate)
	}
}

func TestEmbeddingsRoundTrip(t *testing.T) {
	wire := []*aiv1.EmbeddingVector{
		{Values: []float32{0.1, 0.2, 0.3}},
		{Values: []float32{0.4, 0.5}},
	}

	// Embeddings cross only as response payloads; verify the From direction
	// preserves values exactly.
	back := aitransport.EmbeddingsFromProto(wire)
	if len(back) != 2 || len(back[0]) != 3 || back[1][1] != 0.5 {
		t.Fatalf("embeddings mismatch: %+v", back)
	}
}
