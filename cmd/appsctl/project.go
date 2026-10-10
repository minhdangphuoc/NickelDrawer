package main

import (
	"flag"
	"fmt"
	"io"

	"koboapps/internal/project"
)

const projectUsage = "usage: appsctl project update-config [--file host/config/project.json]"

// runProject serves build-time uses of the project config: `update-config` prints the default
// data/update.json that `make dist` ships, so the repo is named in project.json only.
func runProject(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 || args[0] != "update-config" {
		fmt.Fprintln(stderr, projectUsage)
		return exitFailure
	}
	flags := flag.NewFlagSet("project update-config", flag.ContinueOnError)
	flags.SetOutput(stderr)
	file := flags.String("file", "host/config/project.json", "project config to read")
	if err := flags.Parse(args[1:]); err != nil {
		return exitFailure
	}
	info, err := project.Load(*file)
	if err != nil {
		fmt.Fprintln(stderr, "appsctl:", err)
		return exitFailure
	}
	if _, err := stdout.Write(info.UpdateConfigJSON()); err != nil {
		fmt.Fprintln(stderr, "appsctl: cannot write the update config:", err)
		return exitFailure
	}
	return exitOK
}
