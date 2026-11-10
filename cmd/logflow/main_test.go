package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCLIRequiresExplicitInput(t *testing.T) {
	// An empty stdin keeps this regression test from blocking on the old behavior.
	f, err := os.CreateTemp(t.TempDir(), "stdin")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	previous := os.Stdin
	os.Stdin = f
	defer func() { os.Stdin = previous }()
	var out, errs bytes.Buffer
	if code := run([]string{"-workers", "4"}, &out, &errs); code != 2 || out.Len() != 0 || !strings.Contains(errs.String(), "file or -") {
		t.Fatalf("code=%d stdout=%s stderr=%s", code, &out, &errs)
	}
}

func TestCLIReportsAndExitCodes(t *testing.T) {
	for _, tc := range []struct {
		name, input string
		code        int
	}{
		{"empty", "", 0},
		{"malformed", "bad\n", 2},
		{"missing fields", "{}\n", 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "input.jsonl")
			if err := os.WriteFile(path, []byte(tc.input), 0600); err != nil {
				t.Fatal(err)
			}
			var out, errs bytes.Buffer
			if code := run([]string{path}, &out, &errs); code != tc.code || !strings.Contains(out.String(), `"Complete":`) {
				t.Fatalf("code=%d out=%s err=%s", code, &out, &errs)
			}
		})
	}
}
