package uikit

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"

	"koboapps/internal/i18n"
)

// Issue codes. Each is a stable key for a message, so tools and tests can match on the rule
// that broke instead of on wording.
const (
	CodeNotJSON          = "ui.not_json"
	CodeWrongType        = "ui.wrong_type"
	CodeBadKit           = "ui.bad_kit"
	CodeMissingProp      = "ui.missing_prop"
	CodeUnknownProp      = "ui.unknown_prop"
	CodeUnknownComponent = "ui.unknown_component"
	CodeAmbiguous        = "ui.ambiguous_component"
	CodeBadEnum          = "ui.bad_enum"
	CodeBadBinding       = "ui.bad_binding"
	CodeUnknownVar       = "ui.unknown_var"
	CodeListNotArray     = "ui.list_not_array"
	CodeUnknownAction    = "ui.unknown_action"
	CodeUnknownScreen    = "ui.unknown_screen"
	CodeBadID            = "ui.bad_id"
	CodeMultiplePrimary  = "ui.multiple_primary"
	CodeMissingEnglish   = "ui.missing_english"
	CodeBadLocaleKey     = "ui.bad_locale_key"
	CodeMissingLocaleKey = "ui.missing_locale_key"
	CodeTextTooLong      = "ui.text_too_long"
	CodeTextHasColon     = "ui.text_has_colon"
	CodeTextEmpty        = "ui.text_empty"
	CodeOutOfRange       = "ui.out_of_range"
	CodeTooManyOptions   = "ui.too_many_options"
	CodeDuplicate        = "ui.duplicate"
	CodeNeedsOne         = "ui.needs_exactly_one"
	CodeNeedsTogether    = "ui.needs_together"
	CodeTranslationLong  = "ui.translation_longer"
	CodeSecretField      = "ui.secret_field"
	CodeLocaleMissingVi  = "ui.locale_key_untranslated"
	CodeViewerNotAlone   = "ui.viewer_not_alone"
)

const translationLimitPercent = 150

// Issue is one rule a ui.json breaks (an error) or bends (a warning).
type Issue struct {
	Code    string
	Path    string
	Message string
}

func (i Issue) String() string { return fmt.Sprintf("%s: [%s] %s", i.Path, i.Code, i.Message) }

// Report collects every problem so authors fix them in one pass.
type Report struct {
	Errors   []Issue
	Warnings []Issue
}

// OK reports whether the document may be used.
func (r Report) OK() bool { return len(r.Errors) == 0 }

// HasCode reports whether an error with the code exists.
func (r Report) HasCode(code string) bool {
	for _, issue := range r.Errors {
		if issue.Code == code {
			return true
		}
	}
	return false
}

// Env is what the validator knows about the app around the ui.json.
type Env struct {
	ActionIDs []string
	Catalogs  map[string]i18n.Catalog // locale -> locales/<locale>.json; "en" resolves "@key" texts
}

// ParseAndValidate checks raw ui.json bytes (plus extra screens from screens/*.json, keyed by
// screen id) and returns the document when it has no errors.
func ParseAndValidate(data []byte, extraScreens map[string][]byte, env Env) (*Document, Report) {
	validator := &validator{env: env, vars: map[string]any{}}
	root, ok := validator.decodeObject("ui.json", data)
	if !ok {
		return nil, validator.report
	}
	screens := validator.collectScreens(root, extraScreens)
	validator.checkRoot(root, screens)
	if !validator.report.OK() {
		return nil, validator.report
	}
	return validator.build(root), validator.report
}

type validator struct {
	env          Env
	report       Report
	vars         map[string]any
	screens      map[string]*Screen
	current      *Screen
	allScreenIDs map[string]bool
	primary      int
	galleries    int
	viewers      int
}

func (v *validator) fail(code, path, format string, args ...any) {
	v.report.Errors = append(v.report.Errors, Issue{code, path, fmt.Sprintf(format, args...)})
}

func (v *validator) warn(code, path, format string, args ...any) {
	v.report.Warnings = append(v.report.Warnings, Issue{code, path, fmt.Sprintf(format, args...)})
}

func (v *validator) decodeObject(path string, data []byte) (map[string]any, bool) {
	var decoded any
	if err := json.Unmarshal(data, &decoded); err != nil {
		v.fail(CodeNotJSON, path, "is not valid JSON (%v); fix the syntax", err)
		return nil, false
	}
	object, isObject := decoded.(map[string]any)
	if !isObject {
		v.fail(CodeWrongType, path, "must be a JSON object")
		return nil, false
	}
	return object, true
}

// collectScreens merges the screens of ui.json with screens/<id>.json files.
func (v *validator) collectScreens(root map[string]any, extra map[string][]byte) map[string]any {
	screens := map[string]any{}
	if inline, isObject := root["screens"].(map[string]any); isObject {
		for id, screen := range inline {
			screens[id] = screen
		}
	}
	for id, data := range extra {
		path := "screens/" + id + ".json"
		object, ok := v.decodeObject(path, data)
		if !ok {
			continue
		}
		if _, exists := screens[id]; exists {
			v.fail(CodeDuplicate, path, "screen %q is also defined in ui.json; keep it in one place", id)
			continue
		}
		screens[id] = object
	}
	return screens
}

func (v *validator) checkRoot(root map[string]any, screens map[string]any) {
	for _, key := range sortedKeys(root) {
		if !isKnownRootKey(key) {
			v.fail(CodeUnknownProp, key, "unknown top-level key %q; allowed: kit, start, vars, screens (keys starting with _ are comments)", key)
		}
	}
	v.checkKit(root)
	v.checkVars(root)
	v.checkScreensKey(root, screens)
	start, isString := root["start"].(string)
	if !isString {
		v.fail(CodeMissingProp, "start", "must name the first screen, for example \"start\": \"main\"")
	} else if _, found := screens[start]; !found && len(screens) > 0 {
		v.fail(CodeUnknownScreen, "start", "names screen %q which does not exist; screens: %s", start, strings.Join(sortedKeys(screens), ", "))
	}
	v.screens = map[string]*Screen{}
	v.allScreenIDs = map[string]bool{}
	for id := range screens {
		v.allScreenIDs[id] = true
	}
	for _, id := range sortedKeys(screens) {
		v.checkScreen(id, screens[id])
	}
	v.checkFlow(root, screens)
}

func (v *validator) checkScreensKey(root map[string]any, screens map[string]any) {
	raw, present := root["screens"]
	if _, isObject := raw.(map[string]any); !present || !isObject {
		v.fail(CodeMissingProp, "screens", "must be an object of screens, for example \"screens\": { \"main\": { \"title\": ... } } (use {} when every screen is a file in screens/)")
		return
	}
	if len(screens) == 0 {
		v.fail(CodeMissingProp, "screens", "needs at least one screen")
	}
}

func isKnownRootKey(key string) bool {
	switch key {
	case "kit", "start", "vars", "screens":
		return true
	}
	return strings.HasPrefix(key, "_")
}

func (v *validator) checkKit(root map[string]any) {
	kit, isNumber := root["kit"].(float64)
	if !isNumber || kit != KitVersion {
		v.fail(CodeBadKit, "kit", "must be %d (this tool validates ui.json kit %d only); add \"kit\": %d", KitVersion, KitVersion, KitVersion)
	}
}

func (v *validator) checkVars(root map[string]any) {
	raw, present := root["vars"]
	if !present {
		return
	}
	vars, isObject := raw.(map[string]any)
	if !isObject {
		v.fail(CodeWrongType, "vars", "must be an object of default values, for example {\"status\": \"\"}")
		return
	}
	for _, name := range sortedKeys(vars) {
		if !identPattern.MatchString(name) {
			v.fail(CodeBadID, "vars."+name, "variable names use letters, digits and _ and cannot start with a digit")
		}
	}
	v.vars = vars
}

func (v *validator) checkScreen(id string, raw any) {
	path := "screens." + id
	if !screenIDPattern.MatchString(id) {
		v.fail(CodeBadID, path, "screen id %q must match %s", id, idRegexp)
	}
	object, isObject := raw.(map[string]any)
	if !isObject {
		v.fail(CodeWrongType, path, "must be an object with a title")
		return
	}
	v.current = &Screen{ID: id}
	v.primary, v.galleries, v.viewers = 0, 0, 0
	v.checkObject(path, ScreenShape.Fields, object)
	v.checkHeaderActions(path, object)
	v.current.Title, v.current.Status = object["title"], firstPresent(object["status"], object["badge"])
	v.current.Action, _ = object["action"].(map[string]any)
	v.current.Actions = objectList(object["actions"])
	v.current.Dialogs = objectMap(object["dialogs"])
	v.current.Toasts, _ = object["toasts"].(map[string]any)
	if v.primary > 1 {
		v.fail(CodeMultiplePrimary, path, "has %d primary buttons; a screen has one primary action, make the others \"style\": \"secondary\"", v.primary)
	}
	v.checkViewerAlone(path)
	v.screens[id] = v.current
}

// checkViewerAlone: the viewer owns the whole screen, so nothing else may share it.
func (v *validator) checkViewerAlone(path string) {
	if v.viewers == 0 {
		return
	}
	if len(v.current.Body) != 1 || len(v.current.Footer) != 0 {
		v.fail(CodeViewerNotAlone, path, "a screen with a viewer must have the viewer as its only body component and no footer; the viewer draws its own bottom bar")
	}
}

func firstPresent(values ...any) any {
	for _, value := range values {
		if value != nil {
			return value
		}
	}
	return nil
}

func objectMap(raw any) map[string]map[string]any {
	result := map[string]map[string]any{}
	if entries, isObject := raw.(map[string]any); isObject {
		for key, entry := range entries {
			if object, ok := entry.(map[string]any); ok {
				result[key] = object
			}
		}
	}
	return result
}

func objectList(raw any) []map[string]any {
	var result []map[string]any
	items, _ := raw.([]any)
	for _, item := range items {
		if object, ok := item.(map[string]any); ok {
			result = append(result, object)
		}
	}
	return result
}

// checkHeaderActions: a screen has a text link ("action") or icon buttons ("actions"), not both,
// and each one either pushes a screen or runs an action.
func (v *validator) checkHeaderActions(path string, screen map[string]any) {
	_, hasLink := screen["action"]
	_, hasIcons := screen["actions"]
	if hasLink && hasIcons {
		v.fail(CodeNeedsOne, path, "has both \"action\" (text link) and \"actions\" (icon buttons); keep one")
	}
	if action, isObject := screen["action"].(map[string]any); isObject {
		v.checkExactlyOne(path+".action", action, "push", "action")
	}
	for i, action := range objectList(screen["actions"]) {
		v.checkExactlyOne(fmt.Sprintf("%s.actions[%d]", path, i), action, "push", "action")
	}
}

// checkExactlyOne fails unless exactly one of the keys is present (info counts only when true).
func (v *validator) checkExactlyOne(path string, object map[string]any, keys ...string) {
	present := 0
	for _, key := range keys {
		if value, has := object[key]; has && value != false {
			present++
		}
	}
	if present != 1 {
		v.fail(CodeNeedsOne, path, "needs exactly one of %s", `"`+strings.Join(keys, `", "`)+`"`)
	}
}

// checkObject applies the Props of an object shape to a JSON object.
func (v *validator) checkObject(path string, props []Prop, object map[string]any) {
	known := map[string]Prop{}
	for _, prop := range props {
		known[prop.Name] = prop
	}
	for _, key := range sortedKeys(object) {
		prop, isKnown := known[key]
		if !isKnown {
			v.fail(CodeUnknownProp, path+"."+key, "unknown property %q; allowed: %s", key, strings.Join(propNames(props), ", "))
			continue
		}
		v.checkShape(path+"."+key, prop.Shape, object[key])
	}
	for _, prop := range props {
		if _, present := object[prop.Name]; prop.Required && !present {
			v.fail(CodeMissingProp, path, "is missing required property %q", prop.Name)
		}
	}
}

func propNames(props []Prop) []string {
	names := make([]string, len(props))
	for i, prop := range props {
		names[i] = prop.Name
	}
	return names
}

func (v *validator) checkShape(path string, shape Shape, value any) {
	switch shape.Kind {
	case KindText:
		v.checkText(path, shape, value)
	case KindExpr:
		v.checkExpr(path, shape, value)
	case KindFlag:
		v.checkFlag(path, value)
	case KindBool:
		if _, isBool := value.(bool); !isBool {
			v.fail(CodeWrongType, path, "must be true or false")
		}
	case KindInt:
		v.checkInt(path, shape, value)
	case KindEnum:
		v.checkEnum(path, shape, value)
	case KindAction, KindScreenRef, KindVarName, KindVarRef, KindID:
		v.checkReference(path, shape, value)
	case KindObject:
		v.checkObjectValue(path, shape, value)
	case KindArray:
		v.checkArray(path, shape, value)
	case KindTuple:
		v.checkTuple(path, shape, value)
	case KindMap:
		v.checkMap(path, shape, value)
	case KindComponents:
		v.checkComponents(path, value)
	}
}

func (v *validator) checkObjectValue(path string, shape Shape, value any) {
	object, isObject := value.(map[string]any)
	if !isObject {
		v.fail(CodeWrongType, path, "must be an object")
		return
	}
	v.checkObject(path, shape.Fields, object)
}

func (v *validator) checkArray(path string, shape Shape, value any) {
	items, isArray := value.([]any)
	if !isArray {
		v.fail(CodeWrongType, path, "must be an array")
		return
	}
	if len(items) < shape.Min || (shape.Max > 0 && len(items) > shape.Max) {
		code := CodeOutOfRange
		if shape.Max == MaxRadioOptions {
			code = CodeTooManyOptions
		}
		v.fail(code, path, "has %d entries; it needs %d to %d (more options belong in a list screen)", len(items), shape.Min, shape.Max)
	}
	for i, item := range items {
		v.checkShape(fmt.Sprintf("%s[%d]", path, i), *shape.Item, item)
	}
}

func (v *validator) checkTuple(path string, shape Shape, value any) {
	items, isArray := value.([]any)
	shortest := shape.MinLen
	if shortest == 0 {
		shortest = len(shape.Tuple)
	}
	if !isArray || len(items) < shortest || len(items) > len(shape.Tuple) {
		v.fail(CodeWrongType, path, "must be an array of %s entries", tupleLengthText(shortest, len(shape.Tuple)))
		return
	}
	for i, item := range items {
		v.checkShape(fmt.Sprintf("%s[%d]", path, i), shape.Tuple[i], item)
	}
}

func tupleLengthText(shortest, longest int) string {
	if shortest == longest {
		return fmt.Sprintf("exactly %d", longest)
	}
	return fmt.Sprintf("%d to %d", shortest, longest)
}

func (v *validator) checkMap(path string, shape Shape, value any) {
	object, isObject := value.(map[string]any)
	if !isObject {
		v.fail(CodeWrongType, path, "must be an object")
		return
	}
	for _, key := range sortedKeys(object) {
		if !idPattern.MatchString(key) {
			v.fail(CodeBadID, path+"."+key, "id %q must match %s", key, idRegexp)
		}
		v.checkShape(path+"."+key, *shape.Item, object[key])
	}
}

func (v *validator) checkInt(path string, shape Shape, value any) {
	number, isNumber := value.(float64)
	if !isNumber || number != float64(int(number)) {
		v.fail(CodeWrongType, path, "must be a whole number")
		return
	}
	if int(number) < shape.Min || int(number) > shape.Max {
		v.fail(CodeOutOfRange, path, "is %d; it must be between %d and %d", int(number), shape.Min, shape.Max)
	}
}

func (v *validator) checkEnum(path string, shape Shape, value any) {
	text, isString := value.(string)
	if !isString || !contains(shape.Enum, text) {
		v.fail(CodeBadEnum, path, "must be one of: %s", strings.Join(shape.Enum, ", "))
	}
}

func contains(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}

func (v *validator) checkReference(path string, shape Shape, value any) {
	text, isString := value.(string)
	if !isString {
		v.fail(CodeWrongType, path, "must be a string")
		return
	}
	switch shape.Kind {
	case KindAction:
		v.checkActionID(path, text)
	case KindScreenRef:
		if _, found := v.screenIDs()[text]; !found {
			v.fail(CodeUnknownScreen, path, "names screen %q which does not exist", text)
		}
	case KindVarName:
		v.checkVarName(path, text)
	case KindVarRef:
		v.checkVarRef(path, text)
	case KindID:
		if !idPattern.MatchString(text) {
			v.fail(CodeBadID, path, "id %q must match %s", text, idRegexp)
		}
	}
}

func (v *validator) screenIDs() map[string]bool { return v.allScreenIDs }

func (v *validator) checkActionID(path, id string) {
	if !contains(v.env.ActionIDs, id) {
		v.fail(CodeUnknownAction, path, "action %q is not declared in app.json; actions: %s", id, strings.Join(v.env.ActionIDs, ", "))
	}
}

func (v *validator) checkVarName(path, name string) {
	if !identPattern.MatchString(name) {
		v.fail(CodeBadBinding, path, "%q is not a variable name (letters, digits, _)", name)
		return
	}
	v.requireVar(path, name)
}

func (v *validator) checkVarRef(path, text string) {
	reference, isPure := pureReference(text)
	if !isPure {
		v.fail(CodeBadBinding, path, "must be exactly one binding like \"$items\", got %q", text)
		return
	}
	v.requireVar(path, reference)
}

// requireVar checks the first segment of a dotted path against vars.
func (v *validator) requireVar(path, reference string) {
	root, _, _ := strings.Cut(reference, ".")
	if _, declared := v.vars[root]; !declared {
		v.fail(CodeUnknownVar, path, "refers to variable %q which is not declared in \"vars\"; add a default value for it", root)
	}
}

func (v *validator) checkFlag(path string, value any) {
	switch typed := value.(type) {
	case bool:
	case string:
		reference, _, err := ParseFlag(typed)
		if err != nil {
			v.fail(CodeBadBinding, path, "%v", err)
			return
		}
		v.requireVar(path, reference)
	default:
		v.fail(CodeWrongType, path, "must be true, false, \"$name\" or \"!$name\"")
	}
}

func (v *validator) checkExpr(path string, shape Shape, value any) {
	text, isString := value.(string)
	if !isString {
		v.fail(CodeWrongType, path, "must be a string (it may contain $bindings)")
		return
	}
	v.checkTemplateText(path, shape, "en", text)
}

var tokenLikeName = regexp.MustCompile(`(?i)token|secret|password|passwd|key`)

// ---- components ----

func (v *validator) checkComponents(path string, value any) {
	items, isArray := value.([]any)
	if !isArray {
		v.fail(CodeWrongType, path, "must be an array of components")
		return
	}
	var placed []Component
	for i, item := range items {
		itemPath := fmt.Sprintf("%s[%d]", path, i)
		if component, ok := v.checkComponent(itemPath, item); ok {
			placed = append(placed, component)
		}
	}
	if strings.HasSuffix(path, ".footer") {
		v.current.Footer = placed
	} else {
		v.current.Body = placed
	}
}

func (v *validator) checkComponent(path string, raw any) (Component, bool) {
	object, isObject := raw.(map[string]any)
	if !isObject {
		v.fail(CodeWrongType, path, "must be an object with one component key such as \"text\" or \"button\"")
		return Component{}, false
	}
	spec, ok := v.pickComponent(path, object)
	if !ok {
		return Component{}, false
	}
	v.checkShape(path+"."+spec.Key, spec.Main, object[spec.Key])
	others := map[string]any{}
	for key, value := range object {
		if key != spec.Key {
			others[key] = value
		}
	}
	v.checkObject(path, spec.Props, others)
	component := Component{Kind: spec.Key, Props: object, Path: path}
	v.checkComponentRules(path, component)
	return component, true
}

// pickComponent finds the component key of an object. Keys such as "text" are both a component
// and a property of "empty", so a key that is a property of another present component is skipped.
func (v *validator) pickComponent(path string, object map[string]any) (ComponentSpec, bool) {
	var candidates []ComponentSpec
	for _, key := range sortedKeys(object) {
		if spec, isComponent := componentByKey(key); isComponent {
			candidates = append(candidates, spec)
		}
	}
	candidates = dropProps(candidates)
	switch len(candidates) {
	case 1:
		return candidates[0], true
	case 0:
		v.fail(CodeUnknownComponent, path, "no known component in {%s}; use one of: %s", strings.Join(sortedKeys(object), ", "), strings.Join(componentKeys(), ", "))
	default:
		v.fail(CodeAmbiguous, path, "has %d component keys; one object is one component (split it)", len(candidates))
	}
	return ComponentSpec{}, false
}

func dropProps(candidates []ComponentSpec) []ComponentSpec {
	if len(candidates) < 2 {
		return candidates
	}
	var kept []ComponentSpec
	for _, candidate := range candidates {
		if !isPropOfAnother(candidate.Key, candidates) {
			kept = append(kept, candidate)
		}
	}
	return kept
}

func isPropOfAnother(key string, candidates []ComponentSpec) bool {
	for _, other := range candidates {
		if other.Key == key {
			continue
		}
		for _, prop := range other.Props {
			if prop.Name == key {
				return true
			}
		}
	}
	return false
}

func (v *validator) checkComponentRules(path string, component Component) {
	if component.IsPrimary() {
		v.primary++
	}
	switch component.Kind {
	case "list":
		v.checkListVar(path, component)
	case "gallery":
		v.checkGallery(path, component)
	case "viewer":
		v.checkViewer(path, component)
	case "empty":
		v.checkEmptyButton(path, component)
	case "field":
		v.checkSecretField(path, component)
	case "radio":
		v.checkRadioOptions(path, component)
	}
}

func (v *validator) checkGallery(path string, component Component) {
	v.checkListVar(path, component)
	v.galleries++
	if v.galleries == 2 {
		v.fail(CodeDuplicate, path, "a screen has at most one gallery")
	}
	if tap, isObject := component.Props["on_tap"].(map[string]any); isObject {
		v.checkExactlyOne(path+".on_tap", tap, "push", "action")
	}
	items, _ := component.Props["on_long_press"].([]any)
	for i, item := range items {
		if object, isObject := item.(map[string]any); isObject {
			v.checkExactlyOne(fmt.Sprintf("%s.on_long_press[%d]", path, i), object, "action", "push", "info")
		}
	}
	if empty, isObject := component.Props["empty"].(map[string]any); isObject {
		v.checkEmptyButton(path+".empty", Component{Kind: "empty", Props: empty})
	}
}

func (v *validator) checkViewer(path string, component Component) {
	v.checkListVar(path, component)
	v.viewers++
	if v.viewers == 2 {
		v.fail(CodeDuplicate, path, "a screen has at most one viewer")
	}
}

func (v *validator) checkListVar(path string, component Component) {
	text, _ := component.Main().(string)
	reference, isPure := pureReference(text)
	if !isPure {
		return
	}
	if value, declared := Lookup(v.vars, reference); declared {
		if _, isArray := value.([]any); !isArray {
			v.fail(CodeListNotArray, path+"."+component.Kind, "variable %q must default to an array (for example []) because a %s shows its items", reference, component.Kind)
		}
	}
}

func (v *validator) checkEmptyButton(path string, component Component) {
	_, hasButton := component.Props["button"]
	_, hasAction := component.Props["action"]
	if hasButton != hasAction {
		v.fail(CodeNeedsTogether, path, "\"button\" (label) and \"action\" go together in an empty state")
	}
}

func (v *validator) checkSecretField(path string, component Component) {
	bind, _ := component.Props["bind"].(string)
	if component.Props["secret"] == true && tokenLikeName.MatchString(bind) {
		v.warn(CodeSecretField, path, "secret fields are for short PINs; use phone pairing (\"pair\") for tokens and long secrets")
	}
}

func (v *validator) checkRadioOptions(path string, component Component) {
	options, _ := component.Props["options"].([]any)
	seen := map[string]bool{}
	for i, option := range options {
		pair, _ := option.([]any)
		if len(pair) == 0 {
			continue
		}
		value, _ := pair[0].(string)
		if seen[value] {
			v.fail(CodeDuplicate, fmt.Sprintf("%s.options[%d]", path, i), "option value %q appears twice", value)
		}
		seen[value] = true
	}
}

// ---- texts ----

func (v *validator) checkText(path string, shape Shape, value any) {
	variants, ok := v.textVariants(path, value)
	if !ok {
		return
	}
	for _, locale := range sortedKeys(variants) {
		v.checkTemplateText(path, shape, locale, variants[locale])
	}
	v.checkTranslationLength(path, variants)
}

// textVariants returns locale -> text for a plain string, an "@key" or a locale map.
func (v *validator) textVariants(path string, value any) (map[string]string, bool) {
	switch typed := value.(type) {
	case string:
		if strings.HasPrefix(typed, "@") {
			return v.catalogVariants(path, typed)
		}
		if typed == "" {
			v.fail(CodeTextEmpty, path, "is empty; write the text or remove the property")
			return nil, false
		}
		return map[string]string{i18n.SourceLocale: typed}, true
	case map[string]any:
		return v.mapVariants(path, typed)
	}
	v.fail(CodeWrongType, path, "must be a string, an \"@ui.key\" or a {\"en\": ..., \"vi\": ...} object")
	return nil, false
}

func (v *validator) mapVariants(path string, object map[string]any) (map[string]string, bool) {
	variants := map[string]string{}
	for _, locale := range sortedKeys(object) {
		text, isString := object[locale].(string)
		if !localeCodePattern.MatchString(locale) || !isString || text == "" || strings.HasPrefix(text, "@") {
			v.fail(CodeWrongType, path+"."+locale, "must be a locale code with a non-empty plain string (use \"@ui.key\" for the whole text instead of per-locale keys)")
			continue
		}
		variants[locale] = text
	}
	if _, hasEnglish := variants[i18n.SourceLocale]; !hasEnglish {
		v.fail(CodeMissingEnglish, path, "needs an \"en\" text; English is the source language and the fallback")
		return nil, false
	}
	return variants, true
}

func (v *validator) catalogVariants(path, reference string) (map[string]string, bool) {
	key := strings.TrimPrefix(reference, "@")
	if !localeKeyPattern.MatchString(key) {
		v.fail(CodeBadLocaleKey, path, "locale key %q must look like @ui.start_button (lowercase, digits, _ and .)", reference)
		return nil, false
	}
	english, found := v.env.Catalogs[i18n.SourceLocale][key]
	if !found {
		v.fail(CodeMissingLocaleKey, path, "locale key %q is not in locales/en.json; add it there first", key)
		return nil, false
	}
	variants := map[string]string{i18n.SourceLocale: english}
	for _, locale := range sortedKeys(v.env.Catalogs) {
		translated, has := v.env.Catalogs[locale][key]
		switch {
		case locale == i18n.SourceLocale:
		case has:
			variants[locale] = translated
		default:
			v.warn(CodeLocaleMissingVi, path, "locales/%s.json has no %q; English is shown instead", locale, key)
		}
	}
	return variants, true
}

func (v *validator) checkTemplateText(path string, shape Shape, locale, text string) {
	where := path
	if locale != i18n.SourceLocale {
		where = fmt.Sprintf("%s (%s)", path, locale)
	}
	references, err := ParseTemplate(text)
	if err != nil {
		v.fail(CodeBadBinding, where, "%v", err)
		return
	}
	for _, reference := range references {
		v.requireVar(where, reference)
	}
	if len(references) > 0 || strings.Contains(text, "$$") {
		return // length of a filled-in template is unknown
	}
	v.checkLiteral(where, shape, text)
}

func (v *validator) checkLiteral(where string, shape Shape, text string) {
	if shape.MaxLen > 0 && utf8.RuneCountInString(text) > shape.MaxLen {
		v.fail(CodeTextTooLong, where, "has %d characters; the limit is %d so it fits on the e-ink screen. Shorten it", utf8.RuneCountInString(text), shape.MaxLen)
	}
	if shape.NoColon && strings.Contains(text, ":") {
		v.fail(CodeTextHasColon, where, "must not contain ':' (menus and labels use it as a separator); reword it")
	}
}

func (v *validator) checkTranslationLength(path string, variants map[string]string) {
	english := utf8.RuneCountInString(variants[i18n.SourceLocale])
	for _, locale := range sortedKeys(variants) {
		length := utf8.RuneCountInString(variants[locale])
		if locale != i18n.SourceLocale && english >= 10 && length*100 > english*translationLimitPercent {
			v.warn(CodeTranslationLong, fmt.Sprintf("%s (%s)", path, locale), "is %d characters, over %d%% of the English %d; shorten it so the layout holds", length, translationLimitPercent, english)
		}
	}
}

// ---- building the result ----

func (v *validator) build(root map[string]any) *Document {
	start, _ := root["start"].(string)
	return &Document{Kit: KitVersion, Start: start, Vars: v.vars, Screens: v.screens, catalog: v.env.Catalogs}
}

func sortedKeys[V any](items map[string]V) []string {
	keys := make([]string, 0, len(items))
	for key := range items {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
