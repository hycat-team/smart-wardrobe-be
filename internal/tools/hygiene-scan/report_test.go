package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func sampleRun() *ScanRun {
	return &ScanRun{
		RunID:       "20260830-120000",
		GeneratedAt: time.Date(2026, 8, 30, 12, 0, 0, 0, time.UTC),
		Scope: Scope{
			ScannedFiles: 596,
			TestFiles:    56,
			Exclusions:   []string{"api/swagger/** (generated swagger artifacts)"},
		},
		Summary: summarize([]Violation{
			{RuleID: RuleH1, Location: Location{File: "internal/modules/brand/presentation/handler/big.go"}, Severity: SeverityFail, Status: StatusOpen},
			{RuleID: RuleH4, Location: Location{File: "internal/modules/brand/presentation/handler/x.go", Line: 3}, Severity: SeverityFail, Detail: detailCrossModule + ": ...", Status: StatusOpen},
			{RuleID: RuleH2, Location: Location{File: "internal/modules/fashion/application/usecase/w.go"}, Severity: SeverityWarning, Status: StatusOpen},
			{RuleID: RuleH5, Location: Location{File: "internal/modules/wardrobe/application/usecase/old.go", Line: 10, Symbol: "OldFn"}, Severity: SeverityNeedsConfirmation, Status: StatusOpen},
		}),
		Metrics: Metrics{
			FilesOver500:                 10,
			FilesOver800:                 1,
			CrossModuleProhibitedImports: 1,
			LayerSkips:                   0,
		},
	}
}

func TestSummarize(t *testing.T) {
	s := sampleRun().Summary

	if got := s.ByRule[RuleH1]; got.Fail != 1 {
		t.Fatalf("H1 fail = %d, want 1", got.Fail)
	}
	if got := s.ByModule["brand"]; got.Fail != 2 {
		t.Fatalf("brand fail = %d, want 2", got.Fail)
	}
	if got := s.ByModule["wardrobe"]; got.NeedsConfirmation != 1 {
		t.Fatalf("wardrobe needs-confirmation = %d, want 1", got.NeedsConfirmation)
	}
	if got := s.bySeverity(SeverityFail); got != 2 {
		t.Fatalf("total fail = %d, want 2", got)
	}
}

func TestComputeMetrics(t *testing.T) {
	files := []*parsedFile{
		{scannedFile: scannedFile{Path: "a.go", Type: fileProduction, Lines: 600}},
		{scannedFile: scannedFile{Path: "b.go", Type: fileProduction, Lines: 900}},
		{scannedFile: scannedFile{Path: "t.go", Type: fileTest, Lines: 2000}},
		{scannedFile: scannedFile{Path: "g.go", Type: fileGenerated, Lines: 9000}},
	}
	violations := []Violation{
		{RuleID: RuleH4, Detail: detailCrossModule + ": x"},
		{RuleID: RuleH4, Detail: detailLayerSkip + ": y"},
		{RuleID: RuleH4, Detail: detailLayerSkip + ": z"},
		{RuleID: RuleH1, Detail: "unrelated"},
	}

	m := computeMetrics(files, violations)
	if m.FilesOver500 != 2 || m.FilesOver800 != 1 {
		t.Fatalf("file metrics wrong: %+v", m)
	}
	if m.CrossModuleProhibitedImports != 1 || m.LayerSkips != 2 {
		t.Fatalf("boundary metrics wrong: %+v", m)
	}
}

func TestRenderMarkdownSectionOrder(t *testing.T) {
	md := string(renderMarkdown(sampleRun(), nil))

	want := []string{
		"## Summary",
		"## By Rule",
		"## By Module",
		"## Violations",
		"## Exclusions",
		"## Needs Confirmation",
	}
	last := -1
	for _, header := range want {
		idx := strings.Index(md, header)
		if idx < 0 {
			t.Fatalf("missing section %q", header)
		}
		if idx < last {
			t.Fatalf("section %q out of order", header)
		}
		last = idx
	}
}

func TestRenderMarkdownDeterministic(t *testing.T) {
	run := sampleRun()
	first := string(renderMarkdown(run, nil))
	second := string(renderMarkdown(run, nil))
	if first != second {
		t.Fatalf("renderMarkdown is not deterministic")
	}
}

func TestRenderMarkdownBaselineDeltas(t *testing.T) {
	baseline := sampleRun()
	current := sampleRun()
	current.RunID = "20260830-130000"
	current.Summary.ByRule[RuleH1] = RuleSummary{} // fixed everything in H1
	current.Violations = nil

	md := string(renderMarkdown(current, baseline))
	if !strings.Contains(md, "Compared against baseline `20260830-120000`") {
		t.Fatalf("baseline run id missing:\n%s", md)
	}
	if !strings.Contains(md, "0 (↓-1)") {
		t.Fatalf("expected improvement marker for H1 fail:\n%s", md)
	}
}

func TestSnapshotRoundtrip(t *testing.T) {
	outDir := filepath.Join(t.TempDir(), "reports")
	run := sampleRun()

	path, err := writeSnapshot(outDir, run)
	if err != nil {
		t.Fatalf("writeSnapshot: %v", err)
	}
	if filepath.Base(path) != "snapshot-20260830-120000.json" {
		t.Fatalf("unexpected snapshot name: %s", path)
	}

	loaded, err := loadBaseline(path)
	if err != nil {
		t.Fatalf("loadBaseline: %v", err)
	}
	if loaded.RunID != run.RunID || loaded.Metrics.FilesOver800 != 1 {
		t.Fatalf("roundtrip mismatch: %+v", loaded)
	}
}

func TestLatestSnapshot(t *testing.T) {
	outDir := t.TempDir()
	if _, err := latestSnapshot(outDir); err != nil {
		t.Fatalf("empty dir: %v", err)
	}

	for _, name := range []string{"snapshot-20260830-120000.json", "snapshot-20260830-130000.json", "other.txt"} {
		if err := os.WriteFile(filepath.Join(outDir, name), []byte("{}"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	latest, err := latestSnapshot(outDir)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(latest) != "snapshot-20260830-130000.json" {
		t.Fatalf("latest = %s, want snapshot-20260830-130000.json", latest)
	}
}
