// Package uikit defines ui.json v1 (docs/spec/ui-json.md): the component registry, the
// strict validator and the JSON Schema generated from the same registry, so the three
// cannot drift apart.
package uikit

import "regexp"

// KitVersion is the only supported value of "kit".
const KitVersion = 1

// MaxRadioOptions is the most options a radio list shows (docs/spec/ui-kit.md section 3.9).
const MaxRadioOptions = 8

// Kind says what sort of JSON value a Shape accepts.
type Kind int

const (
	// KindText is a localizable string: plain, "@locale.key" or {"en": ..., "vi": ...}.
	KindText Kind = iota
	// KindExpr is a string that may hold $bindings (a template).
	KindExpr
	// KindFlag is a boolean, or a binding "$name" / "!$name" that evaluates to one.
	KindFlag
	// KindBool is a literal boolean.
	KindBool
	// KindInt is an integer within Min..Max.
	KindInt
	// KindEnum is one of the Enum strings.
	KindEnum
	// KindAction is the id of an action declared in app.json.
	KindAction
	// KindScreenRef is the id of a screen of the same ui.json.
	KindScreenRef
	// KindVarName is the name of a declared variable (used by bind).
	KindVarName
	// KindVarRef is exactly one binding "$name" naming a declared variable.
	KindVarRef
	// KindID is a short identifier such as a menu item id or an option value.
	KindID
	// KindObject is a JSON object with the listed Fields and no others.
	KindObject
	// KindArray is a JSON array of Item values, Min..Max long.
	KindArray
	// KindTuple is a fixed-length JSON array, one Shape per position.
	KindTuple
	// KindMap is a JSON object whose values are all Item.
	KindMap
	// KindComponents is an array of components (screen body and footer).
	KindComponents
	// KindAny accepts any JSON value (the initial values of vars).
	KindAny
)

// Shape describes one JSON value. Only the fields that matter for its Kind are set.
type Shape struct {
	Kind    Kind
	Enum    []string
	Min     int
	Max     int
	MaxLen  int  // longest allowed text in characters (KindText, KindExpr, KindID)
	NoColon bool // ':' is not allowed, because NickelMenu uses it as a separator
	Fields  []Prop
	Item    *Shape
	Tuple   []Shape
	MinLen  int // shortest allowed tuple; 0 means all of Tuple (the tail entries are optional)
}

// Prop is a named property of an object Shape.
type Prop struct {
	Name     string
	Shape    Shape
	Required bool
}

// ComponentSpec is a registry entry. The component's own JSON key (Key) carries Main; Props are
// the sibling keys it accepts.
type ComponentSpec struct {
	Key   string
	Main  Shape
	Props []Prop
}

var (
	idPattern         = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,31}$`)
	screenIDPattern   = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,31}$`)
	actionIDPattern   = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,30}$`)
	identPattern      = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
	localeKeyPattern  = regexp.MustCompile(`^ui\.[a-z0-9][a-z0-9_.]*$`)
	localeCodePattern = regexp.MustCompile(`^[a-z]{2,3}([_-][A-Za-z0-9]{2,8})?$`)
)

// Patterns written into the schema; they must describe exactly what the Go code accepts.
const (
	identRegexp       = `[A-Za-z_][A-Za-z0-9_]*`
	varRefRegexp      = `^\$` + identRegexp + `(\.` + identRegexp + `)*$`
	flagRefRegexp     = `^!?\$` + identRegexp + `(\.` + identRegexp + `)*$`
	idRegexp          = `^[a-z0-9][a-z0-9_-]{0,31}$`
	actionIDRegexp    = `^[a-z0-9][a-z0-9-]{0,30}$`
	screenIDRegexp    = idRegexp
	localeCodeRegexp  = `^[a-z]{2,3}([_-][A-Za-z0-9]{2,8})?$`
	identOnlyRegexp   = `^` + identRegexp + `$`
	localeKeyRegexp   = `^@ui\.[a-z0-9][a-z0-9_.]*$`
	commentKeyRegexp  = `^_`
	maxScreenTitleLen = 40
)

func text(maxLen int) Shape { return Shape{Kind: KindText, MaxLen: maxLen} }

func menuText(maxLen int) Shape { return Shape{Kind: KindText, MaxLen: maxLen, NoColon: true} }

func expr(maxLen int) Shape { return Shape{Kind: KindExpr, MaxLen: maxLen} }

func enum(values ...string) Shape { return Shape{Kind: KindEnum, Enum: values} }

var (
	flagShape   = Shape{Kind: KindFlag}
	boolean     = Shape{Kind: KindBool}
	actionID    = Shape{Kind: KindAction}
	varName     = Shape{Kind: KindVarName}
	varRef      = Shape{Kind: KindVarRef}
	optionalKey = func(name string, shape Shape) Prop { return Prop{Name: name, Shape: shape} }
	requiredKey = func(name string, shape Shape) Prop { return Prop{Name: name, Shape: shape, Required: true} }
)

var confirmShape = Shape{Kind: KindObject, Fields: []Prop{
	requiredKey("text", text(160)),
	requiredKey("ok", menuText(24)),
	optionalKey("cancel", text(24)),
	optionalKey("title", text(40)),
}}

var menuItemShape = Shape{Kind: KindObject, Fields: []Prop{
	requiredKey("id", actionID), // the action the mod runs for this entry
	requiredKey("label", menuText(32)),
	optionalKey("confirm", boolean),
}}

// optionShape is [value, label] or [value, label, sub-line].
var optionShape = Shape{Kind: KindTuple, MinLen: 2, Tuple: []Shape{{Kind: KindID, MaxLen: 32}, text(40), text(60)}}

var kvRowShape = Shape{Kind: KindTuple, Tuple: []Shape{text(24), expr(60)}}

var pairShape = Shape{Kind: KindObject, Fields: []Prop{
	requiredKey("url", expr(120)),
	requiredKey("code", expr(16)),
	optionalKey("title", text(40)),
	optionalKey("hint", text(80)),
	optionalKey("expires", expr(40)),
}}

var stepsShape = Shape{Kind: KindObject, Fields: []Prop{
	requiredKey("label", text(40)),
	requiredKey("done", expr(16)),
	requiredKey("total", expr(16)),
	optionalKey("detail", expr(120)),
}}

var headerActionShape = Shape{Kind: KindObject, Fields: []Prop{
	requiredKey("label", menuText(24)),
	optionalKey("push", Shape{Kind: KindScreenRef}),
	optionalKey("action", actionID),
}}

// headerIconActionShape is one icon button of the header (ui-kit.md "Header icon actions").
var headerIconActionShape = Shape{Kind: KindObject, Fields: []Prop{
	requiredKey("icon", enum(HeaderIcons...)),
	requiredKey("label", menuText(24)),
	optionalKey("push", Shape{Kind: KindScreenRef}),
	optionalKey("action", actionID),
}}

// galleryTapShape is the on_tap target of a gallery: exactly one of push or action.
var galleryTapShape = Shape{Kind: KindObject, Fields: []Prop{
	optionalKey("push", Shape{Kind: KindScreenRef}),
	optionalKey("action", actionID),
}}

// galleryMenuItemShape is one long-press menu entry: exactly one of action, push or info.
var galleryMenuItemShape = Shape{Kind: KindObject, Fields: []Prop{
	requiredKey("label", menuText(32)),
	optionalKey("action", actionID),
	optionalKey("push", Shape{Kind: KindScreenRef}),
	optionalKey("info", boolean),
	optionalKey("confirm", confirmShape),
}}

var galleryEmptyShape = Shape{Kind: KindObject, Fields: []Prop{
	requiredKey("title", text(40)),
	optionalKey("text", text(120)),
	optionalKey("icon", enum("picture")),
	optionalKey("button", menuText(32)),
	optionalKey("action", actionID),
}}

var viewerPrimaryShape = Shape{Kind: KindObject, Fields: []Prop{
	requiredKey("label", menuText(32)),
	optionalKey("label_current", menuText(32)),
	requiredKey("action", actionID),
}}

var viewerSecondaryShape = Shape{Kind: KindObject, Fields: []Prop{
	requiredKey("label", menuText(32)),
	requiredKey("action", actionID),
	optionalKey("confirm", confirmShape),
}}

var dialogShape = Shape{Kind: KindObject, Fields: []Prop{
	requiredKey("title", text(40)),
	requiredKey("text", text(200)),
	requiredKey("ok", menuText(24)),
	optionalKey("cancel", text(24)),
	optionalKey("action", actionID),
}}

// HeaderIcons are the icons a header action can show.
var HeaderIcons = []string{"import", "settings", "refresh", "add"}

// Gallery limits (docs/spec/wallpaper-gallery.md section 3.2).
const (
	MaxGalleryMenuItems = 6
	MaxGridSide         = 6
)

// Components is the closed set of ui.json v1 components, in documentation order.
var Components = []ComponentSpec{
	{Key: "text", Main: text(400), Props: []Prop{
		optionalKey("style", enum("body", "caption", "heading")),
		optionalKey("align", enum("start", "center")),
	}},
	{Key: "section", Main: text(40)},
	{Key: "kv", Main: Shape{Kind: KindArray, Item: &kvRowShape, Min: 1, Max: 12}},
	{Key: "row", Main: text(60), Props: []Prop{
		optionalKey("detail", expr(60)),
		optionalKey("action", actionID),
		optionalKey("push", Shape{Kind: KindScreenRef}),
	}},
	{Key: "toggle", Main: text(60), Props: []Prop{
		requiredKey("bind", varName),
		optionalKey("enabled", flagShape),
		optionalKey("detail", text(60)),
	}},
	{Key: "radio", Main: text(60), Props: []Prop{
		requiredKey("bind", varName),
		requiredKey("options", Shape{Kind: KindArray, Item: &optionShape, Min: 1, Max: MaxRadioOptions}),
		optionalKey("enabled", flagShape),
	}},
	{Key: "field", Main: text(40), Props: []Prop{
		requiredKey("bind", varName),
		optionalKey("keyboard", enum("text", "number", "url")),
		optionalKey("placeholder", text(40)),
		optionalKey("secret", boolean),
	}},
	{Key: "bigcode", Main: expr(16), Props: []Prop{
		optionalKey("digits", Shape{Kind: KindInt, Min: 3, Max: 12}),
	}},
	{Key: "qr", Main: expr(200), Props: []Prop{
		optionalKey("caption", text(60)),
	}},
	{Key: "pair", Main: pairShape},
	{Key: "steps", Main: stepsShape},
	{Key: "notice", Main: text(200), Props: []Prop{
		optionalKey("title", text(40)),
	}},
	{Key: "button", Main: menuText(32), Props: []Prop{
		requiredKey("action", actionID),
		optionalKey("style", enum("primary", "secondary")),
		optionalKey("enabled", flagShape),
		optionalKey("confirm", confirmShape),
		optionalKey("async", boolean),
	}},
	{Key: "list", Main: varRef, Props: []Prop{
		optionalKey("title", text(40)),
		optionalKey("empty_text", text(60)),
		optionalKey("max", Shape{Kind: KindInt, Min: 1, Max: 200}),
		optionalKey("tap", actionID),
		optionalKey("menu", Shape{Kind: KindArray, Item: &menuItemShape, Min: 1, Max: 6}),
	}},
	{Key: "gallery", Main: varRef, Props: []Prop{
		optionalKey("columns", Shape{Kind: KindInt, Min: 1, Max: MaxGridSide}),
		optionalKey("rows", Shape{Kind: KindInt, Min: 1, Max: MaxGridSide}),
		optionalKey("on_tap", galleryTapShape),
		optionalKey("on_long_press", Shape{Kind: KindArray, Item: &galleryMenuItemShape, Min: 1, Max: MaxGalleryMenuItems}),
		optionalKey("pending", expr(8)),
		optionalKey("empty", galleryEmptyShape),
	}},
	{Key: "viewer", Main: varRef, Props: []Prop{
		requiredKey("selected", varName),
		optionalKey("current_status", text(40)),
		requiredKey("primary", viewerPrimaryShape),
		optionalKey("secondary", viewerSecondaryShape),
	}},
	{Key: "empty", Main: text(40), Props: []Prop{
		optionalKey("text", text(120)),
		optionalKey("button", menuText(32)),
		optionalKey("action", actionID),
	}},
}

// screenProps are the keys of a screen object. "badge" is the plan v2 name of "status".
var screenProps = []Prop{
	requiredKey("title", text(maxScreenTitleLen)),
	optionalKey("status", text(40)),
	optionalKey("badge", text(40)),
	optionalKey("action", headerActionShape),
	optionalKey("actions", Shape{Kind: KindArray, Item: &headerIconActionShape, Min: 1, Max: 2}),
	optionalKey("body", Shape{Kind: KindComponents}),
	optionalKey("footer", Shape{Kind: KindComponents}),
	optionalKey("dialogs", Shape{Kind: KindMap, Item: &dialogShape}),
	optionalKey("toasts", Shape{Kind: KindMap, Item: &toastShape}),
}

var toastShape = text(60)

// ScreenShape is the shape of one screen object.
var ScreenShape = Shape{Kind: KindObject, Fields: screenProps}

// componentByKey returns the registry entry for a component key.
func componentByKey(key string) (ComponentSpec, bool) {
	for _, component := range Components {
		if component.Key == key {
			return component, true
		}
	}
	return ComponentSpec{}, false
}

func componentKeys() []string {
	keys := make([]string, len(Components))
	for i, component := range Components {
		keys[i] = component.Key
	}
	return keys
}
