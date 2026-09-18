package main

import (
	"strings"
	"testing"
)

func TestMergeDocsUnionsPathsDefinitionsTags(t *testing.T) {
	account := map[string]any{
		"paths": map[string]any{
			"/api/v1/auth/login": map[string]any{"post": map[string]any{"tags": []any{"Auth"}}},
		},
		"definitions":         map[string]any{"Shared": map[string]any{"type": "object"}},
		"tags":                []any{map[string]any{"name": "Auth"}},
		"securityDefinitions": map[string]any{},
	}
	commerce := map[string]any{
		"paths": map[string]any{
			"/api/v1/brands": map[string]any{"get": map[string]any{"tags": []any{"Brand"}}},
		},
		"definitions": map[string]any{
			"Shared": map[string]any{"type": "object"},
			"OnlyC":  map[string]any{"type": "string"},
		},
		"tags":                []any{map[string]any{"name": "Brand"}},
		"securityDefinitions": map[string]any{},
	}

	combined, err := mergeDocs(map[string]map[string]any{"account": account, "commerce": commerce})
	if err != nil {
		t.Fatal(err)
	}
	if got := len(asMap(combined["paths"])); got != 2 {
		t.Errorf("paths = %d, want 2", got)
	}
	if got := len(asMap(combined["definitions"])); got != 2 {
		t.Errorf("definitions = %d, want 2", got)
	}
	if got := len(asSlice(combined["tags"])); got != 2 {
		t.Errorf("tags = %d, want 2", got)
	}
}

func TestMergeDocsConflictingDefinitionFails(t *testing.T) {
	a := map[string]any{
		"paths":       map[string]any{},
		"definitions": map[string]any{"X": map[string]any{"type": "object"}},
	}
	b := map[string]any{
		"paths":       map[string]any{},
		"definitions": map[string]any{"X": map[string]any{"type": "string"}},
	}
	if _, err := mergeDocs(map[string]map[string]any{"account": a, "commerce": b}); err == nil {
		t.Fatal("expected conflict error for differing definition X")
	}
}

func TestModuleForPath(t *testing.T) {
	tagModule := map[string]string{"Brand": "brand", "Auth": "identity"}
	ops := map[string]any{"get": map[string]any{"tags": []any{"Brand"}}}
	if m, err := moduleForPath(ops, tagModule); err != nil || m != "brand" {
		t.Fatalf("moduleForPath = %q, %v; want brand", m, err)
	}

	if _, err := moduleForPath(map[string]any{"get": map[string]any{"tags": []any{"Nope"}}}, tagModule); err == nil {
		t.Fatal("expected error for unmapped tag")
	}

	span := map[string]any{
		"get":  map[string]any{"tags": []any{"Auth"}},
		"post": map[string]any{"tags": []any{"Brand"}},
	}
	if _, err := moduleForPath(span, tagModule); err == nil {
		t.Fatal("expected error for path spanning two modules")
	}
}

func TestRenderHTMLReplacesCatalog(t *testing.T) {
	html, err := renderHTML(map[string]any{"allUrl": "/api-docs/swagger.json"})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(html, "__CATALOG_JSON__") {
		t.Error("catalog placeholder was not replaced")
	}
	if !strings.Contains(html, `"/api-docs/swagger.json"`) {
		t.Error("catalog value missing from rendered HTML")
	}
	if !strings.Contains(html, `id="service-select"`) {
		t.Error("service selector missing from rendered HTML")
	}
}
