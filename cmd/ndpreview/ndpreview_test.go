package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const (
	localsendDir  = "../../apps/localsend"
	loopbackAuth  = "127.0.0.1:8790"
	localsendVars = `{"running":true,"status":"Đang nhận","pin":"4821","device":"Kobo Libra 2","ip":"192.168.1.23","received":[["Đắc Nhân Tâm.epub","860 KB"]]}`
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// newAppDir makes <temp>/<id>/ with an app.json (one action "go") and its script; the folder
// name must equal the app id.
func newAppDir(t *testing.T, id, script string) string {
	t.Helper()
	appDir := filepath.Join(t.TempDir(), id)
	writeFile(t, filepath.Join(appDir, "app.json"), `{"schema":1,"id":"`+id+`","name":"`+id+`","version":"1.0.0","actions":[{"id":"go","label":"Go","run":"bin/go.sh","output":"show"}]}`)
	scriptPath := filepath.Join(appDir, "bin", "go.sh")
	writeFile(t, scriptPath, script)
	if err := os.Chmod(scriptPath, 0o755); err != nil {
		t.Fatal(err)
	}
	return appDir
}

func newTestServer(t *testing.T, config serverConfig) *server {
	t.Helper()
	instance, err := newServer(config)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(instance.close)
	return instance
}

func get(t *testing.T, instance *server, target string) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(http.MethodGet, target, nil)
	request.Host = loopbackAuth
	recorder := httptest.NewRecorder()
	instance.handler().ServeHTTP(recorder, request)
	return recorder
}

func localsendServer(t *testing.T) *server {
	t.Helper()
	statePath := filepath.Join(t.TempDir(), "state.json")
	writeFile(t, statePath, localsendVars)
	return newTestServer(t, serverConfig{AppDir: localsendDir, Locale: "vi", StatePath: statePath})
}

func TestServeBindsLoopbackOnly(t *testing.T) {
	listener, err := listenLoopback(0)
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	if err := isLoopbackAddress(listener.Addr()); err != nil || !strings.HasPrefix(listener.Addr().String(), "127.0.0.1:") {
		t.Fatalf("listener %s is not on 127.0.0.1", listener.Addr())
	}
}

func TestPageRendersLocalsendInVietnamese(t *testing.T) {
	recorder := get(t, localsendServer(t), "/")
	body := recorder.Body.String()
	if recorder.Code != http.StatusOK {
		t.Fatalf("status %d: %s", recorder.Code, body)
	}
	for name, want := range map[string]string{
		"title":          "LocalSend",
		"vi status":      "Đang nhận",
		"bigcode PIN":    ">4821<",
		"kv key":         "Thiết bị",
		"kv value":       "Kobo Libra 2",
		"vi notice":      "Để màn hình này mở khi đang nhận tệp.",
		"received row":   "Đắc Nhân Tâm.epub",
		"primary button": "Dừng và cập nhật thư viện",
		"stop action":    `&#34;name&#34;:&#34;stop&#34;`,
		"device size":    "width:1264px;height:1680px",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("%s: page lacks %q", name, want)
		}
	}
	if strings.Contains(body, "Stop and update library") {
		t.Error("English text leaked into the vi page")
	}
}

// localsendButtons lays out the LocalSend screen and returns its two footer buttons' boxes.
func localsendButtons(t *testing.T, isRunning bool) (start, stop Item) {
	t.Helper()
	rc := renderContext{doc: mustLoad(t).doc, locale: "en", vars: map[string]any{"running": isRunning}}
	page, err := layoutScreen(rc, "main", 0, Overlay{})
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range page.Items {
		switch {
		case item.Shape == "rect" && item.Label == "Start receiving":
			start = item
		case item.Shape == "rect" && item.Label == "Stop and update library":
			stop = item
		}
	}
	if start.W == 0 || stop.W == 0 {
		t.Fatalf("buttons not found in %d items", len(page.Items))
	}
	return start, stop
}

// A disabled button is dimmed (outline and label) and has no event: no press, taps ignored.
func TestLocalsendButtonsFollowRunningState(t *testing.T) {
	for _, isRunning := range []bool{true, false} {
		start, stop := localsendButtons(t, isRunning)
		enabled, disabled := start, stop
		if isRunning {
			enabled, disabled = stop, start
		}
		if enabled.Event == "" || enabled.Ink != inkBlack {
			t.Errorf("running=%v: enabled button %q has event %q, ink %d", isRunning, enabled.Label, enabled.Event, enabled.Ink)
		}
		if disabled.Event != "" || disabled.Confirm != "" || disabled.Ink != inkDim || disabled.Fill != noFill {
			t.Errorf("running=%v: disabled button %q has event %q, ink %d, fill %d", isRunning, disabled.Label, disabled.Event, disabled.Ink, disabled.Fill)
		}
	}
}

// The header matches the device's PageHeader: sans title at Nickel's header size, the status
// in the same 120 px row at the right, a 1 px divider under it, the body below.
func TestHeaderMatchesNickelPageHeader(t *testing.T) {
	rc := renderContext{doc: mustLoad(t).doc, locale: "en", vars: map[string]any{"status": "Receiving"}}
	screen := rc.doc.Screens["main"]
	items := rc.headerItems(screen)
	byText := map[string]Item{}
	for _, item := range items {
		if len(item.Lines) == 1 {
			byText[item.Lines[0]] = item
		}
	}
	title, status := byText["LocalSend"], byText["Receiving"]
	if title.Family != familySans || title.Size != headerTitleSz || title.Bold || title.Y != 0 || title.H != headerRowHeight {
		t.Errorf("title %+v", title)
	}
	if status.Y != 0 || status.H != headerRowHeight || status.X+status.W != screenWidth-gutter || status.Align != "end" {
		t.Errorf("status not at the right of the header row: %+v", status)
	}
	divider := items[len(items)-1]
	if divider.Y != headerRowHeight || divider.H != headerDivider || divider.W != contentWidth {
		t.Errorf("divider %+v", divider)
	}
}

func TestPageUsesEnglishFallbackAndScaleToggle(t *testing.T) {
	instance := localsendServer(t)
	english := get(t, instance, "/?locale=en&scale=100").Body.String()
	if !strings.Contains(english, "Stop and update library") || !strings.Contains(english, "transform:scale(1)") {
		t.Error("en page at 100% is wrong")
	}
	if half := get(t, instance, "/").Body.String(); !strings.Contains(half, "transform:scale(0.5)") {
		t.Error("default scale is not 50%")
	}
	if grey := get(t, instance, "/?grey=16").Body.String(); !strings.Contains(grey, "filter:url(#q16)") {
		t.Error("16-level grey filter missing")
	}
}

func TestHostGuardRefusesNonLoopbackHosts(t *testing.T) {
	instance := localsendServer(t)
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Host = "evil.example:8790"
	recorder := httptest.NewRecorder()
	instance.handler().ServeHTTP(recorder, request)
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("status %d, want 403", recorder.Code)
	}
	post := httptest.NewRequest(http.MethodPost, "/api/event", strings.NewReader(`{}`))
	post.Host = loopbackAuth
	post.Header.Set("Origin", "http://evil.example")
	recorder = httptest.NewRecorder()
	instance.handler().ServeHTTP(recorder, post)
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("cross-origin status %d, want 403", recorder.Code)
	}
}

// A long list pages: the page keys map to the arrows, and the last page shows the last rows.
func TestLongListPages(t *testing.T) {
	appDir := newAppDir(t, "pager", "echo hi\n")
	var items []string
	for i := 1; i <= 40; i++ {
		items = append(items, fmt.Sprintf(`["File %02d.epub","1 MB"]`, i))
	}
	writeFile(t, filepath.Join(appDir, "ui.json"), `{"kit":1,"start":"main","vars":{"files":[`+strings.Join(items, ",")+`]},
	"screens":{"main":{"title":"Files","body":[{"list":"$files","title":"Files"}],"footer":[{"button":"Go","action":"go","style":"primary"}]}}}`)
	instance := newTestServer(t, serverConfig{AppDir: appDir, Locale: "en"})

	first := screenOnly(get(t, instance, "/").Body.String())
	if !strings.Contains(first, "File 01.epub") || strings.Contains(first, "File 40.epub") {
		t.Error("page 1 should hold the first rows only")
	}
	if !strings.Contains(first, `id="next"`) || strings.Contains(first, `id="prev"`) || !strings.Contains(first, "1 / 4") {
		t.Error("page 1 arrows or indicator wrong")
	}
	last := screenOnly(get(t, instance, "/?page=3").Body.String())
	if !strings.Contains(last, "File 40.epub") || !strings.Contains(last, `id="prev"`) || strings.Contains(last, `id="next"`) {
		t.Error("last page should hold the last rows with only a previous arrow")
	}
	if beyond := get(t, instance, "/?page=99").Body.String(); !strings.Contains(beyond, "4 / 4") {
		t.Error("a page number past the end is clamped to the last page")
	}
}

// screenOnly keeps the pager and the device screen, dropping the sidebar (which lists all vars).
func screenOnly(page string) string {
	return page[strings.Index(page, "<main>"):]
}

func TestInvalidUIShowsReadableError(t *testing.T) {
	appDir := newAppDir(t, "broken", "echo hi\n")
	writeFile(t, filepath.Join(appDir, "ui.json"), `{"kit":1,"start":"main","screens":{"main":{"title":"X","body":[{"carousel":"x"}]}}}`)
	recorder := get(t, newTestServer(t, serverConfig{AppDir: appDir, Locale: "en"}), "/")
	if recorder.Code != http.StatusUnprocessableEntity || !strings.Contains(recorder.Body.String(), "ui.unknown_component") {
		t.Fatalf("status %d: %s", recorder.Code, recorder.Body.String())
	}
}

func postEvent(t *testing.T, instance *server, line, action string) (int, map[string]any) {
	t.Helper()
	body, _ := json.Marshal(eventRequest{Line: line, Action: action})
	request := httptest.NewRequest(http.MethodPost, "/api/event", bytes.NewReader(body))
	request.Host = loopbackAuth
	recorder := httptest.NewRecorder()
	instance.handler().ServeHTTP(recorder, request)
	var response map[string]any
	_ = json.Unmarshal(recorder.Body.Bytes(), &response)
	return recorder.Code, response
}

func TestTapShowsTheActionThatWouldRun(t *testing.T) {
	instance := localsendServer(t)
	code, response := postEvent(t, instance, `{"event":"action","name":"stop"}`, "")
	action, _ := response["action"].(map[string]any)
	if code != http.StatusOK || action["id"] != "stop" || action["run"] != "bin/stop.sh" {
		t.Fatalf("status %d response %v", code, response)
	}
	if _, ran := response["exec"]; ran {
		t.Error("without --exec nothing may run")
	}
	if code, _ := postEvent(t, instance, `{"event":"action","name":"fly"}`, ""); code != http.StatusBadRequest {
		t.Errorf("unknown action status %d, want 400", code)
	}
}

func TestToggleUpdatesState(t *testing.T) {
	instance := localsendServer(t)
	_, response := postEvent(t, instance, `{"event":"toggle","bind":"status","value":"Dừng"}`, "")
	if response["reload"] != true {
		t.Fatalf("toggle should ask for a reload: %v", response)
	}
	if body := get(t, instance, "/").Body.String(); !strings.Contains(body, "Dừng") {
		t.Error("state change not rendered")
	}
}

func TestExecRunsScriptInTemporaryRoot(t *testing.T) {
	appDir := newAppDir(t, "runner", "echo ran in $KOBO_ROOT\npwd\n")
	writeFile(t, filepath.Join(appDir, "ui.json"), `{"kit":1,"start":"main","screens":{"main":{"title":"R","footer":[{"button":"Go","action":"go","style":"primary"}]}}}`)
	instance := newTestServer(t, serverConfig{AppDir: appDir, Locale: "en", ExecEnabled: true})

	_, response := postEvent(t, instance, `{"event":"action","name":"go"}`, "")
	result, _ := response["exec"].(map[string]any)
	output, _ := result["output"].(string)
	root, _ := result["root"].(string)
	if result["exit"] != float64(0) || !strings.Contains(output, "ran in "+root) || root == "" {
		t.Fatalf("exec result %v", result)
	}
	if strings.HasPrefix(root, "/Volumes/") {
		t.Fatal("the fake root must be a temp folder")
	}
}

func TestRenderWritesFullScreenPNG(t *testing.T) {
	out := filepath.Join(t.TempDir(), "screen.png")
	statePath := filepath.Join(t.TempDir(), "state.json")
	writeFile(t, statePath, localsendVars)
	var stdout, stderr bytes.Buffer
	code := run([]string{"render", localsendDir, "--out", out, "--locale", "vi", "--state", statePath, "--grey", "16"}, &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit %d: %s", code, stderr.String())
	}
	file, err := os.Open(out)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	config, err := png.DecodeConfig(file)
	if err != nil || config.Width != 1264 || config.Height != 1680 {
		t.Fatalf("PNG is %dx%d (err %v), want 1264x1680", config.Width, config.Height, err)
	}
}

func TestRenderWithoutFontStillWritesShapes(t *testing.T) {
	fonts := &fontSet{}
	rc := renderContext{doc: mustLoad(t).doc, locale: "en", vars: map[string]any{"pin": "1234"}}
	page, err := layoutScreen(rc, "main", 0, Overlay{})
	if err != nil {
		t.Fatal(err)
	}
	canvas, err := renderPNG(page, fonts, 0)
	if err != nil || canvas.Bounds().Dx() != screenWidth || canvas.Bounds().Dy() != screenHeight {
		t.Fatalf("canvas %v err %v", canvas.Bounds(), err)
	}
}

func mustLoad(t *testing.T) loadedApp {
	t.Helper()
	loaded, err := loadApp(localsendDir, "en")
	if err != nil {
		t.Fatal(err)
	}
	return loaded
}

func TestRenderRejectsUnknownScreen(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run([]string{"render", localsendDir, "--out", filepath.Join(t.TempDir(), "x.png"), "--screen", "nope"}, &stdout, &stderr)
	if code != exitInvalid || !strings.Contains(stderr.String(), `screen "nope" does not exist`) {
		t.Fatalf("exit %d: %s", code, stderr.String())
	}
}

func TestOverlaysRender(t *testing.T) {
	instance := newTestServer(t, serverConfig{AppDir: "../../apps/todo", Locale: "vi"})
	body := get(t, instance, "/?confirm=screens.main.footer[0]").Body.String()
	if !strings.Contains(body, "Xóa mọi việc đã đánh dấu xong?") || !strings.Contains(body, "Hủy") {
		t.Error("confirm dialog not rendered in vi")
	}
}

func TestShippedExamplesRenderEveryScreen(t *testing.T) {
	for _, dir := range []string{"../../apps/localsend", "../../apps/wallpaper", "../../apps/todo"} {
		instance := newTestServer(t, serverConfig{AppDir: dir, Locale: "vi"})
		if recorder := get(t, instance, "/"); recorder.Code != http.StatusOK {
			t.Errorf("%s: status %d: %s", dir, recorder.Code, recorder.Body.String())
		}
	}
}
