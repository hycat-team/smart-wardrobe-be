package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// acceptedEntry is one advisory finding the team reviewed and decided to keep
// (FR-011). Matched by rule + file + symbol suffix; lines are intentionally
// not part of the key because they drift between edits.
type acceptedEntry struct {
	RuleID string `json:"ruleId"`
	File   string `json:"file"`
	Symbol string `json:"symbol"`
	Reason string `json:"reason"`
}

type acceptedList struct {
	Entries []acceptedEntry `json:"entries"`
}

// acceptedFilePath is the keep-list location, relative to the repo root.
const acceptedFilePath = "internal/tools/hygiene-scan/accepted.json"

// loadAcceptedList reads the keep-list; a missing file means nothing is
// accepted yet and every advisory finding stays open.
func loadAcceptedList(root string) *acceptedList {
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(acceptedFilePath)))
	if err != nil {
		return nil
	}
	var list acceptedList
	if json.Unmarshal(data, &list) != nil {
		return nil
	}
	return &list
}

// applyAccepted marks open violations matching the keep-list as
// accepted-with-reason and returns how many were matched.
func applyAccepted(violations []Violation, list *acceptedList) int {
	if list == nil {
		return 0
	}
	matched := 0
	for i := range violations {
		v := &violations[i]
		if v.Status != StatusOpen {
			continue
		}
		for _, e := range list.Entries {
			if e.RuleID != v.RuleID {
				continue
			}
			if !matchAcceptedFile(v.Location.File, e.File) {
				continue
			}
			if !matchAcceptedSymbol(v.Location.Symbol, e.Symbol) {
				continue
			}
			v.Status = StatusAcceptedWithReason
			v.Detail = "accepted-keep: " + e.Reason
			matched++
			break
		}
	}
	return matched
}

func matchAcceptedFile(violationFile, entryFile string) bool {
	return strings.HasSuffix(filepath.ToSlash(violationFile), entryFile)
}

func matchAcceptedSymbol(violationSymbol, entrySymbol string) bool {
	// deadcode hits carry a human prefix, e.g. "unreachable func: Foo.Bar".
	violationSymbol = strings.TrimPrefix(violationSymbol, "unreachable func: ")
	if violationSymbol == entrySymbol {
		return true
	}
	return strings.HasSuffix(violationSymbol, "."+entrySymbol)
}
