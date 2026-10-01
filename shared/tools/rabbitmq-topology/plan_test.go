package main

import (
	"testing"

	"smart-wardrobe-be/shared/application/constants/eventconstants"
	"smart-wardrobe-be/shared/infrastructure/messaging"
)

// TestBuildPlanClassifiesEveryQueue covers the three verdicts that matter: a
// matching queue, one whose type cannot be changed in place, and one the broker
// has never seen.
func TestBuildPlanClassifiesEveryQueue(t *testing.T) {
	desired := map[string]string{
		"subscription_plan_changed_queue":   messaging.QueueTypeQuorum,
		"audit_log_queue":                   messaging.QueueTypeClassic,
		"campaign_grant_user_created_queue": messaging.QueueTypeQuorum,
	}
	actual := map[string]string{
		"subscription_plan_changed_queue": messaging.QueueTypeClassic,
		"audit_log_queue":                 messaging.QueueTypeClassic,
	}
	messages := map[string]int{"subscription_plan_changed_queue": 0, "audit_log_queue": 7}

	plan := BuildPlan(desired, actual, messages)

	if plan.Expected != 3 {
		t.Errorf("Expected = %d, want 3", plan.Expected)
	}
	if plan.Found != 2 {
		t.Errorf("Found = %d, want 2 (campaign_grant is not on the broker yet)", plan.Found)
	}

	want := map[string]QueueState{
		"subscription_plan_changed_queue":   StateDrift,
		"audit_log_queue":                   StateOK,
		"campaign_grant_user_created_queue": StateMissing,
	}
	for _, e := range plan.Entries {
		if e.State != want[e.Queue] {
			t.Errorf("%s state = %s, want %s", e.Queue, e.State, want[e.Queue])
		}
	}
}

// TestMismatchedExcludesMissing is the reason a queue added to the registry does
// not turn CI red: the service declares it at startup, so it is not a blocker.
func TestMismatchedExcludesMissing(t *testing.T) {
	plan := BuildPlan(
		map[string]string{"a_queue": messaging.QueueTypeQuorum, "b_queue": messaging.QueueTypeQuorum},
		map[string]string{"a_queue": messaging.QueueTypeClassic},
		nil,
	)

	got := plan.Mismatched()
	if len(got) != 1 || got[0].Queue != "a_queue" {
		t.Fatalf("Mismatched() = %v, want only a_queue", got)
	}
}

// TestUnknownTypeIsNotTreatedAsDrift guards against a broker or API returning a
// type name this build has never heard of: reporting that as a mismatch would
// send an operator to migrate a queue that is already correct.
func TestUnknownTypeIsNotTreatedAsDrift(t *testing.T) {
	plan := BuildPlan(
		map[string]string{"a_queue": messaging.QueueTypeQuorum},
		map[string]string{"a_queue": "stream"},
		nil,
	)

	if len(plan.Mismatched()) != 0 {
		t.Error("an unrecognised broker type must not be reported as drift")
	}
	if plan.Entries[0].State != StateUnknown {
		t.Errorf("state = %s, want UNKNOWN", plan.Entries[0].State)
	}
}

func TestSelectedReportsUnknownNames(t *testing.T) {
	plan := BuildPlan(
		map[string]string{"a_queue": messaging.QueueTypeQuorum},
		map[string]string{"a_queue": messaging.QueueTypeClassic},
		nil,
	)

	entries, unknown := plan.Selected([]string{"a_queue", "typo_queue"})
	if len(entries) != 1 {
		t.Errorf("entries = %v, want 1", entries)
	}
	if len(unknown) != 1 || unknown[0] != "typo_queue" {
		t.Errorf("unknown = %v, want [typo_queue]", unknown)
	}
}

// TestDesiredQueueTypesMatchesRegistry ties the tool to the registry itself: the
// quorum expectations asserted here are the same ones the service declares, so a
// change to topology.go cannot pass unnoticed.
func TestDesiredQueueTypesMatchesRegistry(t *testing.T) {
	desired := desiredFromRegistry()

	if len(desired) != len(messaging.Topology)+len(messaging.DLQTopology) {
		t.Fatalf("desired = %d entries, want %d (Topology + DLQTopology)",
			len(desired), len(messaging.Topology)+len(messaging.DLQTopology))
	}

	for name, want := range map[string]string{
		eventconstants.QueueSubscriptionPlanChanged:    messaging.QueueTypeQuorum,
		eventconstants.QueueCampaignGrantUserCreated:   messaging.QueueTypeQuorum,
		eventconstants.QueueFashionAnalyze:             messaging.QueueTypeQuorum,
		eventconstants.QueueAuditLog:                   messaging.QueueTypeClassic,
		eventconstants.QueueNotificationEmail + ".dlq": messaging.QueueTypeQuorum,
	} {
		if got := desired[name]; got != want {
			t.Errorf("desired[%s] = %q, want %q", name, got, want)
		}
	}
}

// TestBindingByQueueCoversEveryDeclaredQueue guards the migrate path: a drifted
// queue with no binding could not be recreated with its arguments.
func TestBindingByQueueCoversEveryDeclaredQueue(t *testing.T) {
	bindings := bindingByQueue()
	if len(bindings) != len(messaging.Topology)+len(messaging.DLQTopology) {
		t.Fatalf("bindings = %d, want %d", len(bindings), len(messaging.Topology)+len(messaging.DLQTopology))
	}
	for _, b := range messaging.Topology {
		got, ok := bindings[b.Queue]
		if !ok {
			t.Fatalf("%s has no binding", b.Queue)
		}
		if len(got.RoutingKeys) != len(b.RoutingKeys) || got.Exchange != b.Exchange {
			t.Errorf("%s binding lost its routing keys or exchange", b.Queue)
		}
	}
}

func TestSanitizeKeepsFilenameSafe(t *testing.T) {
	if got := sanitize("campaign_grant_user_created_queue"); got != "campaign_grant_user_created_queue" {
		t.Errorf("sanitize() = %q, want the name unchanged", got)
	}
	if got := sanitize("a/b c"); got != "a_b_c" {
		t.Errorf("sanitize() = %q, want a_b_c", got)
	}
}
