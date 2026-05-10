# ninevigil-acp

> **Agent Context Protocol** — one API call, complete execution context, minimal tokens.
> An intent-resolution and execution-planning layer that **sits on top of MCP** and other tool sources.
>
> **Status:** Public PoC · v0.1 DRAFT · May 2026 &nbsp;**Owner:** Clawdlinux / NineVigil

---

## TL;DR

MCP handles tool *discovery* well. It was never designed to handle *execution context*.

In production agent systems, three problems compound before the agent takes its first action:

1. **Token bloat** — verbose tool schemas consume 70–85% of the context window. With 50 registered tools, a single `tools/list` round-trip costs 9,223 tokens (measured).
2. **Auth in context** — credentials must be present in the agent context to call tools. That's an exploitable attack surface.
3. **Ordering inferred, not declared** — agents reason about `depends_on` chains themselves, burning tokens and introducing errors on multi-step workflows.

**ACP sits on top of MCP** (and any other tool source). The agent sends one intent; ACP returns one scoped manifest with auth pre-injected, ordering pre-computed, and security boundaries declared. Existing MCP servers keep working — ACP ingests their `tools/list` and produces token-minimal manifests for execution.

### How ACP compares

|  | Framework tool-retrievers¹ | Custom MCP proxies² | **ACP** |
| --- | --- | --- | --- |
| Token reduction | ✅ partial | ✅ partial | ✅ **64–97%** |
| Auth out of agent context | ❌ | ❌ | ✅ |
| Execution ordering declared | ❌ | ❌ | ✅ |
| Works across frameworks | ❌ per-framework | ❌ per-app | ✅ |
| Open protocol spec | ❌ | ❌ | ✅ CC BY 4.0 |

¹ LangChain/LlamaIndex ToolRetriever: agent-side embedding selection — auth still in context, framework-specific.
² Ad-hoc `tools/list` filters: app-specific, no auth model, no ordering hints.

**ACP is a protocol, not a library.** Any framework can implement an adapter. The spec (`SPEC.md`) is CC BY 4.0.

### Raw numbers (50 runs/scenario, `tiktoken/cl100k_base`)

|  | MCP today (raw) | ACP on top of MCP |
| --- | --- | --- |
| Round trips before first action | 3–21 (measured) | **1** |
| Tokens for representative workflows | 373–9,223 (measured) | **111–359** |
| Auth in agent context | yes | **never** |
| Execution ordering | agent infers | **server declares** |

Full benchmark data: [`results/2026-05-02-week3-summary.md`](results/2026-05-02-week3-summary.md) · Full positioning: [`docs/positioning.md`](docs/positioning.md).

## Repo layout

```
ninevigil-acp/
├── SPEC.md                       # ACP v0.1 protocol specification
├── cmd/
│   ├── acp-server/               # ACP server entrypoint (Go)
│   └── benchmark/                # Benchmark CLI
├── internal/
│   ├── manifest/                 # Manifest builder + optimizer
│   ├── registry/                 # Tool registry
│   ├── resolver/                 # Intent → capabilities resolver
│   └── proxy/                    # Auth-injection proxy
├── pkg/
│   ├── acp/                      # Public Go SDK: acp.NewClient()
│   └── manifest/                 # Manifest types (shared with adapters)
├── adapters/python/
│   ├── acp_langgraph/            # LangGraph adapter
│   ├── acp_crewai/               # CrewAI adapter
│   └── acp_openai/               # Raw OpenAI function-calling adapter
├── benchmark/
│   ├── scenarios/                # YAML task definitions (S1–S5)
│   ├── baseline/mcp_client.py    # MCP baseline implementation
│   ├── harness.py                # Benchmark orchestrator
│   └── report.py                 # Generates comparison report
├── results/                      # Auto-generated benchmark output + charts
├── docs/
│   ├── architecture.md
│   ├── protocol.md
│   ├── benchmark-methodology.md
│   └── pitch-deck-data.md
└── deploy/
    ├── docker-compose.yaml
    └── k8s/
```

## Phase plan (4 weeks)

| Week | Deliverable | Success criteria |
| --- | --- | --- |
| **1** | ACP server + tool registry + manifest builder; keyword intent resolver; Docker Compose dev stack | `POST /v1/context` returns valid manifest for 3 registered tools |
| **2** | Auth proxy (credential injection); MCP baseline client; benchmark harness for S1 + S2 | Side-by-side runs for simple + multi-tool scenarios |
| **3** | Python adapters (LangGraph, raw OpenAI); S3–S5 scenarios; embedding-based intent resolver | All 5 scenarios benchmarked end-to-end |
| **4** | Report generation; charts; protocol spec finalized; pitch-deck data extraction | Reproducible benchmark, investor-ready numbers |

Phase status: [`docs/phase-log.md`](docs/phase-log.md).

## Quickstart

### Install (pick one)

```bash
# Option A — Go binary, no clone (requires Go 1.25+)
go install github.com/Clawdlinux/ninevigil-acp/cmd/acp-server@v0.1.0-spec
ACP_AUTH_TOKEN=dev-token acp-server --addr :8080

# Option B — Docker (no toolchain needed)
docker run --rm -p 8080:8080 -e ACP_AUTH_TOKEN=dev-token \
  ninevigil/acp-server:v0.1.0-spec

# Option C — Build from source
git clone https://github.com/Clawdlinux/ninevigil-acp && cd ninevigil-acp
make build
ACP_AUTH_TOKEN=dev-token ./bin/acp-server --addr :8080
```

### Request a manifest

```bash
curl -sS -X POST http://localhost:8080/v1/context \
  -H "Authorization: Bearer dev-token" \
  -H "Content-Type: application/json" \
  -d '{"intent":"query customer data, render report, email the team","agent_id":"demo"}' \
  | python3 -m json.tool
```

### Python adapters (pip from git, no PyPI required)

```bash
# Shared client + types (used by every adapter)
pip install "git+https://github.com/Clawdlinux/ninevigil-acp.git@v0.1.0-spec#subdirectory=adapters/python/acp_common"

# Pick the framework you actually use:
pip install "git+https://github.com/Clawdlinux/ninevigil-acp.git@v0.1.0-spec#subdirectory=adapters/python/acp_langgraph"
pip install "git+https://github.com/Clawdlinux/ninevigil-acp.git@v0.1.0-spec#subdirectory=adapters/python/acp_openai"
pip install "git+https://github.com/Clawdlinux/ninevigil-acp.git@v0.1.0-spec#subdirectory=adapters/python/acp_crewai"
```

### Go SDK

```go
client := acp.NewClient("http://localhost:8080", acp.WithToken("dev-token"))
mf, err := client.Context(ctx, manifest.ContextRequest{
    Intent:  "query the customer db",
    AgentID: "agent-01",
})
```

## License

- **Protocol spec** (`SPEC.md`, `docs/protocol.md`): CC BY 4.0
- **Adapters / SDKs** (`pkg/`, `adapters/`): Apache 2.0
- **ACP server runtime** (`cmd/`, `internal/`): **BSL 1.1** → Apache 2.0 after 3 years

See [`LICENSE`](LICENSE) and [`docs/LICENSING.md`](docs/LICENSING.md).
