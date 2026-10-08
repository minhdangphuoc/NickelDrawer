package ghsync

import (
	"encoding/json"
	"os"
	"path"
	"regexp"
	"strings"

	"koboapps/internal/i18n"
)

const (
	defaultDest = "/mnt/onboard/GitHub"
)

var (
	defaultExtensions = []string{".epub", ".kepub.epub", ".pdf", ".cbz", ".cbr"}
	repoPattern       = regexp.MustCompile(`^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$`)
)

// Config is data/config.json.
type Config struct {
	Repo       string   `json:"repo"`
	Branch     string   `json:"branch"`
	Path       string   `json:"path"`
	Dest       string   `json:"dest"`
	Extensions []string `json:"extensions"`
}

// LoadConfig reads, defaults and validates the config file.
func LoadConfig(configPath string) (Config, error) {
	raw, err := os.ReadFile(configPath)
	if err != nil {
		return Config{}, i18n.NewError("msg.err_config_unreadable", "path", configPath).WithCause(err)
	}
	var config Config
	if err := json.Unmarshal(raw, &config); err != nil {
		return Config{}, i18n.NewError("msg.err_config_invalid").WithCause(err)
	}
	return config.withDefaults().validated()
}

func (config Config) withDefaults() Config {
	if config.Dest == "" {
		config.Dest = defaultDest
	}
	if len(config.Extensions) == 0 {
		config.Extensions = defaultExtensions
	}
	config.Path = strings.Trim(config.Path, "/")
	return config
}

func (config Config) validated() (Config, error) {
	if !repoPattern.MatchString(config.Repo) {
		return Config{}, i18n.NewError("msg.err_repo_invalid", "repo", config.Repo)
	}
	if !path.IsAbs(config.Dest) {
		return Config{}, i18n.NewError("msg.err_dest_invalid", "dest", config.Dest)
	}
	if config.Path != "" {
		if _, err := sanitizeRelativePath(config.Path); err != nil {
			return Config{}, i18n.NewError("msg.err_path_invalid").WithCause(err)
		}
	}
	normalized := make([]string, 0, len(config.Extensions))
	for _, extension := range config.Extensions {
		extension = strings.ToLower(strings.TrimSpace(extension))
		if len(extension) < 2 || extension[0] != '.' {
			return Config{}, i18n.NewError("msg.err_extension_invalid", "extension", extension)
		}
		normalized = append(normalized, extension)
	}
	config.Extensions = normalized
	return config, nil
}

// Owner and Name split Repo, which validation guarantees has one slash.
func (config Config) Owner() string { return strings.SplitN(config.Repo, "/", 2)[0] }
func (config Config) Name() string  { return strings.SplitN(config.Repo, "/", 2)[1] }
