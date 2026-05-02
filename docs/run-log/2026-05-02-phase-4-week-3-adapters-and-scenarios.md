# 2026-05-02 - Phase 4 / Week 3: Python Adapters + S3-S5 + MCP Source

## Objective

Make ACP usable from the Python agent ecosystem, push the benchmark to all
5 scenarios, and prove the "ACP on top of MCP" framing with a real Go
source adapter.

## Branch

`feat/week3-adapters-and-scenarios` off `main` (`6521b65`).

## What shipped

### Python adapters (`adapters/python/`, Apache 2.0)

- `acp_common`: shared dataclasses (`Manifest`, `Action`, `Boundary`),
  `ACPClient` over `urllib`, schema translation
  (`expand_schema_field`, `action_to_jsonschema`), `topological_order`
  with cycle detection, `upstream_url_for` helper.
- `acp_openai`: `manifest_to_openai_tools()` renders the manifest as the
  OpenAI Chat Completions `tools=[...]` parameter; `dispatch_tool_call()`
  forwards a tool call (dict or SDK object shape) through the proxy.
- `acp_langgraph`: `make_node()` + `build_graph()` produce a
  `StateGraph` whose nodes execute actions in topological order via the
  proxy. Falls back gracefully if `langgraph` isn't installed.
- `acp_crewai`: `make_tool_callable()` + `manifest_to_crewai_tools()`
  produce CrewAI `BaseTool` instances (or plain callables when crewai
  isn't installed).
- 23 pytest cases covering all four packages with mocked `urlopen`
  transports (no live network, no heavy framework deps required).

### Benchmark expansion

- `Scenario` gained `mcp_noise_tools` field.
- `mcp_context_payload()` accepts `noise_tools=N`, spreading generic
  realistic descriptors across MCP servers to model the
  "many-registered-few-relevant" case.
- Harness ships with S1, S2, S3 (DAG), S4 (scale, 50 tools), S5
  (auth-heavy).

### MCP source adapter (`internal/sources/mcp`, BSL 1.1)

- `Importer.ImportSource()` GETs `<base>/tools/list` and registers every
  returned descriptor as an ACP `registry.Tool`.
- Capability inference: tokenize tool name, add verb synonyms
  (create/post -> write, get/list -> read, etc.).
- Schema compaction: convert JSON-Schema properties into the compact
  ACP mini-language (string, int?, string[], enum:a|b|c). This is what
  produces the token reduction.
- Egress allow-list inferred from the source's BaseURL host.
- Consumer-defined `HTTPDoer` interface; mocked with
  `go.uber.org/mock`.
- 7 test cases covering happy path, transport errors, non-2xx responses,
  validation, enum/array/integer schema translation, and capability
  inference fallbacks.

## First measured S1-S5 numbers

50 runs per scenario, tiktoken cl100k_base, live ACP server:

| Scenario | ACP tok/RT | MCP tok/RT | Reduction |
|---|---|---|---|
| S1 Simple DB query | 111 / 1 | 373 / 3 | **70.2%** |
| S2 Multi-tool workflow | 295 / 1 | 837 / 5 | **64.7%** |
| S3 Complex DAG | 306 / 1 | 1,257 / 7 | **75.6%** |
| **S4 Scale (50 tools, 2 relevant)** | **241 / 1** | **9,223 / 21** | **97.4%** |
| S5 Auth-heavy | 359 / 1 | 1,431 / 7 | **74.9%** |

Headline (slide-ready):

> ACP cuts agent tool-context tokens by 65-97% in head-to-head measurements
> against an MCP baseline built per the official 2024-11 spec. At 50
> registered tools with only 2 relevant to the task, the reduction is
> **97.4%**. Round-trips before the first useful action drop from 3-21
> (MCP) to **1** (ACP).

Raw + summary committed at:

- `results/2026-05-02-week3-baseline.json` (250 raw runs)
- `results/2026-05-02-week3-summary.md`

## Validation

- `go vet ./...` clean.
- `go test -race -count=1 ./...` 9 packages green (incl. new
  `internal/sources/mcp`).
- `staticcheck ./...` clean.
- `govulncheck ./...` clean (zero vulnerabilities).
- `pytest benchmark/tests adapters/python/tests` -> 35 passed in 0.11s.

## Deviations from plan

- Embedding-based intent resolver deferred to Week 4. The keyword
  resolver already produces the right capability tags for all 5
  scenarios, and the benchmark already lands the headline number. The
  embedding upgrade is value-add for production use (when intents get
  more varied) but doesn't change the proof.
- The acp_crewai and acp_langgraph adapter tests don't import the
  framework deps directly to keep CI light; they exercise the
  per-action factories (`make_node`, `make_tool_callable`) which is
  where the adapter logic lives. Optional smoke tests against real
  langgraph/crewai can land later.

## Next phase

Week 4:

1. Write the ACP arxiv preprint with the S1-S5 numbers.
2. Generate charts (matplotlib).
3. Embedding-based intent resolver behind the same `Resolver` interface.
4. Public release of `SPEC.md` (CC BY 4.0).
5. Pitch deck data extraction script.
