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

# Option B — Docker (no toolchain needed) — pick either registry
docker run --rm -p 8080:8080 -e ACP_AUTH_TOKEN=dev-token \
  ghcr.io/clawdlinux/ninevigil-acp:v0.1.0-spec
# or, from Docker Hub:
docker run --rm -p 8080:8080 -e ACP_AUTH_TOKEN=dev-token \
  goodra007/acp-server:v0.1.0-spec

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

## Use with your existing MCP servers

ACP doesn't replace your MCP servers — it sits in front of them. The MCP servers you run today (filesystem, GitHub, Postgres, Slack, your custom ones) keep working unchanged. ACP discovers their tools via standard `tools/list`, deduplicates schemas, attaches auth out-of-context, and emits one scoped manifest per agent intent.

### The shape of the integration

```
   Your agent (Claude / Cursor / LangGraph / custom)
                │
                │  POST /v1/context  {"intent": "..."}
                ▼
   ┌──────────────────────────┐         ┌─────────────────────────┐
   │     ACP server           │ ──GET──▶│  your-mcp-server.local  │  (existing)
   │  (intent → manifest)     │  /tools │   filesystem, GH, etc.  │
   └──────────────────────────┘  /list  └─────────────────────────┘
                │
                │  one manifest, scoped, ordered, auth pre-injected
                ▼
   Agent executes against POST /v1/exec/<action_id>
```

### Pointing ACP at one or more MCP servers

The Go importer at [`internal/sources/mcp`](internal/sources/mcp) reads the standard MCP `tools/list` envelope (2024-11 spec) and registers every tool into ACP's registry. The repo ships a runnable demo at [`cmd/import-demo`](cmd/import-demo):

```bash
# Try it against any MCP server that exposes /tools/list (the 2024-11 envelope).
# Example below uses the bundled fake MCP server for a 60-second smoke test:
python3 -c "$(curl -sSL https://raw.githubusercontent.com/Clawdlinux/ninevigil-acp/v0.1.0-spec/scripts/fake-mcp.py)" 19090 &

go run github.com/Clawdlinux/ninevigil-acp/cmd/import-demo@v0.1.0-spec \
  -source name=files,url=http://127.0.0.1:19090,caps=filesystem \
  -source name=github,url=http://gh-mcp.internal:9100,auth="bearer ghp_xxx",caps=git
```

Output:

```
─── files (http://127.0.0.1:19090) ───────────────────
  ✅ imported 4 tool(s)
─── github (http://gh-mcp.internal:9100) ─────────────
  ✅ imported 12 tool(s)

=== ACP registry now holds 16 tool(s) ===
  files.list_directory   caps=[directory filesystem list read]  endpoint=...
  files.read_file        caps=[file filesystem read]            endpoint=...
  github.create_pr       caps=[create git pr write]              endpoint=...
  github.search_repos    caps=[git read repos search]            endpoint=...
  …
```

Each imported tool gets:
- A namespaced ID (`<source>.<tool>`) so two MCP servers can expose the same tool name without collision.
- Capabilities auto-inferred from the tool name + your explicit `caps=` tag, so the resolver can pick the right one for an intent.
- Endpoint rewired through the ACP auth-injection proxy — your `auth=` value is stored server-side and never enters the agent context.

> **Honest status (v0.1.0-spec):** the `cmd/import-demo` flow above works end-to-end and is what to run for evaluation. The `acp-server` binary still boots with a 5-tool seed registry — wiring the importer behind a `--mcp-source` flag (or a `sources.yaml` config) so you can do it without writing Go is tracked in [#12](https://github.com/Clawdlinux/ninevigil-acp/issues/12) for v0.1.1.

### Migration ladder (for teams already in production with MCP)

| You have today | What to add | What changes for the agent |
|---|---|---|
| 1 MCP server, 5 tools, agent calls `tools/list` every turn | Run ACP in front of it; agent calls `POST /v1/context` instead | One round-trip vs. 3+; tokens drop ~70% |
| 3+ MCP servers, manual fan-out in agent code | Import each into the same ACP registry | Agent stops fanning out; ACP picks the right server per intent |
| MCP servers behind separate auth headers | Set `Source.Auth` per source at import time | Credentials leave the agent context entirely |
| Custom MCP server you wrote | Nothing — ACP just calls your `tools/list` | None — your server is unchanged |

### What ACP does not do

- **Discover MCP servers automatically.** You configure each one explicitly so the egress allow-list is auditable.
- **Re-implement MCP transport.** ACP speaks the MCP HTTP envelope; stdio MCP servers are tracked for v0.2 in [#13](https://github.com/Clawdlinux/ninevigil-acp/issues/13).
- **Cache `tools/list` indefinitely.** Each `ImportSource` call refetches; re-call on a schedule (or on a webhook) when your MCP server's tool set changes.

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
