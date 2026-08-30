// hygiene-scan is a read-only static analysis dev tool that reports codebase
// hygiene violations against the standards defined in
// docs/development/code-hygiene.md (feature 017).
//
// It never mutates source files; it only writes the scan report and a JSON
// snapshot under specs/017-codebase-hygiene-refactor/reports/.
package main

import "time"

// Severity classifies how strongly a violation must be addressed.
type Severity string

const (
	// SeverityFail marks objective violations that must be fixed (e.g. import
	// of a prohibited cross-module package, file above the hard length cap).
	SeverityFail Severity = "fail"
	// SeverityWarning marks heuristic findings that need review but are not
	// hard architectural breaks (e.g. long functions, naming drift).
	SeverityWarning Severity = "warning"
	// SeverityNeedsConfirmation marks findings that require manual
	// verification before action (e.g. dead code candidates).
	SeverityNeedsConfirmation Severity = "needs-confirmation"
)

// ViolationStatus tracks what happened to a finding after review.
type ViolationStatus string

const (
	StatusOpen               ViolationStatus = "open"
	StatusRefactored         ViolationStatus = "refactored"
	StatusAcceptedWithReason ViolationStatus = "accepted-with-reason"
)

// Location points at the exact place a violation was found.
type Location struct {
	File   string `json:"file"`
	Line   int    `json:"line,omitempty"`
	Symbol string `json:"symbol,omitempty"`
}

// Violation is a single finding tied to a checklist rule (H1..H8).
type Violation struct {
	RuleID   string          `json:"ruleId"`
	Location Location        `json:"location"`
	Severity Severity        `json:"severity"`
	Detail   string          `json:"detail"`
	Status   ViolationStatus `json:"status"`
}

// RuleSummary aggregates violation counts per severity.
type RuleSummary struct {
	Fail              int `json:"fail"`
	Warning           int `json:"warning"`
	NeedsConfirmation int `json:"needsConfirmation"`
}

// Summary rolls violations up by rule and by module.
type Summary struct {
	ByRule   map[string]RuleSummary `json:"by_rule"`
	ByModule map[string]RuleSummary `json:"by_module"`
}

// Scope describes what the scan covered and what was excluded.
type Scope struct {
	ScannedFiles int      `json:"scanned_files"`
	TestFiles    int      `json:"test_files"`
	Exclusions   []string `json:"exclusions"`
}

// Metrics holds the headline numbers used by the feature success criteria.
type Metrics struct {
	FilesOver500                 int `json:"files_over_500"`
	FilesOver800                 int `json:"files_over_800"`
	CrossModuleProhibitedImports int `json:"cross_module_prohibited_imports"`
	LayerSkips                   int `json:"layer_skips"`
}

// ScanRun is the full machine-readable result of one scan execution.
type ScanRun struct {
	RunID         string      `json:"run_id"`
	GeneratedAt   time.Time   `json:"generated_at"`
	Scope         Scope       `json:"scope"`
	Summary       Summary     `json:"summary"`
	Metrics       Metrics     `json:"metrics"`
	Violations    []Violation `json:"violations"`
	AcceptedKeeps int         `json:"acceptedKeeps,omitempty"`
	DeadCodeNote  string      `json:"deadCodeNote,omitempty"`
}
