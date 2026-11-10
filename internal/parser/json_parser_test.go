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

func TestJSONLFieldBoundaries(t *testing.T) {
	for _, input := range []string{
		`null`, `[]`, `true`, `{} {}`, `{"timestamp":null,"level":"WARN","message":"x"}`,
		`{"timestamp":"2026-11-06T12:00:00Z","message":"x"}`,
		`{"timestamp":"2026-11-06T12:00:00Z","level":"WARN"}`,
		`{"timestamp":"2026-11-06T12:00:00Z","level":5,"message":"x"}`,
		`{"timestamp":"2026-11-06T12:00:00Z","level":"WARN","message":null}`,
		`{"timestamp":"2026-11-06T12:00:00Z","level":"WARN","message":"x","source_ip":3}`,
	} {
		if _, err := ParseLine([]byte(input)); err == nil {
			t.Fatalf("accepted invalid record: %s", input)
		}
	}
	for _, input := range []string{
		`{"timestamp":"2026-11-06T12:00:00.123456789Z","level":"WARN","message":"x","extra":"ignored"}`,
		`{"timestamp":"2026-11-06T07:00:00-05:00","level":"INFO","message":"unicode: λ"}`,
	} {
		if _, err := ParseLine([]byte(input)); err != nil {
			t.Fatal(input, err)
		}
	}
}

func FuzzParseLine(f *testing.F) {
	f.Add([]byte(`{"timestamp":"2026-11-06T12:00:00Z","level":"WARN","message":"x"}`))
	f.Add([]byte("null"))
	f.Fuzz(func(t *testing.T, input []byte) {
		if len(input) > 64<<10 {
			t.Skip()
		}
		entry, err := ParseLine(input)
		if err == nil && (entry.Timestamp.IsZero() || entry.Level == "" || entry.Message == "") {
			t.Fatal("accepted missing required field", entry)
		}
	})
}
