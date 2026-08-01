package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// swaggerSpec describes the minimal Swagger/OpenAPI 2.0 shape required for splitting paths.
type swaggerSpec struct {
	Swagger             string                    `json:"swagger"`
	Info                map[string]any            `json:"info"`
	Host                string                    `json:"host,omitempty"`
	BasePath            string                    `json:"basePath,omitempty"`
	Schemes             []string                  `json:"schemes,omitempty"`
	Consumes            []string                  `json:"consumes,omitempty"`
	Produces            []string                  `json:"produces,omitempty"`
	SecurityDefinitions map[string]any            `json:"securityDefinitions,omitempty"`
	Security            []map[string]any          `json:"security,omitempty"`
	Tags                []map[string]any          `json:"tags,omitempty"`
	Paths               map[string]map[string]any `json:"paths"`
	Definitions         map[string]any            `json:"definitions,omitempty"`
	Parameters          map[string]any            `json:"parameters,omitempty"`
	Responses           map[string]any            `json:"responses,omitempty"`
	ExternalDocs        map[string]any            `json:"externalDocs,omitempty"`
}

func main() {
	masterPath := flag.String("master", "api/swagger/swagger.json", "path to the master swagger.json")
	mappingPath := flag.String("mapping", "api/swagger/swagger-module-mapping.json", "path to the tag->module mapping (JSON)")
	outDir := flag.String("out", "api/swagger/modules", "output directory for per-module specs")
	flag.Parse()

	if err := run(*masterPath, *mappingPath, *outDir); err != nil {
		fmt.Fprintln(os.Stderr, "swagger-split:", err)
		os.Exit(1)
	}
}

func run(masterPath, mappingPath, outDir string) error {
	masterData, err := os.ReadFile(masterPath)
	if err != nil {
		return fmt.Errorf("read master spec: %w", err)
	}
	var master swaggerSpec
	if err := json.Unmarshal(masterData, &master); err != nil {
		return fmt.Errorf("parse master spec: %w", err)
	}

	mappingData, err := os.ReadFile(mappingPath)
	if err != nil {
		return fmt.Errorf("read mapping: %w", err)
	}
	var moduleTags map[string][]string
	if err := json.Unmarshal(mappingData, &moduleTags); err != nil {
		return fmt.Errorf("parse mapping: %w", err)
	}

	// tagModule maps each tag to its owning module.
	tagModule := map[string]string{}
	for module, tags := range moduleTags {
		for _, t := range tags {
			if prev, exists := tagModule[t]; exists && prev != module {
				return fmt.Errorf("tag '%s' belongs to 2 modules: %s and %s", t, prev, module)
			}
			tagModule[t] = module
		}
	}

	// Group paths by module.
	modulePaths := map[string]map[string]map[string]any{}
	for path, operations := range master.Paths {
		if len(operations) == 0 {
			continue
		}
		module := operationModule(operations, tagModule)
		if module == "" {
			return fmt.Errorf("path '%s' cannot be mapped to any module (tag missing from mapping)", path)
		}
		if modulePaths[module] == nil {
			modulePaths[module] = map[string]map[string]any{}
		}
		modulePaths[module][path] = operations
	}

	// Validate integrity: total paths across modules == master paths.
	masterPathCount := 0
	for p := range master.Paths {
		if len(master.Paths[p]) > 0 {
			masterPathCount++
		}
	}
	splitPathCount := 0
	for _, paths := range modulePaths {
		splitPathCount += len(paths)
	}
	if splitPathCount != masterPathCount {
		return fmt.Errorf("total split paths (%d) differs from master (%d)", splitPathCount, masterPathCount)
	}

	// Warn about tags in the master spec that are not present in the mapping.
	unknownTags := map[string]bool{}
	for _, ops := range master.Paths {
		for _, op := range ops {
			opMap, ok := op.(map[string]any)
			if !ok {
				continue
			}
			for _, t := range stringSlice(opMap["tags"]) {
				if _, exists := tagModule[t]; !exists {
					unknownTags[t] = true
				}
			}
		}
	}
	if len(unknownTags) > 0 {
		keys := make([]string, 0, len(unknownTags))
		for t := range unknownTags {
			keys = append(keys, t)
		}
		sort.Strings(keys)
		fmt.Fprintf(os.Stderr, "swagger-split: warning — tags missing from mapping: %s\n", strings.Join(keys, ", "))
	}

	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return fmt.Errorf("create output directory: %w", err)
	}

	modules := make([]string, 0, len(modulePaths))
	for m := range modulePaths {
		modules = append(modules, m)
	}
	sort.Strings(modules)

	for _, module := range modules {
		spec := master
		spec.Paths = modulePaths[module]
		// Keep only tags belonging to this module.
		spec.Tags = filterTags(master.Tags, moduleTags[module])
		data, err := json.MarshalIndent(spec, "", "  ")
		if err != nil {
			return fmt.Errorf("marshal spec for module %s: %w", module, err)
		}
		dir := filepath.Join(outDir, module)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("create module directory %s: %w", module, err)
		}
		if err := os.WriteFile(filepath.Join(dir, "swagger.json"), data, 0o644); err != nil {
			return fmt.Errorf("write spec for module %s: %w", module, err)
		}
	}

	fmt.Printf("swagger-split: generated %d module specs into %s (master: %d paths)\n", len(modules), outDir, masterPathCount)
	return nil
}

// operationModule returns the module that all tags of the operation belong to.
func operationModule(operations map[string]any, tagModule map[string]string) string {
	module := ""
	assigned := false
	for _, rawOp := range operations {
		opMap, ok := rawOp.(map[string]any)
		if !ok {
			continue
		}
		tags := stringSlice(opMap["tags"])
		if len(tags) == 0 {
			// Operation without tags: keep as-is (handled by caller if module already assigned).
			continue
		}
		for _, t := range tags {
			m, exists := tagModule[t]
			if !exists {
				return ""
			}
			if assigned && m != module {
				return ""
			}
			module = m
			assigned = true
		}
	}
	if !assigned {
		return ""
	}
	return module
}

func stringSlice(v any) []string {
	if v == nil {
		return nil
	}
	arr, ok := v.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(arr))
	for _, item := range arr {
		if s, ok := item.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

func filterTags(all []map[string]any, allowed []string) []map[string]any {
	allowedSet := map[string]bool{}
	for _, a := range allowed {
		allowedSet[a] = true
	}
	var out []map[string]any
	for _, t := range all {
		if name, ok := t["name"].(string); ok && allowedSet[name] {
			out = append(out, t)
		}
	}
	return out
}
