package drawer

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"koboapps/internal/i18n"
	"koboapps/internal/manifest"
	"koboapps/internal/uikit"
)

const (
	stateActionID     = "state"
	defaultListMax    = 50
	defaultCodeDigits = 4
	secretMask        = "•"
)

// ScreenReply is the response of `drawer screen`: one validated ui.json screen with every text
// localized and every binding filled in, so the mod only lays it out.
type ScreenReply struct {
	V       int                     `json:"v"`
	App     string                  `json:"app"`
	Screen  string                  `json:"screen"`
	Title   string                  `json:"title"`
	Status  string                  `json:"status"`
	Action  *HeaderAction           `json:"action,omitempty"`
	Body    []ScreenComponent       `json:"body"`
	Footer  []ScreenComponent       `json:"footer"`
	Dialogs map[string]ScreenDialog `json:"dialogs,omitempty"`
	Toasts  map[string]string       `json:"toasts,omitempty"`
	Live    bool                    `json:"live"` // the app has a state action: refresh while open
}

// HeaderAction is the optional link at the right of the header: another screen or an action.
type HeaderAction struct {
	Label  string `json:"label"`
	Push   string `json:"push,omitempty"`
	Action string `json:"action,omitempty"`
}

// ScreenDialog is a declared dialog or a button's confirm question.
type ScreenDialog struct {
	Title  string `json:"title"`
	Text   string `json:"text"`
	OK     string `json:"ok"`
	Cancel string `json:"cancel,omitempty"`
	Action string `json:"action,omitempty"`
}

// ScreenComponent is one resolved component. Type is the ui.json component key; only the fields
// of that type are set (docs/spec/drawer-protocol.md lists them per type).
type ScreenComponent struct {
	Type        string        `json:"type"`
	Text        string        `json:"text,omitempty"`
	Style       string        `json:"style,omitempty"`
	Align       string        `json:"align,omitempty"`
	Title       string        `json:"title,omitempty"`
	Detail      string        `json:"detail,omitempty"`
	Caption     string        `json:"caption,omitempty"`
	Action      string        `json:"action,omitempty"`
	Bind        string        `json:"bind,omitempty"`
	Value       any           `json:"value,omitempty"`
	Enabled     *bool         `json:"enabled,omitempty"`
	Options     [][2]string   `json:"options,omitempty"`
	Placeholder string        `json:"placeholder,omitempty"`
	Keyboard    string        `json:"keyboard,omitempty"`
	Secret      bool          `json:"secret,omitempty"`
	Digits      int           `json:"digits,omitempty"`
	Rows        [][2]string   `json:"rows,omitempty"`
	URL         string        `json:"url,omitempty"`
	Code        string        `json:"code,omitempty"`
	Hint        string        `json:"hint,omitempty"`
	Expires     string        `json:"expires,omitempty"`
	Label       string        `json:"label,omitempty"`
	Done        string        `json:"done,omitempty"`
	Total       string        `json:"total,omitempty"`
	Confirm     *ScreenDialog `json:"confirm,omitempty"`
	Async       bool          `json:"async,omitempty"`
	EmptyText   string        `json:"empty_text,omitempty"`
	Tap         string        `json:"tap,omitempty"`
	Items       *[][2]string  `json:"items,omitempty"`
	Button      string        `json:"button,omitempty"`
}

// screenContext resolves texts and bindings of one screen.
type screenContext struct {
	doc       *uikit.Document
	locale    string
	vars      map[string]any
	actionIDs map[string]bool
}

func (s screenContext) text(raw any) string {
	if raw == nil {
		return ""
	}
	return uikit.Substitute(s.doc.Text(raw, s.locale), s.vars)
}

func (s screenContext) hasAction(id string) bool { return s.actionIDs[id] }

// Screen loads, validates and resolves one screen of an app (the start screen when screenID is
// empty). Variables come from the app's optional "state" action, else from ui.json "vars".
func (c Config) Screen(appID, screenID string) (ScreenReply, error) {
	app, err := c.loadApp(appID)
	if err != nil {
		return ScreenReply{}, err
	}
	doc, err := loadScreenDocument(app)
	if err != nil {
		return ScreenReply{}, err
	}
	if screenID == "" {
		screenID = doc.Start
	}
	screen, found := doc.Screens[screenID]
	if !found {
		return ScreenReply{}, newError(CodeInvalidApp, "app", appID, "detail", "no screen '"+screenID+"' in ui.json")
	}
	_, isLive := findAction(app, stateActionID)
	context := screenContext{
		doc: doc, locale: screenLocale(app.Dir, c.Locale), vars: c.screenVars(app, doc), actionIDs: actionIDSet(app),
	}
	return context.reply(appID, screen, isLive), nil
}

func loadScreenDocument(app manifest.App) (*uikit.Document, error) {
	if !uikit.HasUI(app.Dir) {
		return nil, newError(CodeInvalidApp, "app", app.ID, "detail", "no "+uikit.FileName)
	}
	doc, report, err := uikit.LoadDir(app.Dir, actionIDList(app))
	if err != nil {
		return nil, newError(CodeIO, "detail", err.Error())
	}
	if !report.OK() || doc == nil {
		return nil, newError(CodeInvalidApp, "app", app.ID, "detail", report.Errors[0].String())
	}
	return doc, nil
}

func actionIDList(app manifest.App) []string {
	ids := make([]string, 0, len(app.Actions))
	for _, action := range app.Actions {
		ids = append(ids, action.ID)
	}
	return ids
}

func actionIDSet(app manifest.App) map[string]bool {
	set := map[string]bool{}
	for _, id := range actionIDList(app) {
		set[id] = true
	}
	return set
}

func findAction(app manifest.App, id string) (manifest.Action, bool) {
	for _, action := range app.Actions {
		if action.ID == id {
			return action, true
		}
	}
	return manifest.Action{}, false
}

// screenLocale picks the most specific locale of the device language that the app translates
// (vi_VN -> vi); texts without a translation fall back to English inside uikit.
func screenLocale(appDir, locale string) string {
	for _, candidate := range i18n.Chain(locale) {
		if _, err := os.Stat(filepath.Join(appDir, "locales", candidate+".json")); err == nil {
			return candidate
		}
	}
	language, _, _ := strings.Cut(locale, "_")
	return language
}

// screenVars starts from the declared defaults and overlays the state action's values for
// declared variables only (a binding can only name a declared variable).
func (c Config) screenVars(app manifest.App, doc *uikit.Document) map[string]any {
	vars := make(map[string]any, len(doc.Vars))
	for name, value := range doc.Vars {
		vars[name] = value
	}
	for name, value := range c.readState(app) {
		if _, declared := vars[name]; declared {
			vars[name] = value
		}
	}
	return vars
}

// readState runs the optional "state" action with the status budget. A missing, slow, failing
// or malformed state leaves the defaults.
func (c Config) readState(app manifest.App) map[string]any {
	action, found := findAction(app, stateActionID)
	if !found {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), c.statusTimeout())
	defer cancel()
	output := c.runScript()(ctx, filepath.Join(app.Dir, filepath.FromSlash(action.Run)))
	if output.Err != nil || output.TimedOut || output.ExitCode != 0 {
		return nil
	}
	return parseState(output.Stdout)
}

// parseState reads one JSON object from the state script's stdout.
func parseState(stdout string) map[string]any {
	var state map[string]any
	if json.Unmarshal([]byte(strings.TrimSpace(stdout)), &state) != nil {
		return nil
	}
	return state
}

func (s screenContext) reply(appID string, screen *uikit.Screen, isLive bool) ScreenReply {
	reply := ScreenReply{
		V: ProtocolVersion, App: appID, Screen: screen.ID, Title: s.text(screen.Title), Status: s.text(screen.Status),
		Body: s.components(screen.Body), Footer: s.components(screen.Footer), Live: isLive,
	}
	if screen.Action != nil {
		reply.Action = &HeaderAction{Label: s.text(screen.Action["label"]), Push: stringProp(screen.Action, "push"),
			Action: stringProp(screen.Action, "action")}
	}
	reply.Dialogs = s.dialogs(screen.Dialogs)
	reply.Toasts = s.toasts(screen.Toasts)
	return reply
}

func (s screenContext) dialogs(declared map[string]map[string]any) map[string]ScreenDialog {
	if len(declared) == 0 {
		return nil
	}
	dialogs := make(map[string]ScreenDialog, len(declared))
	for id, fields := range declared {
		dialogs[id] = s.dialog(fields)
	}
	return dialogs
}

func (s screenContext) dialog(fields map[string]any) ScreenDialog {
	return ScreenDialog{Title: s.text(fields["title"]), Text: s.text(fields["text"]), OK: s.text(fields["ok"]),
		Cancel: s.text(fields["cancel"]), Action: stringProp(fields, "action")}
}

func (s screenContext) toasts(declared map[string]any) map[string]string {
	if len(declared) == 0 {
		return nil
	}
	toasts := make(map[string]string, len(declared))
	for id, raw := range declared {
		toasts[id] = s.text(raw)
	}
	return toasts
}

func stringProp(object map[string]any, key string) string {
	value, _ := object[key].(string)
	return value
}

func (s screenContext) components(components []uikit.Component) []ScreenComponent {
	resolved := make([]ScreenComponent, 0, len(components))
	for _, component := range components {
		resolved = append(resolved, s.component(component))
	}
	return resolved
}

func (s screenContext) component(c uikit.Component) ScreenComponent {
	resolvers := map[string]func(uikit.Component) ScreenComponent{
		"text": s.textComponent, "section": s.mainTextComponent, "notice": s.noticeComponent,
		"kv": s.kvComponent, "row": s.rowComponent, "toggle": s.toggleComponent, "radio": s.radioComponent,
		"field": s.fieldComponent, "bigcode": s.bigCodeComponent, "qr": s.qrComponent, "pair": s.pairComponent,
		"steps": s.stepsComponent, "button": s.buttonComponent, "list": s.listComponent, "empty": s.emptyComponent,
	}
	resolved := resolvers[c.Kind](c)
	resolved.Type = c.Kind
	return resolved
}

func (s screenContext) mainTextComponent(c uikit.Component) ScreenComponent {
	return ScreenComponent{Text: s.text(c.Main())}
}

func (s screenContext) textComponent(c uikit.Component) ScreenComponent {
	return ScreenComponent{Text: s.text(c.Main()), Style: stringProp(c.Props, "style"), Align: stringProp(c.Props, "align")}
}

func (s screenContext) noticeComponent(c uikit.Component) ScreenComponent {
	return ScreenComponent{Text: s.text(c.Main()), Title: s.text(c.Prop("title"))}
}

func (s screenContext) kvComponent(c uikit.Component) ScreenComponent {
	raw, _ := c.Main().([]any)
	rows := make([][2]string, 0, len(raw))
	for _, entry := range raw {
		if pair, isPair := entry.([]any); isPair && len(pair) == 2 {
			rows = append(rows, [2]string{s.text(pair[0]), s.text(pair[1])})
		}
	}
	return ScreenComponent{Rows: rows}
}

func (s screenContext) rowComponent(c uikit.Component) ScreenComponent {
	return ScreenComponent{Text: s.text(c.Main()), Detail: s.text(c.Prop("detail")), Action: stringProp(c.Props, "action")}
}

// boundAction is the action an input control runs: the app action named like its variable.
// Without one the control is shown read-only (enabled false), because nothing could store it.
func (s screenContext) boundAction(bind string) string {
	if s.hasAction(bind) {
		return bind
	}
	return ""
}

func (s screenContext) inputEnabled(c uikit.Component, bind string) *bool {
	enabled := uikit.Truthy(c.Prop("enabled"), s.vars) && s.hasAction(bind)
	return &enabled
}

func (s screenContext) toggleComponent(c uikit.Component) ScreenComponent {
	bind := stringProp(c.Props, "bind")
	return ScreenComponent{Text: s.text(c.Main()), Bind: bind, Value: uikit.Truthy("$"+bind, s.vars),
		Enabled: s.inputEnabled(c, bind), Action: s.boundAction(bind)}
}

func (s screenContext) radioComponent(c uikit.Component) ScreenComponent {
	bind := stringProp(c.Props, "bind")
	raw, _ := c.Prop("options").([]any)
	options := make([][2]string, 0, len(raw))
	for _, entry := range raw {
		if pair, isPair := entry.([]any); isPair && len(pair) == 2 {
			value, _ := pair[0].(string)
			options = append(options, [2]string{value, s.text(pair[1])})
		}
	}
	return ScreenComponent{Text: s.text(c.Main()), Bind: bind, Value: s.varText(bind), Options: options,
		Enabled: s.inputEnabled(c, bind), Action: s.boundAction(bind)}
}

func (s screenContext) varText(path string) string {
	value, _ := uikit.Lookup(s.vars, path)
	return uikit.FormatValue(value)
}

// fieldComponent never sends a secret value to the mod: it gets the same number of dots.
func (s screenContext) fieldComponent(c uikit.Component) ScreenComponent {
	bind := stringProp(c.Props, "bind")
	isSecret := c.Prop("secret") == true
	value := s.varText(bind)
	if isSecret {
		value = strings.Repeat(secretMask, len([]rune(value)))
	}
	keyboard := stringProp(c.Props, "keyboard")
	if keyboard == "" {
		keyboard = "text"
	}
	return ScreenComponent{Text: s.text(c.Main()), Bind: bind, Value: value, Placeholder: s.text(c.Prop("placeholder")),
		Keyboard: keyboard, Secret: isSecret, Enabled: s.inputEnabled(c, bind), Action: s.boundAction(bind)}
}

func (s screenContext) bigCodeComponent(c uikit.Component) ScreenComponent {
	digits := defaultCodeDigits
	if declared, isNumber := c.Prop("digits").(float64); isNumber {
		digits = int(declared)
	}
	return ScreenComponent{Text: s.text(c.Main()), Digits: digits}
}

func (s screenContext) qrComponent(c uikit.Component) ScreenComponent {
	return ScreenComponent{Text: s.text(c.Main()), Caption: s.text(c.Prop("caption"))}
}

func (s screenContext) pairComponent(c uikit.Component) ScreenComponent {
	fields, _ := c.Main().(map[string]any)
	return ScreenComponent{URL: s.text(fields["url"]), Code: s.text(fields["code"]), Title: s.text(fields["title"]),
		Hint: s.text(fields["hint"]), Expires: s.text(fields["expires"]), Digits: defaultCodeDigits}
}

func (s screenContext) stepsComponent(c uikit.Component) ScreenComponent {
	fields, _ := c.Main().(map[string]any)
	return ScreenComponent{Label: s.text(fields["label"]), Done: s.text(fields["done"]), Total: s.text(fields["total"]),
		Detail: s.text(fields["detail"])}
}

func (s screenContext) buttonComponent(c uikit.Component) ScreenComponent {
	style := stringProp(c.Props, "style")
	if style == "" {
		style = "secondary"
	}
	enabled := uikit.Truthy(c.Prop("enabled"), s.vars)
	resolved := ScreenComponent{Text: s.text(c.Main()), Action: stringProp(c.Props, "action"), Style: style,
		Enabled: &enabled, Async: c.Prop("async") == true}
	if confirm, hasConfirm := c.Prop("confirm").(map[string]any); hasConfirm {
		question := s.dialog(confirm)
		resolved.Confirm = &question
	}
	return resolved
}

func (s screenContext) listComponent(c uikit.Component) ScreenComponent {
	reference, _ := c.Main().(string)
	bind := strings.TrimPrefix(reference, "$")
	values, _ := uikit.Lookup(s.vars, bind)
	raw, _ := values.([]any)
	limit := defaultListMax
	if declared, isNumber := c.Prop("max").(float64); isNumber {
		limit = int(declared)
	}
	if len(raw) > limit {
		raw = raw[len(raw)-limit:] // the list keeps the last max items
	}
	items := make([][2]string, 0, len(raw))
	for _, item := range raw {
		items = append(items, listItemTexts(item))
	}
	return ScreenComponent{Bind: bind, Title: s.text(c.Prop("title")), EmptyText: s.text(c.Prop("empty_text")),
		Tap: stringProp(c.Props, "tap"), Items: &items}
}

// listItemTexts reads a list item: a plain value or [primary, secondary].
func listItemTexts(item any) [2]string {
	pair, isPair := item.([]any)
	if !isPair {
		return [2]string{uikit.FormatValue(item), ""}
	}
	var texts [2]string
	for i := 0; i < len(pair) && i < 2; i++ {
		texts[i] = uikit.FormatValue(pair[i])
	}
	return texts
}

func (s screenContext) emptyComponent(c uikit.Component) ScreenComponent {
	return ScreenComponent{Text: s.text(c.Main()), Caption: s.text(c.Prop("text")), Button: s.text(c.Prop("button")),
		Action: stringProp(c.Props, "action")}
}
