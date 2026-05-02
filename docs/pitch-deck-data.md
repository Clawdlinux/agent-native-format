# Pitch Deck Data

This file will hold benchmark numbers that can be pasted directly into investor
materials after the Week 4 report generation pass.

## Core slide

> Perplexity dropped MCP. Independent benchmarks show MCP can burn most of an
> agent's context window before the first user task. We built ACP: one API call,
> complete execution context, auth pre-injected, execution pre-ordered.

## Target proof points

| Proof point | Target |
|---|---|
| Token reduction | 70-85% lower tool-context overhead |
| Cost reduction | 3-5x cheaper at GPT-4o input-token rates |
| Time to first useful action | <500 ms |
| Credential exposure | 0 secrets in agent context |
| Success rate | >95% task completion |

## Demo shape

Live side-by-side:

1. Same task.
2. MCP path loads tools and schemas.
3. ACP path calls `/v1/context` once.
4. Show token counter, latency, and dollar cost.
5. Show manifest boundaries and approval gates.

## Numbers

**First measured benchmark — 2026-05-02, 50 runs/scenario, tiktoken cl100k_base:**

| Scenario | ACP tokens | MCP tokens | Reduction | RT (ACP / MCP) |
|---|---|---|---|---|
| S1 Simple DB query (1 tool) | 111 | 373 | **70.2%** | 1 / 3 |
| S2 Multi-tool workflow (3 tools, 2 servers) | 295 | 837 | **64.7%** | 1 / 5 |

Headline numbers ready to paste into a slide:

> ACP cuts agent tool-context tokens by **65-70%** in head-to-head measurements
> against an MCP baseline built per the official 2024-11 spec. Round-trips
> before the first useful action drop from **3-5 (MCP) to 1 (ACP)**.

Source: [`results/2026-05-02-summary.md`](../results/2026-05-02-summary.md).
Raw data: [`results/2026-05-02-week2-baseline.json`](../results/2026-05-02-week2-baseline.json).
S3-S5 measurements (more tools, intent scoping at scale, auth-heavy paths)
land in Week 3.
