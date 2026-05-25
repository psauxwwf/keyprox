package keyprox

import (
	"bytes"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"keyprox/internal/proxy"
	"keyprox/pkg/config"
)

func TestRootCmdSaveWritesDefaultConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "keyprox.yaml")
	cmd := rootCmd(func(*config.Config, map[string]proxy.ProviderCatalogEntry) error {
		t.Fatalf("runner should not be called when --save is used")
		return nil
	})

	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"--config", path, "--save"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("os.Stat returned error: %v", err)
	}
}

func TestRootCmdLoadsConfigAndInvokesRunner(t *testing.T) {
	path := filepath.Join(t.TempDir(), "keyprox.yaml")
	if err := os.WriteFile(path, []byte(`
runtime:
  listen: ":9090"
  read_header_timeout: 7s
  level: info
provider:
  zai:
    - real-key
`), 0o644); err != nil {
		t.Fatalf("os.WriteFile returned error: %v", err)
	}

	var (
		capturedConfig  *config.Config
		capturedCatalog map[string]proxy.ProviderCatalogEntry
	)
	cmd := rootCmd(func(cfg *config.Config, catalog map[string]proxy.ProviderCatalogEntry) error {
		capturedConfig = cfg
		capturedCatalog = catalog
		return nil
	})

	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"--config", path})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if capturedConfig == nil {
		t.Fatalf("runner was not called")
	}
	if capturedConfig.Runtime.Listen != ":9090" {
		t.Fatalf("Runtime.Listen = %q, want %q", capturedConfig.Runtime.Listen, ":9090")
	}
	if capturedCatalog == nil || len(capturedCatalog) == 0 {
		t.Fatalf("catalog was not passed to runner")
	}
}

func TestRootCmdReturnsInitErrorForMissingConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing.yaml")
	cmd := rootCmd(func(*config.Config, map[string]proxy.ProviderCatalogEntry) error { return nil })
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"--config", path})

	err := cmd.Execute()
	if err == nil {
		t.Fatalf("Execute returned nil error")
	}

	var exitErr *ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("err = %T, want *ExitError", err)
	}
	if exitErr.ExitCode() != initCode {
		t.Fatalf("exitErr.ExitCode() = %d, want %d", exitErr.ExitCode(), initCode)
	}
}

func TestRootCmdRejectsProviderWithoutEndpoint(t *testing.T) {
	path := filepath.Join(t.TempDir(), "keyprox.yaml")
	if err := os.WriteFile(path, []byte(`
runtime:
  listen: ":9090"
  read_header_timeout: 7s
  level: info
provider:
  missing-provider:
    - real-key
`), 0o644); err != nil {
		t.Fatalf("os.WriteFile returned error: %v", err)
	}

	cmd := rootCmd(func(*config.Config, map[string]proxy.ProviderCatalogEntry) error {
		t.Fatalf("runner should not be called when provider endpoint is missing")
		return nil
	})
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"--config", path})

	err := cmd.Execute()
	if err == nil {
		t.Fatalf("Execute returned nil error")
	}

	var exitErr *ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("err = %T, want *ExitError", err)
	}
	if exitErr.ExitCode() != initCode {
		t.Fatalf("exitErr.ExitCode() = %d, want %d", exitErr.ExitCode(), initCode)
	}
	if exitErr.Unwrap() == nil || exitErr.Unwrap().Error() != `provider "missing-provider" has no endpoint in the catalog` {
		t.Fatalf("err = %v, want missing endpoint error", exitErr.Unwrap())
	}
}

func TestRootCmdRejectsInvalidLogLevel(t *testing.T) {
	path := filepath.Join(t.TempDir(), "keyprox.yaml")
	if err := os.WriteFile(path, []byte(`
runtime:
  listen: ":9090"
  read_header_timeout: 7s
  level: verbose
provider:
  zai:
    - real-key
`), 0o644); err != nil {
		t.Fatalf("os.WriteFile returned error: %v", err)
	}

	cmd := rootCmd(func(*config.Config, map[string]proxy.ProviderCatalogEntry) error {
		t.Fatalf("runner should not be called when log level is invalid")
		return nil
	})
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"--config", path})

	err := cmd.Execute()
	if err == nil {
		t.Fatalf("Execute returned nil error")
	}

	var exitErr *ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("err = %T, want *ExitError", err)
	}
	if exitErr.ExitCode() != initCode {
		t.Fatalf("exitErr.ExitCode() = %d, want %d", exitErr.ExitCode(), initCode)
	}
	if exitErr.Unwrap() == nil || !strings.Contains(exitErr.Unwrap().Error(), `invalid log level "verbose"`) {
		t.Fatalf("err = %v, want invalid log level error", exitErr.Unwrap())
	}
}

func TestConfigureLoggerWritesJSONLogFile(t *testing.T) {
	logPath := filepath.Join(t.TempDir(), "logs", "keyprox.jsonl")
	if err := configureLogger("debug", logPath); err != nil {
		t.Fatalf("configureLogger returned error: %v", err)
	}
	t.Cleanup(func() {
		if err := configureLogger("info", ""); err != nil {
			t.Fatalf("reset logger returned error: %v", err)
		}
	})

	slog.Info("test message", "provider", "zai")

	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("os.ReadFile returned error: %v", err)
	}
	text := string(data)
	if !strings.Contains(text, `"msg":"test message"`) {
		t.Fatalf("log file = %q, want JSON message", text)
	}
	if !strings.Contains(text, `"provider":"zai"`) {
		t.Fatalf("log file = %q, want provider field", text)
	}
}
