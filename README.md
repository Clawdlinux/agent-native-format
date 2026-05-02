# ninevigil-acp

> **Agent Context Protocol** — one API call, complete execution context, minimal tokens.
> A successor to MCP for production agent systems.
>
> **Status:** Private PoC · v0.1 DRAFT · May 2026
> **Owner:** Clawdlinux / NineVigil

---

## TL;DR

MCP wastes 70–85% of the context window on tool discovery and schema bloat
before the agent can do any useful work. **ACP** flips the model: the
infrastructure computes the execution context for the agent and returns a
single, intent-scoped manifest with auth pre-injected, ordering pre-computed,
and security boundaries declared.

| | MCP (today) | ACP (this repo) |
|---|---|---|
| Round trips before first action | 5–8 | **1** |
| Tokens for 3-tool workflow | ~6,000+ | **~400** |
| Auth in agent context | yes | **never** |
| Execution ordering | agent infers | **server declares** |

Numbers above are the PoC targets — see [`benchmark/`](./benchmark/) for the
methodology and [`results/`](./results/) for measured outcomes.

## Repo layout (per the source PoC specification §3.3)

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
│   └── pitch-deck-data.md        # Pre-formatted numbers for investor decks
└── deploy/
    ├── docker-compose.yaml       # Local dev stack
    └── k8s/                      # K8s manifests for cluster testing
```

## Phase plan (4 weeks)

| Week | Deliverable | Success criteria |
|---|---|---|
| **1** | ACP server + tool registry + manifest builder; keyword intent resolver; Docker Compose dev stack | `POST /v1/context` returns valid manifest for 3 registered tools |
| **2** | Auth proxy (credential injection); MCP baseline client; benchmark harness for S1 + S2 | Side-by-side runs for simple + multi-tool scenarios |
| **3** | Python adapters (LangGraph, raw OpenAI); S3–S5 scenarios; embedding-based intent resolver | All 5 scenarios benchmarked end-to-end |
| **4** | Report generation; charts; protocol spec finalized; pitch-deck data extraction | Reproducible benchmark, investor-ready numbers |

Phase status is tracked in [`docs/phase-log.md`](./docs/phase-log.md) and
mirrored to the `ACP-PoC/` folder inside the NineVigil Obsidian vault.

## Quickstart

```bash
# 1. Build the server
make build                           # -> bin/acp-server

# 2. Run it (auth required when ACP_AUTH_TOKEN is set)
ACP_AUTH_TOKEN=dev-token ./bin/acp-server --addr :8080

# 3. Request a manifest
curl -sS -X POST http://localhost:8080/v1/context \
  -H "Authorization: Bearer dev-token" \
  -H "Content-Type: application/json" \
  -d '{"intent":"query customer data, render report, email the team","agent_id":"demo"}' \
  | python3 -m json.tool

# 4. Report execution feedback
curl -sS -X POST http://localhost:8080/v1/feedback \
  -H "Authorization: Bearer dev-token" \
  -H "Content-Type: application/json" \
  -d '{"manifest_id":"m-...","action_id":"a1","outcome":"success","latency_ms":42}'
```

The Week 1 server seeds an in-memory registry with five demo tools
(`db.query`, `template.render`, `email.send`, `slack.send_message`,
`audit.log_event`), uses a deterministic keyword resolver, and emits manifests
with depends_on chains, egress allow-lists, and approval gates.

Go SDK consumers can use [`pkg/acp`](./pkg/acp):

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
- **ACP server runtime** (`cmd/`, `internal/`): **BSL 1.1** with 3-year
  conversion to Apache 2.0

See [`LICENSE`](./LICENSE) for the runtime license and [`docs/LICENSING.md`](./docs/LICENSING.md)
for the per-tree breakdown.

## Confidentiality

This repository is **PRIVATE** until the v0.1 spec + benchmark results are
ready for public release. Do not share the SPEC, benchmark numbers, or
architecture diagrams externally without explicit approval.
