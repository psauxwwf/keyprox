package config

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func TestDefaultWithProvidersReturnsIndependentCopy(t *testing.T) {
	t.Parallel()

	source := Providers{
		"zai": {
			Endpoints: []string{"https://api.z.ai/api/coding/paas/v4"},
			DefaultHeaders: map[string]string{
				"X-Test": "one",
			},
			Keys: []string{"key-1"},
		},
	}

	first := DefaultWithProviders(source)
	entry := first.Provider["zai"]
	entry.Endpoints[0] = "https://changed.example/v1"
	entry.DefaultHeaders["X-Test"] = "two"
	entry.Keys[0] = "changed"
	first.Provider["zai"] = entry
	first.Provider["new"] = ProviderConfig{Endpoints: []string{"https://example.com/v1"}, Keys: []string{"key-2"}}

	second := DefaultWithProviders(source)
	if got := second.Provider["zai"].Endpoints[0]; got != "https://api.z.ai/api/coding/paas/v4" {
		t.Fatalf("DefaultWithProviders endpoint mutated: %q", got)
	}
	if got := second.Provider["zai"].DefaultHeaders["X-Test"]; got != "one" {
		t.Fatalf("DefaultWithProviders header mutated: %q", got)
	}
	if got := second.Provider["zai"].Keys[0]; got != "key-1" {
		t.Fatalf("DefaultWithProviders key mutated: %q", got)
	}
	if _, ok := second.Provider["new"]; ok {
		t.Fatalf("DefaultWithProviders unexpectedly contains copied provider")
	}
	if source["zai"].Endpoints[0] != "https://api.z.ai/api/coding/paas/v4" {
		t.Fatalf("source endpoint mutated: %q", source["zai"].Endpoints[0])
	}
	if source["zai"].DefaultHeaders["X-Test"] != "one" {
		t.Fatalf("source header mutated: %q", source["zai"].DefaultHeaders["X-Test"])
	}
	if source["zai"].Keys[0] != "key-1" {
		t.Fatalf("source key mutated: %q", source["zai"].Keys[0])
	}
}

func TestNewLoadsTypedYAMLAndNormalizesProviders(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "keyprox.yaml")
	if err := os.WriteFile(path, []byte(`
runtime:
  listen: " :9090 "
  read_header_timeout: 3s
  upstream_429_retries: 5
  level: " debug "
  log: " logs/keyprox.log "
provider:
  ZAI:
    endpoints:
      - " https://api.z.ai/api/coding/paas/v4 "
    default_headers:
      X-Test: " enabled "
    keys:
      - " key-1 "
      - key-2
  OpenRouter:
    endpoints:
      - " https://openrouter.ai/api/v1 "
    keys: []
`), 0o644); err != nil {
		t.Fatalf("os.WriteFile returned error: %v", err)
	}

	cfg, err := New(path)
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	if cfg.Runtime.Listen != ":9090" {
		t.Fatalf("Runtime.Listen = %q, want %q", cfg.Runtime.Listen, ":9090")
	}
	if cfg.Runtime.ReadHeaderTimeout != 3*time.Second {
		t.Fatalf("Runtime.ReadHeaderTimeout = %v, want %v", cfg.Runtime.ReadHeaderTimeout, 3*time.Second)
	}
	if cfg.Runtime.Upstream429Retries != 5 {
		t.Fatalf("Runtime.Upstream429Retries = %d, want %d", cfg.Runtime.Upstream429Retries, 5)
	}
	if cfg.Runtime.Level != "debug" {
		t.Fatalf("Runtime.Level = %q, want %q", cfg.Runtime.Level, "debug")
	}
	if cfg.Runtime.Log != "logs/keyprox.log" {
		t.Fatalf("Runtime.Log = %q, want %q", cfg.Runtime.Log, "logs/keyprox.log")
	}

	wantProviders := Providers{
		"openrouter": {
			Endpoints: []string{"https://openrouter.ai/api/v1"},
			Keys:      []string{},
		},
		"zai": {
			Endpoints: []string{"https://api.z.ai/api/coding/paas/v4"},
			DefaultHeaders: map[string]string{
				"X-Test": "enabled",
			},
			Keys: []string{"key-1", "key-2"},
		},
	}
	if !reflect.DeepEqual(cfg.Provider, wantProviders) {
		t.Fatalf("Provider = %#v, want %#v", cfg.Provider, wantProviders)
	}
	if got := cfg.EnabledProviderIDs(); !reflect.DeepEqual(got, []string{"zai"}) {
		t.Fatalf("EnabledProviderIDs = %#v, want %#v", got, []string{"zai"})
	}
}

func TestNewReturnsDefaultConfigWhenMissing(t *testing.T) {
	t.Parallel()

	cfg, err := New(filepath.Join(t.TempDir(), "missing.yaml"))
	if !errors.Is(err, ErrNotExists) {
		t.Fatalf("err = %v, want ErrNotExists", err)
	}
	if cfg == nil {
		t.Fatalf("cfg is nil")
	}
	if cfg.Runtime.Listen != defaultListenAddr {
		t.Fatalf("Runtime.Listen = %q, want %q", cfg.Runtime.Listen, defaultListenAddr)
	}
	if cfg.Runtime.Level != defaultConfig.Runtime.Level {
		t.Fatalf("Runtime.Level = %q, want %q", cfg.Runtime.Level, defaultConfig.Runtime.Level)
	}
	if len(cfg.Provider) != 0 {
		t.Fatalf("Provider length = %d, want 0", len(cfg.Provider))
	}
}

func TestValidateAllowsDisabledProvidersWithoutKeys(t *testing.T) {
	t.Parallel()

	cfg := DefaultWithProviders(Providers{
		"zai": {
			Endpoints: []string{"https://api.z.ai/api/coding/paas/v4"},
			Keys:      []string{},
		},
	})

	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate returned error: %v", err)
	}
}

func TestValidateRejectsConfiguredProviderWithoutEndpoint(t *testing.T) {
	t.Parallel()

	cfg := DefaultWithProviders(Providers{
		"zai": {
			Keys: []string{"real-key"},
		},
	})

	err := cfg.Validate()
	if err == nil || err.Error() != `provider "zai" must define at least one endpoint when keys are configured` {
		t.Fatalf("Validate error = %v, want missing endpoint validation", err)
	}
}

func TestValidateRejectsNegativeUpstream429Retries(t *testing.T) {
	t.Parallel()

	cfg := DefaultWithProviders(Providers{
		"zai": {
			Endpoints: []string{"https://api.z.ai/api/coding/paas/v4"},
			Keys:      []string{"real-key"},
		},
	})
	cfg.Runtime.Upstream429Retries = -1

	err := cfg.Validate()
	if err == nil || err.Error() != "runtime.upstream_429_retries must be >= 0" {
		t.Fatalf("Validate error = %v, want runtime.upstream_429_retries validation", err)
	}
}

func TestSaveConfigWritesNormalizedTypedYAML(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "nested", "keyprox.yaml")
	cfg := &Config{
		Runtime: Runtime{
			Listen:             " :8081 ",
			Upstream429Retries: 5,
			Level:              " warn ",
			Log:                " logs/keyprox.jsonl ",
		},
		Provider: Providers{
			" ZAI ": {
				Endpoints: []string{" https://api.z.ai/api/coding/paas/v4 "},
				DefaultHeaders: map[string]string{
					" X-Test ": " enabled ",
				},
				Keys: []string{" key-1 "},
			},
		},
	}
	if err := SaveConfig(path, cfg); err != nil {
		t.Fatalf("SaveConfig returned error: %v", err)
	}

	loaded, err := New(path)
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	if loaded.Runtime.Listen != ":8081" {
		t.Fatalf("Runtime.Listen = %q, want %q", loaded.Runtime.Listen, ":8081")
	}
	if loaded.Runtime.ReadHeaderTimeout != defaultReadHeaderTimeout {
		t.Fatalf("Runtime.ReadHeaderTimeout = %v, want %v", loaded.Runtime.ReadHeaderTimeout, defaultReadHeaderTimeout)
	}
	if loaded.Runtime.Upstream429Retries != 5 {
		t.Fatalf("Runtime.Upstream429Retries = %d, want %d", loaded.Runtime.Upstream429Retries, 5)
	}
	if loaded.Runtime.Level != "warn" {
		t.Fatalf("Runtime.Level = %q, want %q", loaded.Runtime.Level, "warn")
	}
	if loaded.Runtime.Log != "logs/keyprox.jsonl" {
		t.Fatalf("Runtime.Log = %q, want %q", loaded.Runtime.Log, "logs/keyprox.jsonl")
	}
	wantProviders := Providers{
		"zai": {
			Endpoints: []string{"https://api.z.ai/api/coding/paas/v4"},
			DefaultHeaders: map[string]string{
				"X-Test": "enabled",
			},
			Keys: []string{"key-1"},
		},
	}
	if !reflect.DeepEqual(loaded.Provider, wantProviders) {
		t.Fatalf("Provider = %#v, want %#v", loaded.Provider, wantProviders)
	}
}
