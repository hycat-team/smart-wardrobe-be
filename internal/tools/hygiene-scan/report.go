package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// buildRunID formats the run identifier as YYYYMMDD-HHMMSS (contract).
func buildRunID(now time.Time) string {
	return now.Format("20060102-150405")
}

// summarize rolls violations up per rule and per module. Findings accepted
// with a keep-reason (FR-011) are excluded from the counts but remain listed
// in the report.
func summarize(violations []Violation) Summary {
	summary := Summary{
		ByRule:   map[string]RuleSummary{},
		ByModule: map[string]RuleSummary{},
	}
	for _, v := range violations {
		if v.Status == StatusAcceptedWithReason {
			continue
		}
		rule := summary.ByRule[v.RuleID]
		incRule := &rule
		addSeverity(incRule, v.Severity)
		summary.ByRule[v.RuleID] = rule

		module := moduleOf(v.Location.File)
		mod := summary.ByModule[module]
		incMod := &mod
		addSeverity(incMod, v.Severity)
		summary.ByModule[module] = mod
	}
	return summary
}

func addSeverity(s *RuleSummary, sev Severity) {
	switch sev {
	case SeverityFail:
		s.Fail++
	case SeverityWarning:
		s.Warning++
	default:
		s.NeedsConfirmation++
	}
}

// moduleOf maps a violation file path back to the module rollup bucket.
func moduleOf(file string) string {
	parts := strings.Split(file, "/")
	if len(parts) >= 3 && parts[0] == "internal" && parts[1] == "modules" {
		return parts[2]
	}
	if len(parts) >= 2 && parts[0] == "internal" {
		return "internal/" + parts[1]
	}
	return file
}

// computeMetrics derives the headline numbers for SC-001/SC-002.
func computeMetrics(files []*parsedFile, violations []Violation) Metrics {
	m := Metrics{}
	for _, pf := range files {
		if pf.Type != fileProduction {
			continue
		}
		if pf.Lines > warnFileLines {
			m.FilesOver500++
		}
		if pf.Lines > failFileLines {
			m.FilesOver800++
		}
	}
	for _, v := range violations {
		if v.RuleID != RuleH4 {
			continue
		}
		switch {
		case strings.HasPrefix(v.Detail, detailCrossModule):
			m.CrossModuleProhibitedImports++
		case strings.HasPrefix(v.Detail, detailLayerSkip):
			m.LayerSkips++
		}
	}
	return m
}

// writeSnapshot marshals the run as snapshot-<run_id>.json.
func writeSnapshot(outDir string, run *ScanRun) (string, error) {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return "", err
	}
	path := filepath.Join(outDir, "snapshot-"+run.RunID+".json")
	data, err := json.MarshalIndent(run, "", "  ")
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return "", err
	}
	return path, nil
}

// renderMarkdown produces the human-readable report with the contract's fixed
// section order: Summary, By Rule, By Module, Violations, Exclusions, Needs
// Confirmation.
func renderMarkdown(run *ScanRun, baseline *ScanRun) []byte {
	var b strings.Builder

	b.WriteString("# Hygiene Scan Report\n\n")
	b.WriteString(fmt.Sprintf("**Run ID**: `%s` | **Generated**: %s\n\n", run.RunID, run.GeneratedAt.Format(time.RFC3339)))
	b.WriteString(fmt.Sprintf("Scanned %d production files (+%d test files). Violations: **%d fail**, **%d warning**, **%d needs-confirmation**.\n\n",
		run.Scope.ScannedFiles, run.Scope.TestFiles,
		run.Summary.bySeverity(SeverityFail),
		run.Summary.bySeverity(SeverityWarning),
		run.Summary.bySeverity(SeverityNeedsConfirmation)))

	writeSummarySection(&b, run, baseline)
	writeTableSection(&b, "## By Rule", run.Summary.ByRule, ruleKeys(run))
	writeTableSection(&b, "## By Module", run.Summary.ByModule, moduleKeys(run))
	writeViolationsSection(&b, run)
	writeExclusionsSection(&b, run)
	writeNeedsConfirmationSection(&b, run)

	return []byte(b.String())
}

func writeSummarySection(b *strings.Builder, run *ScanRun, baseline *ScanRun) {
	b.WriteString("## Summary\n\n")
	if baseline == nil {
		b.WriteString("No baseline snapshot found; this run becomes the baseline.\n\n")
		return
	}
	b.WriteString("Compared against baseline `" + baseline.RunID + "` (↑ worse, ↓ better, = unchanged).\n\n")
	b.WriteString("| Rule | fail | warning | needs-confirmation |\n")
	b.WriteString("|------|------|---------|--------------------|\n")
	for _, rule := range ruleKeys(run) {
		cur, hasCur := run.Summary.ByRule[rule]
		prev := baseline.Summary.ByRule[rule]
		if !hasCur && prev == (RuleSummary{}) {
			continue
		}
		b.WriteString(fmt.Sprintf("| %s | %s | %s | %s |\n",
			rule,
			deltaCell(cur.Fail, prev.Fail),
			deltaCell(cur.Warning, prev.Warning),
			deltaCell(cur.NeedsConfirmation, prev.NeedsConfirmation)))
	}
	b.WriteString("\n")
}

// deltaCell renders "current (↑/↓/delta)" per the contract notation.
func deltaCell(cur, prev int) string {
	switch {
	case cur > prev:
		return fmt.Sprintf("%d (↑+%d)", cur, cur-prev)
	case cur < prev:
		return fmt.Sprintf("%d (↓-%d)", cur, prev-cur)
	default:
		return fmt.Sprintf("%d (=)", cur)
	}
}

func writeTableSection(b *strings.Builder, title string, by map[string]RuleSummary, keys []string) {
	b.WriteString(title + "\n\n")
	b.WriteString("| Key | fail | warning | needs-confirmation |\n")
	b.WriteString("|-----|------|---------|--------------------|\n")
	for _, k := range keys {
		s := by[k]
		b.WriteString(fmt.Sprintf("| %s | %d | %d | %d |\n", k, s.Fail, s.Warning, s.NeedsConfirmation))
	}
	b.WriteString("\n")
}

func writeViolationsSection(b *strings.Builder, run *ScanRun) {
	b.WriteString("## Violations\n\n")
	if len(run.Violations) == 0 {
		b.WriteString("No violations found.\n\n")
		return
	}
	b.WriteString("| Rule | Severity | Location | Detail |\n")
	b.WriteString("|------|----------|----------|--------|\n")
	for _, v := range run.Violations {
		loc := v.Location.File
		if v.Location.Line > 0 {
			loc = fmt.Sprintf("%s:%d", loc, v.Location.Line)
		}
		if v.Location.Symbol != "" {
			loc += " `" + v.Location.Symbol + "`"
		}
		detail := strings.ReplaceAll(v.Detail, "|", "\\|")
		b.WriteString(fmt.Sprintf("| %s | %s | %s | %s |\n", v.RuleID, v.Severity, loc, detail))
	}
	b.WriteString("\n")
}

func writeExclusionsSection(b *strings.Builder, run *ScanRun) {
	b.WriteString("## Exclusions\n\n")
	b.WriteString("| Pattern | Reason |\n")
	b.WriteString("|---------|--------|\n")
	for _, ex := range exclusions {
		b.WriteString(fmt.Sprintf("| %s | %s |\n", ex.Pattern, ex.Reason))
	}
	b.WriteString("\n")
}

func writeNeedsConfirmationSection(b *strings.Builder, run *ScanRun) {
	b.WriteString("## Needs Confirmation\n\n")
	if run.DeadCodeNote != "" {
		b.WriteString(run.DeadCodeNote + "\n\n")
	}
	count := 0
	for _, v := range run.Violations {
		if v.Severity != SeverityNeedsConfirmation {
			continue
		}
		count++
	}
	if count == 0 {
		b.WriteString("No findings need manual confirmation.\n\n")
		return
	}
	b.WriteString(fmt.Sprintf("%d finding(s) require manual verification before removal (see Violations table, rule H5).\n\n", count))
}

func ruleKeys(run *ScanRun) []string {
	keys := make([]string, 0, len(run.Summary.ByRule))
	for k := range run.Summary.ByRule {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func moduleKeys(run *ScanRun) []string {
	keys := make([]string, 0, len(run.Summary.ByModule))
	for k := range run.Summary.ByModule {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// bySeverity counts violations of one severity across the whole run.
func (s Summary) bySeverity(sev Severity) int {
	total := 0
	for _, rs := range s.ByRule {
		switch sev {
		case SeverityFail:
			total += rs.Fail
		case SeverityWarning:
			total += rs.Warning
		default:
			total += rs.NeedsConfirmation
		}
	}
	return total
}

// latestSnapshot returns the newest snapshot JSON in outDir (by filename,
// which embeds the run timestamp), or nil when outDir is empty.
func latestSnapshot(outDir string) (string, error) {
	matches, err := filepath.Glob(filepath.Join(outDir, "snapshot-*.json"))
	if err != nil {
		return "", err
	}
	if len(matches) == 0 {
		return "", nil
	}
	sort.Strings(matches)
	return matches[len(matches)-1], nil
}

// loadBaseline parses a previous snapshot for comparison.
func loadBaseline(path string) (*ScanRun, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var run ScanRun
	if err := json.Unmarshal(data, &run); err != nil {
		return nil, fmt.Errorf("parse baseline %s: %w", path, err)
	}
	return &run, nil
}
