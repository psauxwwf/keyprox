# OpenCode Providers And Endpoints

Source snapshot:
- Repo: `https://github.com/anomalyco/opencode`
- Provider catalog loaded via `packages/core/src/plugin/models-dev.ts`
- Runtime provider plugins registered in `packages/core/src/plugin/provider/index.ts`
- `models.dev` provider count at capture time: `135`

## Main Providers

| ID | Provider | Endpoint |
| --- | --- | --- |
| 302ai | 302.AI | `https://api.302.ai/v1` |
| abacus | Abacus | `https://routellm.abacus.ai/v1` |
| abliteration-ai | abliteration.ai | `https://api.abliteration.ai/v1` |
| aihubmix | AIHubMix | `not declared in catalog` |
| alibaba | Alibaba | `https://dashscope-intl.aliyuncs.com/compatible-mode/v1` |
| alibaba-cn | Alibaba (China) | `https://dashscope.aliyuncs.com/compatible-mode/v1` |
| alibaba-coding-plan | Alibaba Coding Plan | `https://coding-intl.dashscope.aliyuncs.com/v1` |
| alibaba-coding-plan-cn | Alibaba Coding Plan (China) | `https://coding.dashscope.aliyuncs.com/v1` |
| amazon-bedrock | Amazon Bedrock | `derived from AWS SDK; optional custom endpoint via provider.options.endpoint/baseURL` |
| ambient | Ambient | `https://api.ambient.xyz/v1` |
| anthropic | Anthropic | `SDK default (no fixed base URL in catalog)` |
| atomic-chat | Atomic Chat | `http://127.0.0.1:1337/v1` |
| auriko | Auriko | `https://api.auriko.ai/v1` |
| azure | Azure | `derived from resourceName / baseURL; plugin requires AZURE_RESOURCE_NAME or explicit baseURL` |
| azure-cognitive-services | Azure Cognitive Services | `https://${AZURE_COGNITIVE_SERVICES_RESOURCE_NAME}.cognitiveservices.azure.com/openai` |
| bailing | Bailing | `https://api.tbox.cn/api/llm/v1/chat/completions` |
| baseten | Baseten | `https://inference.baseten.co/v1` |
| berget | Berget.AI | `https://api.berget.ai/v1` |
| cerebras | Cerebras | `SDK default (no fixed base URL in catalog)` |
| chutes | Chutes | `https://llm.chutes.ai/v1` |
| clarifai | Clarifai | `https://api.clarifai.com/v2/ext/openai/v1` |
| claudinio | Claudinio | `https://api.claudin.io/v1` |
| cloudferro-sherlock | CloudFerro Sherlock | `https://api-sherlock.cloudferro.com/openai/v1/` |
| cloudflare-ai-gateway | Cloudflare AI Gateway | `derived from CLOUDFLARE_ACCOUNT_ID + CLOUDFLARE_GATEWAY_ID` |
| cloudflare-workers-ai | Cloudflare Workers AI | `https://api.cloudflare.com/client/v4/accounts/${CLOUDFLARE_ACCOUNT_ID}/ai/v1` |
| cohere | Cohere | `SDK default (no fixed base URL in catalog)` |
| cortecs | Cortecs | `https://api.cortecs.ai/v1` |
| crof | CrofAI | `https://crof.ai/v1` |
| databricks | Databricks | `https://${DATABRICKS_HOST}/ai-gateway/mlflow/v1` |
| deepinfra | Deep Infra | `SDK default (no fixed base URL in catalog)` |
| deepseek | DeepSeek | `https://api.deepseek.com` |
| digitalocean | DigitalOcean | `https://inference.do-ai.run/v1` |
| dinference | DInference | `https://api.dinference.com/v1` |
| drun | D.Run (China) | `https://chat.d.run/v1` |
| evroc | evroc | `https://models.think.evroc.com/v1` |
| fastrouter | FastRouter | `https://go.fastrouter.ai/api/v1` |
| firepass | Fireworks (Firepass) | `https://api.fireworks.ai/inference/v1/` |
| fireworks-ai | Fireworks AI | `https://api.fireworks.ai/inference/v1/` |
| friendli | Friendli | `https://api.friendli.ai/serverless/v1` |
| frogbot | FrogBot | `https://app.frogbot.ai/api/v1` |
| github-copilot | GitHub Copilot | `https://api.githubcopilot.com` |
| github-models | GitHub Models | `https://models.github.ai/inference` |
| gitlab | GitLab Duo | `instanceUrl from config or GITLAB_INSTANCE_URL, default https://gitlab.com` |
| gmicloud | GMI Cloud | `https://api.gmi-serving.com/v1` |
| google | Google | `SDK default (no fixed base URL in catalog)` |
| google-vertex | Vertex | `derived from project/location; templates expanded by plugin` |
| google-vertex-anthropic | Vertex (Anthropic) | `SDK default; for eu/us plugin uses Regional Endpoint Platform URL` |
| groq | Groq | `SDK default (no fixed base URL in catalog)` |
| helicone | Helicone | `https://ai-gateway.helicone.ai/v1` |
| hpc-ai | HPC-AI | `https://api.hpc-ai.com/inference/v1` |
| huggingface | Hugging Face | `https://router.huggingface.co/v1` |
| iflowcn | iFlow | `https://apis.iflow.cn/v1` |
| inception | Inception | `https://api.inceptionlabs.ai/v1/` |
| inceptron | Inceptron | `https://api.inceptron.io/v1` |
| inference | Inference | `https://inference.net/v1` |
| io-net | IO.NET | `https://api.intelligence.io.solutions/api/v1` |
| jiekou | Jiekou.AI | `https://api.jiekou.ai/openai` |
| kilo | Kilo Gateway | `https://api.kilo.ai/api/gateway` |
| kimi-for-coding | Kimi For Coding | `https://api.kimi.com/coding/v1` |
| kuae-cloud-coding-plan | KUAE Cloud Coding Plan | `https://coding-plan-endpoint.kuaecloud.net/v1` |
| lilac | Lilac | `https://api.getlilac.com/v1` |
| llama | Llama | `https://api.llama.com/compat/v1/` |
| llmgateway | LLM Gateway | `https://api.llmgateway.io/v1` |
| lmstudio | LMStudio | `http://127.0.0.1:1234/v1` |
| lucidquery | LucidQuery AI | `https://lucidquery.com/api/v1` |
| meganova | Meganova | `https://api.meganova.ai/v1` |
| minimax | MiniMax (minimax.io) | `https://api.minimax.io/anthropic/v1` |
| minimax-cn | MiniMax (minimaxi.com) | `https://api.minimaxi.com/anthropic/v1` |
| minimax-cn-coding-plan | MiniMax Token Plan (minimaxi.com) | `https://api.minimaxi.com/anthropic/v1` |
| minimax-coding-plan | MiniMax Token Plan (minimax.io) | `https://api.minimax.io/anthropic/v1` |
| mistral | Mistral | `SDK default (no fixed base URL in catalog)` |
| mixlayer | Mixlayer | `https://models.mixlayer.ai/v1` |
| moark | Moark | `https://moark.com/v1` |
| modelscope | ModelScope | `https://api-inference.modelscope.cn/v1` |
| moonshotai | Moonshot AI | `https://api.moonshot.ai/v1` |
| moonshotai-cn | Moonshot AI (China) | `https://api.moonshot.cn/v1` |
| morph | Morph | `https://api.morphllm.com/v1` |
| nano-gpt | NanoGPT | `https://nano-gpt.com/api/v1` |
| nearai | NEAR AI Cloud | `https://cloud-api.near.ai/v1` |
| nebius | Nebius Token Factory | `https://api.tokenfactory.nebius.com/v1` |
| neuralwatt | Neuralwatt | `https://api.neuralwatt.com/v1` |
| nova | Nova | `https://api.nova.amazon.com/v1` |
| novita-ai | NovitaAI | `https://api.novita.ai/openai` |
| nvidia | Nvidia | `https://integrate.api.nvidia.com/v1` |
| ollama-cloud | Ollama Cloud | `https://ollama.com/v1` |
| openai | OpenAI | `SDK default (no fixed base URL in catalog)` |
| opencode | OpenCode Zen | `https://opencode.ai/zen/v1` |
| opencode-go | OpenCode Go | `https://opencode.ai/zen/go/v1` |
| openrouter | OpenRouter | `https://openrouter.ai/api/v1` |
| orcarouter | OrcaRouter | `https://api.orcarouter.ai/v1` |
| ovhcloud | OVHcloud AI Endpoints | `https://oai.endpoints.kepler.ai.cloud.ovh.net/v1` |
| perplexity | Perplexity | `SDK default (no fixed base URL in catalog)` |
| perplexity-agent | Perplexity Agent | `https://api.perplexity.ai/v1` |
| poe | Poe | `https://api.poe.com/v1` |
| poolside | Poolside | `https://inference.poolside.ai/v1` |
| privatemode-ai | Privatemode AI | `http://localhost:8080/v1` |
| qihang-ai | QiHang | `https://api.qhaigc.net/v1` |
| qiniu-ai | Qiniu | `https://api.qnaigc.com/v1` |
| regolo-ai | Regolo AI | `https://api.regolo.ai/v1` |
| requesty | Requesty | `https://router.requesty.ai/v1` |
| routing-run | routing.run | `https://api.routing.run/v1` |
| sap-ai-core | SAP AI Core | `derived from AICORE_SERVICE_KEY and provider config` |
| sarvam | Sarvam AI | `https://api.sarvam.ai/v1` |
| scaleway | Scaleway | `https://api.scaleway.ai/v1` |
| siliconflow | SiliconFlow | `https://api.siliconflow.com/v1` |
| siliconflow-cn | SiliconFlow (China) | `https://api.siliconflow.cn/v1` |
| stackit | STACKIT | `https://api.openai-compat.model-serving.eu01.onstackit.cloud/v1` |
| stepfun | StepFun | `https://api.stepfun.com/v1` |
| stepfun-ai | StepFun | `https://api.stepfun.ai/step_plan/v1` |
| submodel | submodel | `https://llm.submodel.ai/v1` |
| synthetic | Synthetic | `https://api.synthetic.new/openai/v1` |
| tencent-coding-plan | Tencent Coding Plan (China) | `https://api.lkeap.cloud.tencent.com/coding/v3` |
| tencent-tokenhub | Tencent TokenHub | `https://tokenhub.tencentmaas.com/v1` |
| the-grid-ai | The Grid AI | `https://api.thegrid.ai/v1` |
| togetherai | Together AI | `SDK default (no fixed base URL in catalog)` |
| umans-ai-coding-plan | Umans AI Coding Plan | `https://api.code.umans.ai/v1` |
| upstage | Upstage | `https://api.upstage.ai/v1/solar` |
| v0 | v0 | `SDK default (no fixed base URL in catalog)` |
| venice | Venice AI | `SDK default (no fixed base URL in catalog)` |
| vercel | Vercel AI Gateway | `SDK default (AI Gateway; no fixed base URL in catalog)` |
| vivgrid | Vivgrid | `https://api.vivgrid.com/v1` |
| vultr | Vultr | `https://api.vultrinference.com/v1` |
| wafer.ai | Wafer | `https://pass.wafer.ai/v1` |
| wandb | Weights & Biases | `https://api.inference.wandb.ai/v1` |
| xai | xAI | `SDK default (no fixed base URL in catalog)` |
| xiaomi | Xiaomi | `https://api.xiaomimimo.com/v1` |
| xiaomi-token-plan-ams | Xiaomi Token Plan (Europe) | `https://token-plan-ams.xiaomimimo.com/v1` |
| xiaomi-token-plan-cn | Xiaomi Token Plan (China) | `https://token-plan-cn.xiaomimimo.com/v1` |
| xiaomi-token-plan-sgp | Xiaomi Token Plan (Singapore) | `https://token-plan-sgp.xiaomimimo.com/v1` |
| xpersona | Xpersona | `https://www.xpersona.co/v1` |
| zai | Z.AI | `https://api.z.ai/api/paas/v4` |
| zai-coding-plan | Z.AI Coding Plan | `https://api.z.ai/api/coding/paas/v4` |
| zenmux | ZenMux | `https://zenmux.ai/api/v1` |
| zhipuai | Zhipu AI | `https://open.bigmodel.cn/api/paas/v4` |
| zhipuai-coding-plan | Zhipu AI Coding Plan | `https://open.bigmodel.cn/api/coding/paas/v4` |

## Service Providers From Code Only

| ID | Description | Endpoint |
| --- | --- | --- |
| gateway | Generic wrapper around `@ai-sdk/gateway` | `configured in provider options / SDK` |
| openai-compatible | Generic wrapper around `@ai-sdk/openai-compatible` | `uses baseURL from concrete provider` |
| dynamic-provider | Dynamically loads any npm/file provider package | `depends on loaded package` |
