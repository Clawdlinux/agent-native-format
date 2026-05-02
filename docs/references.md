# References

This is a working reference list extracted from the confidential ACP PoC source
document. Each item should be replaced with canonical links before public
release.

## MCP cost and friction evidence

| Source | Finding | Impact |
|---|---|---|
| Perplexity (Mar 2026) | CTO Denis Yarats said Perplexity is moving away from MCP internally | Cited context-window waste and auth friction |
| Apideck benchmark | 3 MCP servers (GitHub, Slack, Sentry) consumed 143K of 200K tokens | 72% context window gone before first user query |
| Benjamin Eckstein | 3 MCP servers consumed 22,000 tokens before any prompt | Replaced with shell scripts and saved 22K tokens |
| Scalekit benchmark | 75 head-to-head tests: MCP costs 4-32x more tokens than CLI | Repo language check: 1,365 tokens (CLI) vs 44,026 (MCP) |
| arXiv 2602.14878 | MCP tool descriptions are verbose, redundant, and ambiguous | Better descriptions can improve efficiency but do not fix discovery tax |

## Alternatives and adjacent work

| Approach | What it solves | Gap ACP targets |
|---|---|---|
| Cloudflare Code Mode | Agents write code against typed SDKs instead of loading schemas | Requires code generation and is Cloudflare-specific |
| CLI progressive disclosure | Replaces upfront schemas with on-demand `--help` | Still multi-step; no auth injection or ordering |
| AgentSpec (`agent.yaml`) | Defines agent model, tools, and guardrails | Describes the agent, not runtime execution context |
| A2A Agent Cards | Agent discovery via well-known JSON | Discovery only; no auth, ordering, or token optimization |
| Oracle Open Agent Spec | Declarative workflow definition | Design-time workflow, not runtime intent-scoped context |

## Verification tasks before publication

- Add canonical URLs for each reference.
- Archive key benchmark pages or quote blocks.
- Separate measured data from opinion / interpretation.
- Mark any unverified claim as internal thesis until confirmed.
