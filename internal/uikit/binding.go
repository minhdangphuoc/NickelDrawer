package uikit

import (
	"fmt"
	"strconv"
	"strings"
)

// Binding syntax (docs/spec/ui-json.md section 5): "$name" and "$name.sub" are replaced by a
// variable, "$$" is a literal dollar sign, and "!$name" negates a flag.

// ParseTemplate returns the variable paths a template refers to, or an error that says what is
// wrong with the first bad "$".
func ParseTemplate(template string) ([]string, error) {
	var refs []string
	for i := 0; i < len(template); i++ {
		if template[i] != '$' {
			continue
		}
		if i+1 < len(template) && template[i+1] == '$' {
			i++
			continue
		}
		ref, next, err := scanReference(template, i+1)
		if err != nil {
			return nil, err
		}
		refs = append(refs, ref)
		i = next - 1
	}
	return refs, nil
}

// scanReference reads ident(.ident)* from start and returns it with the index after it.
func scanReference(template string, start int) (string, int, error) {
	position := start
	for {
		identEnd := scanIdent(template, position)
		if identEnd == position {
			return "", 0, fmt.Errorf("a '$' must be followed by a variable name (write $$ for a literal dollar sign) in %q", template)
		}
		position = identEnd
		if position < len(template) && template[position] == '.' && scanIdent(template, position+1) > position+1 {
			position++
			continue
		}
		break
	}
	if position < len(template) && template[position] == '[' {
		return "", 0, fmt.Errorf("indexing like $list[0] is not supported in kit 1 (in %q)", template)
	}
	return template[start:position], position, nil
}

func scanIdent(text string, start int) int {
	end := start
	for end < len(text) && isIdentByte(text[end], end == start) {
		end++
	}
	return end
}

func isIdentByte(char byte, isFirst bool) bool {
	isLetter := char == '_' || (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z')
	return isLetter || (!isFirst && char >= '0' && char <= '9')
}

// ParseFlag accepts a literal boolean or "$name" / "!$name" and returns the variable path.
func ParseFlag(text string) (path string, isNegated bool, err error) {
	rest := strings.TrimPrefix(text, "!")
	isNegated = rest != text
	path, ok := pureReference(rest)
	if !ok {
		return "", false, fmt.Errorf("%q is not a flag: use true, false, \"$name\" or \"!$name\"", text)
	}
	return path, isNegated, nil
}

// pureReference reports whether text is exactly one "$name[.sub]" binding.
func pureReference(text string) (string, bool) {
	if !strings.HasPrefix(text, "$") {
		return "", false
	}
	path, end, err := scanReference(text, 1)
	if err != nil || end != len(text) {
		return "", false
	}
	return path, true
}

// Lookup finds a dotted path such as "device.name" in vars.
func Lookup(vars map[string]any, path string) (any, bool) {
	var current any = vars
	for _, part := range strings.Split(path, ".") {
		object, isObject := current.(map[string]any)
		if !isObject {
			return nil, false
		}
		next, found := object[part]
		if !found {
			return nil, false
		}
		current = next
	}
	return current, true
}

// Substitute fills the bindings of a template from vars. Unknown variables become empty text so
// a half-configured preview still renders.
func Substitute(template string, vars map[string]any) string {
	var out strings.Builder
	for i := 0; i < len(template); i++ {
		if template[i] != '$' {
			out.WriteByte(template[i])
			continue
		}
		if i+1 < len(template) && template[i+1] == '$' {
			out.WriteByte('$')
			i++
			continue
		}
		path, next, err := scanReference(template, i+1)
		if err != nil {
			out.WriteByte('$')
			continue
		}
		value, _ := Lookup(vars, path)
		out.WriteString(FormatValue(value))
		i = next - 1
	}
	return out.String()
}

// FormatValue renders a JSON value as display text.
func FormatValue(value any) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case string:
		return typed
	case float64:
		return strconv.FormatFloat(typed, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(typed)
	default:
		return fmt.Sprint(typed)
	}
}

// Truthy evaluates a flag: a bool, or "$name" / "!$name" against vars.
func Truthy(flagValue any, vars map[string]any) bool {
	switch typed := flagValue.(type) {
	case nil:
		return true
	case bool:
		return typed
	case string:
		path, isNegated, err := ParseFlag(typed)
		if err != nil {
			return false
		}
		value, _ := Lookup(vars, path)
		return isTruthy(value) != isNegated
	}
	return false
}

func isTruthy(value any) bool {
	switch typed := value.(type) {
	case nil:
		return false
	case bool:
		return typed
	case string:
		return typed != ""
	case float64:
		return typed != 0
	case []any:
		return len(typed) > 0
	}
	return true
}
