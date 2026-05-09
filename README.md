# ninevigil-acp

> **Agent Context Language (ACL)** — a compact, machine-native representation
> of structured data, designed for LLM agents instead of humans.
>
> 90% fewer tokens, same fact-extraction accuracy, on a 1,620-trial
> Anthropic benchmark you can re-run for $1.
>
> **Status:** Private PoC · v0.1 · May 2026 · Owner: Clawdlinux / NineVigil

---

## TL;DR

Every system AI agents consume — Kubernetes APIs, OpenAPI specs, database
schemas, MCP tool catalogs — was designed for human eyes. JSON braces,
field descriptions, vendor extensions, repeated key names: 80–95% of the
bytes don't change the agent's decision. They burn the context window.

**ACL is what those bytes look like when you redesign them for the
consumer.** Three translators ship today (Kubernetes, OpenAPI, Postgres),
each ~250 lines of Go. The compression isn't gzip; it's deliberate field
selection plus a wire format with no JSON ceremony.

| Source | Real fixture | Raw tokens | ACL tokens | Reduction |
|---|---|---:|---:|---:|
| Kubernetes namespace | live kind cluster, 5 pods + 2 deploys + 2 svcs | 19,043 | 145 | **132×** |
| OpenAPI spec | GitHub's full v3 spec, 1,145 endpoints | ~3M | ~44K | **68×** |
| OpenAPI spec | Swagger Petstore (4 endpoints) | 1,492 | 201 | **7.4×** |
| Kubernetes namespace | bundled `state.acl` fixture | 3,671 | 260 | **14.1×** |
| Postgres schema | realistic 30-table `pg_dump -s` | ~5,500 | ~1,600 | **3.5×** |

**Agent accuracy preserved on facts (n=450 each, Claude Haiku 4.5):**

|  | Raw kubectl JSON | ACL | Δ |
|---|---:|---:|---:|
| Fact-extraction accuracy | 93.3% (90.6–95.3) | 93.3% (90.6–95.3) | **+0.0pp** |
| Decision accuracy | 83.3% (79.1–86.8) | 75.0% (70.3–79.2) | −8.3pp |
| Mean prompt tokens | 4,553 | 446 | **−90%** |
| Cost per call | $0.0037 | $0.00037 | **−89%** |

Full reproducible run: [`benchmark/agent_accuracy/results/2026-05-09-094833/summary.md`](./benchmark/agent_accuracy/results/2026-05-09-094833/summary.md)
— including the honest caveat about the 8.3pp decision gap and what
causes it.

## Quickstart (30 seconds)

```bash
git clone https://github.com/Clawdlinux/ninevigil-acp
cd ninevigil-acp
make build-acl

# Compress a real OpenAPI spec
bin/acl tokens pkg/aclhttp/testdata/petstore.json
# tokens:   1492  (cl100k_base)

bin/acl encode openapi pkg/aclhttp/testdata/petstore.json | bin/acl tokens -
# tokens:   201   (cl100k_base)
```

**1,492 → 201 tokens. Same OpenAPI spec.** Full quickstart with three
independent paths (CLI / Go library / Python decoder) at
[**docs/quickstart.md**](./docs/quickstart.md).

## How it works

ACL is two things working together:

1. **A wire format.** Line-oriented, ASCII-only, no JSON ceremony.
   Round-trip stable: `Decode(Encode(d)) == d`. Spec at
   [`docs/acl-spec.md`](./docs/acl-spec.md) (CC BY 4.0, 200 lines).

2. **Hand-written translators.** Each one decides what an agent
   *acting in this domain* needs and discards the rest. The Kubernetes
   translator keeps 8 fields per pod (`name`, `replicas`, `cpu`, `mem`,
   `restarts`, `age`, `warning`, `critical`); a real Pod object has
   ~120. That selection — not encoding tricks — is where the compression
   comes from.

```
┌─────────────────┐     ┌──────────────┐     ┌──────────────┐     ┌────────┐
│ Kubernetes API  │────▶│  Translator  │────▶│ ACL document │────▶│  LLM   │
│ OpenAPI spec    │     │ (~250 LOC Go)│     │  (compact,   │     │  Agent │
│ pg_dump -s      │     │              │     │   round-trip │     │        │
│ MCP tools/list  │     │              │     │   stable)    │     │        │
└─────────────────┘     └──────────────┘     └──────────────┘     └────────┘
   ↑ humans designed             ↑                  ↑ agents consume
     this for humans          you write this        this natively
```

A K8s namespace with 5 pods looks like this in ACL:

```
@cluster prod-east
@ns payments
@source k8s-namespace/v0.1

pods 5/5 ok
  api-7f8d-aaa11 cpu=0 mem=0 r=0 age=3h
  api-7f8d-bbb22 cpu=0 mem=0 r=0 age=3h
  api-7f8d-ccc33 cpu=0 mem=0 r=0 age=3h
  worker-9a2b-aaa11 cpu=0 mem=0 r=0 age=3h
  worker-9a2b-bbb22 cpu=0 mem=0 r=0 age=3h

deploys 2 all-avail
  api replicas=3/3 strategy=rollingupdate image=v2.4.1
  worker replicas=2/2 strategy=rollingupdate image=v1.8.0

svcs 2
  api type=ClusterIP port=8080->8080
  worker type=ClusterIP port=9090->9090

actions
  scale|rollout|restart|logs|describe
```

572 bytes. The kubectl JSON for the same data is 14,506 bytes (25× more).

## Repo layout

```
ninevigil-acp/
├── docs/
│   ├── acl-spec.md               # ACL v0.1 wire-format specification (CC BY 4.0)
│   ├── quickstart.md             # 3-path getting started
│   └── ...
├── pkg/
│   ├── acl/                      # Reference encoder/decoder (Apache 2.0)
│   ├── aclhttp/                  # OpenAPI 3.x → ACL translator
│   ├── aclpg/                    # Postgres schema → ACL translator
│   └── acp/                      # Translator[S] interface + Go client
├── cmd/
│   ├── acl/                      # `acl` CLI binary
│   └── acp-server/               # ACP HTTP server
├── internal/                     # ACP server runtime (BSL 1.1)
│   ├── builder/   manifest/      # Execution Manifest builder
│   ├── registry/                 # Tool registry
│   ├── resolver/                 # Intent → capabilities resolver
│   ├── proxy/                    # Auth-injection proxy
│   └── sources/{mcp,k8s}         # Source adapters
├── benchmark/
│   └── agent_accuracy/           # 540/1620-trial harness (Python + tiktoken)
│       ├── harness.py            # API runner with cost cap + cache
│       ├── fixtures/             # 3 K8s scenarios (healthy/degraded/failing)
│       ├── questions.yaml        # 9 questions (5 fact + 4 decision)
│       └── results/              # Per-run summary.md (committed)
├── python/
│   └── acp_acl/                  # `pip install acp-acl` decoder
├── adapters/python/              # LangGraph / CrewAI / OpenAI adapters
├── examples/
│   ├── 01-k8s-namespace/         # K8s state → ACL demo
│   ├── 02-openapi-petstore/      # OpenAPI → ACL pipeline
│   └── 03-python-decoder/        # Python agent reading ACL
├── SPEC.md                       # ACP wire protocol (separate from ACL spec)
└── Dockerfile.acl                # 15MB distroless multi-arch image
```

The K8s **translator** lives here at `pkg/aclk8s` *interface*; the live
collector that reads from a real cluster lives in
[agentic-operator-core/pkg/aclk8s](https://github.com/Clawdlinux/agentic-operator-core)
because it depends on `kubernetes/client-go`.

## What ships today

| Component | Status | Where |
|---|---|---|
| ACL v0.1 spec | ✅ frozen, CC BY 4.0 | [`docs/acl-spec.md`](./docs/acl-spec.md) |
| Go encoder/decoder | ✅ round-trip stable, fuzzed | [`pkg/acl/`](./pkg/acl/) |
| Translator: Kubernetes | ✅ 132× live, 14× fixture | [agentic-operator-core](https://github.com/Clawdlinux/agentic-operator-core/tree/main/pkg/aclk8s) |
| Translator: OpenAPI 3.x | ✅ 68× on GitHub spec | [`pkg/aclhttp/`](./pkg/aclhttp/) |
| Translator: Postgres schema | ✅ 3.5× vs realistic pg_dump | [`pkg/aclpg/`](./pkg/aclpg/) |
| `acl` CLI | ✅ encode / decode / tokens | [`cmd/acl/`](./cmd/acl/) |
| Distroless container | ✅ 15MB, GHA + cosign | [`Dockerfile.acl`](./Dockerfile.acl) |
| Python decoder | ✅ pure Python, no deps | [`python/acp_acl/`](./python/acp_acl/) |
| Agent-accuracy benchmark | ✅ 1,620 trials, $0.86 | [`benchmark/agent_accuracy/`](./benchmark/agent_accuracy/) |
| ACP server (intent resolution) | ✅ HTTP API + auth proxy | [`cmd/acp-server/`](./cmd/acp-server/), [`SPEC.md`](./SPEC.md) |
| Python framework adapters | ✅ LangGraph, CrewAI, OpenAI | [`adapters/python/`](./adapters/python/) |

## Two products in one repo

This repo ships two complementary things. **ACL is the headline; ACP is
the delivery mechanism.**

### ACL (Agent Context Language) — the format

A representation. Hand-written translators map any structured source
into ACL. Use the Go SDK or the CLI directly; no server required.

### ACP (Agent Context Protocol) — the server

An HTTP server that resolves agent intents into Execution Manifests
with auth pre-injected, dependency ordering pre-computed, and security
boundaries declared. Sits **on top of MCP**: ingests `tools/list` from
existing MCP servers and emits token-minimal manifests in ACL.

ACP server quickstart:

```bash
# 1. Build the server
make build                           # -> bin/acp-server

# 2. Run it
ACP_AUTH_TOKEN=dev-token ./bin/acp-server --addr :8080

# 3. Request a manifest
curl -sS -X POST http://localhost:8080/v1/context \
  -H "Authorization: Bearer dev-token" \
  -H "Content-Type: application/json" \
  -d '{"intent":"query customer data, render report, email the team","agent_id":"demo"}' \
  | python3 -m json.tool
```

Full spec: [`SPEC.md`](./SPEC.md).

Python framework adapters are in [`adapters/python/`](./adapters/python/);
the Go client is `pkg/acp/`:

```go
client := acp.NewClient("http://localhost:8080", acp.WithToken("dev-token"))
mf, _ := client.Context(ctx, manifest.ContextRequest{
    Intent:  "query customer database",
    AgentID: "agent-01",
})
```

## License

- **Spec** ([`docs/acl-spec.md`](./docs/acl-spec.md), [`SPEC.md`](./SPEC.md), [`docs/protocol.md`](./docs/protocol.md)): CC BY 4.0
- **Adapters / SDKs** ([`pkg/`](./pkg/), [`adapters/`](./adapters/), [`python/`](./python/), [`cmd/acl/`](./cmd/acl/)): Apache 2.0
- **ACP server runtime** ([`cmd/acp-server/`](./cmd/acp-server/), [`internal/`](./internal/)): **BSL 1.1** with 3-year conversion to Apache 2.0

See [`LICENSE`](./LICENSE) for the runtime license and [`docs/LICENSING.md`](./docs/LICENSING.md)
for the per-tree breakdown.

## Confidentiality

This repository is **PRIVATE** until the v0.1 spec + benchmark results are
ready for public release. Do not share the SPEC, benchmark numbers, or
architecture diagrams externally without explicit approval.
