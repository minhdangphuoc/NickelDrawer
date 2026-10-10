package uikit

import (
	"fmt"
	"strings"
)

// A `samples` component is a picture of controls in their states (docs/uikit-plan.md 5.1): a
// grid of cells, each a sampleable component with a "look" and a caption. Nothing in it is live,
// so it has no bindings, no navigation and no events, and it never counts as the screen's
// primary action.

// checkSampleItems validates the array of cells.
func (v *validator) checkSampleItems(path string, shape Shape, value any) {
	items, isArray := value.([]any)
	if !isArray {
		v.fail(CodeWrongType, path, "must be an array of sample cells")
		return
	}
	if len(items) < shape.Min || len(items) > shape.Max {
		v.fail(CodeOutOfRange, path, "has %d cells; a samples block needs %d to %d", len(items), shape.Min, shape.Max)
	}
	for i, item := range items {
		v.checkSampleItem(fmt.Sprintf("%s[%d]", path, i), item)
	}
}

// checkSampleItem validates one cell: "look" and "caption" belong to the cell, the other keys to
// the component it holds.
func (v *validator) checkSampleItem(path string, raw any) {
	object, isObject := raw.(map[string]any)
	if !isObject {
		v.fail(CodeWrongType, path, "must be an object with one component key such as \"button\" plus \"look\" and \"caption\"")
		return
	}
	inner := map[string]any{}
	for key, value := range object {
		if key != "look" && key != "caption" {
			inner[key] = value
		}
	}
	look := v.checkSampleLook(path, object)
	if caption, hasCaption := object["caption"]; hasCaption {
		v.checkText(path+".caption", sampleCaptionShape, caption)
	}
	spec, ok := v.pickComponent(path, inner)
	if !ok {
		return
	}
	if !contains(sampleableKeys, spec.Key) {
		v.fail(CodeSampleNotStatic, path+"."+spec.Key, "a %s cannot be sampled (it needs live data or the device); samples hold: %s", spec.Key, strings.Join(sampleableKeys, ", "))
		return
	}
	if look == "selected" && !contains(onStateKeys, spec.Key) {
		v.fail(CodeBadEnum, path+".look", "\"selected\" needs a component with an on state (%s); a %s has none", strings.Join(onStateKeys, ", "), spec.Key)
	}
	v.checkSampleComponent(path, spec, inner)
}

func (v *validator) checkSampleLook(path string, object map[string]any) string {
	raw, hasLook := object["look"]
	if !hasLook {
		return "normal"
	}
	look, isString := raw.(string)
	if !isString || !contains(SampleLooks, look) {
		v.fail(CodeBadEnum, path+".look", "must be one of: %s", strings.Join(SampleLooks, ", "))
		return ""
	}
	return look
}

// checkSampleComponent checks the inner component like a screen component, but without its
// required props, without $bindings and without navigation.
func (v *validator) checkSampleComponent(path string, spec ComponentSpec, inner map[string]any) {
	v.inSample = true
	defer func() { v.inSample = false }()
	v.checkShape(path+"."+spec.Key, spec.Main, inner[spec.Key])
	others := map[string]any{}
	for key, value := range inner {
		if key != spec.Key {
			others[key] = value
		}
	}
	v.checkObject(path, spec.Props, others)
	for _, forbidden := range []string{"push", "bind"} {
		if _, found := others[forbidden]; found {
			v.fail(CodeSampleNotStatic, path+"."+forbidden, "a sample is a picture: it cannot use %q", forbidden)
		}
	}
	switch spec.Key {
	case "radio":
		if options, _ := inner["options"].([]any); len(options) != 1 {
			v.fail(CodeOutOfRange, path+".options", "a radio sample shows exactly one option, not %d", len(options))
		}
	case "kv":
		if rows, _ := inner["kv"].([]any); len(rows) > 1 {
			v.fail(CodeOutOfRange, path+".kv", "a key-value sample shows exactly one row, not %d", len(rows))
		}
	}
}

// checkSampleColumns: more than one column only holds buttons (a 548 px row, field or key-value
// line cannot be read).
func (v *validator) checkSampleColumns(path string, component Component) {
	columns, _ := component.Props["columns"].(float64)
	if columns <= 1 {
		return
	}
	items, _ := component.Props["samples"].([]any)
	for i, item := range items {
		object, _ := item.(map[string]any)
		if _, isButton := object["button"]; !isButton {
			v.fail(CodeOutOfRange, fmt.Sprintf("%s.samples[%d]", path, i), "cell is not a button: use \"columns\": 1 (only buttons fit side by side)")
		}
	}
}
