package assistant

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestProviderEnvelopeContracts(t *testing.T) {
	plan := `{"calls":[{"name":"get_runbook_section","arguments":{"id":"auth-scope"}}]}`
	for _, provider := range []string{"ollama", "openai"} {
		var envelope any
		if provider == "ollama" {
			envelope = map[string]any{"done": true, "done_reason": "stop", "message": map[string]string{"content": plan}}
		} else {
			envelope = map[string]any{"status": "completed", "output": []any{map[string]any{"content": []any{map[string]string{"type": "output_text", "text": plan}}}}}
		}
		b, _ := json.Marshal(envelope)
		text, err := providerPlanText(provider, b)
		if err != nil {
			t.Fatal(err)
		}
		s := fixture(t)
		d := s.Baseline()
		if err = s.applyPlan(&d, text); err != nil || len(d.Guidance) != 1 {
			t.Fatal(err)
		}
	}
	for _, raw := range []string{`{}`, `{"status":"incomplete"}`, `not-json`, `{"done":false,"message":{"content":"ignored"}}`} {
		for _, provider := range []string{"openai", "ollama"} {
			if _, err := providerPlanText(provider, []byte(raw)); err == nil {
				t.Fatal("incomplete envelope accepted", provider, raw)
			}
		}
	}
}
func TestPlanBudgetAndUnavailableSources(t *testing.T) {
	s := fixture(t)
	d := s.Baseline()
	call := `{"name":"get_runbook_section","arguments":{"id":"auth-scope"}}`
	if err := s.applyPlan(&d, []byte(`{"calls":[`+strings.TrimSuffix(strings.Repeat(call+",", 5), ",")+`]}`)); err == nil {
		t.Fatal("tool budget exceeded")
	}
	if _, err := s.Call("get_evidence", json.RawMessage(strings.Repeat("x", 1025))); err == nil {
		t.Fatal("argument budget exceeded")
	}
	if err := s.applyPlan(&d, []byte(`{"calls":[{"name":"get_runbook_section","arguments":{"id":"untrusted-runbook"}}]}`)); err == nil {
		t.Fatal("unknown citation accepted")
	}
	if d.Calls[len(d.Calls)-1].Error == "" {
		t.Fatal("denial missing from trace")
	}
}
