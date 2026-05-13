# Frontier Tool-Context Benchmark Plan

This track extends the current ACP vs MCP token-overhead benchmark to open
function-calling standards and frontier model tiers. It is a plan until result
files are committed under `results/`; do not cite model-specific outcomes from
this directory as measured results.

## Primary open standard: BFCL

The primary benchmark target is the Berkeley Function Calling Leaderboard
(BFCL), because it is open, citable, executable, and explicitly evaluates tool
calling with AST-based correctness metrics across single-turn, parallel, and
multi-turn cases.

ACP adds a protocol-overhead profile around BFCL-style tasks:

1. Convert each selected BFCL function set into an MCP-compatible `tools/list`
   payload.
2. Register the same tools in ACP.
3. For each task, compare:
   - MCP descriptor tokens before the model can act.
   - ACP manifest tokens before the model can act.
   - Tool-call correctness using BFCL's evaluator when available.
   - Round trips before first useful action.
   - Latency and provider-reported cost when API runs are enabled.
4. Store raw inputs, outputs, token counts, and evaluator scores in
   `results/frontier/<date>/`.

## Secondary benchmark: tau-bench / tau^3-bench

The secondary target is tau-bench or its current tau^3-bench successor for
multi-turn tool-agent-user interaction. BFCL answers whether the model called
the right functions. tau-style runs answer whether reduced context overhead
changes end-to-end task success and consistency across repeated attempts.

## Model tiers

Model IDs must be pinned in each result file. The paper should use provider
families only until runs exist.

| Tier | Purpose | Candidate models |
|---|---|---|
| Long-context | Tests whether a 1M+ context window removes the need for ACP or merely hides the cost | Gemini 1M+ context model family, or any available 1M+ context model with API access |
| Medium frontier | Tests production-quality tool use at lower cost | Claude Sonnet family; OpenAI GPT-5.x medium/minimal variants if exact IDs are available; comparable Gemini Flash/Pro variants |
| Heavy frontier | Tests best-available reasoning/tool use | Claude Opus family; highest available OpenAI GPT-5.x model; highest available Gemini Pro/Ultra-class model |
| Open model | Provides a reproducible non-proprietary reference point | Qwen, Llama, GLM, DeepSeek, or xLAM model with tool-calling support and an open/research license |

If a requested alias such as `gpt-5.4` or `gpt-5.5` is not exposed by the
provider API, the harness must record the exact replacement ID used instead of
normalizing it to the requested alias.

## Metrics

| Metric | Definition |
|---|---|
| Descriptor tokens | Tokens the model must read before making the first tool call |
| Context utilization | Descriptor tokens divided by advertised context window |
| Reduction | `1 - acp_descriptor_tokens / mcp_descriptor_tokens` |
| Tool-call accuracy | BFCL AST correctness or equivalent evaluator score |
| Tool-selection precision | Correct tools selected divided by tools supplied to the model |
| Round trips | Network/model turns before first task action |
| Cost | Provider-reported input/output token cost when available |
| Latency | Wall-clock time to first useful action and full task completion |

## Non-goals

- Do not claim ACP improves model reasoning unless BFCL/tau-style correctness
  scores support that claim.
- Do not compare against a character-count token estimate when `tiktoken`, a
  provider tokenizer, or provider usage accounting is available.
- Do not publish API keys, prompts containing secrets, or proprietary tool
  descriptors in result artifacts.
