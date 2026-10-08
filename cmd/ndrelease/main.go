// Command ndrelease is the maintainer's release tool (run on the Mac, never on the Kobo):
// it creates the signing key, signs a dist tree into release.json + signature + zip, and
// verifies such a release. See docs/UPDATES.md.
package main

import (
	"fmt"
	"io"
	"os"
)

const (
	exitOK      = 0
	exitFailure = 1
	exitUsage   = 2

	releaseFilePrefix = "nickeldrawer-"
	defaultPublicKey  = "spec/release-ed25519.pub"
	embeddedPublicKey = "internal/release/release-ed25519.pub"
)

const usage = `usage: ndrelease <command> [flags]

commands:
  keygen   create the signing key (private key stays in ~/.config/nickeldrawer) and write the public key
  sign     --dist dist/onboard --version X.Y.Z   write dist/release/{release.json,release.json.sig,nickeldrawer-X.Y.Z.zip}
  verify   check a dist/release folder against the public key
`

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprint(stderr, usage)
		return exitUsage
	}
	commands := map[string]func([]string, io.Writer, io.Writer) int{
		"keygen": runKeygen,
		"sign":   runSign,
		"verify": runVerify,
	}
	command, found := commands[args[0]]
	if !found {
		fmt.Fprintf(stderr, "unknown command %q\n%s", args[0], usage)
		return exitUsage
	}
	return command(args[1:], stdout, stderr)
}

func fail(stderr io.Writer, err error) int {
	fmt.Fprintln(stderr, "ndrelease:", err)
	return exitFailure
}
