package transport_test

import (
	"testing"

	admintransport "smart-wardrobe-be/services/account/contract/admin/transport"
	admindto "smart-wardrobe-be/services/account/dto/admin"

	"github.com/google/uuid"
)

func TestRecordAuditLogRoundTrip(t *testing.T) {
	targetID := uuid.New()
	ip := "203.0.113.7"
	original := admindto.RecordAuditLogInput{
		ActorUserID: uuid.New(),
		Action:      "brand.update",
		TargetType:  "brand",
		TargetID:    &targetID,
		Payload:     map[string]any{"field": "name", "count": float64(3)},
		RequestIP:   &ip,
	}

	wire, err := admintransport.RecordAuditLogInputToProto(original)
	if err != nil {
		t.Fatalf("to proto: %v", err)
	}
	back, err := admintransport.RecordRequestToInput(wire)
	if err != nil {
		t.Fatalf("from proto: %v", err)
	}

	if back.ActorUserID != original.ActorUserID || back.Action != original.Action {
		t.Fatalf("identity fields mismatch: %+v", back)
	}
	if back.TargetID == nil || *back.TargetID != targetID {
		t.Fatalf("target id mismatch: %+v", back.TargetID)
	}
	if back.Payload["field"] != "name" {
		t.Fatalf("payload mismatch: %+v", back.Payload)
	}
	if back.RequestIP == nil || *back.RequestIP != ip {
		t.Fatalf("request ip mismatch: %+v", back.RequestIP)
	}
}
