// Command wallpaper turns pictures from the Pictures folders into the Kobo's sleep screen.
package main

import (
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
}

func (common *commonFlags) register(flags *flag.FlagSet) {
	flags.StringVar(&common.appDir, "app-dir", defaultAppDir, "app folder")
	flags.StringVar(&common.confPath, "conf", i18n.DefaultConfPath, "Kobo eReader.conf holding CurrentLocale")
	flags.StringVar(&common.locale, "locale", "", "language code overriding the device language, e.g. vi")
	flags.StringVar(&common.versionFile, "version-file", device.DefaultVersionFile, "Kobo .kobo/version file (device id decides the screen size)")
	flags.StringVar(&common.lastFile, "last-file", "/tmp/wallpaper-last.txt", "text the last background import printed (state command)")
	flags.StringVar(&common.pidFile, "pid-file", "/tmp/wallpaper.pid", "pid file of a running background import (state command)")
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
		say(out, messages, "msg.error", "message", messages.Describe(err))
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
	return selected.run(&session{
		service: wallpaper.NewService(paths, size, log), messages: messages, out: out,
		stateFiles: wallpaper.ScreenStateFiles{LastOutput: common.lastFile, PidFile: common.pidFile},
	}, arguments)
}
