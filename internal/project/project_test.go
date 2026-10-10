package project

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"koboapps/internal/device"
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

func TestSupportsModel(t *testing.T) {
	info := loadShipped(t)
	for _, id := range []string{"00000000-0000-0000-0000-000000000388", "384", "00000000-0000-0000-0000-000000000390"} {
		if !info.SupportsModel(id) {
			t.Errorf("%s must be supported", id)
		}
	}
	for _, id := range []string{"00000000-0000-0000-0000-000000000387", "393", "88", ""} {
		if info.SupportsModel(id) {
			t.Errorf("%s must not be supported", id)
		}
	}
}

// Names in project.json are the same as the model names appsctl prints.
func TestSupportedModelNamesMatchTheDeviceTable(t *testing.T) {
	for _, model := range loadShipped(t).SupportedModels {
		if got := device.ModelName(model.ID); got != model.Name {
			t.Errorf("model %s is %q in project.json, %q in internal/device", model.ID, model.Name, got)
		}
	}
}

func TestLoadRejectsAConfigThatCannotWork(t *testing.T) {
	const models = `"supported_models":[{"id":"388","codename":"io"}]`
	cases := map[string]string{
		"not json":      `{`,
		"no name":       `{"repo":"o/r","supported_firmware":["4.1"],` + models + `}`,
		"bad repo":      `{"name":"X","repo":"nope","supported_firmware":["4.1"],` + models + `}`,
		"no firmwares":  `{"name":"X","repo":"o/r",` + models + `}`,
		"repo with url": `{"name":"X","repo":"https://github.com/o/r","supported_firmware":["4.1"],` + models + `}`,
		"no models":     `{"name":"X","repo":"o/r","supported_firmware":["4.1"]}`,
		"bad model id":  `{"name":"X","repo":"o/r","supported_firmware":["4.1"],"supported_models":[{"id":"38","codename":"io"}]}`,
		"no codename":   `{"name":"X","repo":"o/r","supported_firmware":["4.1"],"supported_models":[{"id":"388"}]}`,
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

// supportedModelPairs reads the {"388", "io"} entries of the mod's SUPPORTED_MODELS table as "388:io".
func supportedModelPairs(t *testing.T, path string) []string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	start := strings.Index(string(data), "SUPPORTED_MODELS[] = {")
	if start < 0 {
		t.Fatalf("%s has no SUPPORTED_MODELS table", path)
	}
	table := string(data)[start+len("SUPPORTED_MODELS[] = {"):]
	table = table[:strings.Index(table, "};")]
	var result []string
	for _, entry := range regexp.MustCompile(`\{"([0-9]{3})",\s*"([^"]+)"\}`).FindAllStringSubmatch(table, -1) {
		result = append(result, entry[1]+":"+entry[2])
	}
	sort.Strings(result)
	return result
}

// The drawer mod compiles its own model table; it must match project.json, or the mod would
// load on a model the docs call unsupported (or refuse one they list).
func TestModListsTheSameModelsAsTheProjectConfig(t *testing.T) {
	var want []string
	for _, model := range loadShipped(t).SupportedModels {
		want = append(want, model.ID+":"+model.Codename)
	}
	sort.Strings(want)
	got := supportedModelPairs(t, "../../mod/common/device_gate.cc")
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("mod/common/device_gate.cc lists %v, project.json lists %v", got, want)
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
	for _, source := range []string{"../../mod/common/device_gate.cc", "../../mod/drawer-proto/device_state.cc"} {
		got := quotedBuilds(t, source)
		if strings.Join(got, ",") != strings.Join(want, ",") {
			t.Errorf("%s lists %v, project.json lists %v", source, got, want)
		}
	}
}
