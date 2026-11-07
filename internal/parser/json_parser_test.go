package parser

import "testing"

func TestValidation(t *testing.T) {
	for _, s := range []string{`{}`, `not json`, `{"timestamp":"bad","level":"ERROR","message":"x"}`} {
		if _, err := ParseLine([]byte(s)); err == nil {
			t.Fatalf("accepted %s", s)
		}
	}
	if _, err := ParseLine([]byte(`{"timestamp":"2026-11-06T12:00:00Z","level":"WARN","message":"Failed login attempt","source_ip":"192.0.2.10"}`)); err != nil {
		t.Fatal(err)
	}
}
