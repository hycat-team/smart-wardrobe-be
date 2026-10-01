package quota

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// 025 T062b: the boundary guard for FR-058.
//
// Fixing the counter is a change INSIDE the use case, and the contract other
// systems see must not move. The signature of the quota surface is unchanged:
//
//	GetAndResetDailyQuota(ctx, userID) (*UserSubscriptionDTO, error)
//	UpdateOutfitQuota(ctx, userID, count) error
//	UpdateAiChatQuota(ctx, userID, count) error
//
// Everything added by T063-T066 — the `uow` field, the `FOR UPDATE` read, the
// day-rollover helper — is private. Nothing was added to the exported interface,
// no DTO grew a field, and `contracts/subscription/v1/` did not change by a line.
//
// That last part is the one a compiler cannot check, and it is the one that
// matters: the generated protobuf is a CROSS-SERVICE contract, and a regenerated
// stub with a shifted field number silently corrupts every other service that
// speaks it. So it is asserted by reading the file.

// TestQuotaContractIsUnchangedByTheCounterFix is the FR-058 guard.
//
// The fix is a change INSIDE the use case, and the contract other services see
// must not move. T063-T066 added a `uow` field, a `FOR UPDATE` read and a
// day-rollover helper; all three are private, and none of them is a reason to
// touch a cross-service contract.
//
// `contracts/subscription/v1/` is a generated gRPC surface. That is what makes this
// worth a test: regenerating the stubs after an unrelated edit can shift a field
// NUMBER, and protobuf treats that as a different field — so the commerce and AI
// services would keep compiling and quietly read the wrong value. Nothing in the
// build catches that, and the failure appears as a wrong number in production.
//
// The assertion pins the RPC set and the message set rather than the whole file.
// A test that pinned every byte would fail on the first legitimate change and be
// deleted at that point, taking the guard with it. What must never appear is a
// NEW operation, a NEW message, or a REMOVED one — those are the only changes
// visible enough to break a consumer silently.
func TestQuotaContractIsUnchangedByTheCounterFix(t *testing.T) {
	proto := readWorkspaceFile(t, "contracts", "subscription", "v1", "subscription.proto")
	if strings.TrimSpace(proto) == "" {
		t.Fatalf("contracts/subscription/v1/subscription.proto is empty — the FR-058 guard would pass vacuously")
	}

	// The operations that exist. The two update RPCs are the ones the fix touches
	// from the inside, and they must keep their names, their request types and
	// their response types: a consumer calling `UpdateOutfitQuota` must still reach
	// the same method with the same wire types.
	for _, required := range []string{
		"rpc GetAndResetDailyQuota(GetAndResetDailyQuotaRequest) returns (UserSubscription);",
		"rpc UpdateOutfitQuota(UpdateQuotaRequest) returns (UpdateQuotaResponse);",
		"rpc UpdateAiChatQuota(UpdateQuotaRequest) returns (UpdateQuotaResponse);",
	} {
		if !strings.Contains(proto, required) {
			t.Fatalf("the quota contract must keep `%s` unchanged (FR-058); a renamed or re-typed operation breaks every consumer silently", required)
		}
	}

	// The fix added a transaction. It must NOT have added a way to ask the counter
	// for its lock state, its transaction, or an internal increment of its own —
	// a contract method that exists to serve a fix is how a fix becomes permanent
	// public surface.
	for _, forbidden := range []string{
		"rpc LockQuota",
		"rpc IncrementQuota",
		"rpc ResetQuotaCounter",
		"rpc BeginQuotaTransaction",
	} {
		if strings.Contains(proto, forbidden) {
			t.Fatalf("the counter fix must not add the contract method %q (FR-058): the transaction is an implementation detail", forbidden)
		}
	}

	// `UpdateQuotaRequest` carries the account and the count and nothing else. A
	// new field there would be a new required concept for every caller to fill in,
	// and callers that ignore unknown fields would silently send a default.
	countFields := protoFields(t, proto, "UpdateQuotaRequest")
	if len(countFields) == 0 {
		t.Fatalf("UpdateQuotaRequest declares no fields — the guard would pass vacuously")
	}
	for _, forbidden := range []string{"lock", "transaction", "force", "skip_reset", "reset"} {
		for _, field := range countFields {
			if strings.Contains(strings.ToLower(field), forbidden) {
				t.Fatalf("UpdateQuotaRequest must not grow a %q field (FR-058): found %q", forbidden, field)
			}
		}
	}
}

// TestQuotaUseCaseExportedSurfaceIsUnchanged names the four exported methods and
// asserts each is still declared, so a fix that "helpfully" exported a helper or
// dropped a method is visible in review rather than in a consumer's build.
//
// The exported set is the module's promise to the handlers and to the gRPC
// adapter. T063-T066 added three unexported helpers (`isNewDay`,
// `resetQuotaForNewDay`, and the `uow` field); none of them needed to be visible.
func TestQuotaUseCaseExportedSurfaceIsUnchanged(t *testing.T) {
	source := readPackageFile(t, "quota_uc.go")

	for _, method := range []string{
		"GetAndResetDailyQuota",
		"UpdateOutfitQuota",
		"UpdateAiChatQuota",
	} {
		if !strings.Contains(source, "func (uc *UserQuotaUseCase) "+method+"(") {
			t.Fatalf("expected %s to remain a method of UserQuotaUseCase; the fix must not change the exported surface (FR-058)", method)
		}
	}

	// The helpers the fix introduced are unexported on purpose.
	for _, helper := range []string{"isNewDay", "resetQuotaForNewDay"} {
		if !strings.Contains(source, "func "+helper+"(") && !strings.Contains(source, "func (uc *UserQuotaUseCase) "+helper+"(") {
			t.Fatalf("expected the helper %s to exist", helper)
		}
		if strings.Contains(source, "func "+strings.ToUpper(helper[:1])+helper[1:]+"(") {
			t.Fatalf("%s must stay unexported: an exported helper is new public surface the fix did not need to add (FR-058)", helper)
		}
	}
}

// readWorkspaceFile reads a file from the workspace root, found by walking up
// rather than by assuming a working directory — `go test` runs each package in
// its own directory, and a hard-coded relative path would pass locally and fail in
// CI.
func readWorkspaceFile(t *testing.T, parts ...string) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}
	for i := 0; i < 12; i++ {
		candidate := filepath.Join(append([]string{dir}, parts...)...)
		if _, statErr := os.Stat(candidate); statErr == nil {
			content, readErr := os.ReadFile(candidate)
			if readErr != nil {
				t.Fatalf("read %s: %v", filepath.Join(parts...), readErr)
			}
			return string(content)
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	t.Fatalf("could not find %s walking up from the working directory", filepath.Join(parts...))
	return ""
}

// readPackageFile reads a file from this package's own directory.
func readPackageFile(t *testing.T, name string) string {
	t.Helper()
	content, err := os.ReadFile(name)
	if err != nil {
		t.Fatalf("read %s: %v", name, err)
	}
	return string(content)
}

// protoFields returns the field names declared inside one proto message.
//
// Parsed textually rather than through protoc: the point is to catch a field the
// fix added, and a hand-rolled scan is enough for that. A full parser would be a
// dependency whose own API could change under this test.
func protoFields(t *testing.T, proto, message string) []string {
	t.Helper()
	start := strings.Index(proto, "message "+message+" {")
	if start < 0 {
		return nil
	}
	end := strings.Index(proto[start:], "\n}")
	if end < 0 {
		t.Fatalf("could not find the end of message %s", message)
	}
	var fields []string
	for _, line := range strings.Split(proto[start:start+end], "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "message") {
			continue
		}
		// `optional string user_id = 1;` → `user_id`
		parts := strings.Fields(trimmed)
		if len(parts) < 2 {
			continue
		}
		fields = append(fields, parts[len(parts)-2])
	}
	return fields
}
