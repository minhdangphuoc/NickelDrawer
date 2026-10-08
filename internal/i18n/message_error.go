package i18n

import (
	"errors"
	"sort"
	"strings"
)

// MessageError is an error whose user-facing text lives in a locale file. Error() is the
// stable, language-free form (key and parameters) for logs; Bundle.Describe renders it.
type MessageError struct {
	Key    string
	Params map[string]string
	Cause  error
}

// NewError builds a MessageError from a key and name, value pairs.
func NewError(key string, pairs ...string) *MessageError {
	return &MessageError{Key: key, Params: Params(pairs...)}
}

// WithCause attaches the underlying error; it fills {cause} in the message.
func (err *MessageError) WithCause(cause error) *MessageError {
	err.Cause = cause
	return err
}

func (err *MessageError) Error() string {
	names := make([]string, 0, len(err.Params))
	for name := range err.Params {
		names = append(names, name)
	}
	sort.Strings(names)
	var text strings.Builder
	text.WriteString(err.Key)
	for _, name := range names {
		text.WriteString(" " + name + "=" + err.Params[name])
	}
	if err.Cause != nil {
		text.WriteString(": " + err.Cause.Error())
	}
	return text.String()
}

func (err *MessageError) Unwrap() error { return err.Cause }

// Describe renders err in the bundle's language. Errors that are not MessageErrors, and a nil
// bundle, fall back to err.Error().
func (bundle *Bundle) Describe(err error) string {
	var messageErr *MessageError
	if err == nil || bundle == nil || !errors.As(err, &messageErr) {
		return errorText(err)
	}
	params := map[string]string{}
	for name, value := range messageErr.Params {
		params[name] = value
	}
	if _, hasCause := params["cause"]; !hasCause && messageErr.Cause != nil {
		params["cause"] = bundle.Describe(messageErr.Cause)
	}
	return bundle.Message(messageErr.Key, params)
}

func errorText(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
