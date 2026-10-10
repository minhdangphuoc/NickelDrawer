package i18n

import (
	"regexp"
	"time"
)

// Date and time formats are locale data too: a locale file may set "format.date", "format.datetime"
// and "format.time" with the tokens yyyy, MM, dd, HH and mm (everything else is copied as written),
// so 31/12/2026 14:05 and 12/31/2026 02:05 PM need no code change. Without the keys the
// day-first patterns below are used (design-system.md, section 5).
const (
	DateKey                = "format.date"
	DateTimeKey            = "format.datetime"
	TimeKey                = "format.time"
	DefaultDatePattern     = "dd/MM/yyyy"
	DefaultDateTimePattern = "dd/MM/yyyy HH:mm"
	DefaultTimePattern     = "HH:mm"
)

var patternToken = regexp.MustCompile(`yyyy|MM|dd|HH|mm`)

var goLayoutOfToken = map[string]string{"yyyy": "2006", "MM": "01", "dd": "02", "HH": "15", "mm": "04"}

// FormatPattern renders moment with a pattern of yyyy, MM, dd, HH and mm tokens.
func FormatPattern(pattern string, moment time.Time) string {
	result := ""
	rest := pattern
	for rest != "" {
		location := patternToken.FindStringIndex(rest)
		if location == nil {
			result += rest
			break
		}
		result += rest[:location[0]] + moment.Format(goLayoutOfToken[rest[location[0]:location[1]]])
		rest = rest[location[1]:]
	}
	return result
}

// Date is moment as the locale writes a calendar date, in local time.
func (bundle *Bundle) Date(moment time.Time) string {
	return FormatPattern(bundle.patternOr(DateKey, DefaultDatePattern), moment.Local())
}

// DateTime is moment as the locale writes a date with its time, in local time.
func (bundle *Bundle) DateTime(moment time.Time) string {
	return FormatPattern(bundle.patternOr(DateTimeKey, DefaultDateTimePattern), moment.Local())
}

// Clock is moment as the locale writes a time of day, in local time.
func (bundle *Bundle) Clock(moment time.Time) string {
	return FormatPattern(bundle.patternOr(TimeKey, DefaultTimePattern), moment.Local())
}

func (bundle *Bundle) patternOr(key, fallback string) string {
	if pattern, found := bundle.Lookup(key); found && pattern != "" {
		return pattern
	}
	return fallback
}
