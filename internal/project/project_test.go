package project

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

const shippedConfig = "../../host/config/project.json"

func loadShipped(t *testing.T) Info {
	t.Helper()
	info, err := Load(shippedConfig)
	if err != nil {
		t.Fatal(err)
	}
	return info
}

func TestShippedConfigNamesTheUpstreamProject(t *testing.T) {
	info := loadShipped(t)
	if info.Repo != "minhdangphuoc/NickelDrawer" || info.Maintainer != "Minh Dang" || info.License != "PolyForm Noncommercial 1.0.0" {
		t.Fatalf("info = %+v", info)
	}
	if info.URL() != "https://github.com/minhdangphuoc/NickelDrawer" {
		t.Fatalf("url = %q", info.URL())
	}
	if info.FirmwareRange() != "4.38.23552 – 4.38.23828" {
		t.Fatalf("range = %q", info.FirmwareRange())
	}
}

func TestUpdateConfigComesFromTheProjectRepo(t *testing.T) {
	var config struct{ Repo, Channel string }
	info := loadShipped(t)
	if err := json.Unmarshal(info.UpdateConfigJSON(), &config); err != nil {
		t.Fatal(err)
	}
	if config.Repo != info.Repo || config.Channel != "stable" {
		t.Fatalf("update config = %+v", config)
	}
}

func TestSupportsFirmware(t *testing.T) {
	info := loadShipped(t)
	if !info.SupportsFirmware("4.38.23828") || info.SupportsFirmware("4.38.23429") || info.SupportsFirmware("") {
		t.Fatal("only the listed builds are supported")
	}
}

func TestLoadRejectsAConfigThatCannotWork(t *testing.T) {
	cases := map[string]string{
		"not json":      `{`,
		"no name":       `{"repo":"o/r","supported_firmware":["4.1"]}`,
		"bad repo":      `{"name":"X","repo":"nope","supported_firmware":["4.1"]}`,
		"no firmwares":  `{"name":"X","repo":"o/r"}`,
		"repo with url": `{"name":"X","repo":"https://github.com/o/r","supported_firmware":["4.1"]}`,
	}
	for name, content := range cases {
		path := filepath.Join(t.TempDir(), FileName)
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := Load(path); err == nil {
			t.Errorf("%s: want an error", name)
		}
	}
	if _, err := Load(filepath.Join(t.TempDir(), "missing.json")); err == nil {
		t.Error("a missing file must be an error")
	}
}

// quotedBuilds reads the "4.38.x" strings of a C++ source file.
func quotedBuilds(t *testing.T, path string) []string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	start := strings.Index(string(data), "SUPPORTED_FIRMWARE[] = {")
	if start < 0 {
		t.Fatalf("%s has no SUPPORTED_FIRMWARE list", path)
	}
	list := string(data)[start:]
	list = list[:strings.Index(list, "};")]
	builds := regexp.MustCompile(`"(4\.\d+\.\d+)"`).FindAllStringSubmatch(list, -1)
	var result []string
	for _, build := range builds {
		result = append(result, build[1])
	}
	sort.Strings(result)
	return result
}

// The mods compile their own copy of the list; the test keeps them equal to project.json so
// the About screen never claims a build the mod would refuse.
func TestModsListTheSameFirmwareAsTheProjectConfig(t *testing.T) {
	want := append([]string{}, loadShipped(t).SupportedFirmware...)
	sort.Strings(want)
	for _, source := range []string{"../../mod/drawer/nickel_bridge.cc", "../../mod/drawer-proto/device_state.cc"} {
		got := quotedBuilds(t, source)
		if strings.Join(got, ",") != strings.Join(want, ",") {
			t.Errorf("%s lists %v, project.json lists %v", source, got, want)
		}
	}
}
