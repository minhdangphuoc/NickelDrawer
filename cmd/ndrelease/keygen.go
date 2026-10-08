package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"koboapps/internal/release"
)

func defaultPrivateKeyPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".config", "nickeldrawer", "release-ed25519.key")
	}
	return filepath.Join(home, ".config", "nickeldrawer", "release-ed25519.key")
}

func runKeygen(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("keygen", flag.ContinueOnError)
	flags.SetOutput(stderr)
	keyPath := flags.String("key", defaultPrivateKeyPath(), "where to write the private key (mode 600)")
	publicPaths := publicKeyPaths{}
	flags.StringVar(&publicPaths.spec, "pub", defaultPublicKey, "where to write the public key")
	flags.StringVar(&publicPaths.embedded, "embed-pub", embeddedPublicKey, "copy of the public key that appsctl embeds")
	if err := flags.Parse(args); err != nil {
		return exitUsage
	}
	publicKey, err := generateKeyPair(*keyPath)
	if err != nil {
		return fail(stderr, err)
	}
	if err := publicPaths.write(publicKey); err != nil {
		return fail(stderr, err)
	}
	fmt.Fprintf(stdout, "private key written to %s (mode 600); back it up, it is never printed\n", *keyPath)
	fmt.Fprintf(stdout, "public key written to %s and %s\n", publicPaths.spec, publicPaths.embedded)
	fmt.Fprint(stdout, "public key: ", release.FormatPublicKey(publicKey))
	fmt.Fprintln(stdout, "next: make check build dist, so appsctl embeds the public key")
	return exitOK
}

// generateKeyPair creates the private key file exclusively, so an existing key is never overwritten.
func generateKeyPair(keyPath string) (ed25519.PublicKey, error) {
	if err := os.MkdirAll(filepath.Dir(keyPath), 0o700); err != nil {
		return nil, fmt.Errorf("cannot create %s: %w", filepath.Dir(keyPath), err)
	}
	file, err := os.OpenFile(keyPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if os.IsExist(err) {
		return nil, fmt.Errorf("%s already exists; refusing to overwrite a signing key (move it away on purpose if you really want a new one)", keyPath)
	}
	if err != nil {
		return nil, fmt.Errorf("cannot create %s: %w", keyPath, err)
	}
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err == nil {
		_, err = file.WriteString(release.FormatPrivateKey(privateKey))
	}
	if closeErr := file.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		os.Remove(keyPath)
		return nil, fmt.Errorf("cannot write the private key: %w", err)
	}
	return publicKey, nil
}

type publicKeyPaths struct {
	spec     string // the documented location
	embedded string // the copy go:embed reads
}

func (paths publicKeyPaths) write(publicKey ed25519.PublicKey) error {
	for _, path := range []string{paths.spec, paths.embedded} {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return fmt.Errorf("cannot create %s: %w", filepath.Dir(path), err)
		}
		if err := os.WriteFile(path, []byte(release.FormatPublicKey(publicKey)), 0o644); err != nil {
			return fmt.Errorf("cannot write public key %s: %w", path, err)
		}
	}
	return nil
}
