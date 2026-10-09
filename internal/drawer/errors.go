package drawer

import "fmt"

// Error codes of the protocol; cmd/appsctl localizes them through the key "drawer.err.<code>".
const (
	CodeUsage         = "usage"
	CodeUnknownApp    = "unknown_app"
	CodeUnknownAction = "unknown_action"
	CodeInvalidApp    = "invalid_app"
	CodeHostItem      = "host_item"
	CodeRemoveRefused = "remove_refused"
	CodeNeedConfirm   = "need_confirm"
	CodeExecFailed    = "exec_failed"
	CodeIO            = "io"
)

// Error is a failure the mod shows as a localized error row.
type Error struct {
	Code   string
	Params map[string]string // placeholders for the message: app, action, detail
}

func (e *Error) Error() string { return fmt.Sprintf("drawer: %s %v", e.Code, e.Params) }

func newError(code string, pairs ...string) *Error {
	params := map[string]string{}
	for i := 0; i+1 < len(pairs); i += 2 {
		params[pairs[i]] = pairs[i+1]
	}
	return &Error{Code: code, Params: params}
}
