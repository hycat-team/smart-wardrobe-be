package grant

import (
	"testing"
	"time"

	aiexecutor_entities "smart-wardrobe-be/services/ai/internal/modules/aiexecutor/domain/entities"

	"github.com/google/uuid"
)

// ---------------------------------------------------------------------------
// Shared fixtures for an account whose stored policy snapshot cannot be read
// back. All three user stories of 027 need this scenario, so it is built once
// here rather than three times: US1 delivers a late event onto such a grant,
// US2 repairs it, and US3 proves the repair leaves no unreadable row behind.
//
// "Unreadable" is the shape FR-014 describes: the jsonb document has no
// `envelope` (and therefore no `occurredAt`) for ExtractOccurredAt to read, or
// no `entitlements` for decodeEntitlements. The two unreadable builders below
// cover both forms — an empty object and a populated document missing the two
// keys — and the readable builder is their control, proving the fixture itself
// distinguishes readable from unreadable.
//
// These builders only construct data. The repository fake lives in
// grant_uc_test.go and is deliberately not touched here.
// ---------------------------------------------------------------------------

// grantWithSnapshot builds a grant row whose stored policy_snapshot is raw,
// exactly as the ordering check and the repair path will read it. Other columns
// are left at their zero values so a caller can set whichever ones its scenario
// needs.
func grantWithSnapshot(t *testing.T, raw aiexecutor_entities.JSONDocument) *aiexecutor_entities.AIPolicyGrant {
	t.Helper()
	return &aiexecutor_entities.AIPolicyGrant{
		AuditableEntity: aiexecutor_entities.AuditableEntity{
			BaseEntity: aiexecutor_entities.BaseEntity{ID: uuid.New()},
		},
		PolicySnapshot: raw,
	}
}

// grantWithEmptySnapshot is the first unreadable form: an empty JSON object. It
// carries no `envelope`, so ExtractOccurredAt cannot recover an occurredAt, and
// no `entitlements`, so the entitlement reader cannot recover a limit.
func grantWithEmptySnapshot(t *testing.T) *aiexecutor_entities.AIPolicyGrant {
	t.Helper()
	return grantWithSnapshot(t, aiexecutor_entities.JSONDocument(`{}`))
}

// grantWithSnapshotMissingEnvelope is the second unreadable form: a document
// that carries data but neither `envelope` nor `entitlements`. It is not the
// empty-document case — the payload is non-trivial and still unreadable, which
// is what the production rows looked like — and it pins that the unreadability
// comes from the missing keys rather than from the document being empty.
func grantWithSnapshotMissingEnvelope(t *testing.T) *aiexecutor_entities.AIPolicyGrant {
	t.Helper()
	return grantWithSnapshot(t, aiexecutor_entities.JSONDocument(
		`{"planCode":"premium-monthly","tierRank":2,"note":"written before the envelope existed"}`,
	))
}

// snapshotWithOccurredAt builds a readable two-part snapshot whose envelope
// carries the given occurred_at and all four entitlements. The ordering floor is
// read from this envelope whenever it is present (level 1), so a test that needs a
// specific occurred_at builds the prior row's snapshot from here rather than
// letting it fall through to created_at.
func snapshotWithOccurredAt(t *testing.T, occurredAt time.Time) aiexecutor_entities.JSONDocument {
	t.Helper()

	entitlements := make(map[string]OperationEntitlement, len(entitlementOperations))
	for _, op := range entitlementOperations {
		entitlements[string(op)] = OperationEntitlement{
			NormalMaxInputTokens:  1000,
			NormalMaxOutputTokens: 1000,
		}
	}
	raw, err := (&PolicySnapshot{
		Envelope: SnapshotEnvelope{
			EventVersion: 2,
			OccurredAt:   occurredAt,
			EffectiveAt:  occurredAt,
		},
		Entitlements: entitlements,
	}).Marshal()
	if err != nil {
		t.Fatalf("marshal the readable snapshot: %v", err)
	}
	return raw
}

// grantWithReadableSnapshot is the control: the same kind of row carrying a
// complete document, with an `envelope` (so ExtractOccurredAt succeeds) and all
// four entitlements (so the entitlement reader does not degrade). A test that
// must tell "broken snapshot" apart from "healthy snapshot" builds one of each
// from this file.
func grantWithReadableSnapshot(t *testing.T) *aiexecutor_entities.AIPolicyGrant {
	t.Helper()
	return grantWithSnapshot(t, snapshotWithOccurredAt(t, windowBase))
}
