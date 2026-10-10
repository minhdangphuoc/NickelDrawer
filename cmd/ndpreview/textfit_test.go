package main

import (
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"testing"
	"unicode/utf8"
)

const textVectorsPath = "../../lib/copperui/tests/text-vectors.json"

type fakeMetric struct {
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

// width is the deterministic font of the shared vectors (see the file's comment).
func (m fakeMetric) width(text string, face textFace) int {
	units := 0
	for _, char := range text {
		switch {
		case strings.ContainsRune(m.Wide.Chars, char):
			units += m.Wide.Percent
		case strings.ContainsRune(m.Narrow.Chars, char):
			units += m.Narrow.Percent
		default:
			units += m.DefaultPercent
		}
	}
	width := units * face.size / 100
	width = width * m.FamilyPercent[face.family] / 100
	if face.isBold {
		width = width * m.BoldPercent / 100
	}
	return width
}

type textVector struct {
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

func loadTextVectors(t *testing.T) (fakeMetric, []textVector) {
	t.Helper()
	data, err := os.ReadFile(textVectorsPath)
	if err != nil {
		t.Fatalf("cannot read the shared text vectors: %v", err)
	}
	var file struct {
		Measure fakeMetric   `json:"measure"`
		Cases   []textVector `json:"cases"`
	}
	if err := json.Unmarshal(data, &file); err != nil {
		t.Fatalf("the shared text vectors are not valid JSON: %v", err)
	}
	return file.Measure, file.Cases
}

func useMeter(t *testing.T, meter textMeasurer) {
	t.Helper()
	previous := textMeter
	textMeter = meter
	t.Cleanup(func() { textMeter = previous })
}

func runVector(vector textVector) []string {
	face := textFace{size: vector.Size, family: vector.Family, isBold: vector.Bold}
	switch vector.Op {
	case "wrap":
		return wrapText(vector.Text, vector.Width, face)
	case "wrapLimited":
		return wrapLimited(vector.Text, vector.Width, face, vector.MaxLines)
	case "elide":
		return []string{elide(vector.Text, vector.Width, face, false)}
	case "elideMiddle":
		return []string{elide(vector.Text, vector.Width, face, true)}
	}
	return nil
}

func TestSharedTextVectors(t *testing.T) {
	metric, vectors := loadTextVectors(t)
	useMeter(t, metric)
	for _, vector := range vectors {
		got := runVector(vector)
		if len(got) == 0 && len(vector.Expect) == 0 {
			continue
		}
		if !reflect.DeepEqual(got, vector.Expect) {
			t.Errorf("%s: %s(%q, width %d) = %q, want %q", vector.Name, vector.Op, vector.Text, vector.Width, got, vector.Expect)
		}
	}
}

func TestEveryLineFitsOrIsOneCodePoint(t *testing.T) {
	metric, vectors := loadTextVectors(t)
	useMeter(t, metric)
	for _, vector := range vectors {
		if vector.Op != "wrap" {
			continue
		}
		face := textFace{size: vector.Size, family: vector.Family, isBold: vector.Bold}
		for _, line := range runVector(vector) {
			if utf8.RuneCountInString(line) > 1 && metric.width(line, face) > vector.Width {
				t.Errorf("%s: line %q is %d px wide, wider than %d", vector.Name, line, metric.width(line, face), vector.Width)
			}
		}
	}
}

func TestEstimateMeterKeepsLayoutWorkingWithoutFonts(t *testing.T) {
	useMeter(t, estimatedMeter{})
	lines := wrapText("a few words to wrap in a narrow column", 300, regular(40, familySans))
	if len(lines) < 2 {
		t.Fatalf("expected the text to wrap, got %q", lines)
	}
}

func TestRowDividersAreGreyAndTheSectionRuleStaysBlack(t *testing.T) {
	useMeter(t, estimatedMeter{})
	rc := renderContext{}
	rowItems := valueRow("Key", "Value", "", "", false).draw(0)
	sectionItems := rc.sectionBlock("Section").draw(0)
	if got := dividerFill(t, rowItems); got != lineRow {
		t.Errorf("row divider fill = %d, want %d (line.row)", got, lineRow)
	}
	if got := dividerFill(t, sectionItems); got != lineSection {
		t.Errorf("section rule fill = %d, want %d (black)", got, lineSection)
	}
}

func dividerFill(t *testing.T, items []Item) int {
	t.Helper()
	for _, item := range items {
		if item.Shape == "rect" && item.H == separatorPx && item.W == contentWidth {
			return item.Fill
		}
	}
	t.Fatal("no divider among the items")
	return 0
}

func TestButtonLabelKeepsAtMostTwoLinesAndGrowsTall(t *testing.T) {
	metric, _ := loadTextVectors(t)
	useMeter(t, metric)
	rc := renderContext{}
	longLabel := strings.Repeat("word ", 60)
	block := rc.button(longLabel, "", true, true, "", false)
	if block.height != buttonTallHt {
		t.Errorf("a long label gives a %d px button, want %d", block.height, buttonTallHt)
	}
	for _, item := range block.draw(0) {
		if item.Shape == "text" && len(item.Lines) != buttonLines {
			t.Errorf("button text has %d lines, want %d", len(item.Lines), buttonLines)
		}
	}
	short := rc.button("OK", "", true, true, "", false)
	if short.height != buttonHeight {
		t.Errorf("a short label gives a %d px button, want %d", short.height, buttonHeight)
	}
}

func TestSectionLabelIsMeasuredInCapitals(t *testing.T) {
	metric, _ := loadTextVectors(t)
	useMeter(t, metric)
	rc := renderContext{}
	// Lower case "m" is 32 px wide at size 28 in the fake font; the capital is the same, but the
	// elision must still be decided on the uppercase text.
	label := strings.Repeat("m", 80)
	for _, item := range rc.sectionBlock(label).draw(0) {
		if item.Shape != "text" {
			continue
		}
		face := bold(sectionSize, familySans)
		if width := metric.width(item.Lines[0], face); width > contentWidth {
			t.Errorf("section label is %d px wide, wider than %d", width, contentWidth)
		}
		if item.Lines[0] != strings.ToUpper(item.Lines[0]) {
			t.Errorf("section label %q is not uppercase", item.Lines[0])
		}
	}
}
