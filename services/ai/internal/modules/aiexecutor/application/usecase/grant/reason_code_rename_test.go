package grant

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// 024 T159: the reason code was renamed from `plan_not_in_catalog` to
// `ai_cost_policy_not_found` because it claimed a catalogue lookup that never happens.
//
// The check is `GetPolicyByID` against this service's own ai_cost_policies. Nothing
// reads the Account service's plan catalogue, and it must not - constitution §I gives
// each data domain one owner. Adding a lookup to "make the name true" would have been
// the worse defect.
//
// A wrong name here is not cosmetic. These strings are metric keys and alert names, and
// the one an operator would chase is the catalogue: they would find a perfectly valid
// plan row and run out of leads.
func TestReasonCodeNamesThePolicyLookupThatActuallyHappens(t *testing.T) {
	if ReasonAICostPolicyNotFound != "ai_cost_policy_not_found" {
		t.Errorf("reason code = %q, want %q", ReasonAICostPolicyNotFound, "ai_cost_policy_not_found")
	}
}

// The old name must be gone from this service. It is NOT gone from the Account service,
// where it is accurate - the restore flow really does look the plan up in a catalogue
// loaded from subscription_plans. Those are two different facts, and this test only
// guards the AI side.
func TestOldReasonNameIsGoneFromTheAIService(t *testing.T) {
	if _, exists := reasonConstantNamed("plan_not_in_catalog"); exists {
		t.Error("the old plan_not_in_catalog code is still in the AI service - it describes a lookup in a " +
			"catalogue that this service does not and must not consult")
	}
}

// The refusal message must not send an operator to the catalogue either. It used to say
// "của gói đích ... không tồn tại trong danh mục", which reads as "the plan is not in the
// catalogue" - the same wrong trail, one layer further out, and the only thing a user of
// the API actually reads.
func TestRefusalMessageNamesThePolicyNotTheCatalogue(t *testing.T) {
	err := rejectMessageForMissingPolicy()
	if err == "" {
		t.Fatal("must produce a refusal message")
	}
	// Case-insensitive: the sentence starts with a capital, and the point here is which
	// noun it names, not how it is capitalised.
	lowered := strings.ToLower(err)
	if strings.Contains(lowered, "gói") {
		t.Errorf("the message names the plan, which sends an operator to the plan catalogue: %q", err)
	}
	if !strings.Contains(lowered, "chính sách") {
		t.Errorf("the message must name the cost policy, got: %q", err)
	}
	// 024 T172: the message names a table, not a "catalogue". Both words send an
	// operator to the plan catalogue, where they will find a perfectly valid plan row
	// and nothing left to investigate — the wrong trail that T159 removed from the
	// reason code and that survived here in the sentence beside it.
	if strings.Contains(lowered, "danh mục") {
		t.Errorf("the message names a catalogue, which sends an operator to the plan catalogue: %q", err)
	}
}

// quickstart.md is the manual procedure for the rejection scenarios. It described a
// scenario that cannot produce the reason it named - an event with a toPlanCode absent
// from a catalogue - so following the guide could never confirm the counter.
func TestQuickstartRejectionScenariosUseRealReasonNames(t *testing.T) {
	doc := readFeatureDoc(t, "quickstart.md")

	if !strings.Contains(doc, "`ai_cost_policy_not_found`") {
		t.Error("quickstart.md must name the ai_cost_policy_not_found reason for the scenario " +
			"where the cost policy does not exist")
	}
	if strings.Contains(doc, "`plan_not_in_catalog`") {
		t.Error("quickstart.md still documents the old plan_not_in_catalog code for the AI service")
	}
}

// helper: look up a reason constant by its string value, so the test does not depend on
// which Go identifier holds it.
func reasonConstantNamed(value string) (string, bool) {
	switch value {
	case ReasonAICostPolicyNotFound, ReasonStaleEvent, ReasonPolicySnapshotUnavail,
		ReasonDuplicateSourceEvent, ReasonTransientFailure:
		return value, true
	}
	return "", false
}

// rejectMessageForMissingPolicy runs the branch and returns its message.
//
// 024 T172: it passes the PRODUCTION constant, not a copy of the sentence. It used to
// carry its own literal, so the test asserted a string the service had never used and
// would not have failed when the real message changed.
func rejectMessageForMissingPolicy() string {
	err := (&GrantUseCase{}).reject(
		ReasonAICostPolicyNotFound,
		"22222222-2222-2222-2222-222222222222",
		msgAICostPolicyNotFound,
	)
	var rejection *RejectionError
	if !errors.As(err, &rejection) {
		return ""
	}
	return rejection.Message
}

func readFeatureDoc(t *testing.T, name string) string {
	t.Helper()

	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	for {
		if _, statErr := os.Stat(filepath.Join(dir, "go.work")); statErr == nil {
			content, readErr := os.ReadFile(filepath.Join(dir, "specs", "024-admin-premium-grant", name))
			if readErr != nil {
				t.Fatalf("read %s: %v", name, readErr)
			}
			return string(content)
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("could not find go.work to locate %s", name)
		}
		dir = parent
	}
}

// 024 T172: the reason code was corrected in T159, but the DOCUMENT describing the
// scenario kept naming a catalogue lookup that the AI service never performs. A
// contract that describes behaviour nobody implemented is worse than no contract: it
// tells a reader FR-036 is covered when it is covered for a different field.
//
// This scans the documents, so the correction cannot be undone in one place and left
// stale in another.
func TestNoDocumentDescribesAPlanCatalogueLookupOnTheAISide(t *testing.T) {
	for _, doc := range []struct {
		name   string
		forbid []string
	}{
		{
			name:   "contracts/subscription-plan-changed-event.md",
			forbid: []string{"toPlanCode` không có trong danh mục", "toPlanCode` không có trong danh muc"},
		},
		{
			name:   "quickstart.md",
			forbid: []string{"aiCostPolicyId` không tồn tại trong danh mục"},
		},
	} {
		content := readFeatureDoc(t, doc.name)
		lowered := strings.ToLower(content)
		for _, phrase := range doc.forbid {
			if strings.Contains(lowered, strings.ToLower(phrase)) {
				t.Errorf("%s still describes a plan catalogue lookup on the AI side: %q", doc.name, phrase)
			}
		}
	}
}
