package proxy

import "testing"

func TestLoadCatalogUsesCatwalkProviders(t *testing.T) {
	t.Parallel()

	catalog, err := LoadCatalog()
	if err != nil {
		t.Fatalf("LoadCatalog returned error: %v", err)
	}

	for provider, want := range map[string]string{
		"deepseek":    "https://api.deepseek.com/v1",
		"openrouter":  "https://openrouter.ai/api/v1",
		"zai":         "https://api.z.ai/api/coding/paas/v4",
		"opencode-go": "https://opencode.ai/zen/go/v1",
	} {
		entry, ok := catalog[provider]
		if !ok {
			t.Fatalf("catalog missing provider %q", provider)
		}
		if got := entry.BaseURL.String(); got != want {
			t.Fatalf("catalog[%s].BaseURL = %q, want %q", provider, got, want)
		}
	}

	if got := catalog["cerebras"].DefaultHeaders["X-Cerebras-3rd-Party-Integration"]; got != "crush" {
		t.Fatalf("cerebras default header = %q, want %q", got, "crush")
	}

	for _, provider := range []string{"anthropic", "azure", "gemini", "openai", "opencode", "zhipuai"} {
		if _, ok := catalog[provider]; ok {
			t.Fatalf("catalog unexpectedly contains %q", provider)
		}
	}
}

func TestLoadProviderDefaultsReturnsConfigShape(t *testing.T) {
	t.Parallel()

	providers, err := LoadProviderDefaults()
	if err != nil {
		t.Fatalf("LoadProviderDefaults returned error: %v", err)
	}

	zai, ok := providers["zai"]
	if !ok {
		t.Fatalf("providers missing zai")
	}
	if len(zai.Endpoints) != 1 || zai.Endpoints[0] != "https://api.z.ai/api/coding/paas/v4" {
		t.Fatalf("zai endpoints = %#v, want z.ai endpoint", zai.Endpoints)
	}
	if len(zai.Keys) != 0 {
		t.Fatalf("zai keys = %#v, want empty list", zai.Keys)
	}
	if got := providers["cerebras"].DefaultHeaders["X-Cerebras-3rd-Party-Integration"]; got != "crush" {
		t.Fatalf("cerebras default header = %q, want %q", got, "crush")
	}
}
