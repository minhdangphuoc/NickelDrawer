package main

import (
	"flag"
	"fmt"
	"io"
)

// runHostMsg prints one host message in the device language, for the scripts in koboapps/bin.
// Like msg it never fails over a string: an unknown key prints the key.
func runHostMsg(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("host-msg", flag.ContinueOnError)
	flags.SetOutput(stderr)
	var localeOptions localeFlags
	localeOptions.register(flags)
	if err := flags.Parse(args); err != nil {
		return exitFailure
	}
	if flags.NArg() < 1 {
		fmt.Fprintln(stderr, "usage: appsctl host-msg <key> [name=value ...]")
		return exitFailure
	}
	bundle, err := localeOptions.hostBundle()
	if err != nil {
		fmt.Fprintln(stderr, "warning:", err)
	}
	fmt.Fprintln(stdout, bundle.Message(flags.Arg(0), parseParams(flags.Args()[1:], stderr)))
	return exitOK
}
