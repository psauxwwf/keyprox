package config

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func TestDefaultReturnsIndependentCopy(t *testing.T) {
	t.Parallel()

	first := Default()
	first.Provider["zai"][0] = "changed"
	first.Provider["new"] = []string{"value"}

	second := Default()
	if second.Provider["zai"][0] != "replace-with-real-key-1" {
		t.Fatalf("Default provider key mutated: %q", second.Provider["zai"][0])
	}
	if _, ok := second.Provider["new"]; ok {
		t.Fatalf("Default unexpectedly contains copied provider")
	}
	if second.Runtime.Level != "info" {
		t.Fatalf("Default runtime level = %q, want %q", second.Runtime.Level, "info")
	}
	if second.Runtime.Upstream429Retries != defaultUpstream429Retries {
		t.Fatalf("Default runtime upstream retries = %d, want %d", second.Runtime.Upstream429Retries, defaultUpstream429Retries)
	}
	if second.Runtime.Log != "" {
		t.Fatalf("Default runtime log = %q, want empty string", second.Runtime.Log)
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
    - " key-1 "
    - key-2
  OpenRouter:
    - key-3
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

	wantProviders := ProviderKeys{
		"zai":        {"key-1", "key-2"},
		"openrouter": {"key-3"},
	}
	if !reflect.DeepEqual(cfg.Provider, wantProviders) {
		t.Fatalf("Provider = %#v, want %#v", cfg.Provider, wantProviders)
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
	if cfg.Runtime.Upstream429Retries != defaultUpstream429Retries {
		t.Fatalf("Runtime.Upstream429Retries = %d, want %d", cfg.Runtime.Upstream429Retries, defaultUpstream429Retries)
	}
	if cfg.Runtime.Level != defaultConfig.Runtime.Level {
		t.Fatalf("Runtime.Level = %q, want %q", cfg.Runtime.Level, defaultConfig.Runtime.Level)
	}
}

func TestValidateRejectsExampleKeys(t *testing.T) {
	t.Parallel()

	cfg := Default()
	if err := cfg.Validate(); err == nil {
		t.Fatalf("Validate returned nil error for example keys")
	}
}

func TestValidateRejectsNegativeUpstream429Retries(t *testing.T) {
	t.Parallel()

	cfg := Default()
	cfg.Provider["zai"] = []string{"real-key"}
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
		Provider: ProviderKeys{
			" ZAI ": {" key-1 "},
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
	if !reflect.DeepEqual(loaded.Provider, ProviderKeys{"zai": {"key-1"}}) {
		t.Fatalf("Provider = %#v, want normalized provider map", loaded.Provider)
	}
}
