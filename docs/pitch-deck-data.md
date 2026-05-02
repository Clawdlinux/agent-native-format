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

Pending benchmark execution.
