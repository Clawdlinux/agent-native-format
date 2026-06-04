# June 2026 Research Log

Status: initial Phase 0 validation. Retrieved 2026-06-04.

## Summary

The token-bloat pain is real, but it is not enough for the public thesis.
Claude Code now ships MCP Tool Search on by default. Academic and OSS projects
already report large token reductions. CLI conversion is crowded.

The stronger gap is execution governance for autonomous agents: identity,
policy, TTL, revocation, audit, and bounded blast radius.

## Sources checked

| Source | Date checked | Finding |
|---|---:|---|
| Anthropic Claude Code MCP docs | 2026-06-04 | Tool Search is enabled by default. MCP tools are deferred and discovered on demand. Only used tools enter context. |
| Anthropic tool-use docs | 2026-06-04 | Tool use still adds tokens through tool definitions, tool calls, tool results, and tool-use system prompt overhead. |
| MCP tools spec | 2026-06-04 | Tools are model-controlled. The spec recommends user confirmation, input validation, rate limits, output sanitization, timeouts, and audit, but does not enforce them. |
| Cloudflare remote MCP blog | 2026-06-04 | Remote MCP validates demand for OAuth, token indirection, stateful sessions, and production auth. It competes with parts of ACP's auth-control story. |
| Databricks lakehouse docs | 2026-06-04 | The useful analogy is a contract layer over commodity storage and compute. For agents, the missing contract is governed execution. |
| MCP-Zero, arXiv:2506.01056 | 2026-06-04 | Reports 98% token reduction through active tool discovery over 308 servers and 2,797 tools. |
| Semantic Tool Discovery, arXiv:2603.20313 | 2026-06-04 | Reports 99.6% tool-token reduction, 97.1% hit rate at K=3, MRR 0.91, and sub-100ms retrieval. |
| MCP Tool Descriptions Are Smelly, arXiv:2602.14878 v3 | 2026-06-04 | Finds 97.1% of tool descriptions contain at least one smell. Compact variants can preserve reliability while reducing overhead. |
| GitHub search: mcp2cli | 2026-06-04 | `knowsuchagency/mcp2cli` has 2K+ stars and converts MCP, OpenAPI, and GraphQL to CLI at runtime. |
| GitHub search: MCPorter | 2026-06-04 | `openclaw/mcporter` has 4K+ stars and converts MCPs to TypeScript APIs or CLIs. |

## Layer map

| Layer | Question | Current state |
|---|---|---|
| Discovery | What tools exist? | MCP is the standard. |
| Selection | Which tools fit this intent? | Tool Search, MCP-Zero, semantic retrieval, and mcp2cli-style tools compete here. |
| Schema cost | How much context does the catalog burn? | Being solved by model clients and retrieval layers. |
| Ordering | What sequence is valid? | ACP already declares `depends_on`; many systems leave it to the agent. |
| Identity | Who is the agent acting as? | Cloudflare remote MCP validates this pain; no portable contract found yet. |
| Policy | What is allowed? | MCP recommends controls; enforcement is implementation-specific. |
| Execution | Where is the call isolated and enforced? | Proxies and sandboxes exist, but no common contract shape was found. |
| Audit | Can we prove what happened? | No signed, replayable, portable action log standard found in this pass. |
| Feedback | Did it work? | Most systems have local telemetry, not a shared contract loop. |

## Closest prior art

- **MCP itself:** best catalog and transport standard. It does not define a
  signed execution contract.
- **Claude Code Tool Search:** best client-side mitigation for schema bloat in
  Claude Code. It does not solve non-Claude runtimes or policy/audit contracts.
- **MCP-Zero and Semantic Tool Discovery:** strong tool selection work. They do
  not define identity-bound execution or audit.
- **mcp2cli and MCPorter:** strong CLI/API conversion lane. ACP should not lead
  with CLI conversion.
- **Cloudflare remote MCP:** strong remote auth and token-indirection story. It
  is platform-specific and still centers MCP servers rather than a portable
  execution contract.

## Current conclusion

No checked source appears to own the exact primitive ACP is moving toward:
a source-agnostic, signed, identity-bound, ordered, auditable execution contract
for autonomous agent actions.

This is not final proof. It is enough to continue Phase 1. Before public posting,
repeat this check with broader web, HN, Reddit, and vendor searches if available.

## Tooling limits

The automated deep-research skill was unavailable in this environment, so this
log was compiled manually. It uses direct page fetches, GitHub CLI search, repo
memory, and local repo inspection.