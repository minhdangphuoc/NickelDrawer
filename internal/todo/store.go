// Package todo keeps the To-do app's task list in one JSON file.
//
// The file is rewritten through a temp file, fsync and rename, so a power cut leaves either the
// old list or the new one. A file that cannot be read is moved aside (never deleted) and the list
// starts empty, so the app always opens.
package todo

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"golang.org/x/text/unicode/norm"
)

const (
	fileName         = "todo.json"
	corruptFileName  = "todo.json.corrupt"
	recoveredMarker  = "recovered"
	formatVersion    = 1
	MaxTasks         = 200 // the screen's list shows at most 200 rows
	MaxTextRunes     = 200
	filePermission   = 0o644
	folderPermission = 0o755
)

// Errors a caller maps to a message; each one tells the user what to change.
var (
	ErrEmptyText = errors.New("the task text is empty; type something first")
	ErrDuplicate = errors.New("a task with this text already exists")
	ErrFull      = errors.New("the list is full; remove a task first")
	ErrNotFound  = errors.New("no task has this text")
)

// Task is one row of the list.
type Task struct {
	Text string `json:"text"`
	Done bool   `json:"done"`
}

type fileContent struct {
	Version int    `json:"version"`
	Tasks   []Task `json:"tasks"`
}

// Store reads and writes the task list in one folder (the app's data/).
type Store struct {
	Dir string
}

// List is the loaded tasks plus whether an unreadable file was set aside while loading.
type List struct {
	Tasks     []Task
	Recovered bool
}

func (store Store) path() string { return filepath.Join(store.Dir, fileName) }

// Load returns the tasks. A missing file is an empty list. A file that is not valid JSON or
// breaks the rules (see validate) is renamed to todo.json.corrupt, replacing an older copy,
// and Recovered is true; the recovered marker stays until the next successful change.
func (store Store) Load() (List, error) {
	content, err := os.ReadFile(store.path())
	if errors.Is(err, os.ErrNotExist) {
		return List{}, nil
	}
	if err != nil {
		return List{}, fmt.Errorf("cannot read %s: %w", store.path(), err)
	}
	tasks, parseErr := parseTasks(content)
	if parseErr == nil {
		return List{Tasks: tasks}, nil
	}
	if err := store.setAsideCorruptFile(); err != nil {
		return List{}, fmt.Errorf("%s is damaged (%v) and could not be moved aside: %w", store.path(), parseErr, err)
	}
	return List{Recovered: true}, nil
}

func parseTasks(content []byte) ([]Task, error) {
	var parsed fileContent
	if err := json.Unmarshal(content, &parsed); err != nil {
		return nil, err
	}
	if parsed.Version != formatVersion {
		return nil, fmt.Errorf("unknown format version %d", parsed.Version)
	}
	if len(parsed.Tasks) > MaxTasks {
		return nil, fmt.Errorf("%d tasks, at most %d allowed", len(parsed.Tasks), MaxTasks)
	}
	seen := map[string]bool{}
	for _, task := range parsed.Tasks {
		if task.Text == "" || seen[task.Text] {
			return nil, fmt.Errorf("empty or repeated task text %q", task.Text)
		}
		seen[task.Text] = true
	}
	return parsed.Tasks, nil
}

func (store Store) setAsideCorruptFile() error {
	if err := os.MkdirAll(store.Dir, folderPermission); err != nil {
		return err
	}
	if err := os.Rename(store.path(), filepath.Join(store.Dir, corruptFileName)); err != nil {
		return err
	}
	return store.writeMarker()
}

func (store Store) writeMarker() error {
	return os.WriteFile(filepath.Join(store.Dir, recoveredMarker), nil, filePermission)
}

// HasRecoveredMarker is true from the moment a damaged file was set aside until the next change.
func (store Store) HasRecoveredMarker() bool {
	_, err := os.Stat(filepath.Join(store.Dir, recoveredMarker))
	return err == nil
}

// Save writes the list atomically and clears the recovered marker.
func (store Store) Save(tasks []Task) error {
	if tasks == nil {
		tasks = []Task{}
	}
	content, err := json.MarshalIndent(fileContent{Version: formatVersion, Tasks: tasks}, "", "  ")
	if err != nil {
		return err
	}
	if err := writeFileAtomically(store.path(), append(content, '\n')); err != nil {
		return err
	}
	if err := os.Remove(filepath.Join(store.Dir, recoveredMarker)); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("cannot clear the recovered marker: %w", err)
	}
	return nil
}

func writeFileAtomically(target string, content []byte) error {
	if err := os.MkdirAll(filepath.Dir(target), folderPermission); err != nil {
		return err
	}
	temp, err := os.CreateTemp(filepath.Dir(target), ".tmp-todo-")
	if err != nil {
		return err
	}
	defer os.Remove(temp.Name()) // no-op after the rename
	if _, err := temp.Write(content); err != nil {
		temp.Close()
		return fmt.Errorf("cannot write %s: %w", temp.Name(), err)
	}
	if err := temp.Sync(); err != nil {
		temp.Close()
		return fmt.Errorf("cannot flush %s: %w", temp.Name(), err)
	}
	if err := temp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(temp.Name(), filePermission); err != nil {
		return err
	}
	return os.Rename(temp.Name(), target)
}

// CleanText is what a typed or tapped text becomes: Unicode NFC (the Kobo keyboard can send
// decomposed Vietnamese), control characters and runs of spaces collapsed, ends trimmed, cut to
// MaxTextRunes.
func CleanText(raw string) string {
	normalized := norm.NFC.String(raw)
	words := strings.Fields(strings.Map(replaceControlWithSpace, normalized))
	text := strings.Join(words, " ")
	if utf8.RuneCountInString(text) > MaxTextRunes {
		text = string([]rune(text)[:MaxTextRunes])
	}
	return strings.TrimSpace(text)
}

func replaceControlWithSpace(r rune) rune {
	if r < 0x20 || r == 0x7f {
		return ' '
	}
	return r
}

// Add appends an open task.
func Add(tasks []Task, raw string) ([]Task, error) {
	text := CleanText(raw)
	if text == "" {
		return tasks, ErrEmptyText
	}
	if indexOf(tasks, text) >= 0 {
		return tasks, ErrDuplicate
	}
	if len(tasks) >= MaxTasks {
		return tasks, ErrFull
	}
	return append(append([]Task{}, tasks...), Task{Text: text}), nil
}

// Toggle flips open and done for the task with this text.
func Toggle(tasks []Task, raw string) ([]Task, error) {
	index := indexOf(tasks, CleanText(raw))
	if index < 0 {
		return tasks, ErrNotFound
	}
	changed := append([]Task{}, tasks...)
	changed[index].Done = !changed[index].Done
	return changed, nil
}

// Remove deletes the task with this text.
func Remove(tasks []Task, raw string) ([]Task, error) {
	index := indexOf(tasks, CleanText(raw))
	if index < 0 {
		return tasks, ErrNotFound
	}
	return append(append([]Task{}, tasks[:index]...), tasks[index+1:]...), nil
}

// ClearDone drops every done task and returns how many it dropped.
func ClearDone(tasks []Task) ([]Task, int) {
	kept := make([]Task, 0, len(tasks))
	for _, task := range tasks {
		if !task.Done {
			kept = append(kept, task)
		}
	}
	return kept, len(tasks) - len(kept)
}

// Counts returns how many tasks are open and how many are done.
func Counts(tasks []Task) (open, done int) {
	for _, task := range tasks {
		if task.Done {
			done++
		} else {
			open++
		}
	}
	return open, done
}

func indexOf(tasks []Task, text string) int {
	for index, task := range tasks {
		if task.Text == text {
			return index
		}
	}
	return -1
}
