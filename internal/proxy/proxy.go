package proxy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"

	"keyprox/pkg/config"
)

type Proxy struct {
	catalog            map[string]ProviderCatalogEntry
	keys               map[string][]string
	client             *http.Client
	upstream429Retries int
	counters           sync.Map
}

type providerCounter struct {
	n atomic.Uint64
}
type statusCapturingResponseWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusCapturingResponseWriter) WriteHeader(status int) {
	if w.status == 0 {
		w.status = status
	}
	w.ResponseWriter.WriteHeader(status)
}

func (w *statusCapturingResponseWriter) Write(body []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	return w.ResponseWriter.Write(body)
}
func (w *statusCapturingResponseWriter) Flush() {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	if flusher, ok := w.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

func (w *statusCapturingResponseWriter) StatusCode() int {
	if w.status == 0 {
		return http.StatusOK
	}
	return w.status
}

func NewProxy(cfg config.Config, catalog map[string]ProviderCatalogEntry, client *http.Client) (*Proxy, error) {
	if client == nil {
		client = http.DefaultClient
	}

	keys := make(map[string][]string, len(cfg.Provider))
	for provider, providerKeys := range cfg.Provider {
		entry, ok := catalog[provider]
		if !ok {
			return nil, fmt.Errorf("provider %q is configured but is not available in the catalog", provider)
		}
		if entry.BaseURL == nil || entry.BaseURL.Scheme == "" || entry.BaseURL.Host == "" {
			return nil, fmt.Errorf("provider %q has invalid endpoint", provider)
		}

		keys[provider] = append(keys[provider], providerKeys...)
	}

	return &Proxy{
		catalog:            catalog,
		keys:               keys,
		client:             client,
		upstream429Retries: cfg.Runtime.Upstream429Retries,
	}, nil
}

func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	loggedWriter := &statusCapturingResponseWriter{ResponseWriter: w}
	w = loggedWriter

	var (
		provider string
		target   *url.URL
	)
	defer func() {
		status := loggedWriter.StatusCode()
		switch {
		case provider != "" && target != nil:
			slog.Info("proxy response",
				"provider", provider,
				"method", r.Method,
				"path", r.URL.Path,
				"target", target.Redacted(),
				"status", status,
			)
		case provider != "":
			slog.Info("proxy response",
				"provider", provider,
				"method", r.Method,
				"path", r.URL.Path,
				"status", status,
			)
		default:
			slog.Info("proxy response",
				"method", r.Method,
				"path", r.URL.Path,
				"status", status,
			)
		}
	}()

	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeProxyError(w, http.StatusBadRequest, fmt.Sprintf("read request body: %v", err))
		return
	}

	provider, rewrittenBody, err := p.rewriteRequest(body)
	if err != nil {
		writeProxyError(w, http.StatusBadRequest, err.Error())
		return
	}

	entry, ok := p.catalog[provider]
	if !ok {
		writeProxyError(w, http.StatusBadRequest, fmt.Sprintf("provider %q is not available in the catalog", provider))
		return
	}

	keys, ok := p.keys[provider]
	if !ok {
		writeProxyError(w, http.StatusBadRequest, fmt.Sprintf("provider %q is not configured", provider))
		return
	}

	target = buildTargetURL(entry.BaseURL, r.URL)
	slog.Info("proxy request",
		"provider", provider,
		"method", r.Method,
		"path", r.URL.Path,
		"target", target.Redacted(),
	)
	for attempt := 0; ; attempt++ {
		outbound, err := http.NewRequestWithContext(r.Context(), r.Method, target.String(), bytes.NewReader(rewrittenBody))
		if err != nil {
			writeProxyError(w, http.StatusInternalServerError, fmt.Sprintf("build upstream request: %v", err))
			return
		}

		copyRequestHeaders(outbound.Header, r.Header)
		applyDefaultHeaders(outbound.Header, entry.DefaultHeaders)
		outbound.ContentLength = int64(len(rewrittenBody))
		outbound.Header.Set("Authorization", "Bearer "+p.nextKey(provider, keys))

		response, err := p.client.Do(outbound)
		if err != nil {
			writeProxyError(w, http.StatusBadGateway, fmt.Sprintf("forward request to %s: %v", provider, err))
			return
		}

		if response.StatusCode == http.StatusTooManyRequests && attempt < p.upstream429Retries {
			slog.Warn("upstream returned 429, retrying with next key",
				"provider", provider,
				"status", response.StatusCode,
				"target", target.Redacted(),
				"attempt", attempt+1,
				"max_retries", p.upstream429Retries,
			)
			_ = response.Body.Close()
			continue
		}
		if response.StatusCode >= http.StatusBadRequest {
			slog.Warn("upstream error",
				"provider", provider,
				"status", response.StatusCode,
				"target", target.Redacted(),
			)
		}

		if err := writeUpstreamResponse(w, response); err != nil {
			_ = response.Body.Close()
			return
		}
		_ = response.Body.Close()
		return
	}
}

func (p *Proxy) rewriteRequest(body []byte) (string, []byte, error) {
	if len(bytes.TrimSpace(body)) == 0 {
		return "", nil, fmt.Errorf("request body must be a JSON object with a model field")
	}

	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return "", nil, fmt.Errorf("request body must be valid JSON: %w", err)
	}

	rawModel, ok := payload["model"].(string)
	if !ok {
		return "", nil, fmt.Errorf("request body must contain a string model field")
	}

	provider, model, ok := strings.Cut(strings.TrimSpace(rawModel), "/")
	if !ok || provider == "" || model == "" {
		return "", nil, fmt.Errorf("model must use provider/model format")
	}
	provider = normalizeProviderID(provider)
	payload["model"] = model

	rewrittenBody, err := json.Marshal(payload)
	if err != nil {
		return "", nil, fmt.Errorf("encode rewritten request body: %w", err)
	}

	return provider, rewrittenBody, nil
}

func (p *Proxy) nextKey(provider string, keys []string) string {
	counterAny, _ := p.counters.LoadOrStore(provider, &providerCounter{})
	counter := counterAny.(*providerCounter)
	index := counter.n.Add(1) - 1
	return keys[index%uint64(len(keys))]
}

func buildTargetURL(base *url.URL, incoming *url.URL) *url.URL {
	target := *base
	target.Path = joinURLPath(base.Path, stripOpenAIPrefix(incoming.Path))
	target.RawPath = target.Path
	target.RawQuery = incoming.RawQuery
	return &target
}

func stripOpenAIPrefix(path string) string {
	trimmed := strings.TrimPrefix(path, "/")
	if trimmed == "v1" {
		return ""
	}
	if after, ok := strings.CutPrefix(trimmed, "v1/"); ok {
		return after
	}
	return trimmed
}

func joinURLPath(basePath, appendPath string) string {
	basePath = strings.TrimRight(basePath, "/")
	appendPath = strings.TrimLeft(appendPath, "/")

	switch {
	case basePath == "" && appendPath == "":
		return "/"
	case basePath == "":
		return "/" + appendPath
	case appendPath == "":
		return basePath
	default:
		return basePath + "/" + appendPath
	}
}

func normalizeProviderID(provider string) string {
	return strings.ToLower(strings.TrimSpace(provider))
}

func writeUpstreamResponse(w http.ResponseWriter, response *http.Response) error {
	headers := w.Header()
	copyResponseHeaders(headers, response.Header)

	if response.StatusCode >= http.StatusBadRequest && headers.Get("Content-Type") == "" {
		body, err := io.ReadAll(response.Body)
		if err != nil {
			return err
		}
		if json.Valid(bytes.TrimSpace(body)) {
			headers.Set("Content-Type", "application/json")
		}
		w.WriteHeader(response.StatusCode)
		if len(body) == 0 {
			return nil
		}
		_, err = w.Write(body)
		return err
	}

	w.WriteHeader(response.StatusCode)
	return streamResponse(w, response.Body)
}

func copyRequestHeaders(dst, src http.Header) {
	copyHeaders(dst, src)
	dst.Del("Authorization")
	dst.Del("Connection")
	dst.Del("Proxy-Connection")
	dst.Del("Keep-Alive")
	dst.Del("Proxy-Authenticate")
	dst.Del("Proxy-Authorization")
	dst.Del("Te")
	dst.Del("Trailer")
	dst.Del("Transfer-Encoding")
	dst.Del("Upgrade")
}

func applyDefaultHeaders(dst http.Header, defaults map[string]string) {
	for key, value := range defaults {
		dst.Set(key, value)
	}
}

func copyResponseHeaders(dst, src http.Header) {
	copyHeaders(dst, src)
	dst.Del("Connection")
	dst.Del("Proxy-Connection")
	dst.Del("Keep-Alive")
	dst.Del("Proxy-Authenticate")
	dst.Del("Proxy-Authorization")
	dst.Del("Te")
	dst.Del("Trailer")
	dst.Del("Transfer-Encoding")
	dst.Del("Upgrade")
}

func copyHeaders(dst, src http.Header) {
	for key, values := range src {
		for _, value := range values {
			dst.Add(key, value)
		}
	}
}

func streamResponse(w http.ResponseWriter, body io.Reader) error {
	flusher, _ := w.(http.Flusher)
	buffer := make([]byte, 32*1024)
	for {
		n, err := body.Read(buffer)
		if n > 0 {
			if _, writeErr := w.Write(buffer[:n]); writeErr != nil {
				return writeErr
			}
			if flusher != nil {
				flusher.Flush()
			}
		}
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
	}
}

func writeProxyError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"error": map[string]any{
			"message": message,
			"type":    "invalid_request_error",
		},
	})
}
