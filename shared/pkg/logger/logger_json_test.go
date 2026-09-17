package logger_test

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"testing"

	"smart-wardrobe-be/shared/pkg/logger"
)

// captureStdout swaps os.Stdout, runs fn, and returns what was written.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	original := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	defer func() { os.Stdout = original }()

	fn()

	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	os.Stdout = original

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatal(err)
	}
	return buf.String()
}

// TestManagedEnvEmitsJSON verifies Cloud Run (non-dev env) gets structured
// JSON logs for Cloud Logging collection (T119, FR-057).
func TestManagedEnvEmitsJSON(t *testing.T) {
	out := captureStdout(t, func() {
		l := logger.New("managed", "", "info", false)
		l.Info("hello structured")
	})

	var decoded map[string]any
	if err := json.Unmarshal([]byte(out), &decoded); err != nil {
		t.Fatalf("expected JSON log on managed env, got %q: %v", out, err)
	}
	if decoded["msg"] != "hello structured" {
		t.Fatalf("expected msg field, got %v", decoded)
	}
}

// TestDevEnvKeepsConsole verifies local/dev keeps the colored console format
// (T119: console for local/dev, JSON for managed).
func TestDevEnvKeepsConsole(t *testing.T) {
	out := captureStdout(t, func() {
		l := logger.New("dev", "", "info", false)
		l.Info("hello console")
	})

	var decoded map[string]any
	if err := json.Unmarshal([]byte(out), &decoded); err == nil {
		t.Fatalf("expected console (non-JSON) log on dev env, got JSON %q", out)
	}
}
