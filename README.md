# keyprox

`keyprox` is a local HTTP proxy for OpenAI-compatible APIs with API key rotation across providers.

## What the project does

The proxy accepts a regular OpenAI-compatible request on the local `/v1/...` endpoint, reads the `model` field, expects the format `provider/model`, and then:

- detects the provider from the `provider` prefix;
- strips that prefix from `model` before forwarding upstream;
- resolves the provider endpoint from the saved config;
- picks the provider API key using round-robin;
- proxies the response back to the client, including streaming responses.

Example:

- input: `model: "zai/glm-5.1"`
- upstream request: `model: "glm-5.1"`
- the API key is taken from `provider.zai.keys`
- the upstream base URL is taken from `provider.zai.endpoints`

## Why it exists

The project is useful when you need to:

- expose a single local OpenAI-compatible endpoint for multiple providers;
- keep provider endpoints and keys in one editable config file;
- rotate keys for one provider without changing the client;
- survive per-key rate limits;
- use one local base URL in tools such as OpenCode.

## How 429 retry works

If the upstream returns `429 Too Many Requests`, the proxy:

1. does not return that response to the client immediately;
2. switches to the next key for the same provider;
3. retries the exact same request;
4. writes a warning log about moving to the next key.

The number of such transitions is controlled by `runtime.upstream_429_retries`.
If the retry budget is exhausted, the client receives the last upstream response.

## Configuration

The main config file is `keyprox.yaml`.

Example:

```yaml
runtime:
  listen: :5050
  read_header_timeout: 10s
  upstream_429_retries: 3
  level: info
  log: ""
provider:
  zai:
    endpoints:
      - https://api.z.ai/api/coding/paas/v4
    keys:
      - key-1
      - key-2
      - key-3
```

### Runtime fields

- `listen` — HTTP server address;
- `read_header_timeout` — request header read timeout;
- `upstream_429_retries` — how many times the proxy may move to the next key after `429`;
- `level` — log level;
- `log` — path to a JSON log file; if empty, logs are written only to stdout.

### Provider fields

`provider` is a map of `provider_id -> provider_config`.

Each provider entry may contain:

- `endpoints` — upstream base URLs; the proxy currently uses the first endpoint in the list;
- `keys` — API keys for round-robin and 429 retry fallback;
- `default_headers` — optional headers copied from the provider catalog and sent on every upstream request.

The provider ID must match the prefix used in `model`.

Providers with an empty `keys` list are treated as disabled. This is important because the generated default config contains all supported providers but no secrets.

## Default config generation

If the config file does not exist and you run:

```bash
./bin/keyprox --save
```

`keyprox` loads the provider catalog from Catwalk, keeps only providers with static HTTP endpoints, and writes all of them into `keyprox.yaml`.

The generated file contains:

- the runtime defaults;
- every supported provider ID;
- each provider's upstream endpoint list;
- any provider default headers;
- empty `keys: []` lists.

After the file has been created, runtime provider information is loaded from the saved config instead of querying Catwalk on startup.

## Provider catalog source

The generated default provider list comes from Catwalk via `internal/proxy/endpoints.go`.
Live Catwalk provider data: https://catwalk.charm.land/v2/providers
The repository also keeps `opencode-providers-endpoints.md` as a snapshot reference.

## Running

Build the binary:

```bash
task build
```

Generate a config with defaults:

```bash
./bin/keyprox --save
```

Edit `keyprox.yaml` and fill keys only for the providers you want to enable.

Run the server:

```bash
./bin/keyprox --config keyprox.yaml
```

After startup, the local OpenAI-compatible endpoint is available by default at:

```text
http://127.0.0.1:5050/v1
```

## Example request

```bash
curl http://127.0.0.1:5050/v1/chat/completions \
  -H 'Content-Type: application/json' \
  -d '{
    "model": "zai/glm-5.1",
    "messages": [
      {"role": "user", "content": "ping"}
    ]
  }'
```

## Logs

The proxy writes at least these useful events:

- `proxy request` — outgoing request to the upstream;
- `proxy response` — final HTTP status returned to the client.

On `429`, it also writes this warning:

- `upstream returned 429, retrying with next key`

## Validation

Run tests:

```bash
task test
```
