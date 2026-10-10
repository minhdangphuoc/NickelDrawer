package uikit

import (
	"fmt"
	"strings"
)

// Rules that look at a whole screen (one primary action, the A8 promotion rule, sections) and at
// content style (docs/design-system.md section 5). The structural ones are errors, the style
// ones warnings.

// buttonRule is what the A8 check needs to know about a button.
type buttonRule struct {
	Path      string
	IsPrimary bool
	Enabled   string // the `enabled` binding, "" when the button is always enabled
}

func ruleOf(component Component) buttonRule {
	enabled, _ := component.Props["enabled"].(string)
	return buttonRule{Path: component.Path, IsPrimary: component.Props["style"] == "primary", Enabled: enabled}
}

// showFlagOf is the `show` binding of a component, "" when it is always shown.
func showFlagOf(component Component) string {
	show, _ := component.Props["show"].(string)
	return show
}

// checkPrimaries: a screen has one primary action. Several are allowed only when each one is
// shown by its own `show` flag (the screen then shows whichever applies: A8), and no two share
// the same flag.
func (v *validator) checkPrimaries(path string) {
	if len(v.primaries) <= 1 {
		return
	}
	always, seen := 0, map[string]bool{}
	for _, show := range v.primaries {
		if show == "" {
			always++
		} else if seen[show] {
			v.fail(CodeMultiplePrimary, path, "has two primary actions shown by the same flag %q; give them opposite flags such as \"$x\" and \"!$x\"", show)
			return
		}
		seen[show] = true
	}
	if always > 0 {
		v.fail(CodeMultiplePrimary, path, "has %d primary actions; a screen has one primary action. Make the others \"style\": \"secondary\", or give every primary a \"show\" flag so only one is visible", len(v.primaries))
	}
}

// checkPrimaryDisabled (A8): never a disabled primary while the opposite action is possible. A
// primary enabled by "$x" next to a button enabled by "!$x" (or the reverse) is exactly that:
// promote the possible action to primary and hide the other with `show`.
func (v *validator) checkPrimaryDisabled(path string) {
	for _, primary := range v.buttons {
		if !primary.IsPrimary || primary.Enabled == "" {
			continue
		}
		for _, other := range v.buttons {
			if other.Path != primary.Path && other.Enabled == complementFlag(primary.Enabled) {
				v.warn(CodePrimaryDisabled, primary.Path, "is a primary action that is disabled whenever %s is enabled; promote the action that is possible to primary and show one of them at a time with \"show\" (A8)", other.Path)
			}
		}
	}
}

func complementFlag(flag string) string {
	if rest, isNegated := strings.CutPrefix(flag, "!"); isNegated {
		return rest
	}
	return "!" + flag
}

// checkScreenStyle runs the screen-wide style warnings once the screen is read.
func (v *validator) checkScreenStyle(path string) {
	v.checkPrimaryDisabled(path)
	v.checkStatusWidth(path + ".status")
	v.checkStatusWidth(path + ".badge")
	v.checkSections(v.current.Body)
}

// checkStatusWidth warns when a literal header status is wider than its slot in any language.
func (v *validator) checkStatusWidth(path string) {
	for _, locale := range sortedKeys(v.texts[path]) {
		text := v.texts[path][locale]
		if strings.Contains(text, "$") {
			continue // filled in at run time; the app measures its own state strings
		}
		v.warnIfStatusTooWide(path, locale, text)
	}
}

func (v *validator) warnIfStatusTooWide(path, locale, text string) {
	if width := StatusWidth(text); width > StatusMaxWidth {
		v.warn(CodeStatusTooWide, fmt.Sprintf("%s (%s)", path, locale), "%q is about %d px wide at %d px; the header status has %d px. Shorten it", text, width, StatusTextSize, StatusMaxWidth)
	}
}

// checkCatalogs applies the locale-wide style rules: every status key fits the header, and no
// text is typed in capitals.
func (v *validator) checkCatalogs() {
	for _, locale := range sortedKeys(v.env.Catalogs) {
		catalog := v.env.Catalogs[locale]
		for _, key := range sortedKeys(catalog) {
			where := fmt.Sprintf("locales/%s.json %s", locale, key)
			if words := UppercaseWords(catalog[key]); len(words) > 0 && !isCommandLineKey(key) {
				v.warn(CodeUppercaseText, where, "%q is typed in capitals (%s); write sentence case, the renderer uppercases section labels", catalog[key], strings.Join(words, ", "))
			}
			if IsStatusKey(key) {
				v.warnIfStatusTooWide(where, locale, catalog[key])
			}
		}
	}
}

// checkTextStyle: a literal text typed in capitals, and a one-line caption with a full stop.
func (v *validator) checkTextStyle(path string, shape Shape, variants map[string]string, isReference bool) {
	for _, locale := range sortedKeys(variants) {
		text := variants[locale]
		if !isReference && len(UppercaseWords(text)) > 0 {
			v.warn(CodeUppercaseText, fmt.Sprintf("%s (%s)", path, locale), "%q is typed in capitals; write sentence case", text)
		}
		if shape.Caption {
			v.warnIfCaptionHasStop(path, locale, text)
		}
	}
}

func (v *validator) warnIfCaptionHasStop(path, locale, text string) {
	if EndsWithFullStop(text) {
		v.warn(CodeCaptionPeriod, fmt.Sprintf("%s (%s)", path, locale), "%q is a one-line caption: remove the full stop", text)
	}
}

// checkTextCaption: a `text` in caption style is a caption.
func (v *validator) checkTextCaption(path string, component Component) {
	if component.Props["style"] != "caption" {
		return
	}
	for _, locale := range sortedKeys(v.texts[path+".text"]) {
		v.warnIfCaptionHasStop(path+".text", locale, v.texts[path+".text"][locale])
	}
}

// checkSections: a section label heads a group of rows. A section over nothing, or over a single
// paragraph, is a heading on a sentence (design-system.md 4.3).
func (v *validator) checkSections(body []Component) {
	for i, component := range body {
		if component.Kind != "section" {
			continue
		}
		next := i + 1
		for next < len(body) && body[next].Kind != "section" {
			next++
		}
		members := body[i+1 : next]
		switch {
		case len(members) == 0:
			v.warn(CodeSectionEmpty, component.Path, "is a section label with nothing under it; add rows or remove the label")
		case len(members) == 1 && (members[0].Kind == "text" || members[0].Kind == "notice"):
			v.warn(CodeSectionOverParagraph, component.Path, "heads a single paragraph; a section groups rows, so drop the label (or add rows under it)")
		}
	}
}

// isCommandLineKey: msg.usage is the script's command line (`run|pair|state [--app-dir DIR]`),
// where placeholders are conventionally capitals; it is never shown in the drawer.
func isCommandLineKey(key string) bool { return key == "msg.usage" }
