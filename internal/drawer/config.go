// Package drawer builds the data behind the native drawer and runs its actions
// (protocol: docs/spec/drawer-protocol.md). It has no UI; cmd/appsctl prints its results as JSON.
package drawer

import (
	"net"
	"os"
	"time"

	"koboapps/internal/i18n"
)

// ProtocolVersion is the "v" field of every response.
const ProtocolVersion = 1

const (
	defaultStatusTimeout = time.Second
	defaultCacheTTL      = 30 * time.Second
	defaultShowTimeout   = 30 * time.Second
)

// Config holds everything the drawer needs; the zero values of the optional fields are the
// device defaults, tests inject the rest.
type Config struct {
	AppsDir string // folder with one subfolder per app
	HostDir string // koboapps folder (bin/, data/)
	Locale  string // language for app strings, e.g. "vi"
	Bundle  *i18n.Bundle

	CacheDir      string        // where the status cache lives (default os.TempDir())
	CacheTTL      time.Duration // default 30 s
	StatusTimeout time.Duration // per app, default 1 s
	ShowTimeout   time.Duration // cap for actions of kind show, default 30 s

	RunScript   ScriptRunner       // default: execute with a minimal environment
	StartScript func(string) error // default: detached start
	HasIP       func() bool        // default: a non-loopback IPv4 address exists
	Now         func() time.Time   // default time.Now
}

func (c Config) message(key string, params map[string]string) string {
	return c.Bundle.Message(key, params)
}

func (c Config) statusTimeout() time.Duration {
	return durationOr(c.StatusTimeout, defaultStatusTimeout)
}

func (c Config) cacheTTL() time.Duration { return durationOr(c.CacheTTL, defaultCacheTTL) }

func (c Config) showTimeout() time.Duration { return durationOr(c.ShowTimeout, defaultShowTimeout) }

func (c Config) cacheDir() string {
	if c.CacheDir != "" {
		return c.CacheDir
	}
	return os.TempDir()
}

func (c Config) now() time.Time {
	if c.Now != nil {
		return c.Now()
	}
	return time.Now()
}

func (c Config) runScript() ScriptRunner {
	if c.RunScript != nil {
		return c.RunScript
	}
	return ExecScript
}

func (c Config) startScript() func(string) error {
	if c.StartScript != nil {
		return c.StartScript
	}
	return StartDetached
}

func (c Config) hasIP() bool {
	if c.HasIP != nil {
		return c.HasIP()
	}
	return HasNonLoopbackIP()
}

func durationOr(value, fallback time.Duration) time.Duration {
	if value > 0 {
		return value
	}
	return fallback
}

// HasNonLoopbackIP mirrors current_ip in host/lib/common.sh: any up interface with an IPv4
// address other than 127.0.0.1.
func HasNonLoopbackIP() bool {
	addresses, err := net.InterfaceAddrs()
	if err != nil {
		return false
	}
	for _, address := range addresses {
		network, isNetwork := address.(*net.IPNet)
		if !isNetwork || network.IP.IsLoopback() {
			continue
		}
		if network.IP.To4() != nil {
			return true
		}
	}
	return false
}
