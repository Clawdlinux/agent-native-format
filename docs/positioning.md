# Positioning: ACP on top of MCP

> The strategic positioning of ACP. Companion to the technical spec
> ([SPEC.md §10](../SPEC.md#10-relationship-to-mcp)).
> Status: 2026-05-03.

## One-line statement

> MCP answers "what tools exist?". **ACP sits on top of MCP** and answers
> "what exactly do I need to do right now, and how?".

## The metaphor

MCP is the USB port. ACP is the OS device manager that only loads the
drivers your current task needs.

- **MCP**: catalog protocol. Every server publishes a `tools/list`. Every
  agent must learn that surface up front, every session, every connection.
- **ACP**: execution-context protocol. Agent submits intent + identity.
  Server returns one manifest with the exact tools, compact schemas,
  pre-injected credentials, declared dependency order, and security
  boundaries.

## Why "on top of" beats "successor to"

The earlier draft of ACP framed itself as a successor to MCP. That framing
made for a punchy pitch line but cost us every adoption advantage:

| Issue | "Successor" framing | "On top of" framing |
|---|---|---|
| Anthropic / MCP authors | Adversarial | Aligned |
| Existing MCP server ecosystem | Threat to be replaced | Supply chain to consume |
| "You can't out-ecosystem MCP" objection | Fatal | Doesn't apply |
| Day-1 adoption story | "Migrate off MCP" (high friction) | "Point ACP at your existing MCP servers" (low friction) |
| Future MCP improvements | Threats that erode our edge | Free upgrades to our supply chain |

The technical claims are unchanged. The 70.2% / 64.7% / 75.6% / **97.4%**
/ 74.9% S1-S5 token reductions stand either way. Only the narrative changes
from "we beat MCP" to "we make MCP economical at production scale."

## How ACP layers on MCP in code

The reference implementation (this repo) ships an MCP source adapter at
[`internal/sources/mcp`](../internal/sources/mcp/source.go). The flow:

```
agent ── POST /v1/context ──> ACP server ── reads MCP tools/list ──> any MCP server
       <── ExecutionManifest ──
```

The `Importer.ImportSource(Source)` call:

1. `GET <base>/tools/list` from any MCP-compliant server.
2. For each descriptor: infer capability tags from name + verb synonyms.
3. Compact the verbose JSON-Schema into the ACP mini-language (defined in
   [SPEC.md §4.4](../SPEC.md#44-schema-mini-language)).
4. Register the tool as `<source_name>.<tool_name>` with the source's host
   on the egress allow-list and `auth: pre-injected` so the proxy handles
   credentials.

The compaction is what produces the token reduction. A real MCP `tools/list`
descriptor for `github.issues.create` weighs ~1.5 KB; the compact ACP form
is ~80 bytes for the same executable meaning. Across a 5-tool workflow
this is the difference between 1,431 tokens and 359 tokens (S5 measured).

## What we keep, what we open, what we don't ship

| Layer | License | Why |
|---|---|---|
| Protocol spec (`SPEC.md`, `docs/protocol.md`) | CC BY 4.0 | Maximize ecosystem adoption |
| SDKs and adapters (`pkg/`, `adapters/`) | Apache 2.0 | Make it trivial to consume from any agent framework |
| Server runtime (`cmd/`, `internal/`) | BSL 1.1, 3-year Apache conversion | Protect the optimized runtime + data flywheel until adoption catches up |

We do **not** ship: a competing tool catalog, an MCP-server runtime, or any
piece of the MCP stack. Anthropic's MCP team continues to own that surface.
ACP is purely the layer above it.

## What "compatible" means concretely

If an organization is already running MCP servers:

1. They can keep them running unchanged.
2. They point ACP at the MCP server (one-line registration).
3. Agents call ACP instead of the MCP server directly.
4. Token cost drops 65-97% (measured); auth no longer enters agent context.
5. New tools added to the MCP server appear in the ACP registry on next
   refresh (no re-deploy of ACP).

## SDK / ecosystem strategy

We do **not** plan to compete with Anthropic on SDK breadth. ACP is one
HTTP endpoint plus a typed manifest parser per language:

- The "client SDK" is a thin HTTP wrapper (~200 lines per language).
- We ship Python (`acp_common`, `acp_openai`, `acp_langgraph`,
  `acp_crewai`) and Go (`pkg/acp`) initially.
- Anything that can `POST /v1/context` and parse JSON can consume ACP.

The proprietary value is server-side: intent resolution, schema compaction,
credential injection, dependency ordering, and the optimizer that gets
better with usage data.

## Concrete fundraising / pitch implications

- We are **not** asking investors to bet on us replacing MCP.
- We are asking them to bet on the **execution-context layer** being a
  category, with us as the reference implementation.
- The MCP ecosystem is a feature, not a threat. Every new MCP server is
  another tool ACP can scope intent over.
- The data flywheel (manifest -> execution -> feedback -> better manifest)
  is the long-term moat, not the protocol itself.

## References inside this repo

- [SPEC.md §10 Relationship to MCP](../SPEC.md#10-relationship-to-mcp)
- [internal/sources/mcp/source.go](../internal/sources/mcp/source.go)
- [tests/integration/mcp_source_to_acp_test.go](../tests/integration/mcp_source_to_acp_test.go)
- [docs/architecture.md](./architecture.md)
- [docs/pitch-deck-data.md](./pitch-deck-data.md)
- [docs/LICENSING.md](./LICENSING.md)
