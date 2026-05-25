package keyprox

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"charm.land/fang/v2"
	"github.com/spf13/cobra"

	"keyprox/internal/proxy"
	"keyprox/pkg/config"
)

const (
	_ int = iota
	initCode
	fatalCode
)

type ExitError struct {
	code int
	err  error
}

func (e *ExitError) Error() string {
	if e == nil || e.err == nil {
		return ""
	}
	return e.err.Error()
}

func (e *ExitError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.err
}

func (e *ExitError) ExitCode() int {
	if e == nil {
		return fatalCode
	}
	return e.code
}

var (
	loggerMu       sync.Mutex
	currentLogFile *os.File
)

func Execute(ctx context.Context) error {
	return fang.Execute(ctx, rootCmd(runProxyServer), fang.WithoutVersion())
}

func newExitError(code int, err error) error {
	if err == nil {
		return nil
	}
	return &ExitError{code: code, err: err}
}

func rootCmd(run func(*config.Config) error) *cobra.Command {
	var (
		saveConf bool
		confPath = "keyprox.yaml"
		cfg      *config.Config
	)

	cmd := &cobra.Command{
		Use:           "keyprox",
		Short:         "OpenAI-compatible provider key proxy",
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(_ *cobra.Command, _ []string) error {
			loaded, err := config.New(confPath)
			if err != nil {
				if errors.Is(err, config.ErrNotExists) && saveConf {
					providerDefaults, loadErr := proxy.LoadProviderDefaults()
					if loadErr != nil {
						return newExitError(initCode, loadErr)
					}
					cfgCopy := config.DefaultWithProviders(providerDefaults)
					cfg = &cfgCopy
				} else {
					return newExitError(initCode, err)
				}
			} else {
				cfg = loaded
			}

			return newExitError(initCode, configureLogger(cfg.Runtime.Level, cfg.Runtime.Log))
		},
		RunE: func(_ *cobra.Command, _ []string) error {
			if saveConf {
				if err := config.SaveConfig(confPath, cfg); err != nil {
					return newExitError(fatalCode, fmt.Errorf("save config %q: %w", confPath, err))
				}
				slog.Info("config saved", "path", confPath)
				return nil
			}

			if err := cfg.Validate(); err != nil {
				return newExitError(initCode, err)
			}

			slog.Info("starting keyprox",
				"listen", cfg.Runtime.Listen,
				"level", cfg.Runtime.Level,
				"log", cfg.Runtime.Log,
				"providers", configuredProviders(*cfg),
			)
			return newExitError(fatalCode, run(cfg))
		},
	}

	cmd.Flags().StringVar(&confPath, "config", "keyprox.yaml", "path to config file")
	cmd.Flags().BoolVar(&saveConf, "save", false, "save resolved config to --config and exit")
	return cmd
}

func runProxyServer(cfg *config.Config) error {
	proxyHandler, err := proxy.NewProxy(*cfg, &http.Client{})
	if err != nil {
		return err
	}

	server := &http.Server{
		Addr:              cfg.Runtime.Listen,
		Handler:           proxyHandler,
		ReadHeaderTimeout: cfg.Runtime.ReadHeaderTimeout,
	}

	return server.ListenAndServe()
}

func configureLogger(levelText, logPath string) error {
	loggerMu.Lock()
	defer loggerMu.Unlock()

	if currentLogFile != nil {
		_ = currentLogFile.Close()
		currentLogFile = nil
	}

	var parsedLevel slog.Level
	if err := parsedLevel.UnmarshalText([]byte(levelText)); err != nil {
		return fmt.Errorf("invalid log level %q: %w", levelText, err)
	}

	stdoutHandler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: parsedLevel})
	if strings.TrimSpace(logPath) == "" {
		slog.SetDefault(slog.New(stdoutHandler))
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(logPath), 0o755); err != nil {
		return fmt.Errorf("create log dir for %q: %w", logPath, err)
	}

	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("open log file %q: %w", logPath, err)
	}
	currentLogFile = logFile

	jsonHandler := slog.NewJSONHandler(logFile, &slog.HandlerOptions{
		AddSource: true,
		Level:     parsedLevel,
	})
	slog.SetDefault(slog.New(newMultiHandler(stdoutHandler, jsonHandler)))
	return nil
}

func configuredProviders(cfg config.Config) []string {
	return cfg.EnabledProviderIDs()
}

type multiHandler struct {
	handlers []slog.Handler
}

func newMultiHandler(handlers ...slog.Handler) slog.Handler {
	filtered := make([]slog.Handler, 0, len(handlers))
	for _, handler := range handlers {
		if handler != nil {
			filtered = append(filtered, handler)
		}
	}
	return &multiHandler{handlers: filtered}
}

func (h *multiHandler) Enabled(ctx context.Context, level slog.Level) bool {
	for _, handler := range h.handlers {
		if handler.Enabled(ctx, level) {
			return true
		}
	}
	return false
}

func (h *multiHandler) Handle(ctx context.Context, record slog.Record) error {
	var errs []error
	for _, handler := range h.handlers {
		if !handler.Enabled(ctx, record.Level) {
			continue
		}
		if err := handler.Handle(ctx, record.Clone()); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func (h *multiHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	next := make([]slog.Handler, 0, len(h.handlers))
	for _, handler := range h.handlers {
		next = append(next, handler.WithAttrs(attrs))
	}
	return &multiHandler{handlers: next}
}

func (h *multiHandler) WithGroup(name string) slog.Handler {
	next := make([]slog.Handler, 0, len(h.handlers))
	for _, handler := range h.handlers {
		next = append(next, handler.WithGroup(name))
	}
	return &multiHandler{handlers: next}
}
