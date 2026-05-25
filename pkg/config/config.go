package config

import (
	"errors"
	"fmt"
	"io"
	"maps"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	defaultListenAddr         = ":5050"
	defaultReadHeaderTimeout  = 10 * time.Second
	defaultUpstream429Retries = 3
)

var (
	defaultConfig = Config{
		Runtime: Runtime{
			Listen:             defaultListenAddr,
			ReadHeaderTimeout:  defaultReadHeaderTimeout,
			Upstream429Retries: defaultUpstream429Retries,
			Level:              "info",
			Log:                "",
		},
		Provider: make(Providers),
	}
	ErrNotExists = fmt.Errorf("config not found: %w", os.ErrNotExist)
)

type Config struct {
	Runtime  Runtime   `yaml:"runtime"`
	Provider Providers `yaml:"provider"`
}

type Runtime struct {
	Listen             string        `yaml:"listen"`
	ReadHeaderTimeout  time.Duration `yaml:"read_header_timeout"`
	Upstream429Retries int           `yaml:"upstream_429_retries"`
	Level              string        `yaml:"level"`
	Log                string        `yaml:"log"`
}

type ProviderConfig struct {
	Endpoints      []string          `yaml:"endpoints"`
	DefaultHeaders map[string]string `yaml:"default_headers,omitempty"`
	Keys           []string          `yaml:"keys"`
}

type Providers map[string]ProviderConfig

func Default() Config {
	cfg := Config{
		Runtime: defaultConfig.Runtime,
	}
	cfg.Provider = cloneProviders(defaultConfig.Provider)
	return cfg
}

func DefaultWithProviders(providers Providers) Config {
	cfg := Default()
	cfg.Provider = cloneProviders(providers)
	return cfg
}

func Save(filename string) error {
	cfg := Default()
	return save(&cfg, filename)
}

func SaveConfig(filename string, cfg *Config) error {
	if cfg == nil {
		return fmt.Errorf("config is nil")
	}
	return save(cfg, filename)
}

func New(filename string) (*Config, error) {
	cfg := Config{}

	data, err := readFile(filename)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			defaults := Default()
			return &defaults, ErrNotExists
		}
		return nil, err
	}

	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config %q: %w", filename, err)
	}

	normalize(&cfg)
	applyDefaults(&cfg)
	return &cfg, nil
}

func (cfg *Config) Validate() error {
	if cfg == nil {
		return fmt.Errorf("config is nil")
	}
	if strings.TrimSpace(cfg.Runtime.Listen) == "" {
		return fmt.Errorf("runtime.listen must not be empty")
	}
	if cfg.Runtime.ReadHeaderTimeout <= 0 {
		return fmt.Errorf("runtime.read_header_timeout must be > 0")
	}
	if cfg.Runtime.Upstream429Retries < 0 {
		return fmt.Errorf("runtime.upstream_429_retries must be >= 0")
	}
	if strings.TrimSpace(cfg.Runtime.Level) == "" {
		return fmt.Errorf("runtime.level must not be empty")
	}
	if len(cfg.Provider) == 0 {
		return fmt.Errorf("provider must define at least one provider entry")
	}

	for providerID, provider := range cfg.Provider {
		if providerID == "" {
			return fmt.Errorf("provider id must not be empty")
		}
		if len(provider.Keys) == 0 {
			continue
		}
		if len(provider.Endpoints) == 0 {
			return fmt.Errorf("provider %q must define at least one endpoint when keys are configured", providerID)
		}
		for _, endpoint := range provider.Endpoints {
			if endpoint == "" {
				return fmt.Errorf("provider %q contains an empty endpoint", providerID)
			}
			if err := validateEndpoint(endpoint); err != nil {
				return fmt.Errorf("provider %q endpoint %q: %w", providerID, endpoint, err)
			}
		}
		for _, key := range provider.Keys {
			if key == "" {
				return fmt.Errorf("provider %q contains an empty key", providerID)
			}
			if strings.HasPrefix(key, "replace-with-real-key") {
				return fmt.Errorf("provider %q still uses the example key %q", providerID, key)
			}
		}
	}

	return nil
}

func (cfg Config) ProviderIDs() []string {
	providers := make([]string, 0, len(cfg.Provider))
	for provider := range cfg.Provider {
		providers = append(providers, provider)
	}
	sort.Strings(providers)
	return providers
}

func (cfg Config) EnabledProviderIDs() []string {
	providers := make([]string, 0, len(cfg.Provider))
	for provider, providerCfg := range cfg.Provider {
		if len(providerCfg.Keys) == 0 {
			continue
		}
		providers = append(providers, provider)
	}
	sort.Strings(providers)
	return providers
}

func readFile(filename string) ([]byte, error) {
	if filename == "-" {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return nil, fmt.Errorf("read config from stdin: %w", err)
		}
		return data, nil
	}

	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("read config %q: %w", filename, err)
	}
	return data, nil
}

func save(cfg *Config, filename string) error {
	if filename == "-" {
		return fmt.Errorf("cannot save config to stdin")
	}

	normalized := *cfg
	normalized.Provider = cloneProviders(cfg.Provider)
	normalize(&normalized)
	applyDefaults(&normalized)

	data, err := yaml.Marshal(&normalized)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(filename), 0o755); err != nil {
		return fmt.Errorf("create config directory for %q: %w", filename, err)
	}
	if err := os.WriteFile(filename, data, 0o644); err != nil {
		return fmt.Errorf("write config %q: %w", filename, err)
	}
	return nil
}

func applyDefaults(cfg *Config) {
	if strings.TrimSpace(cfg.Runtime.Listen) == "" {
		cfg.Runtime.Listen = defaultListenAddr
	}
	if cfg.Runtime.ReadHeaderTimeout <= 0 {
		cfg.Runtime.ReadHeaderTimeout = defaultReadHeaderTimeout
	}
	if cfg.Runtime.Upstream429Retries == 0 {
		cfg.Runtime.Upstream429Retries = defaultUpstream429Retries
	}
	if strings.TrimSpace(cfg.Runtime.Level) == "" {
		cfg.Runtime.Level = defaultConfig.Runtime.Level
	}
	if cfg.Provider == nil {
		cfg.Provider = make(Providers)
	}
}

func normalize(cfg *Config) {
	cfg.Runtime.Listen = strings.TrimSpace(cfg.Runtime.Listen)
	cfg.Runtime.Level = strings.TrimSpace(cfg.Runtime.Level)
	cfg.Runtime.Log = strings.TrimSpace(cfg.Runtime.Log)
	cfg.Provider = normalizeProviders(cfg.Provider)
}

func normalizeProviders(src Providers) Providers {
	if len(src) == 0 {
		return make(Providers)
	}

	dst := make(Providers, len(src))
	for providerID, provider := range src {
		dst[normalizeProviderID(providerID)] = ProviderConfig{
			Endpoints:      normalizeStrings(provider.Endpoints),
			DefaultHeaders: normalizeStringMap(provider.DefaultHeaders),
			Keys:           normalizeStrings(provider.Keys),
		}
	}
	return dst
}

func cloneProviders(src Providers) Providers {
	if len(src) == 0 {
		return make(Providers)
	}

	dst := make(Providers, len(src))
	for providerID, provider := range src {
		dst[providerID] = ProviderConfig{
			Endpoints:      cloneStrings(provider.Endpoints),
			DefaultHeaders: cloneStringMap(provider.DefaultHeaders),
			Keys:           cloneStrings(provider.Keys),
		}
	}
	return dst
}

func normalizeStrings(src []string) []string {
	if len(src) == 0 {
		return []string{}
	}

	dst := make([]string, 0, len(src))
	for _, value := range src {
		dst = append(dst, strings.TrimSpace(value))
	}
	return dst
}

func cloneStrings(src []string) []string {
	if len(src) == 0 {
		return []string{}
	}
	return append([]string(nil), src...)
}

func normalizeStringMap(src map[string]string) map[string]string {
	if len(src) == 0 {
		return nil
	}

	dst := make(map[string]string, len(src))
	for key, value := range src {
		trimmedKey := strings.TrimSpace(key)
		if trimmedKey == "" {
			continue
		}
		dst[trimmedKey] = strings.TrimSpace(value)
	}
	if len(dst) == 0 {
		return nil
	}
	return dst
}

func cloneStringMap(src map[string]string) map[string]string {
	if len(src) == 0 {
		return nil
	}

	dst := make(map[string]string, len(src))
	maps.Copy(dst, src)
	return dst
}

func validateEndpoint(raw string) error {
	parsed, err := url.Parse(raw)
	if err != nil {
		return err
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("invalid HTTP endpoint")
	}
	if parsed.Host == "" {
		return fmt.Errorf("invalid HTTP endpoint")
	}
	return nil
}

func normalizeProviderID(provider string) string {
	return strings.ToLower(strings.TrimSpace(provider))
}
