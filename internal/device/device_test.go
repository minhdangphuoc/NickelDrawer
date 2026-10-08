package device

import (
	"os"
	"path/filepath"
	"testing"
)

const sampleLine = "N000000000000,4.1.15,4.38.23828,4.1.15,4.1.15,00000000-0000-0000-0000-000000000388\n"

func TestParseSample(t *testing.T) {
	info, err := Parse(sampleLine)
	if err != nil {
		t.Fatal(err)
	}
	if info.Model != "Kobo Libra 2" || info.Firmware != "4.38.23828" || info.ID != "00000000-0000-0000-0000-000000000388" {
		t.Fatalf("info = %+v", info)
	}
}

func TestUnknownIDFallsBack(t *testing.T) {
	info, err := Parse("S,1,2,3,4,00000000-0000-0000-0000-000000000999")
	if err != nil || info.Model != FallbackName {
		t.Fatalf("info = %+v err = %v", info, err)
	}
	if ModelName("") != FallbackName || ModelName("7") != FallbackName {
		t.Fatal("short ids must fall back")
	}
}

func TestParseRejectsShortLine(t *testing.T) {
	if _, err := Parse("only,three,fields"); err == nil {
		t.Fatal("want an error")
	}
}

func TestReadFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "version")
	if err := os.WriteFile(path, []byte(sampleLine), 0o644); err != nil {
		t.Fatal(err)
	}
	info, err := Read(path)
	if err != nil || info.Model != "Kobo Libra 2" {
		t.Fatalf("info = %+v err = %v", info, err)
	}
	if _, err := Read(filepath.Join(t.TempDir(), "missing")); err == nil {
		t.Fatal("want an error for a missing file")
	}
}
