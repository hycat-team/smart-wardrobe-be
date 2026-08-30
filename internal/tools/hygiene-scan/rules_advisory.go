package main

import (
	"fmt"
	"go/ast"
	"go/token"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

// Advisory rule IDs (docs/development/code-hygiene.md).
const (
	RuleH5 = "H5" // dead code (needs-confirmation only)
	RuleH6 = "H6" // duplication clusters
	RuleH7 = "H7" // naming consistency
	RuleH8 = "H8" // error handling consistency
)

// maxDuplicationSuggestions caps H6 output so the report stays readable.
const maxDuplicationSuggestions = 40

// runAdvisoryRules executes the heuristic rules H6, H7 and H8 that work off
// parsed files. H5 (dead code) needs the external deadcode tool and runs
// separately via runDeadCodeScan.
func runAdvisoryRules(files []*parsedFile) []Violation {
	var out []Violation
	out = append(out, checkDuplicationClusters(files)...)
	for _, pf := range files {
		if pf.Type != fileProduction {
			continue
		}
		out = append(out, checkNaming(pf)...)
		out = append(out, checkErrorHandling(pf)...)
	}
	sort.Slice(out, func(i, j int) bool { return lessViolation(out[i], out[j]) })
	return out
}

// fileOccurrence records where a candidate file lives across modules.
type fileOccurrence struct {
	module string
	path   string
}

// checkDuplicationClusters implements H6 as a suggestion engine: files with
// identical base names living in different modules hint at copy-paste
// helpers. Whether to merge is a manual audit call — module boundaries take
// precedence over de-duplication (CHK013).
func checkDuplicationClusters(files []*parsedFile) []Violation {
	byBase := map[string][]fileOccurrence{}

	for _, pf := range files {
		if pf.Type != fileProduction || !strings.HasPrefix(pf.Module, "internal/modules/") {
			continue
		}
		base := filepath.Base(pf.Path)
		byBase[base] = append(byBase[base], fileOccurrence{module: pf.Module, path: pf.Path})
	}

	bases := make([]string, 0, len(byBase))
	for base, occs := range byBase {
		if sameModule(occs) {
			continue
		}
		bases = append(bases, base)
	}
	sort.Strings(bases)

	var out []Violation
	for _, base := range bases {
		if len(out) >= maxDuplicationSuggestions {
			break
		}
		occs := byBase[base]
		modules := distinctModules(occs)
		sort.Strings(modules)
		out = append(out, Violation{
			RuleID: RuleH6,
			Location: Location{
				File:   occs[0].path,
				Symbol: base,
			},
			Severity: SeverityWarning,
			Detail: fmt.Sprintf(
				"potential duplication: file %s exists in %d modules (%s); audit before merging",
				base, len(modules), strings.Join(modules, ", ")),
			Status: StatusOpen,
		})
	}
	return out
}

func sameModule(occs []fileOccurrence) bool {
	for _, o := range occs {
		if o.module != occs[0].module {
			return false
		}
	}
	return true
}

func distinctModules(occs []fileOccurrence) []string {
	seen := map[string]bool{}
	var out []string
	for _, o := range occs {
		if !seen[o.module] {
			seen[o.module] = true
			out = append(out, o.module)
		}
	}
	return out
}

// checkNaming implements H7: repository persistence methods should prefer the
// Get... prefix over Find... (design-pattern.md Naming Conventions).
func checkNaming(pf *parsedFile) []Violation {
	if !strings.Contains(pf.Path, "/infrastructure/") ||
		!strings.Contains(pf.Path, "persistence") {
		return nil
	}
	var out []Violation
	ast.Inspect(pf.AST, func(n ast.Node) bool {
		fn, ok := n.(*ast.FuncDecl)
		if !ok || fn.Recv == nil || !strings.HasPrefix(fn.Name.Name, "Find") {
			return true
		}
		out = append(out, Violation{
			RuleID:   RuleH7,
			Location: Location{File: pf.Path, Line: pf.Fset.Position(fn.Pos()).Line, Symbol: fn.Name.Name},
			Severity: SeverityWarning,
			Detail:   "repository method uses Find... prefix; prefer Get... (design-pattern.md)",
			Status:   StatusOpen,
		})
		return true
	})
	return out
}

// checkErrorHandling implements H8 heuristics (warning level):
//   - presentation handlers must not construct raw business errors
//     (errors.New / fmt.Errorf) — they delegate to use cases;
//   - use cases must not create bare errors.New — use apperror factories or
//     module domain error constructors, EXCEPT package-level sentinels named
//     with the "Err" prefix (standard Go convention, see FR-003);
//   - use cases must not drop an error cause in fmt.Errorf (error-typed
//     argument without %w). Message-only fmt.Errorf (no error argument) is
//     accepted — there is nothing to chain.
func checkErrorHandling(pf *parsedFile) []Violation {
	isHandler := strings.Contains(pf.Path, "/presentation/handler/")
	isUseCase := strings.Contains(pf.Path, "/application/usecase/")
	if !isHandler && !isUseCase {
		return nil
	}

	sentinelPos := collectSentinelPositions(pf.AST)

	var out []Violation
	ast.Inspect(pf.AST, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		var callee string
		switch fn := call.Fun.(type) {
		case *ast.SelectorExpr:
			if ident, ok := fn.X.(*ast.Ident); ok {
				callee = ident.Name + "." + fn.Sel.Name
			}
		case *ast.Ident:
			callee = fn.Name
		}

		flag := false
		detail := ""
		switch {
		case isHandler && (callee == "errors.New" || callee == "fmt.Errorf"):
			flag = true
			detail = "handler constructs raw error; delegate business errors to the use case"
		case isUseCase && callee == "errors.New" && !sentinelPos[call.Pos()]:
			flag = true
			detail = "use case constructs bare errors.New; use apperror factory or domain error constructor"
		case isUseCase && callee == "fmt.Errorf" && !hasWrapVerb(call) && hasErrorLikeArg(call):
			flag = true
			detail = "use case drops the error cause in fmt.Errorf; wrap it with %w or use apperror"
		}
		if !flag {
			return true
		}
		out = append(out, Violation{
			RuleID:   RuleH8,
			Location: Location{File: pf.Path, Line: pf.Fset.Position(call.Pos()).Line, Symbol: callee},
			Severity: SeverityWarning,
			Detail:   detail,
			Status:   StatusOpen,
		})
		return true
	})
	return out
}

// collectSentinelPositions records the call positions of package-level
// `var ErrXxx = errors.New(...)` declarations so the sentinel convention is
// exempt from the bare errors.New rule.
func collectSentinelPositions(node ast.Node) map[token.Pos]bool {
	positions := make(map[token.Pos]bool)
	ast.Inspect(node, func(n ast.Node) bool {
		spec, ok := n.(*ast.ValueSpec)
		if !ok || len(spec.Names) == 0 || len(spec.Values) != len(spec.Names) {
			return true
		}
		for i, name := range spec.Names {
			if !strings.HasPrefix(name.Name, "Err") {
				continue
			}
			call, ok := spec.Values[i].(*ast.CallExpr)
			if !ok {
				continue
			}
			if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
				if ident, ok := sel.X.(*ast.Ident); ok && ident.Name == "errors" && sel.Sel.Name == "New" {
					positions[call.Pos()] = true
				}
			}
		}
		return true
	})
	return positions
}

// hasErrorLikeArg reports whether any argument identifier looks like an
// error value by naming convention (err, errXxx, xxErr, xxError).
func hasErrorLikeArg(call *ast.CallExpr) bool {
	for _, arg := range call.Args {
		if ident, ok := arg.(*ast.Ident); ok {
			name := ident.Name
			if name == "err" || strings.HasPrefix(name, "err") ||
				strings.HasSuffix(name, "Err") || strings.HasSuffix(name, "Error") {
				return true
			}
		}
	}
	return false
}

// hasWrapVerb reports whether an fmt.Errorf call includes the %w verb.
func hasWrapVerb(call *ast.CallExpr) bool {
	for _, arg := range call.Args {
		if lit, ok := arg.(*ast.BasicLit); ok && strings.Contains(lit.Value, "%w") {
			return true
		}
	}
	return false
}

// deadCodeHit is one parsed line of the deadcode tool output.
type deadCodeHit struct {
	File   string
	Line   int
	Symbol string
}

// runDeadCodeScan shells out to golang.org/x/tools/cmd/deadcode as an
// advisory source. It never deletes anything: every hit becomes a
// needs-confirmation finding (FR-008). If the tool is unavailable the note
// explains why and the report stays valid.
func runDeadCodeScan(root string) ([]deadCodeHit, string) {
	cmd := exec.Command("go", "run", "golang.org/x/tools/cmd/deadcode@latest", "./...")
	cmd.Dir = root
	output, err := cmd.CombinedOutput()
	if err != nil {
		note := fmt.Sprintf("deadcode tool unavailable (%v); H5 skipped this run", err)
		return nil, note
	}

	var hits []deadCodeHit
	for _, line := range strings.Split(string(output), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || !strings.Contains(line, ":") {
			continue
		}
		// deadcode line format: path:line:col: <symbol>
		parts := strings.SplitN(line, ":", 4)
		if len(parts) < 4 {
			continue
		}
		file := filepath.ToSlash(parts[0])
		var lineNo int
		if _, err := fmt.Sscanf(parts[1], "%d", &lineNo); err != nil {
			continue
		}
		symbol := strings.TrimSpace(parts[3])
		hits = append(hits, deadCodeHit{File: file, Line: lineNo, Symbol: symbol})
	}
	return hits, ""
}

// deadCodeViolations converts deadcode hits into needs-confirmation findings.
// Hits inside test files are skipped — test helpers are intentionally
// uncalled from production entry points.
func deadCodeViolations(root string, hits []deadCodeHit) []Violation {
	var out []Violation
	for _, hit := range hits {
		rel := hit.File
		if rel != "" && !strings.HasPrefix(rel, ".") {
			// deadcode reports OS-specific absolute-ish paths; normalize.
			if normalized, err := filepath.Rel(root, hit.File); err == nil {
				rel = filepath.ToSlash(normalized)
			}
		}
		if strings.HasSuffix(rel, "_test.go") {
			continue
		}
		if isExcludedPath(rel) {
			continue
		}
		out = append(out, Violation{
			RuleID:   RuleH5,
			Location: Location{File: rel, Line: hit.Line, Symbol: hit.Symbol},
			Severity: SeverityNeedsConfirmation,
			Detail:   "unreachable from any entry point per deadcode; verify references (reflection/interfaces) before removing",
			Status:   StatusOpen,
		})
	}
	sort.Slice(out, func(i, j int) bool { return lessViolation(out[i], out[j]) })
	return out
}

// isExcludedPath mirrors the walk exclusions for tool outputs that bypass
// the file walker (e.g. generated swagger files reported by deadcode).
func isExcludedPath(rel string) bool {
	for _, ex := range exclusions {
		prefix := strings.TrimRight(ex.Pattern, "/*")
		if prefix != "" && (rel == prefix || strings.HasPrefix(rel, prefix+"/")) {
			return true
		}
	}
	return false
}
