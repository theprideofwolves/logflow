// Package assistant builds evidence-grounded drafts. It has no Jira or shell capability.
package assistant

import (
	"context"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/netip"
	"strings"
	"time"

	"github.com/theprideofwolves/logflow/internal/analysis"
)

//go:embed runbooks.json
var runbooksJSON []byte

type Section struct{ ID, Version, Title, Text, Source string }
type Claim struct {
	Text      string
	Citations []string
}
type Reference struct{ ID, Text string }
type ToolCall struct {
	Name      string
	Arguments json.RawMessage
	Result    json.RawMessage
	Error     string `json:",omitempty"`
}
type Draft struct {
	Version, ID, Snapshot, AlertID, Mode, Status, Warning string
	At                                                    time.Time
	Facts                                                 []Claim
	Limitations                                           []string
	References                                            []Reference
	Guidance                                              []Section
	Calls                                                 []ToolCall
	PromptVersion, Model                                  string
	Prompt                                                string
	CorpusSHA256                                          string
	ProviderContext                                       json.RawMessage `json:",omitempty"`
}
type Session struct {
	Report   *analysis.Report
	Alert    analysis.Alert
	Snapshot string
}

func Open(r *analysis.Report, id string) (*Session, error) {
	if r == nil || len(r.Alerts) > 1000 || r.Options.Validate() != nil {
		return nil, errors.New("valid report required")
	}
	if r.Accepted < 0 || r.Rejected < 0 || r.Unattributed < 0 || r.Unattributed > r.Accepted || r.Lines != r.Accepted+r.Rejected || (r.Complete && (r.Rejected > 0 || r.Limit != "")) {
		return nil, errors.New("inconsistent report completeness or counts")
	}
	b, err := json.Marshal(r)
	if err != nil || len(b) > 16<<20 {
		return nil, errors.New("report exceeds assistant snapshot limit")
	}
	// Copy to prevent caller mutations during an investigation.
	var copy analysis.Report
	if json.Unmarshal(b, &copy) != nil {
		return nil, errors.New("invalid snapshot")
	}
	sum := sha256.Sum256(b)
	for _, a := range copy.Alerts {
		if a.ID == id {
			if raw, e := hex.DecodeString(a.ID); e != nil || len(raw) != 32 {
				return nil, errors.New("invalid alert identity")
			}
			if a.ID == "" || len(a.Evidence) == 0 || len(a.Evidence) > 100 || a.Count != len(a.Evidence) {
				return nil, errors.New("missing or inconsistent contributing evidence")
			}
			ip, e := netip.ParseAddr(a.IP)
			if e != nil || ip.Unmap().String() != a.IP {
				return nil, errors.New("invalid alert IP")
			}
			for i, v := range a.Evidence {
				if v.Line < 1 || v.Timestamp.IsZero() || v.Message != "Failed login attempt" || v.SourceIP != a.IP || (i > 0 && v.Timestamp.Before(a.Evidence[i-1].Timestamp)) {
					return nil, errors.New("invalid contributing evidence")
				}
			}
			if a.Start != a.Evidence[0].Timestamp || a.End != a.Evidence[len(a.Evidence)-1].Timestamp || a.End.Sub(a.Start) > r.Options.Window || a.Count != r.Options.Threshold {
				return nil, errors.New("inconsistent trigger window")
			}
			return &Session{&copy, a, hex.EncodeToString(sum[:])}, nil
		}
	}
	return nil, errors.New("alert not found")
}
func Sections() []Section { var s []Section; _ = json.Unmarshal(runbooksJSON, &s); return s }

// Call exposes only bounded reads, never paths, URLs, commands or external writes.
func (s *Session) Call(name string, args json.RawMessage) (json.RawMessage, error) {
	if len(args) > 1024 {
		return nil, errors.New("tool arguments too large")
	}
	var in struct {
		ID string `json:"id"`
	}
	dec := json.NewDecoder(strings.NewReader(string(args)))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&in); err != nil {
		return nil, errors.New("invalid tool arguments")
	}
	var extra any
	if dec.Decode(&extra) != io.EOF {
		return nil, errors.New("trailing tool arguments")
	}
	var value any
	switch name {
	case "get_finding":
		if in.ID != s.Alert.ID {
			return nil, errors.New("finding outside this snapshot")
		}
		value = map[string]any{"Alert": s.Alert, "Options": s.Report.Options, "InputHash": s.Report.InputHash, "Complete": s.Report.Complete, "Rejected": s.Report.Rejected, "Limit": s.Report.Limit, "Snapshot": s.Snapshot}
	case "get_evidence":
		if in.ID != s.Alert.ID {
			return nil, errors.New("evidence outside this snapshot")
		}
		value = s.Alert.Evidence
	case "get_runbook_section":
		for _, section := range Sections() {
			if section.ID == in.ID {
				value = section
				break
			}
		}
		if value == nil {
			return nil, errors.New("unknown runbook section")
		}
	default:
		return nil, errors.New("tool is not allowed")
	}
	return json.Marshal(value)
}
func (s *Session) Baseline() Draft {
	a := s.Alert
	d := Draft{Version: "logflow-assistant/v1", Snapshot: s.Snapshot, AlertID: a.ID, Mode: "offline", Status: "draft", At: time.Now().UTC(), PromptVersion: "selection-v1", Calls: []ToolCall{},
		Facts: []Claim{{fmt.Sprintf("%d observed failures from %s triggered this rule.", a.Count, a.IP), []string{"finding"}},
			{fmt.Sprintf("The retained trigger spans %s to %s.", a.Start.Format(time.RFC3339Nano), a.End.Format(time.RFC3339Nano)), []string{"finding"}},
			{fmt.Sprintf("Processing complete: %t; rejected records: %d; missing IP records: %d.", s.Report.Complete, s.Report.Rejected, s.Report.Unattributed), []string{"report"}}},
		Limitations: []string{"This is a draft for analyst review, not a compromise verdict.", "No account, destination or successful-login correlation is performed.", "Sensitivity covers retained trigger events only.", "No action or Jira request is performed by the assistant."},
		References:  []Reference{{"finding", a.ID}, {"report", s.Snapshot}}, Guidance: Sections()}
	if !s.Report.Complete {
		d.Limitations = append(d.Limitations, "Input is incomplete. Do not infer whole-file absence.")
	}
	for _, e := range a.Evidence {
		d.References = append(d.References, Reference{fmt.Sprintf("line:%d", e.Line), fmt.Sprintf("%s %s %s", e.Timestamp.Format(time.RFC3339Nano), e.SourceIP, e.Message)})
	}
	for _, q := range []struct{ name, id string }{{"get_finding", a.ID}, {"get_evidence", a.ID}} {
		args, _ := json.Marshal(map[string]string{"id": q.id})
		v, _ := s.Call(q.name, args)
		d.Calls = append(d.Calls, ToolCall{Name: q.name, Arguments: args, Result: v})
	}
	for _, sec := range d.Guidance {
		args, _ := json.Marshal(map[string]string{"id": sec.ID})
		v, _ := s.Call("get_runbook_section", args)
		d.Calls = append(d.Calls, ToolCall{Name: "get_runbook_section", Arguments: args, Result: v})
	}
	return d
}
func (s *Session) Build(ctx context.Context, provider string) Draft {
	d := s.Baseline()
	hash := sha256.Sum256(runbooksJSON)
	d.CorpusSHA256 = hex.EncodeToString(hash[:])
	if ctx.Err() != nil {
		d.Warning = "Draft generation cancelled; no provider request performed"
		d.Status = "offline fallback"
		provider = "offline"
	}
	if provider != "offline" {
		if err := s.enrich(ctx, &d, provider); err != nil {
			d.Warning = err.Error()
			d.Status = "offline fallback"
			d.Guidance = Sections()
		}
	}
	b, _ := json.Marshal(d)
	sum := sha256.Sum256(b)
	d.ID = hex.EncodeToString(sum[:])
	return d
}
