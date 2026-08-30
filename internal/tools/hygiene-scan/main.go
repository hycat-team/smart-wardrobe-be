package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// defaultOutDir is where the scan artifacts land (feature 017 spec).
const defaultOutDir = "specs/017-codebase-hygiene-refactor/reports"

func main() {
	root := flag.String("root", ".", "repository root to scan (must contain go.mod)")
	out := flag.String("out", defaultOutDir, "output directory for report and snapshot")
	baseline := flag.String("baseline", "", "baseline snapshot JSON (default: newest snapshot in -out)")
	format := flag.String("format", "all", "output format: md | json | all")
	flag.Parse()

	exitCode := run(*root, *out, *baseline, *format)
	if exitCode != 0 {
		os.Exit(exitCode)
	}
}

// run executes the scan pipeline and returns the process exit code
// (0 success, 2 tool error — per contracts/hygiene-scan-cli.md).
func run(rootFlag, outFlag, baselineFlag, format string) int {
	root, err := findRepoRoot(rootFlag)
	if err != nil {
		fmt.Fprintf(os.Stderr, "hygiene-scan: %v\n", err)
		return 2
	}
	outDir := outFlag
	if !filepath.IsAbs(outDir) {
		outDir = filepath.Join(root, outDir)
	}

	files, err := scanTree(root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "hygiene-scan: scan failed: %v\n", err)
		return 2
	}

	violations := runStructuralRules(files)
	violations = append(violations, runBoundaryRules(files)...)
	violations = append(violations, runAdvisoryRules(files)...)

	hits, deadCodeNote := runDeadCodeScan(root)
	violations = append(violations, deadCodeViolations(root, hits)...)

	run := &ScanRun{
		RunID:        buildRunID(time.Now()),
		GeneratedAt:  time.Now(),
		Scope:        buildScope(files),
		Summary:      summarize(violations),
		Metrics:      computeMetrics(files, violations),
		Violations:   violations,
		DeadCodeNote: deadCodeNote,
	}

	base, err := resolveBaseline(root, outDir, baselineFlag)
	if err != nil {
		fmt.Fprintf(os.Stderr, "hygiene-scan: baseline unavailable: %v\n", err)
		return 2
	}

	if format == "all" || format == "json" {
		if _, err := writeSnapshot(outDir, run); err != nil {
			fmt.Fprintf(os.Stderr, "hygiene-scan: write snapshot failed: %v\n", err)
			return 2
		}
	}
	if format == "all" || format == "md" {
		reportPath := filepath.Join(outDir, "scan-report.md")
		if err := os.WriteFile(reportPath, renderMarkdown(run, base), 0o644); err != nil {
			fmt.Fprintf(os.Stderr, "hygiene-scan: write report failed: %v\n", err)
			return 2
		}
	}

	printStdoutSummary(run, outDir)
	return 0
}

// findRepoRoot walks up from start until a go.mod is found.
func findRepoRoot(start string) (string, error) {
	dir, err := filepath.Abs(start)
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("no go.mod found from %s", start)
		}
		dir = parent
	}
}

// buildScope rolls file counts and exclusion documentation into the run.
func buildScope(files []*parsedFile) Scope {
	scope := Scope{Exclusions: make([]string, 0, len(exclusions))}
	for _, ex := range exclusions {
		scope.Exclusions = append(scope.Exclusions, ex.Pattern+" ("+ex.Reason+")")
	}
	for _, pf := range files {
		switch pf.Type {
		case fileTest:
			scope.TestFiles++
		case fileProduction:
			scope.ScannedFiles++
		}
	}
	return scope
}

// resolveBaseline picks the explicit flag, else the newest existing snapshot
// in outDir, else nil (first run — no comparison).
func resolveBaseline(root, outDir, baselineFlag string) (*ScanRun, error) {
	if baselineFlag == "" {
		latest, err := latestSnapshot(outDir)
		if err != nil || latest == "" {
			return nil, err
		}
		baselineFlag = latest
	} else if baselineFlag == "none" {
		return nil, nil
	}
	if baselineFlag == "" {
		return nil, nil
	}
	if !filepath.IsAbs(baselineFlag) {
		baselineFlag = filepath.Join(root, baselineFlag)
	}
	if _, err := os.Stat(baselineFlag); err != nil {
		return nil, nil // no baseline yet — first run
	}
	return loadBaseline(baselineFlag)
}

// printStdoutSummary gives the operator instant feedback per the contract.
func printStdoutSummary(run *ScanRun, outDir string) {
	fmt.Printf("run %s: %d production files, %d test files\n",
		run.RunID, run.Scope.ScannedFiles, run.Scope.TestFiles)
	fmt.Printf("violations: %d fail, %d warning, %d needs-confirmation\n",
		run.Summary.bySeverity(SeverityFail),
		run.Summary.bySeverity(SeverityWarning),
		run.Summary.bySeverity(SeverityNeedsConfirmation))

	type row struct {
		rule string
		RuleSummary
	}
	rows := make([]row, 0, len(run.Summary.ByRule))
	for rule, s := range run.Summary.ByRule {
		rows = append(rows, row{rule, s})
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].rule < rows[j].rule })
	for _, r := range rows {
		fmt.Printf("  %s: fail=%d warning=%d needs-confirmation=%d\n",
			r.rule, r.Fail, r.Warning, r.NeedsConfirmation)
	}
	fmt.Printf("metrics: files_over_500=%d files_over_800=%d cross_module_prohibited_imports=%d layer_skips=%d\n",
		run.Metrics.FilesOver500, run.Metrics.FilesOver800,
		run.Metrics.CrossModuleProhibitedImports, run.Metrics.LayerSkips)
	fmt.Printf("report: %s\n", filepath.Join(outDir, "scan-report.md"))
}
