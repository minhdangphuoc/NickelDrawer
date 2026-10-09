package uikit

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"
	"unicode/utf8"

	"koboapps/internal/manifest"
)

var updateSchema = flag.Bool("update", false, "rewrite spec/ui.schema.json from the component registry")

const repoRoot = "../.."

var schemaPath = filepath.Join(repoRoot, "spec", "ui.schema.json")

func TestSchemaFileIsCurrent(t *testing.T) {
	want, err := SchemaJSON()
	if err != nil {
		t.Fatal(err)
	}
	if *updateSchema {
		if err := os.WriteFile(schemaPath, want, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	got, err := os.ReadFile(schemaPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatal("spec/ui.schema.json is stale; run: go test ./internal/uikit -run TestSchemaFileIsCurrent -update")
	}
}

// exampleAppDirs are the apps whose ui.json must satisfy both the schema and the validator.
func exampleAppDirs(t *testing.T) []string {
	t.Helper()
	var dirs []string
	for _, pattern := range []string{"apps/*/ui.json", "examples/*/ui.json"} {
		matches, err := filepath.Glob(filepath.Join(repoRoot, pattern))
		if err != nil {
			t.Fatal(err)
		}
		for _, match := range matches {
			dirs = append(dirs, filepath.Dir(match))
		}
	}
	if len(dirs) < 3 {
		t.Fatalf("expected at least todo, localsend and wallpaper ui.json, found %v", dirs)
	}
	return dirs
}

func TestExamplesSatisfySchemaAndValidator(t *testing.T) {
	raw, err := os.ReadFile(schemaPath)
	if err != nil {
		t.Fatal(err)
	}
	var schema map[string]any
	if err := json.Unmarshal(raw, &schema); err != nil {
		t.Fatal(err)
	}
	for _, dir := range exampleAppDirs(t) {
		t.Run(filepath.Base(dir), func(t *testing.T) {
			data, err := os.ReadFile(filepath.Join(dir, FileName))
			if err != nil {
				t.Fatal(err)
			}
			var instance any
			if err := json.Unmarshal(data, &instance); err != nil {
				t.Fatal(err)
			}
			if problems := (&schemaChecker{root: schema}).check(schema, instance, "$"); len(problems) > 0 {
				t.Errorf("schema rejects the example: %v", problems)
			}
			app, _, err := manifest.Load(dir)
			if err != nil {
				t.Fatal(err)
			}
			ids := make([]string, len(app.Actions))
			for i, action := range app.Actions {
				ids[i] = action.ID
			}
			if _, report, err := LoadDir(dir, ids); err != nil || !report.OK() {
				t.Errorf("validator rejects the example: err=%v errors=%v", err, report.Errors)
			}
		})
	}
}

// Schema and validator must reject the same broken documents.
func TestSchemaAndValidatorAgreeOnBrokenDocuments(t *testing.T) {
	raw, _ := os.ReadFile(schemaPath)
	var schema map[string]any
	if err := json.Unmarshal(raw, &schema); err != nil {
		t.Fatal(err)
	}
	broken := map[string]string{
		"unknown component": screenJSON(`[{"carousel":"x"}]`, ""),
		"missing prop":      screenJSON(`[{"toggle":"Wi-Fi"}]`, ""),
		"unknown prop":      screenJSON(`[{"text":"a","colour":"red"}]`, ""),
		"bad enum":          screenJSON(`[{"text":"a","style":"huge"}]`, ""),
		"bad binding flag":  screenJSON(`[{"toggle":"x","bind":"on","enabled":"yes"}]`, ""),
		"bad button flag":   screenJSON(`[]`, `[{"button":"Go","action":"start","enabled":"on"}]`),
		"colon in label":    screenJSON(`[]`, `[{"button":"Step 1: go","action":"start"}]`),
		"too long":          screenJSON(`[{"section":"`+strings.Repeat("x", 41)+`"}]`, ""),
		"missing english":   screenJSON(`[{"text":{"vi":"Chào"}}]`, ""),
		"nine options":      screenJSON(`[{"radio":"L","bind":"pin","options":[["a","A"],["b","B"],["c","C"],["d","D"],["e","E"],["f","F"],["g","G"],["h","H"],["i","I"]]}]`, ""),
	}
	for name, document := range broken {
		t.Run(name, func(t *testing.T) {
			var instance any
			if err := json.Unmarshal([]byte(document), &instance); err != nil {
				t.Fatal(err)
			}
			if problems := (&schemaChecker{root: schema}).check(schema, instance, "$"); len(problems) == 0 {
				t.Error("schema accepts a document the validator rejects")
			}
			if validate(t, document).OK() {
				t.Error("validator accepts a broken document")
			}
		})
	}
}

// Every registry component must be documented in docs/spec/ui-json.md.
func TestSpecDocumentNamesEveryComponent(t *testing.T) {
	doc, err := os.ReadFile(filepath.Join(repoRoot, "docs", "spec", "ui-json.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, spec := range Components {
		if !strings.Contains(string(doc), "`"+spec.Key+"`") {
			t.Errorf("docs/spec/ui-json.md does not document component %q", spec.Key)
		}
	}
}

// schemaChecker implements the subset of JSON Schema 2020-12 that SchemaJSON emits.
type schemaChecker struct{ root map[string]any }

func (c *schemaChecker) check(schema map[string]any, value any, path string) []string {
	if ref, isRef := schema["$ref"].(string); isRef {
		return c.check(c.resolve(ref), value, path)
	}
	var problems []string
	add := func(format string, args ...any) { problems = append(problems, path+": "+fmt.Sprintf(format, args...)) }
	if want, has := schema["const"]; has && !reflect.DeepEqual(want, value) {
		add("must equal %v", want)
	}
	if allowed, has := schema["enum"].([]any); has && !containsValue(allowed, value) {
		add("must be one of %v", allowed)
	}
	if kind, has := schema["type"].(string); has && !hasType(kind, value) {
		return append(problems, path+": must be "+kind)
	}
	problems = append(problems, c.checkCombinators(schema, value, path)...)
	problems = append(problems, c.checkString(schema, value, path)...)
	problems = append(problems, c.checkNumber(schema, value, path)...)
	problems = append(problems, c.checkArray(schema, value, path)...)
	problems = append(problems, c.checkObject(schema, value, path)...)
	return problems
}

func (c *schemaChecker) resolve(ref string) map[string]any {
	node := any(c.root)
	for _, part := range strings.Split(strings.TrimPrefix(ref, "#/"), "/") {
		node = node.(map[string]any)[part]
	}
	return node.(map[string]any)
}

func (c *schemaChecker) checkCombinators(schema map[string]any, value any, path string) []string {
	var problems []string
	for _, keyword := range []string{"oneOf", "anyOf"} {
		variants, has := schema[keyword].([]any)
		if !has {
			continue
		}
		matches := 0
		for _, variant := range variants {
			if len(c.check(variant.(map[string]any), value, path)) == 0 {
				matches++
			}
		}
		if matches == 0 || (keyword == "oneOf" && matches > 1) {
			problems = append(problems, fmt.Sprintf("%s: matches %d of the %s variants", path, matches, keyword))
		}
	}
	return problems
}

func (c *schemaChecker) checkString(schema map[string]any, value any, path string) []string {
	text, isString := value.(string)
	if !isString {
		return nil
	}
	var problems []string
	if limit, has := schema["maxLength"].(float64); has && float64(utf8.RuneCountInString(text)) > limit {
		problems = append(problems, fmt.Sprintf("%s: longer than %v", path, limit))
	}
	if limit, has := schema["minLength"].(float64); has && float64(utf8.RuneCountInString(text)) < limit {
		problems = append(problems, fmt.Sprintf("%s: shorter than %v", path, limit))
	}
	if pattern, has := schema["pattern"].(string); has && !regexp.MustCompile(pattern).MatchString(text) {
		problems = append(problems, fmt.Sprintf("%s: %q does not match %s", path, text, pattern))
	}
	return problems
}

func (c *schemaChecker) checkNumber(schema map[string]any, value any, path string) []string {
	number, isNumber := value.(float64)
	if !isNumber {
		return nil
	}
	var problems []string
	if low, has := schema["minimum"].(float64); has && number < low {
		problems = append(problems, fmt.Sprintf("%s: below %v", path, low))
	}
	if high, has := schema["maximum"].(float64); has && number > high {
		problems = append(problems, fmt.Sprintf("%s: above %v", path, high))
	}
	return problems
}

func (c *schemaChecker) checkArray(schema map[string]any, value any, path string) []string {
	items, isArray := value.([]any)
	if !isArray {
		return nil
	}
	var problems []string
	if low, has := schema["minItems"].(float64); has && float64(len(items)) < low {
		problems = append(problems, fmt.Sprintf("%s: fewer than %v items", path, low))
	}
	if high, has := schema["maxItems"].(float64); has && float64(len(items)) > high {
		problems = append(problems, fmt.Sprintf("%s: more than %v items", path, high))
	}
	prefix, _ := schema["prefixItems"].([]any)
	for i, item := range items {
		itemPath := fmt.Sprintf("%s[%d]", path, i)
		switch {
		case i < len(prefix):
			problems = append(problems, c.check(prefix[i].(map[string]any), item, itemPath)...)
		case schema["items"] == false:
			problems = append(problems, itemPath+": extra item")
		case schema["items"] != nil:
			problems = append(problems, c.check(schema["items"].(map[string]any), item, itemPath)...)
		}
	}
	return problems
}

func (c *schemaChecker) checkObject(schema map[string]any, value any, path string) []string {
	object, isObject := value.(map[string]any)
	if !isObject {
		return nil
	}
	var problems []string
	required, _ := schema["required"].([]any)
	for _, name := range required {
		if _, present := object[name.(string)]; !present {
			problems = append(problems, fmt.Sprintf("%s: missing %q", path, name))
		}
	}
	properties, _ := schema["properties"].(map[string]any)
	patterns, _ := schema["patternProperties"].(map[string]any)
	names, _ := schema["propertyNames"].(map[string]any)
	for _, key := range sortedKeys(object) {
		keyPath := path + "." + key
		if names != nil && len(c.check(names, key, keyPath)) > 0 {
			problems = append(problems, keyPath+": bad property name")
		}
		problems = append(problems, c.checkProperty(schema, properties, patterns, key, object[key], keyPath)...)
	}
	return problems
}

func (c *schemaChecker) checkProperty(schema, properties, patterns map[string]any, key string, value any, path string) []string {
	if sub, known := properties[key].(map[string]any); known {
		return c.check(sub, value, path)
	}
	for pattern, sub := range patterns {
		if regexp.MustCompile(pattern).MatchString(key) {
			return c.check(sub.(map[string]any), value, path)
		}
	}
	switch extra := schema["additionalProperties"].(type) {
	case bool:
		if !extra {
			return []string{path + ": unknown property"}
		}
	case map[string]any:
		return c.check(extra, value, path)
	}
	return nil
}

func hasType(kind string, value any) bool {
	switch kind {
	case "string":
		_, ok := value.(string)
		return ok
	case "boolean":
		_, ok := value.(bool)
		return ok
	case "object":
		_, ok := value.(map[string]any)
		return ok
	case "array":
		_, ok := value.([]any)
		return ok
	case "integer":
		number, ok := value.(float64)
		return ok && number == math.Trunc(number)
	}
	return false
}

func containsValue(allowed []any, value any) bool {
	for _, candidate := range allowed {
		if reflect.DeepEqual(candidate, value) {
			return true
		}
	}
	return false
}
