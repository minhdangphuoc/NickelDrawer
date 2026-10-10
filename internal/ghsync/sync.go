package ghsync

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"hash"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"koboapps/internal/i18n"
	"koboapps/internal/inbox"
)

// partPattern names the temp files of downloads in progress; a killed run leaves them behind.
const partPattern = ".ghsync-*.part"

// Paths locates the files of an app folder.
type Paths struct{ AppDir string }

func (paths Paths) Config() string { return filepath.Join(paths.AppDir, "data", "config.json") }
func (paths Paths) Token() string  { return filepath.Join(paths.AppDir, "data", "token") }
func (paths Paths) State() string  { return filepath.Join(paths.AppDir, "data", "state.json") }
func (paths Paths) Log() string    { return filepath.Join(paths.AppDir, "data", "log.txt") }

// Summary is the outcome of one run.
type Summary struct {
	New           int
	Existing      int
	Deleted       int // recorded in state but removed by the user; not downloaded again
	Skipped       int // not on the inbox allowlist, too large, or content that does not match its type
	Errors        int
	Truncated     bool
	Unchanged     bool // GitHub said nothing changed since the last complete run
	FolderMissing bool // the configured folder holds no files on this branch
}

// Progress is what a running sync leaves for the app screen.
type Progress struct {
	Done  int    `json:"done"`
	Total int    `json:"total"`
	File  string `json:"file"`
}

// syncer holds everything one run needs.
type syncer struct {
	config Config
	rules  inbox.Rules
	client *Client
	state  State
	log    *Logger
	dryRun bool
	branch string

	messages *i18n.Bundle // renders errors saved to state.json for the status screen

	redownloadDeleted bool
	progressFile      string
}

// Options tune a run.
type Options struct {
	Paths   Paths
	APIBase string
	DryRun  bool
	Backoff time.Duration
	Log     *Logger // nil: open data/log.txt

	// Rules is the inbox allowlist; nil loads the host's routing rules (and the user's override).
	Rules *inbox.Rules

	// Messages is the user's language; nil keeps message keys, which is fine for tests.
	Messages *i18n.Bundle

	// RedownloadDeleted fetches files again that state.json knows but the user deleted.
	RedownloadDeleted bool

	// ProgressFile receives the download count while the run works; it is removed at the end.
	ProgressFile string
	// RescanFlag is created when books were added, so the screen can ask Nickel to rescan once.
	RescanFlag string
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
	rules, err := loadRules(options.Rules, log)
	if err != nil {
		return summary, err
	}
	state, err := LoadState(options.Paths.State())
	if err != nil {
		return summary, err
	}
	run := &syncer{
		config: config,
		rules:  rules,
		client: NewClient(options.APIBase, token, options.Backoff),
		state:  state,
		log:    log,
		dryRun: options.DryRun,

		messages: options.Messages,

		redownloadDeleted: options.RedownloadDeleted,
		progressFile:      options.ProgressFile,
	}
	defer run.clearProgress()
	if !options.DryRun {
		run.removeStaleParts()
	}
	summary, err = run.execute(ctx)
	if !options.DryRun {
		if saveErr := run.saveState(summary, err, options.Paths.State()); saveErr != nil {
			err = errors.Join(err, saveErr)
		}
		if summary.New > 0 {
			markRescanNeeded(options.RescanFlag, log)
		}
	}
	return summary, scrubbed(err, token)
}

func loadRules(given *inbox.Rules, log *Logger) (inbox.Rules, error) {
	if given != nil {
		return *given, nil
	}
	rules, overrideErr, err := inbox.LoadRules(inbox.ShippedRulesPath(), inbox.OverrideRulesPath())
	if err != nil {
		return inbox.Rules{}, i18n.NewError("msg.err_rules_unreadable").WithCause(err)
	}
	if overrideErr != nil {
		log.Printf("warning: the user's routing override is ignored: %v", overrideErr)
	}
	return rules, nil
}

func markRescanNeeded(flagPath string, log *Logger) {
	if flagPath == "" {
		return
	}
	if err := os.WriteFile(flagPath, nil, 0o644); err != nil {
		log.Printf("warning: cannot leave the rescan marker: %v", err)
	}
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

	treeKey := run.treeKey()
	listing, err := run.client.ListTree(ctx, run.config, branch, run.knownETag(treeKey))
	if err != nil {
		return summary, err
	}
	if listing.NotModified {
		summary.Unchanged = true
		summary.Existing = len(run.state.Files)
		run.log.Printf("repo %s@%s: unchanged since the last run", run.config.Repo, branch)
		return summary, nil
	}
	if listing.Truncated {
		summary.Truncated = true
		run.log.Printf("warning: repo tree truncated; use a smaller subfolder")
	}
	candidates := run.selectCandidates(listing.Entries)
	run.log.Printf("repo %s@%s: %d matching files", run.config.Repo, branch, len(candidates))
	summary.FolderMissing = run.config.Path != "" && !run.hasFilesUnderPath(listing.Entries)
	if summary.FolderMissing {
		run.log.Printf("warning: folder %q has no files on %s", run.config.Path, branch)
	}

	pending := run.classify(candidates, &summary)
	if err := run.downloadAll(ctx, pending, &summary); err != nil {
		return summary, err
	}
	run.rememberTree(listing.ETag, treeKey, summary)
	run.log.Printf("done: %d new, %d existing, %d skipped, %d errors", summary.New, summary.Existing, summary.Skipped, summary.Errors)
	return summary, nil
}

// treeKey names the settings a listing was fetched with; a changed folder or extension list must
// not be answered with "nothing changed".
func (run *syncer) treeKey() string {
	parts := []string{run.config.Repo, run.branch, run.config.Path, run.config.Dest}
	return strings.Join(append(parts, run.config.Extensions...), "|")
}

// knownETag is the validator worth sending: only after a clean run with the same settings, and
// never when the user asks for deleted files back.
func (run *syncer) knownETag(treeKey string) string {
	wasClean := run.state.LastRun != "" && run.state.LastError == "" && run.state.LastErrors == 0
	if run.redownloadDeleted || !wasClean || run.state.TreeKey != treeKey {
		return ""
	}
	return run.state.TreeETag
}

func (run *syncer) rememberTree(etag, treeKey string, summary Summary) {
	run.state.TreeETag, run.state.TreeKey = "", ""
	if etag != "" && summary.Errors == 0 && !summary.Truncated && !run.dryRun {
		run.state.TreeETag, run.state.TreeKey = etag, treeKey
	}
}

// candidate is a repo file with an allowed extension, below the configured folder.
type candidate struct {
	entry TreeEntry
	rule  inbox.Rule
}

// pendingFile is a candidate that still has to be downloaded.
type pendingFile struct {
	candidate
	localPath string
}

func (run *syncer) selectCandidates(entries []TreeEntry) []candidate {
	var found []candidate
	for _, entry := range entries {
		if entry.Type != "blob" || !run.isUnderPath(entry.Path) || !run.hasConfiguredExtension(entry.Path) {
			continue
		}
		rule, _, isAllowed := run.rules.Match(entry.Path)
		if !isAllowed {
			run.log.Printf("skipped, type not on the allowlist: %s", entry.Path)
			continue
		}
		found = append(found, candidate{entry: entry, rule: rule})
	}
	return found
}

func (run *syncer) isUnderPath(repoPath string) bool {
	return run.config.Path == "" || strings.HasPrefix(repoPath, run.config.Path+"/")
}

func (run *syncer) hasFilesUnderPath(entries []TreeEntry) bool {
	for _, entry := range entries {
		if entry.Type == "blob" && run.isUnderPath(entry.Path) {
			return true
		}
	}
	return false
}

func (run *syncer) hasConfiguredExtension(repoPath string) bool {
	lower := strings.ToLower(repoPath)
	for _, extension := range run.config.Extensions {
		if strings.HasSuffix(lower, extension) {
			return true
		}
	}
	return false
}

// classify decides for every candidate without touching the network: already here, deleted by the
// user, refused by the allowlist, or still to download. Counts go into summary.
func (run *syncer) classify(candidates []candidate, summary *Summary) []pendingFile {
	var pending []pendingFile
	for _, item := range candidates {
		localPath, err := localPathFor(run.config.DestinationFor(item.rule), run.config.Path, item.entry.Path)
		if err != nil {
			summary.Errors++
			run.log.Printf("error %s: %s", item.entry.Path, run.client.redact(err.Error()))
			continue
		}
		if !run.isDownloadable(item, localPath, summary) {
			continue
		}
		if run.dryRun {
			run.log.Printf("would download: %s", item.entry.Path)
			summary.New++
			continue
		}
		pending = append(pending, pendingFile{candidate: item, localPath: localPath})
	}
	return pending
}

// isDownloadable is false for files that need no download; it counts why in summary.
func (run *syncer) isDownloadable(item candidate, localPath string, summary *Summary) bool {
	entry := item.entry
	if entry.Size > item.rule.MaxBytes {
		summary.Skipped++
		run.log.Printf("skipped, larger than %d bytes: %s", item.rule.MaxBytes, entry.Path)
		return false
	}
	if run.state.Rejected[entry.Path] == entry.SHA {
		summary.Skipped++
		return false
	}
	if _, err := os.Lstat(localPath); err == nil {
		run.rememberExisting(entry)
		summary.Existing++
		return false
	}
	if _, wasSynced := run.state.Files[entry.Path]; wasSynced && !run.redownloadDeleted {
		run.log.Printf("deleted locally, skipped: %s", entry.Path)
		summary.Deleted++
		return false
	}
	return true
}

// rememberExisting records files the user copied by hand; never touches them.
func (run *syncer) rememberExisting(entry TreeEntry) {
	if run.dryRun {
		return
	}
	if _, alreadyRecorded := run.state.Files[entry.Path]; alreadyRecorded {
		return // repo content may have changed; add-only keeps the local copy
	}
	run.state.Files[entry.Path] = FileState{SHA: entry.SHA, Size: entry.Size, SyncedAt: time.Now().Format(time.RFC3339)}
}

func (run *syncer) downloadAll(ctx context.Context, pending []pendingFile, summary *Summary) error {
	for index, file := range pending {
		run.writeProgress(Progress{Done: index, Total: len(pending), File: filepath.Base(file.localPath)})
		err := run.fetch(ctx, file, summary)
		if err == nil {
			continue
		}
		var stopRun fatalError
		if errors.As(err, &stopRun) {
			return err
		}
		summary.Errors++
		run.log.Printf("error %s: %s", file.entry.Path, run.client.redact(err.Error()))
	}
	return nil
}

// fetch downloads one file and records the result in state and summary.
func (run *syncer) fetch(ctx context.Context, file pendingFile, summary *Summary) error {
	isAccepted, err := run.download(ctx, file)
	if err != nil {
		return err
	}
	entry := file.entry
	if !isAccepted {
		run.state.Rejected[entry.Path] = entry.SHA
		summary.Skipped++
		run.log.Printf("skipped, content does not match a %s file: %s", file.rule.Type, entry.Path)
		return nil
	}
	run.state.Files[entry.Path] = FileState{SHA: entry.SHA, Size: entry.Size, SyncedAt: time.Now().Format(time.RFC3339)}
	run.log.Printf("downloaded: %s (%d bytes)", entry.Path, entry.Size)
	summary.New++
	return nil
}

// download fetches into a temp file in the destination folder, verifies the git blob sha and the
// allowlist's content check, then renames into place. The final name never holds a partial file.
// It reports false when the content is not what the extension promises.
func (run *syncer) download(ctx context.Context, file pendingFile) (isAccepted bool, err error) {
	if err := os.MkdirAll(filepath.Dir(file.localPath), 0o755); err != nil {
		return false, i18n.NewError("msg.err_mkdir").WithCause(err)
	}
	temp, err := os.CreateTemp(filepath.Dir(file.localPath), partPattern)
	if err != nil {
		return false, i18n.NewError("msg.err_tempfile").WithCause(err)
	}
	defer os.Remove(temp.Name()) // no-op after a successful rename

	var gotSHA string
	err = run.client.Download(ctx, run.config, run.branch, file.entry.Path, func(body io.Reader) error {
		sha, err := writeAndHash(temp, body, file.entry.Size)
		gotSHA = sha
		return err
	})
	syncErr := temp.Sync()
	closeErr := temp.Close()
	if err != nil {
		return false, err
	}
	if err := errors.Join(syncErr, closeErr); err != nil {
		return false, i18n.NewError("msg.err_write").WithCause(err)
	}
	if gotSHA != file.entry.SHA {
		return false, i18n.NewError("msg.err_checksum", "want", file.entry.SHA, "got", gotSHA)
	}
	matches, err := file.rule.ContentMatches(temp.Name())
	if err != nil || !matches {
		return false, err
	}
	return true, moveWithoutOverwrite(temp.Name(), file.localPath)
}

// moveWithoutOverwrite renames into place unless a file appeared at the name meanwhile (the user
// may copy the same book over USB while a sync runs); add-only never replaces it.
func moveWithoutOverwrite(from, to string) error {
	if _, err := os.Lstat(to); err == nil {
		return nil
	}
	if err := os.Rename(from, to); err != nil {
		return i18n.NewError("msg.err_write").WithCause(err)
	}
	return nil
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

// removeStaleParts deletes temp files a killed run left in the destination folders. Only one
// sync runs at a time (the start script checks), so none of them is in use.
func (run *syncer) removeStaleParts() {
	for _, folder := range run.destinationFolders() {
		walkErr := filepath.WalkDir(folder, func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				return nil // a missing folder simply has nothing to clean
			}
			if matched, _ := filepath.Match(partPattern, entry.Name()); matched && !entry.IsDir() {
				if removeErr := os.Remove(path); removeErr == nil {
					run.log.Printf("removed an unfinished download: %s", path)
				}
			}
			return nil
		})
		if walkErr != nil {
			run.log.Printf("warning: cannot clean %s: %v", folder, walkErr)
		}
	}
}

func (run *syncer) destinationFolders() []string {
	seen := map[string]bool{}
	var folders []string
	for _, rule := range run.rules.Rules {
		folder := run.config.DestinationFor(rule)
		if !seen[folder] {
			seen[folder] = true
			folders = append(folders, folder)
		}
	}
	return folders
}

func (run *syncer) writeProgress(progress Progress) {
	if run.progressFile == "" {
		return
	}
	encoded, err := json.Marshal(progress)
	if err == nil {
		err = writeFileAtomically(run.progressFile, encoded)
	}
	if err != nil {
		run.log.Printf("warning: cannot write the progress: %v", err)
	}
}

func (run *syncer) clearProgress() {
	if run.progressFile != "" {
		os.Remove(run.progressFile)
	}
}
