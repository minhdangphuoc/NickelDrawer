package uikit

import "strings"

// The App updates screen of the host lists apps from the device, which ui.json cannot express.
// Its ui.json holds one marker row; the state variable AppRowsVar holds the sections, rows and
// installed versions to put in its place. The drawer and ndpreview both expand it, so the mod
// receives plain components and the preview shows the same screen.
const (
	// AppRowsMarkerAction is the action of the marker row; the host never runs it.
	AppRowsMarkerAction = "apps-list"
	// AppRowsVar is the state variable with the items; each is an object with a "kind".
	AppRowsVar = "apps_rows"

	appRowSection   = "section"
	appRowTap       = "row"
	appRowInstalled = "kv"
	maxKVPairs      = 12
)

// ExpandAppRows replaces the marker row of components by the items in vars[AppRowsVar]:
// {"kind":"section","text"}, {"kind":"row","text","detail","action","confirm":{"title","text",
// "ok","cancel"}} and {"kind":"kv","key","value"}; consecutive kv items become one component.
func ExpandAppRows(components []Component, vars map[string]any) []Component {
	expanded := make([]Component, 0, len(components))
	for _, component := range components {
		if component.Kind == "row" && component.Props["action"] == AppRowsMarkerAction {
			expanded = append(expanded, appRowComponents(vars)...)
			continue
		}
		expanded = append(expanded, component)
	}
	return expanded
}

func appRowComponents(vars map[string]any) []Component {
	items, _ := vars[AppRowsVar].([]any)
	var components []Component
	var pairs []any
	flushPairs := func() {
		for len(pairs) > 0 {
			count := min(len(pairs), maxKVPairs)
			components = append(components, Component{Kind: "kv", Props: map[string]any{"kv": pairs[:count:count]}})
			pairs = pairs[count:]
		}
	}
	for _, raw := range items {
		item, _ := raw.(map[string]any)
		kind, _ := item["kind"].(string)
		if kind != appRowInstalled {
			flushPairs()
		}
		switch kind {
		case appRowSection:
			components = append(components, Component{Kind: "section", Props: map[string]any{"section": literal(item["text"])}})
		case appRowTap:
			components = append(components, tapRowComponent(item))
		case appRowInstalled:
			pairs = append(pairs, []any{literal(item["key"]), literal(item["value"])})
		}
	}
	flushPairs()
	return components
}

func tapRowComponent(item map[string]any) Component {
	props := map[string]any{"row": literal(item["text"]), "detail": literal(item["detail"]), "action": item["action"]}
	if confirm, hasConfirm := item["confirm"].(map[string]any); hasConfirm {
		props["confirm"] = map[string]any{"title": literal(confirm["title"]), "text": literal(confirm["text"]),
			"ok": literal(confirm["ok"]), "cancel": literal(confirm["cancel"])}
	}
	return Component{Kind: "row", Props: props}
}

// literal makes device text safe to pass through the binding substitution: "$" would otherwise
// start a variable reference.
func literal(value any) string {
	text, _ := value.(string)
	return strings.ReplaceAll(text, "$", "$$")
}
