# ACP vs MCP - Context-Window Token Cost

- Token counter: `tiktoken/cl100k_base`
- MCP `initialize` response size (per server, included in MCP totals): 257 bytes

## Per-scenario summary

| Scenario | Runs | ACP tokens (mean / p95) | ACP round-trips | MCP tokens (mean / p95) | MCP round-trips | Token reduction |
|---|---|---|---|---|---|---|
| S1 Simple DB query | 50 | 111 / 114 | 1 | 373 / 373 | 3 | **70.2%** |
| S2 Multi-tool enterprise workflow | 50 | 295 / 297 | 1 | 837 / 837 | 5 | **64.7%** |
| S3 Complex DAG (research, render, audit, email) | 50 | 306 / 308 | 1 | 1257 / 1257 | 7 | **75.6%** |
| S4 Scale: 50 registered tools, 2 relevant | 50 | 241 / 244 | 1 | 9223 / 9223 | 21 | **97.4%** |
| S5 Auth-heavy: cross-service workflow with credential injection | 50 | 359 / 362 | 1 | 1431 / 1431 | 7 | **74.9%** |

## Headline

Across measured scenarios, ACP reduces tool-context token cost by **64.7% to 97.4%** compared to an MCP `initialize` + `tools/list` baseline using the same tool set.

Round-trip count drops from up to 5-21 (MCP) to **1** (ACP) before the agent can take its first task action.

## Methodology

- ACP measurements come from real `POST /v1/context` calls against the live ACP server.
- MCP measurements come from a faithful reproduction of the MCP `initialize` + `tools/list` payloads for the same tool set, built per the MCP 2024-11 spec.
- Tool descriptors used in the MCP baseline are the verbose JSON-Schema form emitted by real MCP servers (with `description`, examples, constraints, and `$schema`).
- Token counts use the same encoder for both paths (apples-to-apples).
