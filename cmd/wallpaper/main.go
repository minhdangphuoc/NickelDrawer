// Command wallpaper turns pictures from the Pictures folders into the Kobo's sleep screen.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"koboapps/internal/device"
	"koboapps/internal/i18n"
	"koboapps/internal/wallpaper"
)

const (
	defaultAppDir = "/mnt/onboard/.adds/apps/wallpaper"
	exitUsage     = 2
	exitNotDue    = 1 // first-import-due: nothing to import now
)

func main() {
	os.Exit(execute(os.Args[1:], os.Stdout))
}

type commonFlags struct {
	appDir      string
	confPath    string
	locale      string
	versionFile string
	lastFile    string
	pidFile     string
	thumbsPid   string

	progressFile   string
	confirm        bool
	deleteOriginal bool
	saveOnly       bool
	signal         bool
}

func (common *commonFlags) register(flags *flag.FlagSet) {
	flags.StringVar(&common.appDir, "app-dir", defaultAppDir, "app folder")
	flags.StringVar(&common.confPath, "conf", i18n.DefaultConfPath, "Kobo eReader.conf holding CurrentLocale")
	flags.StringVar(&common.locale, "locale", "", "language code overriding the device language, e.g. vi")
	flags.StringVar(&common.versionFile, "version-file", device.DefaultVersionFile, "Kobo .kobo/version file (device id decides the screen size)")
	flags.StringVar(&common.lastFile, "last-file", "/tmp/wallpaper-last.txt", "text the last background import printed (state command)")
	flags.StringVar(&common.pidFile, "pid-file", "/tmp/wallpaper.pid", "pid file of a running background import (state and thumbnails commands)")
	flags.StringVar(&common.thumbsPid, "thumbs-pid-file", defaultThumbsPidFile, "pid file of the background thumbnail job (state and thumbnails commands)")
	flags.StringVar(&common.progressFile, "progress-file", "/tmp/wallpaper-progress.json", "progress of a running import or re-fit (import, mode, state commands)")
	flags.BoolVar(&common.confirm, "confirm", false, "remove: the user confirmed")
	flags.BoolVar(&common.deleteOriginal, "delete-original", false, "remove: also delete the original in Pictures when it is the same file")
	flags.BoolVar(&common.signal, "signal", false, "remove: print the drawer toast id of the outcome instead of a sentence")
	flags.BoolVar(&common.saveOnly, "save-only", false, "mode: only store the mode, do not re-fit now (a running job re-fits when it ends)")
}

func (common commonFlags) messages() *i18n.Bundle {
	locale := common.locale
	if locale == "" {
		locale = i18n.CurrentLocale(common.confPath)
	}
	bundle, err := i18n.LoadBundle(filepath.Join(common.appDir, "locales"), locale)
	if err != nil {
		fmt.Fprintln(os.Stderr, "warning:", err)
	}
	return bundle
}

func say(out io.Writer, bundle *i18n.Bundle, key string, pairs ...string) {
	fmt.Fprintln(out, bundle.Message(key, i18n.Params(pairs...)))
}

// command is one subcommand; run gets the positional arguments after the flags.
type command struct {
	arguments   int  // required positional arguments
	needsScreen bool // the command converts pictures, so it must know the screen size
	run         func(session *session, arguments []string) error
}

var commands = map[string]command{
	"import":   {0, true, (*session).importPictures},
	"apply":    {0, false, (*session).apply},
	"next":     {0, false, (*session).next},
	"mode":     {1, true, (*session).setMode},
	"rotation": {1, false, (*session).setRotation},
	"off":      {0, false, (*session).turnOff},
	"status":   {0, false, (*session).status},
	"state":    {0, false, (*session).state},

	"thumbnails": {0, false, (*session).makeThumbnails},

	"first-import-due": {0, false, (*session).firstImportDue},

	"set-current": {1, false, (*session).setCurrent},
	"remove":      {1, false, (*session).remove},
}

func execute(args []string, out io.Writer) int {
	if len(args) == 0 {
		say(out, commonFlags{appDir: defaultAppDir, confPath: i18n.DefaultConfPath}.messages(), "msg.usage")
		return exitUsage
	}
	selected, known := commands[args[0]]
	flags := flag.NewFlagSet(args[0], flag.ContinueOnError)
	flags.SetOutput(out)
	var common commonFlags
	common.register(flags)
	if !known {
		say(out, common.messages(), "msg.unknown_command", "command", args[0])
		return exitUsage
	}
	if err := flags.Parse(args[1:]); err != nil {
		return exitUsage
	}
	messages := common.messages()
	if flags.NArg() != selected.arguments {
		say(out, messages, "msg.usage")
		return exitUsage
	}
	if err := runSession(selected, common, messages, out, flags.Args()); err != nil {
		if errors.Is(err, errNotDue) {
			return exitNotDue
		}
		say(out, messages, "msg.error", "message", messages.Describe(err))
		if args[0] == "first-import-due" {
			return exitUsage
		}
		return 1
	}
	return 0
}

func runSession(selected command, common commonFlags, messages *i18n.Bundle, out io.Writer, arguments []string) error {
	paths := wallpaper.Paths{AppDir: common.appDir}
	if err := os.MkdirAll(paths.Data(), 0o755); err != nil {
		return i18n.NewError("msg.err_mkdir", "path", paths.Data()).WithCause(err)
	}
	log, err := wallpaper.NewLogFile(paths.Log())
	if err != nil {
		return i18n.NewError("msg.err_log_unopenable").WithCause(err)
	}
	var size wallpaper.Size
	if selected.needsScreen {
		if size, err = wallpaper.ResolutionFromVersionFile(common.versionFile); err != nil {
			return err
		}
	}
	service := wallpaper.NewService(paths, size, log)
	service.ProgressFile = common.progressFile
	return selected.run(&session{
		service: service, messages: messages, out: out, flags: common,
		confPath: common.confPath, stateFiles: wallpaper.ScreenStateFiles{LastOutput: common.lastFile, PidFile: common.pidFile, ProgressFile: common.progressFile, ConfPath: common.confPath},
	}, arguments)
}
