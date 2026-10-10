// Command todo is the To-do app's helper: it keeps the task list in data/todo.json.
//
//	todo state [--notice TEXT]    print the screen variables as one JSON object
//	todo add|toggle|remove --text TEXT
//	todo clear-done
//
// Every change prints one outcome word (and a count for clear-done) that the action script
// turns into a declared toast. Exit status 1 means an I/O problem, with the reason on stderr.
package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"koboapps/internal/todo"
)

const (
	exitOK      = 0
	exitFailure = 1
	exitUsage   = 2

	defaultDataDir = "/mnt/onboard/.adds/apps/todo/data"
	usage          = "usage: todo state [--notice TEXT] | add|toggle|remove --text TEXT | clear-done  (flag --data-dir DIR)"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, usage)
		return exitUsage
	}
	flags := flag.NewFlagSet(args[0], flag.ContinueOnError)
	flags.SetOutput(stderr)
	dataDir := flags.String("data-dir", defaultDataDir, "folder holding todo.json")
	text := flags.String("text", "", "the task text (add, toggle, remove)")
	notice := flags.String("notice", "", "state: localized text shown while a damaged list was set aside")
	if err := flags.Parse(args[1:]); err != nil {
		return exitUsage
	}
	store := todo.Store{Dir: *dataDir}
	var err error
	switch args[0] {
	case "state":
		err = printState(store, *notice, stdout)
	case "add":
		err = change(store, stdout, func(tasks []todo.Task) ([]todo.Task, string, error) {
			changed, err := todo.Add(tasks, *text)
			return changed, "added", err
		})
	case "toggle":
		err = change(store, stdout, func(tasks []todo.Task) ([]todo.Task, string, error) {
			changed, err := todo.Toggle(tasks, *text)
			return changed, "toggled", err
		})
	case "remove":
		err = change(store, stdout, func(tasks []todo.Task) ([]todo.Task, string, error) {
			changed, err := todo.Remove(tasks, *text)
			return changed, "removed", err
		})
	case "clear-done":
		err = change(store, stdout, clearDone)
	default:
		fmt.Fprintln(stderr, usage)
		return exitUsage
	}
	if err != nil {
		fmt.Fprintln(stderr, "todo:", err)
		return exitFailure
	}
	return exitOK
}

func printState(store todo.Store, noticeText string, stdout io.Writer) error {
	list, err := store.Load()
	if err != nil {
		return err
	}
	if !store.HasRecoveredMarker() {
		noticeText = ""
	}
	encoded, err := todo.StateJSON(list.Tasks, noticeText)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(stdout, "%s\n", encoded)
	return err
}

func clearDone(tasks []todo.Task) ([]todo.Task, string, error) {
	kept, removedCount := todo.ClearDone(tasks)
	if removedCount == 0 {
		return tasks, "nothing_done", nil
	}
	return kept, fmt.Sprintf("cleared %d", removedCount), nil
}

// userOutcomes maps a rule violation to the outcome word the script reports.
var userOutcomes = map[error]string{
	todo.ErrEmptyText: "empty",
	todo.ErrDuplicate: "duplicate",
	todo.ErrFull:      "full",
	todo.ErrNotFound:  "missing",
}

// change loads the list, applies one edit and saves it; a rule violation is an outcome, not a
// failure. A list that had to be set aside reports "recovered" whatever the edit was.
func change(store todo.Store, stdout io.Writer, edit func([]todo.Task) ([]todo.Task, string, error)) error {
	list, err := store.Load()
	if err != nil {
		return err
	}
	changed, outcome, editErr := edit(list.Tasks)
	if editErr != nil {
		word, isRuleViolation := userOutcomes[editErr]
		if !isRuleViolation {
			return editErr
		}
		return report(stdout, list.Recovered, word)
	}
	if outcome == "nothing_done" {
		return report(stdout, list.Recovered, outcome)
	}
	if err := store.Save(changed); err != nil {
		return err
	}
	return report(stdout, list.Recovered, outcome)
}

func report(stdout io.Writer, recovered bool, outcome string) error {
	if recovered {
		outcome = "recovered"
	}
	_, err := fmt.Fprintln(stdout, outcome)
	return err
}
