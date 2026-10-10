// Command ndrelease is the maintainer's release tool (run on the Mac, never on the Kobo):
// it creates the signing key, packs the per-app zips, signs a dist tree (host + mod) and the app
// zips into release.json + signature + zip, and verifies such a release. See docs/UPDATES.md.
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
  pack-apps  --tree build/apps-tree --out dist/apps   write one deterministic app-<id>-<version>.zip per app of the tree
  verify   check a dist/release folder against the public key
  sign-apps    --apps-dir dist/apps --sequence N [--only ID...]   write dist/apps-release/{apps.json,apps.json.sig} (+ the zips to upload)
  verify-apps  check dist/apps-release (and, with --apps-dir, every catalog entry against its zip)
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
		"keygen":    runKeygen,
		"sign":      runSign,
		"pack-apps": runPackApps,
		"verify":    runVerify,

		"sign-apps":   runSignApps,
		"verify-apps": runVerifyApps,
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
