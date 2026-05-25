package proxy

import (
	"fmt"
	"maps"
	"net/url"
	"strings"

	"charm.land/catwalk/pkg/catwalk"
	"charm.land/catwalk/pkg/embedded"

	"keyprox/pkg/config"
)

type ProviderCatalogEntry struct {
	BaseURL        *url.URL
	DefaultHeaders map[string]string
}

var supportedCatwalkProviderTypes = map[catwalk.Type]struct{}{
	catwalk.TypeOpenAI:       {},
	catwalk.TypeOpenAICompat: {},
	catwalk.TypeOpenRouter:   {},
}

func LoadProviderDefaults() (config.Providers, error) {
	providers := embedded.GetAll()
	defaults := make(config.Providers, len(providers))
	for _, provider := range providers {
		if !supportsOpenAIProxy(provider) {
			continue
		}

		rawURL := strings.TrimSpace(provider.APIEndpoint)
		if !isStaticHTTPEndpoint(rawURL) {
			continue
		}

		defaults[normalizeProviderID(string(provider.ID))] = config.ProviderConfig{
			Endpoints:      []string{rawURL},
			DefaultHeaders: cloneStringMap(provider.DefaultHeaders),
			Keys:           []string{},
		}
	}

	if len(defaults) == 0 {
		return nil, fmt.Errorf("catwalk embedded catalog contains no compatible providers with static endpoints")
	}

	return defaults, nil
}

func LoadCatalog() (map[string]ProviderCatalogEntry, error) {
	defaults, err := LoadProviderDefaults()
	if err != nil {
		return nil, err
	}

	catalog := make(map[string]ProviderCatalogEntry, len(defaults))
	for providerID, provider := range defaults {
		if len(provider.Endpoints) == 0 {
			return nil, fmt.Errorf("provider %q has no endpoints", providerID)
		}

		entry, err := newProviderCatalogEntry(provider.Endpoints[0], provider.DefaultHeaders)
		if err != nil {
			return nil, fmt.Errorf("provider %q endpoint %q: %w", providerID, provider.Endpoints[0], err)
		}
		catalog[providerID] = entry
	}

	return catalog, nil
}

func supportsOpenAIProxy(provider catwalk.Provider) bool {
	_, ok := supportedCatwalkProviderTypes[provider.Type]
	return ok
}

func newProviderCatalogEntry(rawURL string, defaultHeaders map[string]string) (ProviderCatalogEntry, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return ProviderCatalogEntry{}, err
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return ProviderCatalogEntry{}, fmt.Errorf("invalid HTTP endpoint")
	}

	return ProviderCatalogEntry{
		BaseURL:        parsed,
		DefaultHeaders: cloneStringMap(defaultHeaders),
	}, nil
}

func cloneStringMap(src map[string]string) map[string]string {
	if len(src) == 0 {
		return nil
	}

	dst := make(map[string]string, len(src))
	maps.Copy(dst, src)
	return dst
}

func isStaticHTTPEndpoint(endpoint string) bool {
	if !strings.HasPrefix(endpoint, "http://") && !strings.HasPrefix(endpoint, "https://") {
		return false
	}
	if strings.Contains(endpoint, "${") || strings.ContainsAny(endpoint, "{}") {
		return false
	}
	return true
}

/*
Legacy opencode snapshot catalog from opencode-providers-endpoints.md.
Uncomment loadLegacyCatalog and switch LoadCatalog to it if you want the previous provider table.

func loadLegacyCatalog() map[string]ProviderCatalogEntry {
	return map[string]ProviderCatalogEntry{
		"302ai": mustNewProviderCatalogEntry("https://api.302.ai/v1", nil),
		"abacus": mustNewProviderCatalogEntry("https://routellm.abacus.ai/v1", nil),
		"abliteration-ai": mustNewProviderCatalogEntry("https://api.abliteration.ai/v1", nil),
		"alibaba": mustNewProviderCatalogEntry("https://dashscope-intl.aliyuncs.com/compatible-mode/v1", nil),
		"alibaba-cn": mustNewProviderCatalogEntry("https://dashscope.aliyuncs.com/compatible-mode/v1", nil),
		"alibaba-coding-plan": mustNewProviderCatalogEntry("https://coding-intl.dashscope.aliyuncs.com/v1", nil),
		"alibaba-coding-plan-cn": mustNewProviderCatalogEntry("https://coding.dashscope.aliyuncs.com/v1", nil),
		"ambient": mustNewProviderCatalogEntry("https://api.ambient.xyz/v1", nil),
		"atomic-chat": mustNewProviderCatalogEntry("http://127.0.0.1:1337/v1", nil),
		"auriko": mustNewProviderCatalogEntry("https://api.auriko.ai/v1", nil),
		"bailing": mustNewProviderCatalogEntry("https://api.tbox.cn/api/llm/v1/chat/completions", nil),
		"baseten": mustNewProviderCatalogEntry("https://inference.baseten.co/v1", nil),
		"berget": mustNewProviderCatalogEntry("https://api.berget.ai/v1", nil),
		"chutes": mustNewProviderCatalogEntry("https://llm.chutes.ai/v1", nil),
		"clarifai": mustNewProviderCatalogEntry("https://api.clarifai.com/v2/ext/openai/v1", nil),
		"claudinio": mustNewProviderCatalogEntry("https://api.claudin.io/v1", nil),
		"cloudferro-sherlock": mustNewProviderCatalogEntry("https://api-sherlock.cloudferro.com/openai/v1/", nil),
		"cortecs": mustNewProviderCatalogEntry("https://api.cortecs.ai/v1", nil),
		"crof": mustNewProviderCatalogEntry("https://crof.ai/v1", nil),
		"deepseek": mustNewProviderCatalogEntry("https://api.deepseek.com", nil),
		"digitalocean": mustNewProviderCatalogEntry("https://inference.do-ai.run/v1", nil),
		"dinference": mustNewProviderCatalogEntry("https://api.dinference.com/v1", nil),
		"drun": mustNewProviderCatalogEntry("https://chat.d.run/v1", nil),
		"evroc": mustNewProviderCatalogEntry("https://models.think.evroc.com/v1", nil),
		"fastrouter": mustNewProviderCatalogEntry("https://go.fastrouter.ai/api/v1", nil),
		"firepass": mustNewProviderCatalogEntry("https://api.fireworks.ai/inference/v1/", nil),
		"fireworks-ai": mustNewProviderCatalogEntry("https://api.fireworks.ai/inference/v1/", nil),
		"friendli": mustNewProviderCatalogEntry("https://api.friendli.ai/serverless/v1", nil),
		"frogbot": mustNewProviderCatalogEntry("https://app.frogbot.ai/api/v1", nil),
		"github-copilot": mustNewProviderCatalogEntry("https://api.githubcopilot.com", nil),
		"github-models": mustNewProviderCatalogEntry("https://models.github.ai/inference", nil),
		"gmicloud": mustNewProviderCatalogEntry("https://api.gmi-serving.com/v1", nil),
		"helicone": mustNewProviderCatalogEntry("https://ai-gateway.helicone.ai/v1", nil),
		"hpc-ai": mustNewProviderCatalogEntry("https://api.hpc-ai.com/inference/v1", nil),
		"huggingface": mustNewProviderCatalogEntry("https://router.huggingface.co/v1", nil),
		"iflowcn": mustNewProviderCatalogEntry("https://apis.iflow.cn/v1", nil),
		"inception": mustNewProviderCatalogEntry("https://api.inceptionlabs.ai/v1/", nil),
		"inceptron": mustNewProviderCatalogEntry("https://api.inceptron.io/v1", nil),
		"inference": mustNewProviderCatalogEntry("https://inference.net/v1", nil),
		"io-net": mustNewProviderCatalogEntry("https://api.intelligence.io.solutions/api/v1", nil),
		"jiekou": mustNewProviderCatalogEntry("https://api.jiekou.ai/openai", nil),
		"kilo": mustNewProviderCatalogEntry("https://api.kilo.ai/api/gateway", nil),
		"kimi-for-coding": mustNewProviderCatalogEntry("https://api.kimi.com/coding/v1", nil),
		"kuae-cloud-coding-plan": mustNewProviderCatalogEntry("https://coding-plan-endpoint.kuaecloud.net/v1", nil),
		"lilac": mustNewProviderCatalogEntry("https://api.getlilac.com/v1", nil),
		"llama": mustNewProviderCatalogEntry("https://api.llama.com/compat/v1/", nil),
		"llmgateway": mustNewProviderCatalogEntry("https://api.llmgateway.io/v1", nil),
		"lmstudio": mustNewProviderCatalogEntry("http://127.0.0.1:1234/v1", nil),
		"lucidquery": mustNewProviderCatalogEntry("https://lucidquery.com/api/v1", nil),
		"meganova": mustNewProviderCatalogEntry("https://api.meganova.ai/v1", nil),
		"minimax": mustNewProviderCatalogEntry("https://api.minimax.io/anthropic/v1", nil),
		"minimax-cn": mustNewProviderCatalogEntry("https://api.minimaxi.com/anthropic/v1", nil),
		"minimax-cn-coding-plan": mustNewProviderCatalogEntry("https://api.minimaxi.com/anthropic/v1", nil),
		"minimax-coding-plan": mustNewProviderCatalogEntry("https://api.minimax.io/anthropic/v1", nil),
		"mixlayer": mustNewProviderCatalogEntry("https://models.mixlayer.ai/v1", nil),
		"moark": mustNewProviderCatalogEntry("https://moark.com/v1", nil),
		"modelscope": mustNewProviderCatalogEntry("https://api-inference.modelscope.cn/v1", nil),
		"moonshotai": mustNewProviderCatalogEntry("https://api.moonshot.ai/v1", nil),
		"moonshotai-cn": mustNewProviderCatalogEntry("https://api.moonshot.cn/v1", nil),
		"morph": mustNewProviderCatalogEntry("https://api.morphllm.com/v1", nil),
		"nano-gpt": mustNewProviderCatalogEntry("https://nano-gpt.com/api/v1", nil),
		"nearai": mustNewProviderCatalogEntry("https://cloud-api.near.ai/v1", nil),
		"nebius": mustNewProviderCatalogEntry("https://api.tokenfactory.nebius.com/v1", nil),
		"neuralwatt": mustNewProviderCatalogEntry("https://api.neuralwatt.com/v1", nil),
		"nova": mustNewProviderCatalogEntry("https://api.nova.amazon.com/v1", nil),
		"novita-ai": mustNewProviderCatalogEntry("https://api.novita.ai/openai", nil),
		"nvidia": mustNewProviderCatalogEntry("https://integrate.api.nvidia.com/v1", nil),
		"ollama-cloud": mustNewProviderCatalogEntry("https://ollama.com/v1", nil),
		"opencode": mustNewProviderCatalogEntry("https://opencode.ai/zen/v1", nil),
		"opencode-go": mustNewProviderCatalogEntry("https://opencode.ai/zen/go/v1", nil),
		"openrouter": mustNewProviderCatalogEntry("https://openrouter.ai/api/v1", nil),
		"orcarouter": mustNewProviderCatalogEntry("https://api.orcarouter.ai/v1", nil),
		"ovhcloud": mustNewProviderCatalogEntry("https://oai.endpoints.kepler.ai.cloud.ovh.net/v1", nil),
		"perplexity-agent": mustNewProviderCatalogEntry("https://api.perplexity.ai/v1", nil),
		"poe": mustNewProviderCatalogEntry("https://api.poe.com/v1", nil),
		"poolside": mustNewProviderCatalogEntry("https://inference.poolside.ai/v1", nil),
		"privatemode-ai": mustNewProviderCatalogEntry("http://localhost:8080/v1", nil),
		"qihang-ai": mustNewProviderCatalogEntry("https://api.qhaigc.net/v1", nil),
		"qiniu-ai": mustNewProviderCatalogEntry("https://api.qnaigc.com/v1", nil),
		"regolo-ai": mustNewProviderCatalogEntry("https://api.regolo.ai/v1", nil),
		"requesty": mustNewProviderCatalogEntry("https://router.requesty.ai/v1", nil),
		"routing-run": mustNewProviderCatalogEntry("https://api.routing.run/v1", nil),
		"sarvam": mustNewProviderCatalogEntry("https://api.sarvam.ai/v1", nil),
		"scaleway": mustNewProviderCatalogEntry("https://api.scaleway.ai/v1", nil),
		"siliconflow": mustNewProviderCatalogEntry("https://api.siliconflow.com/v1", nil),
		"siliconflow-cn": mustNewProviderCatalogEntry("https://api.siliconflow.cn/v1", nil),
		"stackit": mustNewProviderCatalogEntry("https://api.openai-compat.model-serving.eu01.onstackit.cloud/v1", nil),
		"stepfun": mustNewProviderCatalogEntry("https://api.stepfun.com/v1", nil),
		"stepfun-ai": mustNewProviderCatalogEntry("https://api.stepfun.ai/step_plan/v1", nil),
		"submodel": mustNewProviderCatalogEntry("https://llm.submodel.ai/v1", nil),
		"synthetic": mustNewProviderCatalogEntry("https://api.synthetic.new/openai/v1", nil),
		"tencent-coding-plan": mustNewProviderCatalogEntry("https://api.lkeap.cloud.tencent.com/coding/v3", nil),
		"tencent-tokenhub": mustNewProviderCatalogEntry("https://tokenhub.tencentmaas.com/v1", nil),
		"the-grid-ai": mustNewProviderCatalogEntry("https://api.thegrid.ai/v1", nil),
		"umans-ai-coding-plan": mustNewProviderCatalogEntry("https://api.code.umans.ai/v1", nil),
		"upstage": mustNewProviderCatalogEntry("https://api.upstage.ai/v1/solar", nil),
		"vivgrid": mustNewProviderCatalogEntry("https://api.vivgrid.com/v1", nil),
		"vultr": mustNewProviderCatalogEntry("https://api.vultrinference.com/v1", nil),
		"wafer.ai": mustNewProviderCatalogEntry("https://pass.wafer.ai/v1", nil),
		"wandb": mustNewProviderCatalogEntry("https://api.inference.wandb.ai/v1", nil),
		"xiaomi": mustNewProviderCatalogEntry("https://api.xiaomimimo.com/v1", nil),
		"xiaomi-token-plan-ams": mustNewProviderCatalogEntry("https://token-plan-ams.xiaomimimo.com/v1", nil),
		"xiaomi-token-plan-cn": mustNewProviderCatalogEntry("https://token-plan-cn.xiaomimimo.com/v1", nil),
		"xiaomi-token-plan-sgp": mustNewProviderCatalogEntry("https://token-plan-sgp.xiaomimimo.com/v1", nil),
		"xpersona": mustNewProviderCatalogEntry("https://www.xpersona.co/v1", nil),
		"zai": mustNewProviderCatalogEntry("https://api.z.ai/api/paas/v4", nil),
		"zai-coding-plan": mustNewProviderCatalogEntry("https://api.z.ai/api/coding/paas/v4", nil),
		"zenmux": mustNewProviderCatalogEntry("https://zenmux.ai/api/v1", nil),
		"zhipuai": mustNewProviderCatalogEntry("https://open.bigmodel.cn/api/paas/v4", nil),
		"zhipuai-coding-plan": mustNewProviderCatalogEntry("https://open.bigmodel.cn/api/coding/paas/v4", nil),
	}
}
*/
