# Agent Native Format (ANF)

> **ANF** is a token-minimal view format for AI agents.
> Translate live system state into far fewer tokens. Same facts, a fraction of
> the context window.
>
> **Status:** Public PoC · v0.1 spec DRAFT · 2026 &nbsp;**Owner:** Clawdlinux

Not Zed's Agent Client Protocol. Not IBM's Agent Communication Protocol. ANF is
a data format, not a transport.

---

## TL;DR

JSON, YAML, and HTML are built for humans. Agents pay for that in tokens.

ANF is a line-oriented representation built for LLM consumption. It encodes the
decision-relevant state of a system, a Kubernetes namespace, a database, a SaaS
dashboard, in the fewest tokens that still carry the facts.

- No quotes, braces, or commas. Indentation and newlines are the structure.
- Health, alerts, and available actions surface first.
- Self-describing. An LLM parses it with no schema.

A Kubernetes namespace that costs about 12,000 tokens as raw API JSON costs
about 350 tokens as ANF. The full spec is in [`FORMAT.md`](FORMAT.md).

## What ships today

- **Spec.** [`FORMAT.md`](FORMAT.md), CC BY 4.0. The format definition.
- **Go encoder.** [`pkg/anf`](pkg/anf), Apache 2.0. Build and emit ANF documents.
- **Kubernetes translator.** [`translators/kubernetes`](translators/kubernetes).
  Live cluster state to ANF.
- **MCP server.** [`cmd/anf-mcp`](cmd/anf-mcp), Apache 2.0. A stateless stdio MCP
  server exposing ANF encoding tools to any MCP client (Claude, Cursor, VS Code,
  Codex). `go install github.com/Clawdlinux/agent-native-format/cmd/anf-mcp@latest`.
- **Benchmarks.** Reproducible token measurements against raw and filtered JSON.

## Governed execution runtime

The repo also ships a reference runtime that turns an agent intent into a
scoped, auditable execution contract. This was the original Agent Contract
Protocol work. It stays here as the execution layer that consumes tool
discovery and enforces policy at the boundary.

Before an autonomous agent touches a real system, the runtime answers:

1. **Who is it acting as?** The contract binds execution to an agent identity
   and a credential alias. Raw credentials never enter the model context.
2. **What is it allowed to do?** The contract declares egress, approval gates,
   TTL, rate limits, and audit level.
3. **What happened afterward?** Every action ties back to a contract, action
   id, principal, outcome, and audit record.

The runtime is MCP-compatible. It consumes existing `tools/list` output,
normalizes MCP, Kubernetes, and future OpenAPI/gRPC/CLI sources into one
contract shape, enforces egress and approval at the proxy, and attaches audit
events to the same contract id.

Scoped contracts also cut tool-context tokens, because a contract only carries
the capabilities the intent needs. Measured against the MCP baseline:

| Scenario | runtime / MCP tokens | runtime / MCP round trips | Reduction |
|---|---:|---:|---:|
| S1 Simple DB query | 111 / 373 | 1 / 3 | 70.2% |
| S2 Multi-tool workflow | 295 / 837 | 1 / 5 | 64.7% |
| S3 Complex DAG | 306 / 1,257 | 1 / 7 | 75.6% |
| S4 Scale, 50 tools and 2 relevant | 241 / 9,223 | 1 / 21 | 97.4% |
| S5 Auth-heavy | 359 / 1,431 | 1 / 7 | 74.9% |

Full data in
[`results/2026-05-02-week3-summary.md`](results/2026-05-02-week3-summary.md).

## Prior art and honest positioning

ANF is not a novel serialization format, and this is not published research. The
token-efficient-format space is already crowded: TOON, TRON, and TSLN cover
compact JSON-style encodings. Independent work is also sobering about the
approach. The "Notation Matters" agentic benchmark (arXiv 2605.29676) finds
compact notations save roughly 18-27% inside real agent loops and cost accuracy.
State-in-context minification (arXiv 2606.01326) reports about 42% with a 12pp
accuracy drop.

So ANF does not compete on notation. Most of its token savings come from view
extraction: dropping irrelevant fields and surfacing the decision-relevant state
(health, alerts, available actions), not from compressing the syntax. Treat ANF
as a supporting internal tool for that view extraction, grounded in the
literature above, not as a groundbreaking format.

## Repository layout

```text
agent-native-format/
├── FORMAT.md                     # ANF format specification (CC BY 4.0)
├── SPEC.md                       # Execution runtime protocol specification
├── pkg/
│   ├── anf/                      # ANF encoder and types
│   ├── acp/                      # Execution runtime Go SDK
│   └── manifest/                 # Wire types
├── translators/
│   └── kubernetes/               # Live cluster state to ANF
├── cmd/
│   ├── acp-server/               # Execution runtime entrypoint
│   ├── acp-bridge/               # MCP bridge for IDE-style clients
│   └── benchmark/                # Benchmark CLI
├── internal/
│   ├── builder/                  # Contract builder and ordering
│   ├── proxy/                    # Auth-injection and policy enforcement
│   ├── registry/                 # Tool registry
│   ├── resolver/                 # Intent to capabilities resolver
│   └── sources/                  # MCP and Kubernetes source adapters
├── adapters/python/              # Python adapters for common agent stacks
├── benchmark/                    # Reproducible token benchmark harness
├── docs/                         # Architecture, positioning, validation
└── deploy/                       # Docker Compose and Kubernetes assets
```

## Quickstart

### Build from source

```bash
git clone https://github.com/Clawdlinux/agent-native-format
cd agent-native-format
make build
ACP_AUTH_TOKEN=dev-token ./bin/acp-server --addr :8080
```

### Plug into VS Code MCP

For normal MCP users, install the bridge and register one MCP server in VS Code:

```bash
go install github.com/Clawdlinux/agent-native-format/cmd/acp-bridge@latest
```

`~/Library/Application Support/Code/User/mcp.json`:

```json
{
  "servers": {
    "acp-bridge": {
      "type": "stdio",
      "command": "acp-bridge",
      "args": ["--import-vscode"]
    }
  }
}
```

`acp-bridge` imports your existing VS Code stdio MCP servers, skips its own
entry, starts each downstream server as a child process, compacts `tools/list`,
narrows the tool surface after observed calls, and forwards real `tools/call`
requests to the original MCP server. VS Code HTTP MCP entries are skipped until
HTTP forwarding lands.

For an explicit config instead of auto-import:

```bash
acp-bridge --config ./examples/bridge.json
```

### Request an execution contract


```bash
curl -sS -X POST http://localhost:8080/v1/context \
  -H "Authorization: Bearer dev-token" \
  -H "Content-Type: application/json" \
  -d '{"intent":"query customer data, render report, email the team","agent_id":"demo"}' \
  | python3 -m json.tool
```

### Go install

```bash
go install github.com/Clawdlinux/agent-native-format/cmd/acp-server@main
ACP_AUTH_TOKEN=dev-token acp-server --addr :8080
```

Use a release tag like `@v0.2.0-paper` for reproducible installs.

## Use with existing MCP servers

ACP does not replace MCP. It uses MCP as a discovery supply chain.

```text
   Agent runtime
        │
        │  POST /v1/context  {"intent":"...", "agent_id":"..."}
        ▼
   ┌──────────────────────────────┐       ┌─────────────────────────┐
   │ ACP server                   │ ────▶ │ Existing MCP server     │
   │ intent → execution contract  │ tools │ GitHub, Slack, DB, etc. │
   └──────────────────────────────┘ /list └─────────────────────────┘
        │
        │  scoped actions, policy, auth alias, ordering
        ▼
   Agent executes through /v1/exec/{contract_id}/{action_id}
```

The MCP importer at [`internal/sources/mcp`](internal/sources/mcp) reads the
standard MCP `tools/list` envelope and registers each tool in ACP's registry.

```bash
python3 scripts/fake-mcp.py 19090 &

go run ./cmd/import-demo \
  -source name=files,url=http://127.0.0.1:19090,caps=filesystem
```

Each imported tool gets:

- A namespaced id like `<source>.<tool>`.
- Capability tags for intent resolution.
- An endpoint routed through the ACP proxy.
- An auth mode that keeps raw credentials out of the model context.

> **Honest status:** `acp-bridge` is the plug-and-play IDE path. It can import
> VS Code MCP config, start stdio MCP servers, and forward real tool calls.
> HTTP MCP entries are skipped by the auto-import path for now.
> `acp-server` still uses the manifest/proxy path and a seeded registry unless
> you run the importer flow above.

## Current status

Working today:

- `POST /v1/context` returns scoped contracts using the existing v0.1
  `ExecutionManifest` wire type.
- The proxy strips agent-supplied auth headers and injects server-side creds.
- Egress and approval boundaries are represented in the contract.
- MCP and Kubernetes source adapters exist.
- Python adapters cover OpenAI function calling, LangGraph, and CrewAI.
- Benchmarks reproduce the MCP baseline and token deltas.

### What ACP does not do

- **Resolve interactive VS Code `${input:...}` prompts.** Put secrets in the
  environment or in your existing downstream MCP server config.
- **Replace your downstream MCP servers.** ACP starts and forwards to them. It
  does not re-implement their provider APIs.
- **Cache `tools/list` indefinitely.** Each `ImportSource` call refetches; re-call on a schedule (or on a webhook) when your MCP server's tool set changes.

### Next hardening work

- Enforce contract TTL at the proxy.
- Persist append-only audit logs.
- Sign contracts and verify signatures before execution.
- Bind each contract to principal and credential alias.
- Add an OpenAPI source adapter to prove source-agnostic execution.

See [`docs/positioning.md`](docs/positioning.md) and
[`docs/validation/signals.md`](docs/validation/signals.md).

## Python adapters

```bash
pip install "git+https://github.com/Clawdlinux/agent-native-format.git@main#subdirectory=adapters/python/acp_common"
pip install "git+https://github.com/Clawdlinux/agent-native-format.git@main#subdirectory=adapters/python/acp_langgraph"
pip install "git+https://github.com/Clawdlinux/agent-native-format.git@main#subdirectory=adapters/python/acp_openai"
pip install "git+https://github.com/Clawdlinux/agent-native-format.git@main#subdirectory=adapters/python/acp_crewai"
```

## Go SDK

```go
client := acp.NewClient("http://localhost:8080", acp.WithToken("dev-token"))
contract, err := client.Context(ctx, manifest.ContextRequest{
    Intent:  "query the customer db",
    AgentID: "agent-01",
})
```

## License

- **Protocol spec** (`SPEC.md`, `docs/protocol.md`): CC BY 4.0
- **Adapters / SDKs** (`pkg/`, `adapters/`): Apache 2.0
- **ACP server runtime** (`cmd/`, `internal/`): BSL 1.1, converts to Apache 2.0 after 3 years

See [`LICENSE`](LICENSE) and [`docs/LICENSING.md`](docs/LICENSING.md).