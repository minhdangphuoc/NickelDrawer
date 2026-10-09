package main

import (
	"fmt"
	"io"
	"strings"

	"koboapps/internal/manifest"
	"koboapps/internal/uikit"
)

// reportUI validates the app's ui.json (and screens/*.json) when it has one. It returns
// whether a ui.json exists and whether it is usable.
func reportUI(out io.Writer, app manifest.App) (hasUI, isValid bool) {
	if !uikit.HasUI(app.Dir) {
		return false, true
	}
	actionIDs := make([]string, len(app.Actions))
	for i, action := range app.Actions {
		actionIDs[i] = action.ID
	}
	_, report, err := uikit.LoadDir(app.Dir, actionIDs)
	if err != nil {
		fmt.Fprintf(out, "INVALID %s: %v\n", app.ID, err)
		return true, false
	}
	for _, warning := range report.Warnings {
		fmt.Fprintf(out, "warning %s: %s: %s\n", app.ID, uikit.FileName, warning)
	}
	if len(report.Errors) > 0 {
		fmt.Fprintf(out, "INVALID %s: %s: %s\n", app.ID, uikit.FileName, joinIssues(report.Errors))
	}
	return true, report.OK()
}

func joinIssues(issues []uikit.Issue) string {
	lines := make([]string, len(issues))
	for i, issue := range issues {
		lines[i] = issue.String()
	}
	return strings.Join(lines, "; ")
}
