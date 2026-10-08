package ghsync

import (
	"os"
	"path/filepath"
	"strings"

	"koboapps/internal/i18n"
)

// Token is the GitHub PAT. Its String method never reveals the value.
type Token string

func (Token) String() string { return "***" }

// LoadToken reads the token file. warning is non-empty when file modes cannot
// be enforced (vfat) so the caller can log it once.
func LoadToken(tokenPath string) (token Token, warning string, err error) {
	info, err := os.Stat(tokenPath)
	if err != nil {
		return "", "", i18n.NewError("msg.err_token_missing", "path", tokenPath)
	}
	if info.Mode().Perm()&0o077 != 0 {
		canEnforce, probeErr := filesystemEnforcesModes(filepath.Dir(tokenPath))
		if probeErr != nil {
			return "", "", probeErr
		}
		if canEnforce {
			return "", "", i18n.NewError("msg.err_token_permissions", "path", tokenPath)
		}
		warning = "filesystem cannot enforce file modes (vfat); token permissions not checked"
	}
	raw, err := os.ReadFile(tokenPath)
	if err != nil {
		return "", "", i18n.NewError("msg.err_token_unreadable").WithCause(err)
	}
	value := strings.TrimSpace(string(raw))
	if value == "" {
		return "", "", i18n.NewError("msg.err_token_empty", "path", tokenPath)
	}
	return Token(value), warning, nil
}

// filesystemEnforcesModes checks whether chmod 600 sticks in dir, using a
// probe file so the token's own mode is never modified.
func filesystemEnforcesModes(dir string) (bool, error) {
	probe, err := os.CreateTemp(dir, ".ghsync-probe-*")
	if err != nil {
		return false, i18n.NewError("msg.err_data_dir_unwritable").WithCause(err)
	}
	defer os.Remove(probe.Name())
	defer probe.Close()
	if err := probe.Chmod(0o600); err != nil {
		return false, nil
	}
	info, err := probe.Stat()
	if err != nil {
		return false, err
	}
	return info.Mode().Perm()&0o077 == 0, nil
}
