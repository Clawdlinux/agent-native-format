# Pitch Deck Data

This file will hold benchmark numbers that can be pasted directly into investor
materials after the Week 4 report generation pass.

## Core slide

> MCP answers “what tools exist?”. **ACP sits on top of MCP** and answers
> “what exactly do I need to do right now, and how?” — in one API call,
> with auth pre-injected, ordering pre-computed, and security boundaries
> declared. Existing MCP servers keep working unchanged. Independent
> benchmarks show MCP can burn most of an agent’s context window before the
> first user task; ACP cuts that overhead by 65–97%.

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

**Headline benchmark - 2026-05-02, 50 runs/scenario, tiktoken cl100k_base, all 5 scenarios:**

| Scenario | ACP tokens | MCP tokens | Reduction | RT (ACP / MCP) |
|---|---|---|---|---|
| S1 Simple DB query (1 tool) | 111 | 373 | **70.2%** | 1 / 3 |
| S2 Multi-tool workflow (3 tools, 2 servers) | 295 | 837 | **64.7%** | 1 / 5 |
| S3 Complex DAG (4 tools, 3 servers) | 306 | 1,257 | **75.6%** | 1 / 7 |
| **S4 Scale (50 registered, 2 relevant)** | **241** | **9,223** | **97.4%** | **1 / 21** |
| S5 Auth-heavy (5 tools across 3 servers) | 359 | 1,431 | **74.9%** | 1 / 7 |

Headline numbers ready to paste into a slide:

> ACP cuts agent tool-context tokens by **65-97%** in head-to-head measurements
> against an MCP baseline built per the official 2024-11 spec. The bigger the
> tool registry gets, the bigger the gap: at 50 registered tools (only 2
> relevant to the task), ACP delivers a **97.4% reduction** vs MCP's full
> `tools/list` payload. Round-trips before the first useful action drop from
> **3-21 (MCP) to 1 (ACP)**.

Source: [`results/2026-05-02-week3-summary.md`](../results/2026-05-02-week3-summary.md).
Raw data: [`results/2026-05-02-week3-baseline.json`](../results/2026-05-02-week3-baseline.json).
