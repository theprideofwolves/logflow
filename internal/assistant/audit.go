package assistant

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
)

var auditMu sync.Mutex

// Persist saves an immutable draft and an append-only run record in local app data.
// This is local provenance, not a tamper-proof or multi-user audit service.
func Persist(dir string, d Draft) error {
	auditMu.Lock()
	defer auditMu.Unlock()
	if raw, e := hex.DecodeString(d.ID); e != nil || len(raw) != 32 {
		return errors.New("invalid draft identity")
	}
	path := filepath.Join(dir, "assistant-audit.jsonl")
	if stat, e := os.Stat(path); e == nil && stat.Size() > 32<<20 {
		return errors.New("assistant audit reached 32 MiB; archive it before generating more drafts")
	}
	root := filepath.Join(dir, "investigations")
	if err := os.MkdirAll(root, 0700); err != nil {
		return errors.New("cannot create investigation storage")
	}
	b, err := json.Marshal(d)
	if err != nil || len(b) > 1<<20 {
		return errors.New("draft exceeds storage limit")
	}
	f, err := os.OpenFile(filepath.Join(root, d.ID+".json"), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return errors.New("cannot save investigation draft")
	}
	_, err = f.Write(b)
	if err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err == nil {
		err = closeErr
	}
	if err != nil {
		return errors.New("draft persistence failed")
	}
	f, err = os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return errors.New("cannot append assistant audit")
	}
	b, err = json.Marshal(d)
	if err == nil {
		_, err = f.Write(append(b, '\n'))
	}
	if err == nil {
		err = f.Sync()
	}
	closeErr = f.Close()
	if err == nil {
		err = closeErr
	}
	if err != nil {
		return errors.New("assistant audit write failed; draft was not presented as complete")
	}
	return nil
}
