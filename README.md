# Agent Contract Protocol

> **ACP** is the governed execution layer for autonomous agents.
> Any tool source in. Signed, identity-bound, ordered, auditable execution out.
>
> **Status:** Public PoC · v0.2 DRAFT · June 2026 &nbsp;**Owner:** Clawdlinux / NineVigil

---

## TL;DR

MCP answers **what tools exist**. Claude Code Tool Search and semantic retrieval
are making tool selection cheaper. That is good. It also means raw token
reduction is not a durable product thesis.

The hard problem is now execution trust.

Before an autonomous agent touches a real system, you need to know:

1. **Who is it acting as?** The contract binds execution to an agent identity
   and a credential alias. Raw credentials never enter the model context.
2. **What is it allowed to do?** The contract declares egress, approval gates,
   TTL, rate limits, and audit level.
3. **What happened afterward?** Every action can be tied back to a contract,
   action id, principal, outcome, and audit record.

ACP takes an agent intent and returns an **Execution Contract**. The current
wire type is still `ExecutionManifest` for v0.1 compatibility, but the contract
is the product primitive: scoped capabilities, ordered actions, auth handled at
the proxy, and policy declared before the first tool call.

## Why this exists

Autonomous agents are stuck in demos because teams cannot bound their blast
radius. Tool discovery is getting solved. Production execution governance is
not.

ACP is a small open contract format and reference runtime for that gap:

- **MCP-compatible.** Existing MCP servers keep working. ACP consumes their
  `tools/list` and emits a governed execution contract.
- **Source-agnostic.** MCP, Kubernetes Services, and future OpenAPI/gRPC/CLI
  adapters all normalize into the same contract shape.
- **Policy at the boundary.** The proxy enforces egress and approval. The
  model does not get to self-police.
- **Audit by construction.** Feedback and proxy execution events attach to the
  same contract id.

## Token efficiency is a side effect

ACP still reduces tool-context tokens because contracts only include the
capabilities needed for the intent.

Measured benchmark data is in
[`results/2026-05-02-week3-summary.md`](results/2026-05-02-week3-summary.md).

| Scenario | ACP / MCP tokens | ACP / MCP round trips | Reduction |
|---|---:|---:|---:|
| S1 Simple DB query | 111 / 373 | 1 / 3 | 70.2% |
| S2 Multi-tool workflow | 295 / 837 | 1 / 5 | 64.7% |
| S3 Complex DAG | 306 / 1,257 | 1 / 7 | 75.6% |
| S4 Scale, 50 tools and 2 relevant | 241 / 9,223 | 1 / 21 | 97.4% |
| S5 Auth-heavy | 359 / 1,431 | 1 / 7 | 74.9% |

Those numbers are useful. They are not the moat. The moat is the governed
execution contract.

## Repository layout

```text
agent-contract-protocol/
├── SPEC.md                       # ACP protocol specification
├── cmd/
│   ├── acp-server/               # ACP server entrypoint
│   ├── acp-bridge/               # MCP bridge for IDE-style clients
│   └── benchmark/                # Benchmark CLI
├── internal/
│   ├── builder/                  # Contract builder and ordering
│   ├── proxy/                    # Auth-injection and policy enforcement
│   ├── registry/                 # Tool registry
│   ├── resolver/                 # Intent to capabilities resolver
│   └── sources/                  # MCP and Kubernetes source adapters
├── pkg/
│   ├── acp/                      # Public Go SDK
│   └── manifest/                 # Wire types
├── adapters/python/              # Python adapters for common agent stacks
├── benchmark/                    # Reproducible MCP vs ACP harness
├── docs/                         # Architecture, positioning, validation
└── deploy/                       # Docker Compose and Kubernetes assets
```

## Quickstart

### Build from source

```bash
git clone https://github.com/Clawdlinux/agent-contract-protocol
cd agent-contract-protocol
make build
ACP_AUTH_TOKEN=dev-token ./bin/acp-server --addr :8080
```

### Plug into VS Code MCP

For normal MCP users, install the bridge and register one MCP server in VS Code:

```bash
go install github.com/Clawdlinux/agent-contract-protocol/cmd/acp-bridge@latest
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

### Go install after the rename lands

```bash
go install github.com/Clawdlinux/agent-contract-protocol/cmd/acp-server@main
ACP_AUTH_TOKEN=dev-token acp-server --addr :8080
```

Use a release tag once the first Agent Contract Protocol tag is cut.

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
pip install "git+https://github.com/Clawdlinux/agent-contract-protocol.git@main#subdirectory=adapters/python/acp_common"
pip install "git+https://github.com/Clawdlinux/agent-contract-protocol.git@main#subdirectory=adapters/python/acp_langgraph"
pip install "git+https://github.com/Clawdlinux/agent-contract-protocol.git@main#subdirectory=adapters/python/acp_openai"
pip install "git+https://github.com/Clawdlinux/agent-contract-protocol.git@main#subdirectory=adapters/python/acp_crewai"
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