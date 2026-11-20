package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/theprideofwolves/logflow/internal/analysis"
	"github.com/theprideofwolves/logflow/internal/assistant"
)

func TestAssistantCLIAndRemoteOptIn(t *testing.T) {
	r, err := analysis.Process(context.Background(), strings.NewReader(analysis.Demo()), analysis.DefaultOptions())
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "investigation.json")
	b, _ := json.Marshal(saved{Report: r, Demo: true})
	if err = os.WriteFile(path, b, 0600); err != nil {
		t.Fatal(err)
	}
	args := []string{"-assist", path, "-alert", r.Alerts[0].ID}
	var out, errs bytes.Buffer
	if code := run(args, &out, &errs); code != 0 {
		t.Fatal(code, errs.String())
	}
	var draft assistant.Draft
	if err = json.Unmarshal(out.Bytes(), &draft); err != nil || len(draft.Facts) != 3 || draft.Mode != "offline" || draft.ID == "" {
		t.Fatal(err, out.String())
	}
	bare, _ := json.Marshal(r)
	if err = os.WriteFile(path, bare, 0600); err != nil {
		t.Fatal(err)
	}
	if code := run(args, &bytes.Buffer{}, &bytes.Buffer{}); code != 0 {
		t.Fatal("bare CLI report was rejected", code)
	}
	for _, extra := range [][]string{{"-provider", "openai"}, {"-threshold", "2"}, {"-provider", "other"}} {
		var rejected, diag bytes.Buffer
		code := run(append(append([]string{}, args...), extra...), &rejected, &diag)
		if code != 2 || rejected.Len() != 0 {
			t.Fatal(extra, code, rejected.String())
		}
	}
}
