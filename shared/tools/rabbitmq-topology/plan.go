package main

import (
	"sort"

	"smart-wardrobe-be/shared/infrastructure/messaging"
)

// QueueState is how a declared queue compares to the queue the broker actually
// holds.
type QueueState int

const (
	// StateOK — broker queue type matches the registry.
	StateOK QueueState = iota
	// StateDrift — broker holds a different type. RabbitMQ cannot change
	// x-queue-type in place, so this needs the drain/delete/redeclare
	// migration before any service can start.
	StateDrift
	// StateMissing — no such queue on the broker. Normal on a fresh
	// environment: the service creates it at startup, so it is not an error.
	StateMissing
	// StateUnknown — the management API reported a type this tool does not
	// recognise, so equality cannot be decided.
	StateUnknown
)

func (s QueueState) String() string {
	switch s {
	case StateOK:
		return "OK"
	case StateDrift:
		return "DRIFT"
	case StateMissing:
		return "MISSING"
	default:
		return "UNKNOWN"
	}
}

// Drift is one queue's verdict.
type Drift struct {
	Queue   string
	Want    string
	Got     string
	State   QueueState
	Message int
}

// Plan is the full comparison of the registry against the broker.
type Plan struct {
	Entries  []Drift
	Found    int
	Expected int
}

// Mismatched returns only the entries that block a deploy. MISSING is
// deliberately excluded: a queue the broker has never seen is created by the
// service itself, so failing on it would turn every first deploy — and every
// newly added queue — red for no reason.
func (p Plan) Mismatched() []Drift {
	var out []Drift
	for _, e := range p.Entries {
		if e.State == StateDrift {
			out = append(out, e)
		}
	}
	return out
}

// Selected returns the entries for the queues named in the filter, or every
// entry when the filter is empty. A name that matches nothing is reported as an
// error by the caller so a typo cannot silently produce a no-op migrate.
func (p Plan) Selected(names []string) ([]Drift, []string) {
	if len(names) == 0 {
		return p.Entries, nil
	}
	wanted := make(map[string]bool, len(names))
	for _, n := range names {
		wanted[n] = true
	}
	var out []Drift
	var unknown []string
	for _, n := range names {
		found := false
		for _, e := range p.Entries {
			if e.Queue == n {
				out = append(out, e)
				found = true
			}
		}
		if !found {
			unknown = append(unknown, n)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Queue < out[j].Queue })
	return out, unknown
}

// BuildPlan compares the queue types declared by the messaging registry against
// the types the broker reports. actual maps queue name to broker-reported type;
// a queue absent from actual is StateMissing.
func BuildPlan(desired map[string]string, actual map[string]string, messages map[string]int) Plan {
	names := make([]string, 0, len(desired))
	for name := range desired {
		names = append(names, name)
	}
	sort.Strings(names)

	plan := Plan{Expected: len(desired), Entries: make([]Drift, 0, len(desired))}
	for _, name := range names {
		want := desired[name]
		got, present := actual[name]
		if !present {
			plan.Entries = append(plan.Entries, Drift{Queue: name, Want: want, State: StateMissing})
			continue
		}
		state := StateDrift
		switch {
		case got == want:
			state = StateOK
		case !isKnownQueueType(got):
			// The broker reported a type this build does not recognise.
			// Equality cannot be decided, so reporting drift would send an
			// operator to migrate a queue that may already be correct.
			state = StateUnknown
		}
		plan.Entries = append(plan.Entries, Drift{Queue: name, Want: want, Got: got, State: state, Message: messages[name]})
	}
	plan.Found = len(actual)
	return plan
}

// isKnownQueueType reports whether the registry can reason about a queue type.
// A type outside this set is left as UNKNOWN rather than treated as drift.
func isKnownQueueType(t string) bool {
	return t == messaging.QueueTypeClassic || t == messaging.QueueTypeQuorum
}

// desiredFromRegistry reads the live registry rather than a copy, so the tool
// can never drift from topology.go.
func desiredFromRegistry() map[string]string {
	return messaging.DesiredQueueTypes()
}
