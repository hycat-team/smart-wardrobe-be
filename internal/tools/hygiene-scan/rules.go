package main

import (
	"fmt"
	"go/ast"
	"sort"
	"strings"
)

// Checklist rule IDs (docs/development/code-hygiene.md).
const (
	RuleH1 = "H1" // file & function length
	RuleH2 = "H2" // struct-per-file (behavior-carrying types)
	RuleH4 = "H4" // module boundary & layering
)

// Finalized thresholds (spec 017 Clarifications, 2026-08-30).
const (
	warnFileLines      = 500
	failFileLines      = 800
	warnTestFileLines  = 750
	failTestFileLines  = 1200
	warnFuncLines      = 80
	warnBehaviorStruct = 2 // files with more behavior-carrying structs are flagged
)

// runStructuralRules executes H1, H2 and H4 over every production file.
func runStructuralRules(files []*parsedFile) []Violation {
	var out []Violation
	for _, pf := range files {
		if pf.Type != fileProduction {
			if pf.Type == fileTest {
				out = append(out, checkFileLength(pf)...)
			}
			continue
		}
		out = append(out, checkFileLength(pf)...)
		out = append(out, checkFuncLength(pf)...)
		out = append(out, checkBehaviorStructs(pf)...)
	}
	sort.Slice(out, func(i, j int) bool { return lessViolation(out[i], out[j]) })
	return out
}

// checkFileLength implements H1: production >500 warning / >800 fail, test
// files relaxed by 50% (>750 warning / >1200 fail). Generated files exempt.
func checkFileLength(pf *parsedFile) []Violation {
	warn, fail := warnFileLines, failFileLines
	if pf.Type == fileTest {
		warn, fail = warnTestFileLines, failTestFileLines
	}
	switch {
	case pf.Lines > fail:
		return []Violation{{
			RuleID:   RuleH1,
			Location: Location{File: pf.Path},
			Severity: SeverityFail,
			Detail:   fmt.Sprintf("file has %d lines (fail threshold %d)", pf.Lines, fail),
			Status:   StatusOpen,
		}}
	case pf.Lines > warn:
		return []Violation{{
			RuleID:   RuleH1,
			Location: Location{File: pf.Path},
			Severity: SeverityWarning,
			Detail:   fmt.Sprintf("file has %d lines (warning threshold %d)", pf.Lines, warn),
			Status:   StatusOpen,
		}}
	default:
		return nil
	}
}

// checkFuncLength flags production functions longer than warnFuncLines.
func checkFuncLength(pf *parsedFile) []Violation {
	var out []Violation
	ast.Inspect(pf.AST, func(n ast.Node) bool {
		fn, ok := n.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			return true
		}
		start := pf.Fset.Position(fn.Pos())
		end := pf.Fset.Position(fn.End())
		lines := end.Line - start.Line + 1
		if lines > warnFuncLines {
			name := fn.Name.Name
			if fn.Recv != nil && len(fn.Recv.List) > 0 {
				name = recvTypeName(fn.Recv.List[0].Type) + "." + name
			}
			out = append(out, Violation{
				RuleID:   RuleH1,
				Location: Location{File: pf.Path, Line: start.Line, Symbol: name},
				Severity: SeverityWarning,
				Detail:   fmt.Sprintf("function spans %d lines (threshold %d)", lines, warnFuncLines),
				Status:   StatusOpen,
			})
		}
		return true
	})
	return out
}

// checkBehaviorStructs implements H2: a production file should carry behavior
// for at most one type. Pure data-shape files (DTOs, entities) are exempt by
// design — only structs that own methods count as behavior-carrying.
func checkBehaviorStructs(pf *parsedFile) []Violation {
	structsWithMethods := map[string]bool{}
	ast.Inspect(pf.AST, func(n ast.Node) bool {
		if fn, ok := n.(*ast.FuncDecl); ok && fn.Recv != nil && len(fn.Recv.List) > 0 {
			if name := recvTypeName(fn.Recv.List[0].Type); name != "" {
				structsWithMethods[name] = true
			}
		}
		return true
	})
	if len(structsWithMethods) <= warnBehaviorStruct {
		return nil
	}

	names := make([]string, 0, len(structsWithMethods))
	for name := range structsWithMethods {
		names = append(names, name)
	}
	sort.Strings(names)

	return []Violation{{
		RuleID:   RuleH2,
		Location: Location{File: pf.Path},
		Severity: SeverityWarning,
		Detail: fmt.Sprintf(
			"file carries behavior for %d types (%s); split by responsibility",
			len(names), strings.Join(names, ", ")),
		Status: StatusOpen,
	}}
}

// boundaryRuleIDs maps H4 detail prefixes so metrics can be derived.
const (
	detailCrossModule = "cross-module prohibited import"
	detailLayerSkip   = "layer skip"
)

// runBoundaryRules implements H4 against Constitution §II/§IV:
//   - no module may import another module's domain/entities,
//     domain/repositories or infrastructure (cross-module, fail);
//   - within a module, presentation and application must not import
//     infrastructure, and domain must not import any other layer
//     (layer skip, fail).
//
// provider.go at a module root, internal/di and internal/bootstrap are exempt
// from layer-skip rules (they are the Wire wiring points) but never from the
// cross-module prohibition.
func runBoundaryRules(files []*parsedFile) []Violation {
	var out []Violation
	for _, pf := range files {
		if pf.Type != fileProduction || pf.AST == nil {
			continue
		}
		for _, imp := range pf.AST.Imports {
			importPath := strings.Trim(imp.Path.Value, `"`)
			target := internalTarget(importPath)
			if target == nil {
				continue
			}

			if pf.Module != target.module &&
				strings.HasPrefix(pf.Module, "internal/modules/") &&
				strings.HasPrefix(target.module, "internal/modules/") &&
				(target.prohibitedEntities || target.prohibitedRepos || target.infrastructure) {
				out = append(out, Violation{
					RuleID:   RuleH4,
					Location: Location{File: pf.Path, Line: pf.Fset.Position(imp.Pos()).Line},
					Severity: SeverityFail,
					Detail: fmt.Sprintf(
						"%s: %s imports %s", detailCrossModule, pf.Module, importPath),
					Status: StatusOpen,
				})
				continue
			}

			if exemptFromLayerSkip(pf.Path) || pf.Module != target.module {
				continue
			}
			fileLayer := layerOf(pf.Path)
			if fileLayer == "" {
				continue
			}
			if layerSkip(fileLayer, target) {
				out = append(out, Violation{
					RuleID:   RuleH4,
					Location: Location{File: pf.Path, Line: pf.Fset.Position(imp.Pos()).Line},
					Severity: SeverityFail,
					Detail: fmt.Sprintf(
						"%s: %s imports %s", detailLayerSkip, fileLayer, importPath),
					Status: StatusOpen,
				})
			}
		}
	}
	sort.Slice(out, func(i, j int) bool { return lessViolation(out[i], out[j]) })
	return out
}

// internalTarget classifies an import path against the importing rules.
// Returns nil for anything outside this module's own namespace.
func internalTarget(importPath string) *targetInfo {
	const marker = "internal/"
	idx := strings.Index(importPath, marker)
	if idx < 0 {
		return nil
	}
	rel := importPath[idx:]
	parts := strings.Split(rel, "/")

	t := &targetInfo{module: strings.Join(parts[:min(2, len(parts))], "/")}
	if len(parts) >= 3 && parts[0] == "internal" && parts[1] == "modules" {
		t.module = "internal/modules/" + parts[2]
	}
	for _, seg := range parts {
		switch seg {
		case "entities":
			t.prohibitedEntities = true
		case "repositories":
			t.prohibitedRepos = true
		case "infrastructure":
			t.infrastructure = true
		case "application":
			t.application = true
		case "presentation":
			t.presentation = true
		case "constants":
			// Pure enum/error-factory packages (domain/constants per §II,
			// shared application constants like apperror/eventconstants per
			// §V) are sanctioned cross-cutting values without behavior.
			t.constants = true
		}
	}
	return t
}

type targetInfo struct {
	module             string
	prohibitedEntities bool
	prohibitedRepos    bool
	infrastructure     bool
	application        bool
	presentation       bool
	constants          bool
}

// layerOf returns the architecture layer of a module file path.
func layerOf(relPath string) string {
	switch {
	case strings.Contains(relPath, "/presentation/"):
		return "presentation"
	case strings.Contains(relPath, "/application/"):
		return "application"
	case strings.Contains(relPath, "/domain/"):
		return "domain"
	case strings.Contains(relPath, "/infrastructure/"):
		return "infrastructure"
	default:
		return ""
	}
}

// layerSkip reports whether a file in fileLayer importing t violates §IV.
// Constants packages are exempt: they are pure values/error factories
// sanctioned by §II (domain/constants) and §V (apperror usage mandate).
func layerSkip(fileLayer string, t *targetInfo) bool {
	if t.constants {
		return false
	}
	switch fileLayer {
	case "presentation", "application":
		return t.infrastructure
	case "domain":
		return t.infrastructure || t.application || t.presentation
	default:
		return false
	}
}

// exemptFromLayerSkip marks the DI wiring points that are allowed to touch
// infrastructure (constitution governance: archived/wiring code excluded).
func exemptFromLayerSkip(relPath string) bool {
	if strings.HasPrefix(relPath, "internal/di/") ||
		strings.HasPrefix(relPath, "internal/bootstrap/") {
		return true
	}
	parts := strings.Split(relPath, "/")
	return len(parts) == 4 &&
		parts[0] == "internal" && parts[1] == "modules" &&
		parts[3] == "provider.go"
}

// recvTypeName extracts the type name from a receiver type expression.
func recvTypeName(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return recvTypeName(t.X)
	case *ast.IndexExpr: // generic receivers
		return recvTypeName(t.X)
	case *ast.IndexListExpr:
		return recvTypeName(t.X)
	case *ast.SelectorExpr:
		return t.Sel.Name
	default:
		return ""
	}
}

// lessViolation orders violations deterministically: severity, rule, file,
// line, symbol.
func lessViolation(a, b Violation) bool {
	if rankSeverity(a.Severity) != rankSeverity(b.Severity) {
		return rankSeverity(a.Severity) < rankSeverity(b.Severity)
	}
	if a.RuleID != b.RuleID {
		return a.RuleID < b.RuleID
	}
	if a.Location.File != b.Location.File {
		return a.Location.File < b.Location.File
	}
	if a.Location.Line != b.Location.Line {
		return a.Location.Line < b.Location.Line
	}
	return a.Location.Symbol < b.Location.Symbol
}

func rankSeverity(s Severity) int {
	switch s {
	case SeverityFail:
		return 0
	case SeverityWarning:
		return 1
	default:
		return 2
	}
}
