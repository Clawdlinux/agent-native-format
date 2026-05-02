# Show HN: ACP — the execution-context layer above MCP (cuts tool tokens 65–97%)

Production agent systems built on MCP routinely spend most of their context
window on tool discovery and verbose JSON-Schema descriptors before the user's
task even begins. We measured this at 65–97% across five workflows.

I built **ACP** — an intent-resolution and execution-planning layer that sits
**on top of** MCP and other tool sources. The agent submits an intent; ACP
returns one intent-scoped Execution Manifest with auth pre-injected, dependency
order pre-computed, and security boundaries declared. Existing MCP servers keep
working unchanged.

**Repository:** https://github.com/Clawdlinux/ninevigil-acp
**Spec (CC BY 4.0):** https://github.com/Clawdlinux/ninevigil-acp/blob/main/SPEC.md
**Paper draft:** https://github.com/Clawdlinux/ninevigil-acp/blob/main/paper/acp.md
**arxiv:** TBD (added at submission time)

## The numbers (50 runs/scenario, `tiktoken/cl100k_base`)

| Scenario | ACP / MCP tokens | RT | Reduction |
|---|---|---|---|
| S1 Simple DB query | 111 / 373 | 1 / 3 | 70.2% |
| S2 Multi-tool workflow | 295 / 837 | 1 / 5 | 64.7% |
| S3 Complex DAG | 306 / 1,257 | 1 / 7 | 75.6% |
| **S4 Scale (50 tools, 2 relevant)** | **241 / 9,223** | **1 / 21** | **97.4%** |
| S5 Auth-heavy | 359 / 1,431 | 1 / 7 | 74.9% |

Mean reduction: **76.6%**. The bigger the tool registry gets, the bigger the
gap: at 50 registered tools (only 2 relevant to the task), ACP returns the
two tools, MCP returns all 50.

## Why "on top of" MCP, not "instead of"

MCP is great at *catalog* (what tools exist?). It's not designed for
production *execution* (what does this specific agent need to do right now?).
Those are different problems. ACP doesn't replace MCP — it consumes MCP's
`tools/list`, compacts the schemas into a typed mini-language, scopes by
intent, injects credentials at a proxy boundary, and pre-computes the
dependency order.

This means:

- Existing MCP servers (GitHub, Slack, Sentry, your internal ones) work
  with ACP day one. No migration.
- The MCP ecosystem becomes ACP's supply chain, not its competitor.
- Adding more sources (REST/OpenAPI, gRPC reflection, Kubernetes
  Services) is a new adapter, not a fork.

The reference implementation includes both an MCP source adapter (Go,
~300 lines) and a Kubernetes Services source adapter (Go, ~200 lines).
The Python adapter suite covers OpenAI function-calling, LangGraph, and
CrewAI.

## Architecture sketch

```
agent ── POST /v1/context ──> ACP server ──┬── reads MCP tools/list
                                           ├── reads REST/gRPC catalogs
                                           └── reads Kubernetes Services
       <── ExecutionManifest ──

agent ── POST /v1/exec/{mid}/{aid} ──> ACP auth proxy ──> upstream tool
        (proxy strips client Authorization, injects server-side creds)
```

## What's in the repo

- Go server (`cmd/acp-server`, ~4,500 LOC).
- Auth-injection reverse proxy (`internal/proxy`).
- Intent resolvers: deterministic keyword (default) + opt-in hash-TF-IDF
  embedding (no model downloads, no GPU).
- Source adapters for MCP and Kubernetes.
- Python adapter suite: OpenAI / LangGraph / CrewAI.
- Benchmark harness with `tiktoken` token counting.
- Faithful MCP 2024-11 payload reproducer for the baseline (so the
  comparison is apples-to-apples, not a strawman).
- 95% Go coverage, 96.8% Python coverage, fuzz tests, live integration
  test that asserts ACP manifests are strictly smaller than the
  upstream MCP `tools/list`.

## Licenses

- Spec (`SPEC.md`): **CC BY 4.0**.
- SDKs and adapters (`pkg/`, `adapters/`): **Apache 2.0**.
- Reference server runtime (`cmd/`, `internal/`): **BSL 1.1**, converts to
  Apache 2.0 on 2029-05-02.

## What I'd love feedback on

1. The schema mini-language. Is the cost/value of dropping JSON-Schema
   constraint metadata (`maxLength`, `pattern`, etc.) the right
   tradeoff? Should there be a verbose mode for cases where it isn't?
2. The auth-injection proxy contract. Particularly the
   `X-ACP-Approval-Required` semantics for human-in-the-loop actions.
3. The MCP source adapter's capability inference (currently splits the
   tool name on `./_-` plus verb synonyms). What does this miss in
   real-world MCP servers you've deployed?
4. Operator integration. The Kubernetes source adapter uses Service
   annotations; would you want a CRD instead? Why?

(For background: I'm a solo founder working on the broader
[NineVigil](https://github.com/Clawdlinux/agentic-operator-core)
agent-orchestration story. ACP is the execution-context primitive
that everything else stacks on.)
