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
  gen-menu [--apps-dir DIR] [--out FILE]   write the NickelMenu config in the device language
  refresh                                  gen-menu with device defaults, short summary
  errors                                   show apps that failed validation
  msg --app ID KEY [NAME=VALUE...]         print an app message in the device language
  device-name                              print the Kobo model name
  host-msg KEY [NAME=VALUE...]             print a host message in the device language
  update check|apply|status|rollback       signed updates from GitHub Releases (see docs/UPDATES.md)
  --version                                print the version

update takes --onboard-root, --api-base, --tmp-dir and --allow-downgrade (apply).
gen-menu, refresh, errors, msg, host-msg and update also take --locale, --conf and --koboapps-dir.
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
		"gen-menu": runGenMenu,
		"refresh":  runRefresh,
		"errors":   runErrors,
		"msg":      runMsg,
		"host-msg": runHostMsg,
		"update":   runUpdate,

		"device-name": runDeviceName,
	}
	command, found := commands[args[0]]
	if !found {
		fmt.Fprintf(stderr, "unknown command %q\n%s", args[0], usage)
		return exitFailure
	}
	return command(args[1:], stdout, stderr)
}
