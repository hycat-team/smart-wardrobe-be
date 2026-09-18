package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// fileType classifies a discovered .go source file.
type fileType string

const (
	fileProduction fileType = "production"
	fileTest       fileType = "test"
	fileGenerated  fileType = "generated"
)

// scannedFile is one .go file classified by role and owning module.
type scannedFile struct {
	// Path is repo-relative with forward slashes.
	Path   string
	Type   fileType
	Module string
	Lines  int
}

// parsedFile pairs a scanned file with its AST so rules parse each file once.
type parsedFile struct {
	scannedFile
	Fset *token.FileSet
	AST  *ast.File
}

// exclusion documents one exclusion pattern and why it is skipped.
type exclusion struct {
	Pattern string
	Reason  string
}

// exclusions mirrors research.md R8: what the scan skips entirely.
var exclusions = []exclusion{
	{Pattern: "api/swagger/**", Reason: "generated swagger artifacts"},
	{Pattern: "bin/**", Reason: "build output"},
	{Pattern: "deployments/**", Reason: "deployment configuration, no Go source"},
	{Pattern: "specs/**", Reason: "specification documents"},
	{Pattern: "docs/**", Reason: "documentation"},
	{Pattern: "database/**", Reason: "goose SQL migrations"},
	{Pattern: ".git/**", Reason: "vcs metadata"},
}

// excludedDirs are directory names skipped during the walk.
var excludedDirs = map[string]bool{
	"bin":         true,
	"deployments": true,
	"specs":       true,
	"docs":        true,
	"database":    true,
	".git":        true,
}

// excludedSubPaths are nested paths excluded from the walk.
var excludedSubPaths = map[string]bool{
	"api/swagger": true,
}

// scanTree walks root, classifies every .go file and parses it once.
func scanTree(root string) ([]*parsedFile, error) {
	var files []*parsedFile

	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return relErr
		}
		rel = filepath.ToSlash(rel)

		if d.IsDir() {
			if rel == "." {
				return nil
			}
			if excludedDirs[rel] || excludedSubPaths[rel] {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(rel, ".go") {
			return nil
		}

		src, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}

		fset := token.NewFileSet()
		ast, parseErr := parser.ParseFile(fset, path, src, parser.ParseComments)
		if parseErr != nil {
			return fmt.Errorf("parse %s: %w", rel, parseErr)
		}

		files = append(files, &parsedFile{
			scannedFile: scannedFile{
				Path:   rel,
				Type:   classifyFile(rel, src),
				Module: classifyModule(rel),
				Lines:  countLines(src),
			},
			Fset: fset,
			AST:  ast,
		})
		return nil
	})
	if err != nil {
		return nil, err
	}

	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	return files, nil
}

// classifyFile marks generated and test files; everything else is production.
func classifyFile(rel string, src []byte) fileType {
	base := filepath.Base(rel)
	if strings.HasSuffix(base, "_test.go") {
		return fileTest
	}
	if isGenerated(src) {
		return fileGenerated
	}
	return fileProduction
}

// isGenerated follows the Go convention: a "// Code generated ... DO NOT
// EDIT." comment line before the package clause.
func isGenerated(src []byte) bool {
	for _, line := range strings.SplitN(string(src), "\n", 32) {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "package ") {
			return false
		}
		if strings.HasPrefix(trimmed, "// Code generated") && strings.Contains(trimmed, "DO NOT EDIT") {
			return true
		}
	}
	return false
}

// classifyModule maps a repo-relative path to its owning module segment used
// in boundary rules and per-module report rollups. Business modules use the
// canonical "internal/modules/<name>" form; support trees keep their root
// segment.
func classifyModule(rel string) string {
	parts := strings.Split(rel, "/")
	switch {
	case len(parts) >= 3 && parts[0] == "internal" && parts[1] == "modules":
		return "internal/modules/" + parts[2]
	case len(parts) >= 2 && parts[0] == "internal":
		return "internal/" + parts[1]
	case parts[0] == "cmd":
		return "cmd"
	default:
		return "root"
	}
}

// countLines counts newline-terminated lines (mirrors wc -l).
func countLines(src []byte) int {
	n := strings.Count(string(src), "\n")
	if len(src) > 0 && src[len(src)-1] != '\n' {
		n++
	}
	return n
}
