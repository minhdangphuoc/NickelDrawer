package main

import (
	"flag"
	"fmt"
	"io"

	"koboapps/internal/device"
)

// runDeviceName prints the model name; scripts use it as the device alias, so it always prints something.
func runDeviceName(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("device-name", flag.ContinueOnError)
	flags.SetOutput(stderr)
	versionFile := flags.String("version-file", device.DefaultVersionFile, "Kobo .kobo/version file")
	if err := flags.Parse(args); err != nil {
		return exitFailure
	}
	info, err := device.Read(*versionFile)
	if err != nil {
		fmt.Fprintln(stderr, "warning:", err)
		fmt.Fprintln(stdout, device.FallbackName)
		return exitOK
	}
	fmt.Fprintln(stdout, info.Model)
	return exitOK
}
