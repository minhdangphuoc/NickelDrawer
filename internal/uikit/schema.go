package uikit

import (
	"encoding/json"
	"fmt"
)

// SchemaID names the generated schema file.
const SchemaID = "https://koboapps.invalid/spec/ui.schema.json"

type object = map[string]any

// SchemaJSON renders spec/ui.schema.json (JSON Schema draft 2020-12) from the component
// registry. A test keeps the checked-in file equal to this output.
func SchemaJSON() ([]byte, error) {
	data, err := json.MarshalIndent(buildSchema(), "", "  ")
	if err != nil {
		return nil, fmt.Errorf("cannot render the ui.json schema: %w", err)
	}
	return append(data, '\n'), nil
}

func buildSchema() object {
	return object{
		"$schema":     "https://json-schema.org/draft/2020-12/schema",
		"$id":         SchemaID,
		"title":       "NickelDrawer ui.json, kit 1",
		"description": "Generated from internal/uikit by `go test ./internal/uikit -run TestSchemaFileIsCurrent -update`. Rules a schema cannot express (bindings name declared vars, action ids exist in app.json, one primary button per screen, @ui.key texts exist in locales/en.json) are checked by `appsctl validate`. The same screen object is the schema of screens/<id>.json (#/$defs/screen).",
		"type":        "object",
		"required":    []string{"kit", "start", "screens"},
		"properties": object{
			"kit":     object{"const": KitVersion},
			"start":   object{"type": "string", "pattern": screenIDRegexp},
			"vars":    object{"type": "object", "propertyNames": object{"pattern": identOnlyRegexp}},
			"screens": object{"type": "object", "propertyNames": object{"pattern": screenIDRegexp}, "additionalProperties": object{"$ref": "#/$defs/screen"}},
		},
		"patternProperties":    object{commentKeyRegexp: object{}},
		"additionalProperties": false,
		"$defs": object{
			"screen":    shapeSchema(ScreenShape),
			"component": componentSchema(),
		},
	}
}

func componentSchema() object {
	variants := make([]any, len(Components))
	for i, spec := range Components {
		variants[i] = componentVariantSchema(spec)
	}
	return object{"oneOf": variants}
}

func componentVariantSchema(spec ComponentSpec) object {
	properties := object{spec.Key: shapeSchema(spec.Main)}
	required := []string{spec.Key}
	for _, prop := range spec.Props {
		properties[prop.Name] = shapeSchema(prop.Shape)
		if prop.Required {
			required = append(required, prop.Name)
		}
	}
	return object{"type": "object", "required": required, "properties": properties, "additionalProperties": false}
}

func shapeSchema(shape Shape) object {
	switch shape.Kind {
	case KindText:
		return textSchema(shape)
	case KindExpr:
		return object{"type": "string", "maxLength": shape.MaxLen}
	case KindFlag:
		return object{"anyOf": []any{object{"type": "boolean"}, object{"type": "string", "pattern": flagRefRegexp}}}
	case KindBool:
		return object{"type": "boolean"}
	case KindInt:
		return object{"type": "integer", "minimum": shape.Min, "maximum": shape.Max}
	case KindEnum:
		return object{"enum": shape.Enum}
	case KindAction:
		return object{"type": "string", "pattern": actionIDRegexp}
	case KindScreenRef:
		return object{"type": "string", "pattern": screenIDRegexp}
	case KindVarName:
		return object{"type": "string", "pattern": identOnlyRegexp}
	case KindVarRef:
		return object{"type": "string", "pattern": varRefRegexp}
	case KindID:
		return object{"type": "string", "pattern": idRegexp}
	case KindComponents:
		return object{"type": "array", "items": object{"$ref": "#/$defs/component"}}
	case KindAny:
		return object{}
	}
	return structuredSchema(shape)
}

func structuredSchema(shape Shape) object {
	switch shape.Kind {
	case KindObject:
		return objectSchema(shape)
	case KindArray:
		return object{"type": "array", "items": shapeSchema(*shape.Item), "minItems": shape.Min, "maxItems": shape.Max}
	case KindTuple:
		prefix := make([]any, len(shape.Tuple))
		for i, item := range shape.Tuple {
			prefix[i] = shapeSchema(item)
		}
		return object{"type": "array", "prefixItems": prefix, "items": false, "minItems": len(prefix), "maxItems": len(prefix)}
	case KindMap:
		return object{"type": "object", "propertyNames": object{"pattern": idRegexp}, "additionalProperties": shapeSchema(*shape.Item)}
	}
	return object{}
}

func objectSchema(shape Shape) object {
	properties := object{}
	var required []string
	for _, prop := range shape.Fields {
		properties[prop.Name] = shapeSchema(prop.Shape)
		if prop.Required {
			required = append(required, prop.Name)
		}
	}
	schema := object{"type": "object", "properties": properties, "additionalProperties": false}
	if len(required) > 0 {
		schema["required"] = required
	}
	return schema
}

// textSchema accepts a plain string, an "@ui.key" reference or a locale map with "en".
func textSchema(shape Shape) object {
	plain := object{"type": "string", "minLength": 1, "maxLength": shape.MaxLen}
	localized := object{"type": "string", "minLength": 1, "maxLength": shape.MaxLen}
	if shape.NoColon {
		plain["pattern"] = `^[^:]+$`
		localized["pattern"] = `^[^:]+$`
	}
	return object{"anyOf": []any{
		object{"type": "string", "pattern": localeKeyRegexp},
		plain,
		object{
			"type":                 "object",
			"required":             []string{"en"},
			"propertyNames":        object{"pattern": localeCodeRegexp},
			"additionalProperties": localized,
		},
	}}
}
