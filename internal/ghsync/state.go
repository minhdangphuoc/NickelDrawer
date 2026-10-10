package ghsync

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	"koboapps/internal/i18n"
)

// FileState records one synced file.
type FileState struct {
	SHA      string `json:"sha"`
	Size     int64  `json:"size"`
	SyncedAt string `json:"synced_at"`
}

// State is data/state.json.
type State struct {
	LastRun      string               `json:"last_run"`
	LastNew      int                  `json:"last_new"`
	LastExisting int                  `json:"last_existing"`
	LastErrors   int                  `json:"last_errors"`
	LastError    string               `json:"last_error,omitempty"`
	Files        map[string]FileState `json:"files"`

	// Rejected remembers files that failed the content check (repo path -> blob sha), so an
	// unchanged bad file is not downloaded again on every run.
	Rejected map[string]string `json:"rejected,omitempty"`

	// TreeETag is the validator of the last complete listing; TreeKey says which repo, branch and
	// settings it belongs to. GitHub answers 304 (free of rate limit) while nothing changed.
	TreeETag string `json:"tree_etag,omitempty"`
	TreeKey  string `json:"tree_key,omitempty"`
}

// LoadState returns an empty state when the file does not exist yet.
func LoadState(statePath string) (State, error) {
	state := State{Files: map[string]FileState{}, Rejected: map[string]string{}}
	raw, err := os.ReadFile(statePath)
	if errors.Is(err, os.ErrNotExist) {
		return state, nil
	}
	if err != nil {
		return state, i18n.NewError("msg.err_state_unreadable").WithCause(err)
	}
	if err := json.Unmarshal(raw, &state); err != nil {
		return state, i18n.NewError("msg.err_state_corrupt").WithCause(err)
	}
	if state.Files == nil {
		state.Files = map[string]FileState{}
	}
	if state.Rejected == nil {
		state.Rejected = map[string]string{}
	}
	return state, nil
}

// Save writes the state atomically (temp file, fsync, rename).
func (state State) Save(statePath string) error {
	encoded, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	return writeFileAtomically(statePath, encoded)
}

func writeFileAtomically(target string, content []byte) error {
	temp, err := os.CreateTemp(filepath.Dir(target), ".ghsync-*.part")
	if err != nil {
		return err
	}
	_, writeErr := temp.Write(content)
	syncErr := temp.Sync()
	closeErr := temp.Close()
	if err := errors.Join(writeErr, syncErr, closeErr); err != nil {
		os.Remove(temp.Name())
		return err
	}
	if err := os.Rename(temp.Name(), target); err != nil {
		os.Remove(temp.Name())
		return err
	}
	return nil
}
