package uikit

import (
	"sort"
	"strings"

	"koboapps/internal/i18n"
)

// Document is a parsed and validated ui.json (plus any screens/*.json).
type Document struct {
	Kit     int
	Start   string
	Vars    map[string]any
	Screens map[string]*Screen
	catalog map[string]i18n.Catalog
}

// Screen is one screen. Texts stay raw (string or locale map) until Document.Text resolves them.
type Screen struct {
	ID      string
	Title   any
	Status  any
	Action  map[string]any
	Actions []map[string]any
	Body    []Component
	Footer  []Component
	Dialogs map[string]map[string]any
	Toasts  map[string]any
}

// Component is one component on a screen; Props holds every key of its JSON object,
// the component's own key (Kind) included.
type Component struct {
	Kind  string
	Props map[string]any
	Path  string
}

// Main returns the value under the component's own key.
func (c Component) Main() any { return c.Props[c.Kind] }

// Prop returns another key of the component, nil when absent.
func (c Component) Prop(name string) any { return c.Props[name] }

// IsPrimary reports whether the component is the screen's primary action.
func (c Component) IsPrimary() bool {
	switch c.Kind {
	case "button":
		return c.Props["style"] == "primary"
	case "empty":
		return c.Props["button"] != nil
	case "gallery":
		empty, _ := c.Props["empty"].(map[string]any)
		return empty["button"] != nil
	case "viewer":
		return true
	}
	return false
}

// ScreenIDs lists screen ids with the start screen first, the rest sorted.
func (d *Document) ScreenIDs() []string {
	ids := make([]string, 0, len(d.Screens))
	for id := range d.Screens {
		if id != d.Start {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	return append([]string{d.Start}, ids...)
}

// Text resolves a text value (plain, "@key" or locale map) for locale, falling back to English.
// Bindings are left in place; call Substitute on the result.
func (d *Document) Text(value any, locale string) string {
	switch typed := value.(type) {
	case string:
		if key, isKey := strings.CutPrefix(typed, "@"); isKey {
			return d.catalogText(key, locale)
		}
		return typed
	case map[string]any:
		for _, wanted := range []string{locale, i18n.SourceLocale} {
			if text, isString := typed[wanted].(string); isString {
				return text
			}
		}
	}
	return ""
}

func (d *Document) catalogText(key, locale string) string {
	for _, wanted := range []string{locale, i18n.SourceLocale} {
		if text, found := d.catalog[wanted][key]; found {
			return text
		}
	}
	return "@" + key
}

// WithCatalogs returns d resolving "@key" texts from the given catalogs (locale -> catalog).
func (d *Document) WithCatalogs(catalogs map[string]i18n.Catalog) *Document {
	copied := *d
	copied.catalog = catalogs
	return &copied
}
