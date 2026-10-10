package todo

import (
	"encoding/json"
	"strconv"
)

const doneMark = "✓"

// screenState is the object the state action prints: the variables of ui.json.
type screenState struct {
	Open    string      `json:"open"`
	Done    string      `json:"done"`
	HasDone bool        `json:"hasdone"`
	NewItem string      `json:"newitem"`
	Notice  string      `json:"notice"`
	Items   [][2]string `json:"items"`
}

// StateJSON renders the screen variables. notice is the localized "list was damaged" text; it
// is shown only while the recovered marker exists.
func StateJSON(tasks []Task, notice string) ([]byte, error) {
	open, done := Counts(tasks)
	rows := make([][2]string, 0, len(tasks))
	for _, task := range tasks {
		mark := ""
		if task.Done {
			mark = doneMark
		}
		rows = append(rows, [2]string{task.Text, mark})
	}
	return json.Marshal(screenState{
		Open: strconv.Itoa(open), Done: strconv.Itoa(done), HasDone: done > 0, Notice: notice, Items: rows,
	})
}
