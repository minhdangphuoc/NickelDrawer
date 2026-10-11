package todo

import (
	"encoding/json"
	"strconv"
)

const doneMark = "✓"

// screenState is the object the state action prints: the variables of ui.json.
type screenState struct {
	Open    string `json:"open"`
	Done    string `json:"done"`
	HasDone bool   `json:"hasdone"`
	NewItem string `json:"newitem"`
	Notice  string `json:"notice"`
	// OpenItems and DoneItems are the two lists of the main screen; a done row carries the check mark.
	OpenItems []string    `json:"open_items"`
	DoneItems [][2]string `json:"done_items"`
}

// StateJSON renders the screen variables. notice is the localized "list was damaged" text; it
// is shown only while the recovered marker exists.
func StateJSON(tasks []Task, notice string) ([]byte, error) {
	open, done := Counts(tasks)
	openItems := make([]string, 0, open)
	doneItems := make([][2]string, 0, done)
	for _, task := range tasks {
		if task.Done {
			doneItems = append(doneItems, [2]string{task.Text, doneMark})
		} else {
			openItems = append(openItems, task.Text)
		}
	}
	return json.Marshal(screenState{
		Open: strconv.Itoa(open), Done: strconv.Itoa(done), HasDone: done > 0, Notice: notice,
		OpenItems: openItems, DoneItems: doneItems,
	})
}
