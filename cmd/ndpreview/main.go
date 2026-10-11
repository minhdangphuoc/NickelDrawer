// Command ndpreview previews NickelDrawer app screens (ui.json) on a Mac or Linux computer.
//
//	ndpreview serve <app-dir> [--locale vi] [--state state.json] [--port 8790] [--exec]
//	ndpreview render <app-dir> --out screen.png [--screen id] [--locale vi] [--state state.json]
//
// serve answers on 127.0.0.1 only. See docs/APPS.md, "Design a native screen".
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"time"
)

const (
	exitOK      = 0
	exitInvalid = 1 // the app or its ui.json cannot be previewed
	exitFailure = 2 // usage error or I/O failure
)

const usage = `usage: ndpreview <command> <app-dir> [flags]

commands:
  serve   web page with every screen at 1264x1680 (shown at 50%, toggle 100%); 127.0.0.1 only
          --locale en|vi  --state FILE  --port N (default 8790)  --exec
  render  headless PNG of one screen
          --out FILE (required)  --screen ID  --locale en|vi  --state FILE  --page N  --grey 16
          --value ID (screen value, e.g. the picture a viewer opens on)  --chrome shown|hidden (viewer)
          --font FILE  --font-serif FILE  --font-mono FILE  --icons DIR

  drawer-grid  headless PNG of the drawer's grid view with sample apps
  drawer-list  headless PNG of the drawer's list view with sample apps (--problems, --qt6)
  drawer-empty headless PNG of the drawer's empty state (--loading for the Loading state)
  extras-tweaks headless PNG of the Extras helper's "Tweaks" modal
          --out FILE (required)  --apps DIR (icons)  --locale en|vi  --count N  --installed (real apps of --apps)  --font FILE  --icons DIR

Every command lays out in the real page area (1264x1426 below Nickel's 128 px status bar, a 126 px
tab bar under it; both drawn as plain strips) and draws icons from the repository's Tabler PNGs
(host/icons/ui, found by walking up from the working directory; --icons DIR overrides).

State files: with --locale en, --state preview-state.json uses preview-state.en.json when that file
exists next to it (English sample data); other locales use the file as given.

--exec runs the app's action scripts on this computer inside a temporary folder when you tap
a button. Without a font file render draws shapes only and prints a warning.
`

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprint(stderr, usage)
		return exitFailure
	}
	switch args[0] {
	case "serve":
		return runServe(args[1:], stdout, stderr)
	case "render":
		return runRender(args[1:], stdout, stderr)
	case "drawer-grid":
		return runDrawerGrid(args[1:], stdout, stderr)
	case "drawer-list":
		return runDrawerList(args[1:], stdout, stderr)
	case "drawer-empty":
		return runDrawerEmpty(args[1:], stdout, stderr)
	case "extras-tweaks":
		return runExtrasTweaks(args[1:], stdout, stderr)
	case "-h", "--help", "help":
		fmt.Fprint(stdout, usage)
		return exitOK
	}
	fmt.Fprintf(stderr, "ndpreview: unknown command %q\n\n%s", args[0], usage)
	return exitFailure
}

// parseInterspersed lets flags come before or after the positional app folder.
func parseInterspersed(flags *flag.FlagSet, args []string) ([]string, error) {
	var positional []string
	for {
		if err := flags.Parse(args); err != nil {
			return nil, err
		}
		if flags.NArg() == 0 {
			return positional, nil
		}
		positional = append(positional, flags.Arg(0))
		args = flags.Args()[1:]
	}
}

func appDirArgument(positional []string, stderr io.Writer) (string, bool) {
	if len(positional) != 1 {
		fmt.Fprintf(stderr, "ndpreview: give exactly one app folder, for example apps/localsend\n")
		return "", false
	}
	return positional[0], true
}

func runServe(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("serve", flag.ContinueOnError)
	flags.SetOutput(stderr)
	locale := flags.String("locale", "en", "language of the preview (en or vi)")
	statePath := flags.String("state", "", "JSON file with variable values")
	iconsDir := flags.String("icons", "", "folder with the Tabler PNGs (default: host/icons/ui of the repository)")
	port := flags.Int("port", defaultPort, "port on 127.0.0.1")
	isExec := flags.Bool("exec", false, "run the app's action scripts when a button is tapped")
	positional, err := parseInterspersed(flags, args)
	if err != nil {
		return exitFailure
	}
	appDir, ok := appDirArgument(positional, stderr)
	if !ok {
		return exitFailure
	}
	setIconDir(*iconsDir)
	fonts, err := loadFonts("", "", "") // system fonts: the layout measures with what the PNG preview draws with
	if err != nil {
		fmt.Fprintln(stderr, "ndpreview:", err)
		return exitFailure
	}
	useFonts(fonts, stderr)
	instance, err := newServer(serverConfig{AppDir: appDir, Locale: *locale, StatePath: localizedStatePath(*statePath, *locale), ExecEnabled: *isExec})
	if err != nil {
		fmt.Fprintln(stderr, "ndpreview:", err)
		return exitFailure
	}
	defer instance.close()
	return serveUntilInterrupted(instance, *port, stdout, stderr)
}

func serveUntilInterrupted(instance *server, port int, stdout, stderr io.Writer) int {
	listener, err := listenLoopback(port)
	if err != nil {
		fmt.Fprintln(stderr, "ndpreview:", err)
		return exitFailure
	}
	httpServer := &http.Server{Handler: instance.handler(), ReadHeaderTimeout: 10 * time.Second}
	fmt.Fprintf(stdout, "ndpreview: %s at http://%s/ (Ctrl-C to stop)\n", instance.config.AppDir, listener.Addr())
	interrupts := make(chan os.Signal, 1)
	signal.Notify(interrupts, os.Interrupt)
	go func() {
		<-interrupts
		httpServer.Close()
	}()
	if err := httpServer.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
		fmt.Fprintln(stderr, "ndpreview:", err)
		return exitFailure
	}
	return exitOK
}

func runRender(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("render", flag.ContinueOnError)
	flags.SetOutput(stderr)
	options := renderOptions{}
	flags.StringVar(&options.OutPath, "out", "", "PNG file to write (required)")
	flags.StringVar(&options.Screen, "screen", "", "screen id (default: the start screen)")
	flags.StringVar(&options.Locale, "locale", "en", "language of the preview (en or vi)")
	flags.StringVar(&options.StatePath, "state", "", "JSON file with variable values")
	flags.StringVar(&options.IconsDir, "icons", "", "folder with the Tabler PNGs (default: host/icons/ui of the repository)")
	flags.Func("value", "screen value, e.g. the picture id a viewer opens on (sets the variable selected)", func(text string) error {
		options.Value = &text
		return nil
	})
	flags.StringVar(&options.Chrome, "chrome", chromeShown, "viewer chrome: shown or hidden")
	flags.IntVar(&options.Page, "page", 1, "page number of a long screen (1 is the first)")
	flags.IntVar(&options.Grey, "grey", greyLevelsOff, "16 snaps the picture to 16 grey levels")
	flags.StringVar(&options.FontSans, "font", "", "TrueType or OpenType file for text (default: a system font if found)")
	flags.StringVar(&options.FontSerif, "font-serif", "", "font for titles (default: the system serif or --font)")
	flags.StringVar(&options.FontMono, "font-mono", "", "font for codes (default: the system mono or --font)")
	positional, err := parseInterspersed(flags, args)
	if err != nil {
		return exitFailure
	}
	appDir, ok := appDirArgument(positional, stderr)
	if !ok || options.OutPath == "" {
		fmt.Fprintln(stderr, "ndpreview render needs an app folder and --out FILE")
		return exitFailure
	}
	if options.Chrome != chromeShown && options.Chrome != chromeHidden {
		fmt.Fprintf(stderr, "ndpreview: --chrome must be %s or %s, got %q\n", chromeShown, chromeHidden, options.Chrome)
		return exitFailure
	}
	options.AppDir = appDir
	return renderToFile(options, stdout, stderr)
}
