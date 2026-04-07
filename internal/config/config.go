// Package config loads and validates pagerduty-client configuration.
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
	koToml "github.com/knadh/koanf/parsers/toml"
	"github.com/knadh/koanf/providers/confmap"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
	"github.com/matcra587/pagerduty-client/internal/credential"
	"github.com/matcra587/pagerduty-client/internal/dirs"
)

// DefaultTabs is the default set of TUI tabs when none are configured.
var DefaultTabs = []string{"incidents", "escalation-policies", "services", "teams"}

// ValidTabs lists all recognised tab names.
var ValidTabs = map[string]bool{
	"incidents":           true,
	"escalation-policies": true,
	"services":            true,
	"teams":               true,
}

// TUI holds dashboard display preferences.
type TUI struct {
	Theme        string   `koanf:"theme"`
	ShowResolved bool     `koanf:"show_resolved"`
	PageSize     int      `koanf:"page_size"`
	Tabs         []string `koanf:"tabs"`
}

// CustomField maps a PagerDuty custom field to a display column.
type CustomField struct {
	Label   string `koanf:"label"`
	Path    string `koanf:"path"`
	Display string `koanf:"display"`
}

// Config holds all configuration for pagerduty-client.
type Config struct {
	Token            string            `koanf:"-"`
	BaseURL          string            `koanf:"base_url"`
	Email            string            `koanf:"-"`
	Team             string            `koanf:"-"`
	Service          string            `koanf:"-"`
	Format           string            `koanf:"-"`
	RefreshInterval  int               `koanf:"-"`
	Debug            bool              `koanf:"-"`
	AgentMode        bool              `koanf:"-"`
	Interactive      bool              `koanf:"-"`
	TUI              TUI               `koanf:"tui"`
	CredentialSource credential.Source `koanf:"credential_source"`

	CustomFields []CustomField `koanf:"custom_fields"`

	tokenOptional bool
}

var validFormats = map[string]bool{
	"table": true,
	"json":  true,
}

// SetTokenOptional marks the token as not required for validation.
// Used by commands that can operate without authentication.
func (c *Config) SetTokenOptional() { c.tokenOptional = true }

// Validate checks required fields and value constraints.
func (c *Config) Validate() error {
	if c.Token == "" && !c.tokenOptional {
		return errors.New("token is required: set PDC_TOKEN or configure a credential source (run \"pdc config init\")")
	}
	if !validFormats[c.Format] {
		return fmt.Errorf("invalid format %q: must be \"table\" or \"json\"", c.Format)
	}
	if c.RefreshInterval <= 0 {
		return errors.New("refresh interval must be positive")
	}
	return nil
}

// Default returns a Config with sensible defaults. The base URL is not set
// here; the API client owns that default.
func Default() *Config {
	return &Config{
		Format:          "table",
		RefreshInterval: 30,
	}
}

// Option configures Load behaviour.
type Option func(*loadOptions)

type loadOptions struct {
	path    string
	token   string
	team    string
	service string
}

// WithPath overrides the config file path.
func WithPath(path string) Option {
	return func(o *loadOptions) { o.path = path }
}

// WithToken sets the API token directly.
func WithToken(token string) Option {
	return func(o *loadOptions) { o.token = token }
}

// WithTeam sets the team filter.
func WithTeam(team string) Option {
	return func(o *loadOptions) { o.team = team }
}

// WithService sets the service filter.
func WithService(service string) Option {
	return func(o *loadOptions) { o.service = service }
}

// DefaultConfigPath returns the path to the config file, using the
// platform-aware directory from dirs.PdcConfigDir.
func DefaultConfigPath() string {
	dir, err := dirs.PdcConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "config.toml")
}

// Load reads configuration from file, environment and options.
// Precedence (highest wins): option > env > file > built-in default.
func Load(opts ...Option) (*Config, error) {
	lo := &loadOptions{
		path: DefaultConfigPath(),
	}
	for _, o := range opts {
		o(lo)
	}

	k := koanf.New(".")

	if err := k.Load(confmap.Provider(map[string]any{
		"defaults.format":           "table",
		"defaults.refresh_interval": 30,
	}, "."), nil); err != nil {
		return nil, fmt.Errorf("loading defaults: %w", err)
	}

	if lo.path != "" {
		if _, err := os.Stat(lo.path); err == nil {
			if err := k.Load(file.Provider(lo.path), koToml.Parser()); err != nil {
				return nil, fmt.Errorf("loading config file: %w", err)
			}
		}
	}

	cfg := &Config{}
	if err := k.Unmarshal("", cfg); err != nil {
		return nil, fmt.Errorf("unmarshalling config: %w", err)
	}

	// Promote [defaults] values to top-level fields. koanf tags can't
	// map dotted keys to flat struct fields, so we read them manually.
	if v := k.String("defaults.email"); v != "" {
		cfg.Email = v
	}
	if v := k.String("defaults.team"); v != "" {
		cfg.Team = v
	}
	if v := k.String("defaults.service"); v != "" {
		cfg.Service = v
	}
	if v := k.String("defaults.format"); v != "" {
		cfg.Format = v
	}
	if v := k.Int("defaults.refresh_interval"); v != 0 {
		cfg.RefreshInterval = v
	}
	if k.Bool("defaults.interactive") {
		cfg.Interactive = true
	}

	// Load .env from the working directory if present. Ignore errors
	// (file is optional). Variables set here are picked up by applyEnv.
	_ = godotenv.Load()

	applyEnv(cfg)

	if lo.token != "" {
		cfg.Token = lo.token
	}
	if lo.team != "" {
		cfg.Team = lo.team
	}
	if lo.service != "" {
		cfg.Service = lo.service
	}

	return cfg, nil
}

func applyEnv(cfg *Config) {
	if v := os.Getenv("PDC_EMAIL"); v != "" {
		cfg.Email = v
	}
	if v := os.Getenv("PDC_TEAM"); v != "" {
		cfg.Team = v
	}
	if v := os.Getenv("PDC_SERVICE"); v != "" {
		cfg.Service = v
	}
	if v := os.Getenv("PDC_BASE_URL"); v != "" {
		cfg.BaseURL = v
	}
	if v := os.Getenv("PDC_FORMAT"); v != "" {
		cfg.Format = v
	}
	if v := os.Getenv("PDC_DEBUG"); v != "" {
		cfg.Debug = v == "1" || v == "true"
	}
	if v := os.Getenv("PDC_INTERACTIVE"); v != "" {
		cfg.Interactive = v == "1" || v == "true"
	}
}
