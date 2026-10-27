package dto

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
)

// 033 T024 (D7) - the outfit-save payload is the contract between Commerce's
// outbox publisher and the style learning worker. It travels through RabbitMQ
// as JSON, so the field names on the wire are part of the contract: a rename
// here silently stops the worker from learning anything, with no compile error
// anywhere in between.

func TestOutfitSavedEventPayloadMarshalsCamelCaseContract(t *testing.T) {
	signalID := uuid.New()
	occurredAt := time.Date(2026, 10, 27, 9, 15, 30, 0, time.UTC)
	payload := OutfitSavedEventPayload{
		SignalID:       signalID,
		UserID:         uuid.New(),
		OutfitID:       uuid.New(),
		FashionItemIDs: []uuid.UUID{uuid.New(), uuid.New()},
		Source:         "outfit_saved",
		OccurredAt:     occurredAt,
	}

	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal the payload: %v", err)
	}

	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		t.Fatalf("decode the payload as a generic object: %v", err)
	}
	wantKeys := []string{"signalId", "userId", "outfitId", "fashionItemIds", "source", "occurredAt"}
	for _, key := range wantKeys {
		if _, ok := fields[key]; !ok {
			t.Fatalf("payload is missing %q: %s", key, string(raw))
		}
	}
	if len(fields) != len(wantKeys) {
		t.Fatalf("payload carries %d keys (%s), want exactly %d", len(fields), string(raw), len(wantKeys))
	}

	// signalId must stay the UUID string the worker turns back into the
	// idempotency key of fashion.style_signals.id.
	if got := string(fields["signalId"]); got != `"`+signalID.String()+`"` {
		t.Fatalf("signalId = %s, want %q", got, signalID.String())
	}
	if got := string(fields["source"]); got != `"outfit_saved"` {
		t.Fatalf("source = %s, want %q", got, "outfit_saved")
	}

	// The timestamps cross the wire as RFC 3339 with the offset preserved.
	if got := string(fields["occurredAt"]); got != `"2026-10-27T09:15:30Z"` {
		t.Fatalf("occurredAt = %s, want RFC 3339 UTC", got)
	}
}

func TestOutfitSavedEventPayloadRoundTrip(t *testing.T) {
	itemIDs := []uuid.UUID{uuid.New(), uuid.New(), uuid.New()}
	occurredAt := time.Now().UTC().Truncate(time.Microsecond)
	original := OutfitSavedEventPayload{
		SignalID:       uuid.New(),
		UserID:         uuid.New(),
		OutfitID:       uuid.New(),
		FashionItemIDs: itemIDs,
		Source:         "outfit_saved",
		OccurredAt:     occurredAt,
	}

	raw, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal the payload: %v", err)
	}
	var decoded OutfitSavedEventPayload
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("unmarshal the payload: %v", err)
	}

	if decoded.SignalID != original.SignalID {
		t.Fatalf("signalId round-trip = %s, want %s", decoded.SignalID, original.SignalID)
	}
	if decoded.UserID != original.UserID || decoded.OutfitID != original.OutfitID {
		t.Fatalf("identity round-trip mismatch: %+v", decoded)
	}
	if decoded.Source != original.Source {
		t.Fatalf("source round-trip = %q, want %q", decoded.Source, original.Source)
	}
	if !decoded.OccurredAt.Equal(occurredAt) {
		t.Fatalf("occurredAt round-trip = %s, want %s", decoded.OccurredAt, occurredAt)
	}
	if len(decoded.FashionItemIDs) != len(itemIDs) {
		t.Fatalf("fashionItemIds round-trip kept %d of %d ids", len(decoded.FashionItemIDs), len(itemIDs))
	}
	for i := range itemIDs {
		if decoded.FashionItemIDs[i] != itemIDs[i] {
			t.Fatalf("fashionItemIds[%d] = %s, want %s (the order is part of the snapshot)",
				i, decoded.FashionItemIDs[i], itemIDs[i])
		}
	}
}

// TestOutfitSavedEventPayloadIgnoresUnknownFields documents the rollout
// direction: a worker redeployed after a publisher that added a field must
// still learn from the payload it understands.
func TestOutfitSavedEventPayloadIgnoresUnknownFields(t *testing.T) {
	original := OutfitSavedEventPayload{
		SignalID:       uuid.New(),
		UserID:         uuid.New(),
		OutfitID:       uuid.New(),
		FashionItemIDs: []uuid.UUID{uuid.New()},
		Source:         "outfit_saved",
		OccurredAt:     time.Date(2026, 10, 27, 9, 15, 30, 0, time.UTC),
	}
	raw, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal the payload: %v", err)
	}

	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		t.Fatalf("decode the payload: %v", err)
	}
	fields["learnedWeight"] = json.RawMessage(`0.25`)
	extended, err := json.Marshal(fields)
	if err != nil {
		t.Fatalf("re-encode the extended payload: %v", err)
	}

	var decoded OutfitSavedEventPayload
	if err := json.Unmarshal(extended, &decoded); err != nil {
		t.Fatalf("an unknown field must not break decoding: %v", err)
	}
	if decoded.SignalID != original.SignalID || len(decoded.FashionItemIDs) != 1 {
		t.Fatalf("known fields were lost while ignoring the unknown one: %+v", decoded)
	}
}
