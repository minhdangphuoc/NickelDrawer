package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// The shared text vectors (lib/copperui/tests/text-vectors.json) are read by the Go tests of
// cmd/ndpreview directly. The mod's host tests have no JSON parser and no C++ standard library,
// so this file turns the same vectors into mod/drawer/tests/text_vectors.inc.

type vectorMeasure struct {
	DefaultPercent int `json:"defaultPercent"`
	Wide           struct {
		Chars   string `json:"chars"`
		Percent int    `json:"percent"`
	} `json:"wide"`
	Narrow struct {
		Chars   string `json:"chars"`
		Percent int    `json:"percent"`
	} `json:"narrow"`
	FamilyPercent map[string]int `json:"familyPercent"`
	BoldPercent   int            `json:"boldPercent"`
}

type vectorCase struct {
	Name     string   `json:"name"`
	Op       string   `json:"op"`
	Text     string   `json:"text"`
	Width    int      `json:"width"`
	Size     int      `json:"size"`
	Family   string   `json:"family"`
	Bold     bool     `json:"bold"`
	MaxLines int      `json:"maxLines"`
	Expect   []string `json:"expect"`
}

type vectorFile struct {
	Comment string        `json:"comment"`
	Measure vectorMeasure `json:"measure"`
	Cases   []vectorCase  `json:"cases"`
}

var vectorOps = map[string]string{"wrap": "OP_WRAP", "wrapLimited": "OP_WRAP_LIMITED", "elide": "OP_ELIDE", "elideMiddle": "OP_ELIDE_MIDDLE"}
var vectorFamilies = map[string]string{"sans": "FAMILY_SANS", "serif": "FAMILY_SERIF", "mono": "FAMILY_MONO"}

const maxExpectedLines = 6

func loadVectors(path string) (*vectorFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("cannot read the text vectors %s: %w", path, err)
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var file vectorFile
	if err := decoder.Decode(&file); err != nil {
		return nil, fmt.Errorf("the text vectors %s are not valid: %w", path, err)
	}
	for _, vector := range file.Cases {
		if err := validateVector(vector); err != nil {
			return nil, err
		}
	}
	return &file, nil
}

func validateVector(vector vectorCase) error {
	if _, known := vectorOps[vector.Op]; !known {
		return fmt.Errorf("text vector %q has unknown op %q (wrap, wrapLimited, elide, elideMiddle)", vector.Name, vector.Op)
	}
	if _, known := vectorFamilies[vector.Family]; !known {
		return fmt.Errorf("text vector %q has unknown family %q (sans, serif, mono)", vector.Name, vector.Family)
	}
	if len(vector.Expect) > maxExpectedLines {
		return fmt.Errorf("text vector %q expects %d lines; the C++ table holds %d", vector.Name, len(vector.Expect), maxExpectedLines)
	}
	return nil
}

// cString is a C string literal with every non-ASCII byte as a 3-digit octal escape, which
// cannot swallow the characters that follow it.
func cString(text string) string {
	var out strings.Builder
	out.WriteByte('"')
	for _, b := range []byte(text) {
		switch {
		case b == '"' || b == '\\':
			out.WriteByte('\\')
			out.WriteByte(b)
		case b == '\n':
			out.WriteString(`\n`)
		case b < 0x20 || b >= 0x7f:
			fmt.Fprintf(&out, `\%03o`, b)
		default:
			out.WriteByte(b)
		}
	}
	out.WriteByte('"')
	return out.String()
}

func emitVectors(file *vectorFile) []byte {
	var out bytes.Buffer
	fmt.Fprintf(&out, "// %s\n", generatedNote)
	out.WriteString("// Source: lib/copperui/tests/text-vectors.json\n\n")
	measure := file.Measure
	fmt.Fprintf(&out, "static const int FAKE_DEFAULT_PERCENT = %d;\n", measure.DefaultPercent)
	fmt.Fprintf(&out, "static const char *const FAKE_WIDE_CHARS = %s;\nstatic const int FAKE_WIDE_PERCENT = %d;\n", cString(measure.Wide.Chars), measure.Wide.Percent)
	fmt.Fprintf(&out, "static const char *const FAKE_NARROW_CHARS = %s;\nstatic const int FAKE_NARROW_PERCENT = %d;\n", cString(measure.Narrow.Chars), measure.Narrow.Percent)
	fmt.Fprintf(&out, "static const int FAKE_FAMILY_PERCENT[] = {%d, %d, %d}; // sans, serif, mono\n", measure.FamilyPercent["sans"], measure.FamilyPercent["serif"], measure.FamilyPercent["mono"])
	fmt.Fprintf(&out, "static const int FAKE_BOLD_PERCENT = %d;\n\n", measure.BoldPercent)
	fmt.Fprintf(&out, "static const int MAX_EXPECTED_LINES = %d;\n\n", maxExpectedLines)
	out.WriteString("static const TextVector TEXT_VECTORS[] = {\n")
	for _, vector := range file.Cases {
		fmt.Fprintf(&out, "    {%s, %s, %s, %d, %d, %s, %t, %d, %d, {", cString(vector.Name), vectorOps[vector.Op], cString(vector.Text),
			vector.Width, vector.Size, vectorFamilies[vector.Family], vector.Bold, vector.MaxLines, len(vector.Expect))
		for index, line := range vector.Expect {
			if index > 0 {
				out.WriteString(", ")
			}
			out.WriteString(cString(line))
		}
		out.WriteString("}},\n")
	}
	out.WriteString("};\n")
	return out.Bytes()
}
