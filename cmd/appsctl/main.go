// Command appsctl validates koboapps app manifests and generates the NickelMenu config.
package main

import (
	"fmt"
	"io"
	"os"
)

const (
	exitOK      = 0
	exitInvalid = 1 // at least one app manifest is invalid
	exitFailure = 2 // usage error or I/O failure
)

// version is the release this binary belongs to; the Makefile sets it with -ldflags.
var version = "0.0.0"

const usage = `usage: appsctl <command> [flags]

commands:
  validate [--apps-dir DIR] [APP_DIR...]   check app manifests
  msg --app ID KEY [NAME=VALUE...]         print an app message in the device language
  device-name                              print the Kobo model name
  host-msg KEY [NAME=VALUE...]             print a host message in the device language
  inbox route|sort|status|purge            sort received files into Books, Documents, Comics, Pictures (docs/spec/inbox.md)
  new ID [--name N] [--with-ui] [--dir D]  scaffold a new app folder from templates/app
  drawer model|run|screen|event|info|remove|tweaks|tweak  JSON for the native drawer (docs/spec/drawer-protocol.md)
  project update-config                    print the default update.json from host/config/project.json
  update check|apply|status|rollback       signed updates from GitHub Releases (see docs/UPDATES.md)
  --version                                print the version

update takes --onboard-root, --api-base, --tmp-dir and --allow-downgrade (apply).
msg, host-msg and update also take --locale, --conf and --koboapps-dir.
`

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprint(stderr, usage)
		return exitFailure
	}
	if args[0] == "--version" {
		fmt.Fprintln(stdout, "appsctl", version)
		return exitOK
	}
	commands := map[string]func([]string, io.Writer, io.Writer) int{
		"validate": runValidate,
		"msg":      runMsg,
		"host-msg": runHostMsg,
		"update":   runUpdate,
		"inbox":    runInbox,
		"drawer":   runDrawer,
		"project":  runProject,
		"new":      runNew,

		"device-name": runDeviceName,
	}
	command, found := commands[args[0]]
	if !found {
		fmt.Fprintf(stderr, "unknown command %q\n%s", args[0], usage)
		return exitFailure
	}
	return command(args[1:], stdout, stderr)
}
