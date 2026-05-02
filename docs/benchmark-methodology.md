# Benchmark Methodology

The benchmark must produce numbers strong enough for an investor pitch and
credible enough for technical scrutiny.

## Claim to test

ACP reduces tool-context overhead by **70-85%** versus MCP for identical agent
tasks while improving time to first useful action and reducing auth-related
failures.

## Scenarios

| Scenario | Task | Tools involved | Purpose |
|---|---|---|---|
| S1 Simple | Query a database, return rows | 1 DB tool | Shows MCP overhead even for one tool |
| S2 Multi-tool | Query DB + send Slack + log audit | 3 tools, 2 servers | Typical enterprise workflow |
| S3 Complex DAG | Research -> analyze -> draft -> review -> email | 5 tools, 3 servers | Shows ordering advantage |
| S4 Scale | Same task with 10 MCP servers registered | 50+ tools, 2 relevant | Shows intent-scoping advantage |
| S5 Auth-heavy | Cross-service task with 3 credential types | OAuth + API key + mTLS | Shows auth injection advantage |

## Metrics

| Metric | Measurement | Target |
|---|---|---|
| Token overhead | Tool discovery + schema loading + auth flows, excluding task work | 70-85% reduction |
| Context utilization | Available context after setup | >80% ACP vs ~35% MCP |
| Round trips | Calls before first task action | 1 ACP vs 5-8 MCP |
| Wall-clock latency | Start to first useful action | <500 ms ACP vs 2-5 s MCP |
| Task success rate | Completed without selection/ordering/auth failure | >95% ACP |
| Credential exposure | Credentials in agent context | 0 for ACP |
| Cost per task | GPT-4o pricing equivalent | 3-5x cheaper |

## Controls

- Run each scenario 50 times for ACP and MCP.
- Use the same LLM model and prompt budget for both paths.
- Use `tiktoken` with `cl100k_base` for exact token counting.
- Use the official MCP SDK for the baseline.
- Record OpenTelemetry traces for each run.
- Generate markdown, CSV, chart, and PDF outputs.

## Investor-grade bar

A result is pitch-ready only when:

1. Scenario definitions are checked into `benchmark/scenarios/`.
2. Raw traces and aggregate summaries are reproducible from a clean clone.
3. The report states confidence intervals, not just means.
4. A reviewer can inspect every payload counted as overhead.
