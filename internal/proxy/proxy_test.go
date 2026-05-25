package proxy

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"

	"keyprox/pkg/config"
)

type capturedRequest struct {
	Path           string
	RawQuery       string
	Authorization  string
	Model          string
	ProviderHeader string
}

func TestProxyRoundRobinUsesNextKeyPerRequest(t *testing.T) {
	t.Parallel()

	var (
		mu       sync.Mutex
		requests []capturedRequest
	)

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("io.ReadAll returned error: %v", err)
		}

		var payload map[string]any
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Fatalf("json.Unmarshal returned error: %v", err)
		}

		mu.Lock()
		requests = append(requests, capturedRequest{
			Path:           r.URL.Path,
			RawQuery:       r.URL.RawQuery,
			Authorization:  r.Header.Get("Authorization"),
			Model:          payload["model"].(string),
			ProviderHeader: r.Header.Get("X-Provider-Default"),
		})
		mu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer upstream.Close()

	baseURL := mustParseURL(t, upstream.URL+"/api/paas/v4")
	catalog := map[string]ProviderCatalogEntry{
		"zhipu": {
			BaseURL: baseURL,
			DefaultHeaders: map[string]string{
				"X-Provider-Default": "enabled",
			},
		},
	}

	proxy, err := NewProxy(config.Config{
		Provider: config.ProviderKeys{
			"zhipu": {"key-1", "key-2"},
		},
	}, catalog, upstream.Client())
	if err != nil {
		t.Fatalf("NewProxy returned error: %v", err)
	}

	proxyServer := httptest.NewServer(proxy)
	defer proxyServer.Close()

	for range 3 {
		resp, err := http.Post(proxyServer.URL+"/v1/chat/completions?stream=true", "application/json", bytes.NewBufferString(`{"model":"zhipu/glm-5.1","messages":[{"role":"user","content":"ping"}]}`))
		if err != nil {
			t.Fatalf("http.Post returned error: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			t.Fatalf("StatusCode = %d, body = %s", resp.StatusCode, string(body))
		}
	}

	mu.Lock()
	defer mu.Unlock()

	if len(requests) != 3 {
		t.Fatalf("captured %d requests, want 3", len(requests))
	}

	if requests[0].Path != "/api/paas/v4/chat/completions" {
		t.Fatalf("first path = %q, want %q", requests[0].Path, "/api/paas/v4/chat/completions")
	}
	if requests[0].RawQuery != "stream=true" {
		t.Fatalf("first RawQuery = %q, want %q", requests[0].RawQuery, "stream=true")
	}
	if requests[0].Authorization != "Bearer key-1" {
		t.Fatalf("first Authorization = %q, want %q", requests[0].Authorization, "Bearer key-1")
	}
	if requests[1].Authorization != "Bearer key-2" {
		t.Fatalf("second Authorization = %q, want %q", requests[1].Authorization, "Bearer key-2")
	}
	if requests[2].Authorization != "Bearer key-1" {
		t.Fatalf("third Authorization = %q, want %q", requests[2].Authorization, "Bearer key-1")
	}
	if requests[0].ProviderHeader != "enabled" || requests[1].ProviderHeader != "enabled" || requests[2].ProviderHeader != "enabled" {
		t.Fatalf("provider headers = %#v, want enabled on all requests", requests)
	}
	if requests[0].Model != "glm-5.1" || requests[1].Model != "glm-5.1" || requests[2].Model != "glm-5.1" {
		t.Fatalf("models = %#v, want glm-5.1 on all requests", requests)
	}
}

func TestProxyRejectsMissingProviderPrefix(t *testing.T) {
	t.Parallel()

	proxy, err := NewProxy(config.Config{
		Provider: config.ProviderKeys{
			"zai": {"key-1"},
		},
	}, map[string]ProviderCatalogEntry{
		"zai": {
			BaseURL: mustParseURL(t, "https://api.z.ai/api/coding/paas/v4"),
		},
	}, http.DefaultClient)
	if err != nil {
		t.Fatalf("NewProxy returned error: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewBufferString(`{"model":"glm-5.1"}`))
	resp := httptest.NewRecorder()

	proxy.ServeHTTP(resp, req)

	if resp.Code != http.StatusBadRequest {
		t.Fatalf("Code = %d, want %d", resp.Code, http.StatusBadRequest)
	}

	body := resp.Body.String()
	if !bytes.Contains(resp.Body.Bytes(), []byte("provider/model format")) {
		t.Fatalf("body = %q, want provider/model format error", body)
	}
}
func TestProxyLogsResponseStatusCode(t *testing.T) {
	logs := captureProxyLogs(t)

	client := &http.Client{
		Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusAccepted,
				Header: http.Header{
					"Content-Type": {"application/json"},
				},
				Body:    io.NopCloser(bytes.NewBufferString(`{"ok":true}`)),
				Request: req,
			}, nil
		}),
	}

	proxy, err := NewProxy(config.Config{
		Provider: config.ProviderKeys{
			"zai": {"key-1"},
		},
	}, map[string]ProviderCatalogEntry{
		"zai": {
			BaseURL: mustParseURL(t, "https://api.z.ai/api/coding/paas/v4"),
		},
	}, client)
	if err != nil {
		t.Fatalf("NewProxy returned error: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/v1/logging/success", bytes.NewBufferString(`{"model":"zai/glm-5.1"}`))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()

	proxy.ServeHTTP(resp, req)

	if resp.Code != http.StatusAccepted {
		t.Fatalf("Code = %d, want %d", resp.Code, http.StatusAccepted)
	}

	logText := logs.String()
	if !strings.Contains(logText, `"msg":"proxy response"`) {
		t.Fatalf("logs = %q, want proxy response entry", logText)
	}
	if !strings.Contains(logText, `"path":"/v1/logging/success"`) {
		t.Fatalf("logs = %q, want response path", logText)
	}
	if !strings.Contains(logText, `"status":202`) {
		t.Fatalf("logs = %q, want response status", logText)
	}
}

func TestProxyLogsRejectedRequestStatusCode(t *testing.T) {
	logs := captureProxyLogs(t)

	proxy, err := NewProxy(config.Config{
		Provider: config.ProviderKeys{
			"zai": {"key-1"},
		},
	}, map[string]ProviderCatalogEntry{
		"zai": {
			BaseURL: mustParseURL(t, "https://api.z.ai/api/coding/paas/v4"),
		},
	}, http.DefaultClient)
	if err != nil {
		t.Fatalf("NewProxy returned error: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/v1/logging/rejected", bytes.NewBufferString(`{"model":"glm-5.1"}`))
	resp := httptest.NewRecorder()

	proxy.ServeHTTP(resp, req)

	if resp.Code != http.StatusBadRequest {
		t.Fatalf("Code = %d, want %d", resp.Code, http.StatusBadRequest)
	}

	logText := logs.String()
	if !strings.Contains(logText, `"msg":"proxy response"`) {
		t.Fatalf("logs = %q, want proxy response entry", logText)
	}
	if !strings.Contains(logText, `"path":"/v1/logging/rejected"`) {
		t.Fatalf("logs = %q, want response path", logText)
	}
	if !strings.Contains(logText, `"status":400`) {
		t.Fatalf("logs = %q, want response status", logText)
	}
}
func TestProxyRetries429WithNextKeyAndLogsTransition(t *testing.T) {
	logs := captureProxyLogs(t)

	var authorizations []string
	client := &http.Client{
		Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			authorizations = append(authorizations, req.Header.Get("Authorization"))

			status := http.StatusAccepted
			body := `{"ok":true}`
			if len(authorizations) == 1 {
				status = http.StatusTooManyRequests
				body = `{"error":"rate limited"}`
			}

			return &http.Response{
				StatusCode: status,
				Header: http.Header{
					"Content-Type": {"application/json"},
				},
				Body:    io.NopCloser(bytes.NewBufferString(body)),
				Request: req,
			}, nil
		}),
	}

	proxy, err := NewProxy(config.Config{
		Runtime: config.Runtime{
			Upstream429Retries: 1,
		},
		Provider: config.ProviderKeys{
			"zai": {"key-1", "key-2"},
		},
	}, map[string]ProviderCatalogEntry{
		"zai": {
			BaseURL: mustParseURL(t, "https://api.z.ai/api/coding/paas/v4"),
		},
	}, client)
	if err != nil {
		t.Fatalf("NewProxy returned error: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewBufferString(`{"model":"zai/glm-5.1"}`))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()

	proxy.ServeHTTP(resp, req)

	if resp.Code != http.StatusAccepted {
		t.Fatalf("Code = %d, want %d", resp.Code, http.StatusAccepted)
	}
	if got := resp.Body.String(); got != `{"ok":true}` {
		t.Fatalf("Body = %q, want success body from retried request", got)
	}
	if len(authorizations) != 2 {
		t.Fatalf("authorization attempts = %d, want 2", len(authorizations))
	}
	if authorizations[0] != "Bearer key-1" {
		t.Fatalf("first Authorization = %q, want %q", authorizations[0], "Bearer key-1")
	}
	if authorizations[1] != "Bearer key-2" {
		t.Fatalf("second Authorization = %q, want %q", authorizations[1], "Bearer key-2")
	}

	logText := logs.String()
	if !strings.Contains(logText, `"msg":"upstream returned 429, retrying with next key"`) {
		t.Fatalf("logs = %q, want 429 retry log entry", logText)
	}
	if !strings.Contains(logText, `"status":429`) {
		t.Fatalf("logs = %q, want 429 status in retry log", logText)
	}
	if !strings.Contains(logText, `"attempt":1`) {
		t.Fatalf("logs = %q, want retry attempt counter", logText)
	}
}

func TestProxyStopsRetryingAfterConfigured429Limit(t *testing.T) {
	t.Parallel()

	var authorizations []string
	client := &http.Client{
		Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			authorizations = append(authorizations, req.Header.Get("Authorization"))
			return &http.Response{
				StatusCode: http.StatusTooManyRequests,
				Header: http.Header{
					"Content-Type": {"application/json"},
				},
				Body:    io.NopCloser(bytes.NewBufferString(`{"error":"rate limited"}`)),
				Request: req,
			}, nil
		}),
	}

	proxy, err := NewProxy(config.Config{
		Runtime: config.Runtime{
			Upstream429Retries: 1,
		},
		Provider: config.ProviderKeys{
			"zai": {"key-1", "key-2", "key-3"},
		},
	}, map[string]ProviderCatalogEntry{
		"zai": {
			BaseURL: mustParseURL(t, "https://api.z.ai/api/coding/paas/v4"),
		},
	}, client)
	if err != nil {
		t.Fatalf("NewProxy returned error: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewBufferString(`{"model":"zai/glm-5.1"}`))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()

	proxy.ServeHTTP(resp, req)

	if resp.Code != http.StatusTooManyRequests {
		t.Fatalf("Code = %d, want %d", resp.Code, http.StatusTooManyRequests)
	}
	if len(authorizations) != 2 {
		t.Fatalf("authorization attempts = %d, want 2", len(authorizations))
	}
	if authorizations[0] != "Bearer key-1" {
		t.Fatalf("first Authorization = %q, want %q", authorizations[0], "Bearer key-1")
	}
	if authorizations[1] != "Bearer key-2" {
		t.Fatalf("second Authorization = %q, want %q", authorizations[1], "Bearer key-2")
	}
}
func TestStatusCapturingResponseWriterPreservesFlush(t *testing.T) {
	recorder := &flushCountingResponseWriter{ResponseRecorder: httptest.NewRecorder()}
	writer := &statusCapturingResponseWriter{ResponseWriter: recorder}

	if err := streamResponse(writer, bytes.NewBufferString("chunk")); err != nil {
		t.Fatalf("streamResponse returned error: %v", err)
	}
	if recorder.flushes == 0 {
		t.Fatalf("flushes = %d, want at least one flush", recorder.flushes)
	}
	if writer.StatusCode() != http.StatusOK {
		t.Fatalf("StatusCode() = %d, want %d", writer.StatusCode(), http.StatusOK)
	}
}
func TestProxyPreservesJSONErrorContentTypeWithoutUpstreamHeader(t *testing.T) {
	t.Parallel()

	body := `{"error":{"code":"1113","message":"Insufficient balance"}}`
	client := &http.Client{
		Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusTooManyRequests,
				Header:     make(http.Header),
				Body:       io.NopCloser(bytes.NewBufferString(body)),
				Request:    req,
			}, nil
		}),
	}

	proxy, err := NewProxy(config.Config{
		Provider: config.ProviderKeys{
			"zai": {"key-1"},
		},
	}, map[string]ProviderCatalogEntry{
		"zai": {
			BaseURL: mustParseURL(t, "https://api.z.ai/api/coding/paas/v4"),
		},
	}, client)
	if err != nil {
		t.Fatalf("NewProxy returned error: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewBufferString(`{"model":"zai/glm-5.1","stream":true}`))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()

	proxy.ServeHTTP(resp, req)

	if resp.Code != http.StatusTooManyRequests {
		t.Fatalf("Code = %d, want %d", resp.Code, http.StatusTooManyRequests)
	}
	if got := resp.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("Content-Type = %q, want %q", got, "application/json")
	}
	if got := resp.Body.String(); got != body {
		t.Fatalf("Body = %q, want upstream JSON error", got)
	}
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (fn roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func mustParseURL(t *testing.T, raw string) *url.URL {
	t.Helper()

	parsed, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("url.Parse returned error: %v", err)
	}
	return parsed
}
func captureProxyLogs(t *testing.T) *bytes.Buffer {
	t.Helper()

	var buffer bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&buffer, nil)))
	t.Cleanup(func() {
		slog.SetDefault(previous)
	})
	return &buffer
}

type flushCountingResponseWriter struct {
	*httptest.ResponseRecorder
	flushes int
}

func (w *flushCountingResponseWriter) Flush() {
	w.flushes++
	w.ResponseRecorder.Flush()
}
