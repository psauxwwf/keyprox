package config

import (
	"errors"
	"fmt"
	"io"
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
		Provider: ProviderKeys{
			"zai": {
				"replace-with-real-key-1",
				"replace-with-real-key-2",
			},
		},
	}
	ErrNotExists = fmt.Errorf("config not found: %w", os.ErrNotExist)
)

type Config struct {
	Runtime  Runtime      `yaml:"runtime"`
	Provider ProviderKeys `yaml:"provider"`
}

type Runtime struct {
	Listen             string        `yaml:"listen"`
	ReadHeaderTimeout  time.Duration `yaml:"read_header_timeout"`
	Upstream429Retries int           `yaml:"upstream_429_retries"`
	Level              string        `yaml:"level"`
	Log                string        `yaml:"log"`
}

type ProviderKeys map[string][]string

func Default() Config {
	cfg := Config{
		Runtime: defaultConfig.Runtime,
	}
	if defaultConfig.Provider != nil {
		cfg.Provider = cloneProviderKeys(defaultConfig.Provider)
	}
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
		return fmt.Errorf("provider must define at least one provider key list")
	}

	for provider, keys := range cfg.Provider {
		if provider == "" {
			return fmt.Errorf("provider id must not be empty")
		}
		if len(keys) == 0 {
			return fmt.Errorf("provider %q has no keys", provider)
		}
		for _, key := range keys {
			if key == "" {
				return fmt.Errorf("provider %q contains an empty key", provider)
			}
			if strings.HasPrefix(key, "replace-with-real-key") {
				return fmt.Errorf("provider %q still uses the example key %q", provider, key)
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
	normalized.Provider = cloneProviderKeys(cfg.Provider)
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
		cfg.Provider = make(ProviderKeys)
	}
}

func normalize(cfg *Config) {
	cfg.Runtime.Listen = strings.TrimSpace(cfg.Runtime.Listen)
	cfg.Runtime.Level = strings.TrimSpace(cfg.Runtime.Level)
	cfg.Runtime.Log = strings.TrimSpace(cfg.Runtime.Log)
	cfg.Provider = normalizeProviderKeys(cfg.Provider)
}

func normalizeProviderKeys(src ProviderKeys) ProviderKeys {
	if len(src) == 0 {
		return make(ProviderKeys)
	}

	dst := make(ProviderKeys, len(src))
	for provider, keys := range src {
		normalizedProvider := normalizeProviderID(provider)
		normalizedKeys := make([]string, 0, len(keys))
		for _, key := range keys {
			normalizedKeys = append(normalizedKeys, strings.TrimSpace(key))
		}
		dst[normalizedProvider] = append(dst[normalizedProvider], normalizedKeys...)
	}
	return dst
}

func cloneProviderKeys(src ProviderKeys) ProviderKeys {
	if len(src) == 0 {
		return make(ProviderKeys)
	}

	dst := make(ProviderKeys, len(src))
	for provider, keys := range src {
		dst[provider] = append([]string(nil), keys...)
	}
	return dst
}

func normalizeProviderID(provider string) string {
	return strings.ToLower(strings.TrimSpace(provider))
}
