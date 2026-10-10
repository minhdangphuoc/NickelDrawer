package main

import (
	"strconv"

	"koboapps/internal/drawer"
	"koboapps/internal/i18n"
	"koboapps/internal/manifest"
)

// refreshHostItem rescans the apps folder. The drawer model is rebuilt from disk on every
// call, so there is nothing to write; the reply only tells the user what was found.
func refreshHostItem(session drawerSession) (any, error) {
	scan, err := manifest.LoadAll(session.options.appsDir)
	if err != nil {
		return nil, &drawer.Error{Code: drawer.CodeIO, Params: map[string]string{"detail": err.Error()}}
	}
	return drawer.RunResult{
		V: drawer.ProtocolVersion, OK: true, Rescan: manifest.RescanNone,
		Text: refreshSummary(session.bundle, len(scan.Apps), len(scan.Failures)),
	}, nil
}

func refreshSummary(bundle *i18n.Bundle, appCount, failureCount int) string {
	summary := bundle.Message("refresh.summary", i18n.Params("apps", strconv.Itoa(appCount)))
	if failureCount > 0 {
		summary += "\n" + bundle.Message("refresh.failures", i18n.Params("count", strconv.Itoa(failureCount)))
	}
	return summary
}
