package release

import (
	"crypto/ed25519"
	_ "embed"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
)

// embeddedPublicKey is a copy of spec/release-ed25519.pub (go:embed cannot reach outside the
// package folder); `make` and `ndrelease keygen` keep the copies equal and a test checks it.
//
//go:embed release-ed25519.pub
var embeddedPublicKey string

// ErrNoPublicKey means no real release key is built into this binary, so updates are refused.
var ErrNoPublicKey = errors.New("no release public key is embedded in this build")

// EmbeddedPublicKey returns the key this build trusts for updates.
func EmbeddedPublicKey() (ed25519.PublicKey, error) {
	return ParsePublicKey(embeddedPublicKey)
}

// ParsePublicKey decodes the base64 text of a raw 32-byte ed25519 public key.
// Empty text or the word "placeholder" gives ErrNoPublicKey.
func ParsePublicKey(text string) (ed25519.PublicKey, error) {
	text = strings.TrimSpace(text)
	if text == "" || strings.HasPrefix(strings.ToLower(text), "placeholder") {
		return nil, ErrNoPublicKey
	}
	raw, err := base64.StdEncoding.DecodeString(text)
	if err != nil || len(raw) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("public key must be base64 of %d raw bytes", ed25519.PublicKeySize)
	}
	return ed25519.PublicKey(raw), nil
}

// FormatPublicKey is the inverse of ParsePublicKey, with a trailing newline.
func FormatPublicKey(key ed25519.PublicKey) string {
	return base64.StdEncoding.EncodeToString(key) + "\n"
}

// ParsePrivateKey decodes the base64 text of a 32-byte ed25519 seed.
func ParsePrivateKey(text string) (ed25519.PrivateKey, error) {
	seed, err := base64.StdEncoding.DecodeString(strings.TrimSpace(text))
	if err != nil || len(seed) != ed25519.SeedSize {
		return nil, fmt.Errorf("private key must be base64 of a %d-byte seed", ed25519.SeedSize)
	}
	return ed25519.NewKeyFromSeed(seed), nil
}

// FormatPrivateKey writes the seed as base64 with a trailing newline.
func FormatPrivateKey(key ed25519.PrivateKey) string {
	return base64.StdEncoding.EncodeToString(key.Seed()) + "\n"
}

// Sign returns the detached signature of the exact bytes, as base64 of the raw 64 bytes.
func Sign(key ed25519.PrivateKey, data []byte) string {
	return base64.StdEncoding.EncodeToString(ed25519.Sign(key, data)) + "\n"
}

// Verify checks a detached base64 signature over the exact bytes.
func Verify(key ed25519.PublicKey, data []byte, signatureText string) error {
	signature, err := base64.StdEncoding.DecodeString(strings.TrimSpace(signatureText))
	if err != nil || len(signature) != ed25519.SignatureSize {
		return errors.New("signature is not base64 of 64 raw bytes")
	}
	if !ed25519.Verify(key, data, signature) {
		return errors.New("signature does not match the release manifest")
	}
	return nil
}
