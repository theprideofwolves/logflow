package assistant

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"
)

// RecordAction records approval before external delivery. No credential enters it.
func RecordAction(dir, event, approval, snapshot, disclosure string) error {
	auditMu.Lock()
	defer auditMu.Unlock()
	if len(disclosure) > 16384 {
		return errors.New("handoff audit disclosure exceeds limit")
	}
	h := sha256.Sum256([]byte(disclosure))
	record := struct {
		At                                                      time.Time
		Event, Approval, Snapshot, DisclosureSHA256, Disclosure string
	}{time.Now().UTC(), event, approval, snapshot, hex.EncodeToString(h[:]), disclosure}
	path := filepath.Join(dir, "handoff-audit.jsonl")
	if info, e := os.Stat(path); e == nil && info.Size() > 8<<20 {
		return errors.New("handoff audit limit reached; archive it before sending")
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return errors.New("handoff audit unavailable; no approval recorded")
	}
	b, _ := json.Marshal(record)
	_, err = f.Write(append(b, '\n'))
	if err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err == nil {
		err = closeErr
	}
	if err != nil {
		return errors.New("handoff audit could not be saved")
	}
	return nil
}
