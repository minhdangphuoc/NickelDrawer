package ghsync

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"koboapps/internal/i18n"
)

// Paths locates the files of an app folder.
type Paths struct{ AppDir string }

func (paths Paths) Config() string { return filepath.Join(paths.AppDir, "data", "config.json") }
func (paths Paths) Token() string  { return filepath.Join(paths.AppDir, "data", "token") }
func (paths Paths) State() string  { return filepath.Join(paths.AppDir, "data", "state.json") }
func (paths Paths) Log() string    { return filepath.Join(paths.AppDir, "data", "log.txt") }

// Summary is the outcome of one run.
type Summary struct {
	New       int
	Existing  int
	Deleted   int // recorded in state but removed by the user; not downloaded again
	Errors    int
	Truncated bool
}

// syncer holds everything one run needs.
type syncer struct {
	config Config
	client *Client
	state  State
	log    *Logger
	dryRun bool
	branch string

	messages *i18n.Bundle // renders errors saved to state.json for the status screen

	redownloadDeleted bool
}

// Options tune a run.
type Options struct {
	Paths   Paths
	APIBase string
	DryRun  bool
	Backoff time.Duration
	Log     *Logger // nil: open data/log.txt

	// Messages is the user's language; nil keeps message keys, which is fine for tests.
	Messages *i18n.Bundle

	// RedownloadDeleted fetches files again that state.json knows but the user deleted.
	RedownloadDeleted bool
}

// Run performs one add-only sync. The state file is saved even when the run
// fails, so completed downloads are remembered.
func Run(ctx context.Context, options Options) (summary Summary, err error) {
	config, err := LoadConfig(options.Paths.Config())
	if err != nil {
		return summary, err
	}
	token, warning, err := LoadToken(options.Paths.Token())
	if err != nil {
		return summary, err
	}
	log := options.Log
	if log == nil {
		logFile, openErr := NewLogFile(options.Paths.Log())
		if openErr != nil {
			return summary, i18n.NewError("msg.err_log_unopenable").WithCause(openErr)
		}
		log = NewLogger(logFile, token)
	}
	if warning != "" {
		log.Printf("warning: %s", warning)
	}
	state, err := LoadState(options.Paths.State())
	if err != nil {
		return summary, err
	}
	run := &syncer{
		config: config,
		client: NewClient(options.APIBase, token, options.Backoff),
		state:  state,
		log:    log,
		dryRun: options.DryRun,

		messages: options.Messages,

		redownloadDeleted: options.RedownloadDeleted,
	}
	summary, err = run.execute(ctx)
	if !options.DryRun {
		if saveErr := run.saveState(summary, err, options.Paths.State()); saveErr != nil {
			err = errors.Join(err, saveErr)
		}
	}
	return summary, err
}

func (run *syncer) saveState(summary Summary, runErr error, statePath string) error {
	run.state.LastRun = time.Now().Format(time.RFC3339)
	run.state.LastNew = summary.New
	run.state.LastExisting = summary.Existing
	run.state.LastErrors = summary.Errors
	run.state.LastError = ""
	if runErr != nil {
		run.state.LastError = run.client.redact(run.messages.Describe(runErr))
	}
	return run.state.Save(statePath)
}

func (run *syncer) execute(ctx context.Context) (Summary, error) {
	var summary Summary
	branch := run.config.Branch
	if branch == "" {
		defaultBranch, err := run.client.DefaultBranch(ctx, run.config)
		if err != nil {
			return summary, err
		}
		branch = defaultBranch
	}
	run.branch = branch

	tree, err := run.client.ListTree(ctx, run.config, branch)
	if err != nil {
		return summary, err
	}
	if tree.Truncated {
		summary.Truncated = true
		run.log.Printf("warning: repo tree truncated; use a smaller subfolder")
	}
	books := run.selectBooks(tree.Entries)
	run.log.Printf("repo %s@%s: %d matching files", run.config.Repo, branch, len(books))

	for _, entry := range books {
		if err := run.syncEntry(ctx, entry, &summary); err != nil {
			var fatal fatalError
			if errors.As(err, &fatal) {
				return summary, err
			}
			summary.Errors++
			run.log.Printf("error %s: %s", entry.Path, run.client.redact(err.Error()))
		}
	}
	run.log.Printf("done: %d new, %d existing, %d errors", summary.New, summary.Existing, summary.Errors)
	return summary, nil
}

func (run *syncer) selectBooks(entries []TreeEntry) []TreeEntry {
	var books []TreeEntry
	for _, entry := range entries {
		if entry.Type == "blob" && run.isUnderPath(entry.Path) && run.hasAllowedExtension(entry.Path) {
			books = append(books, entry)
		}
	}
	return books
}

func (run *syncer) isUnderPath(repoPath string) bool {
	return run.config.Path == "" || strings.HasPrefix(repoPath, run.config.Path+"/")
}

func (run *syncer) hasAllowedExtension(repoPath string) bool {
	lower := strings.ToLower(repoPath)
	for _, extension := range run.config.Extensions {
		if strings.HasSuffix(lower, extension) {
			return true
		}
	}
	return false
}

func (run *syncer) syncEntry(ctx context.Context, entry TreeEntry, summary *Summary) error {
	localPath, err := localPathFor(run.config.Dest, run.config.Path, entry.Path)
	if err != nil {
		return err
	}
	if _, err := os.Stat(localPath); err == nil {
		run.rememberExisting(entry, localPath)
		summary.Existing++
		return nil
	}
	if _, wasSynced := run.state.Files[entry.Path]; wasSynced && !run.redownloadDeleted {
		run.log.Printf("deleted locally, skipped: %s", entry.Path)
		summary.Deleted++
		return nil
	}
	if run.dryRun {
		run.log.Printf("would download: %s", entry.Path)
		summary.New++
		return nil
	}
	if err := run.download(ctx, entry, localPath); err != nil {
		return err
	}
	run.state.Files[entry.Path] = FileState{SHA: entry.SHA, Size: entry.Size, SyncedAt: time.Now().Format(time.RFC3339)}
	run.log.Printf("downloaded: %s (%d bytes)", entry.Path, entry.Size)
	summary.New++
	return nil
}

// rememberExisting records files the user copied by hand; never touches them.
func (run *syncer) rememberExisting(entry TreeEntry, localPath string) {
	if run.dryRun {
		return
	}
	if _, alreadyRecorded := run.state.Files[entry.Path]; alreadyRecorded {
		return // repo content may have changed; add-only keeps the local copy
	}
	run.state.Files[entry.Path] = FileState{SHA: entry.SHA, Size: entry.Size, SyncedAt: time.Now().Format(time.RFC3339)}
}

// download fetches into a temp file in the destination folder, verifies the
// git blob sha, then renames into place.
func (run *syncer) download(ctx context.Context, entry TreeEntry, localPath string) error {
	if err := os.MkdirAll(filepath.Dir(localPath), 0o755); err != nil {
		return i18n.NewError("msg.err_mkdir").WithCause(err)
	}
	temp, err := os.CreateTemp(filepath.Dir(localPath), ".ghsync-*.part")
	if err != nil {
		return i18n.NewError("msg.err_tempfile").WithCause(err)
	}
	defer os.Remove(temp.Name()) // no-op after a successful rename

	var gotSHA string
	err = run.client.Download(ctx, run.config, run.branch, entry.Path, func(body io.Reader) error {
		sha, err := writeAndHash(temp, body, entry.Size)
		gotSHA = sha
		return err
	})
	syncErr := temp.Sync()
	closeErr := temp.Close()
	if err != nil {
		return err
	}
	if err := errors.Join(syncErr, closeErr); err != nil {
		return i18n.NewError("msg.err_write").WithCause(err)
	}
	if gotSHA != entry.SHA {
		return i18n.NewError("msg.err_checksum", "want", entry.SHA, "got", gotSHA)
	}
	return os.Rename(temp.Name(), localPath)
}

// writeAndHash restarts the file, streams body into it and returns the git
// blob sha1 of what was written. Size comes from the tree listing.
func writeAndHash(file *os.File, body io.Reader, size int64) (string, error) {
	if err := file.Truncate(0); err != nil {
		return "", err
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return "", err
	}
	hasher := newBlobHasher(size)
	if _, err := io.Copy(io.MultiWriter(file, hasher), body); err != nil {
		return "", err
	}
	return hex.EncodeToString(hasher.Sum(nil)), nil
}

func newBlobHasher(size int64) hash.Hash {
	hasher := sha1.New()
	fmt.Fprintf(hasher, "blob %d\x00", size)
	return hasher
}
