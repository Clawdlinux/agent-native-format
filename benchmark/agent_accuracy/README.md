# Agent-accuracy benchmark

This benchmark answers the **behavioural** claim of ACL: do LLM agents
actually perform better when fed Agent Context Language than when fed
the same data as raw kubectl JSON?

The compression number alone (`--bytes / +bytes`) is a *cost* claim.
The accuracy number is the *capability* claim: smaller context AND
better answers. The paper needs both.

## Method

For each of three Kubernetes scenarios — **healthy**, **degraded**,
**failing** — we generate two representations of the *same underlying
state*:

- **`raw_json/`** — the bytes an agent gets today from a kubectl MCP
  server: pretty-printed `kubectl get pods,deploy,svc -o json`.
- **`acl/`** — the same data after running through `pkg/aclk8s`.

We then ask each model the same 10 questions per scenario (5 fact +
5 decision) under both conditions. Each (model × scenario × question ×
condition) cell is repeated `n=30` times to drive sampling noise down.

Per cell we record:

| Metric              | What it measures                                     |
|---------------------|------------------------------------------------------|
| `correct`           | grader judgement (0 or 1)                            |
| `prompt_tokens`     | counted via the model's native tokenizer             |
| `completion_tokens` | from the API response                                |
| `latency_ms`        | wall time for the API call                           |
| `usd`               | `prompt * in_rate + completion * out_rate`           |

Aggregated by `(model, condition, question_kind)`:
- accuracy = mean(correct), with binomial 95% CI
- mean prompt tokens
- p50 / p95 latency
- total USD

## Models

- `gpt-4o-mini` (OpenAI; tokenizer: `o200k_base` via tiktoken)
- `claude-3-5-haiku-20241022` (Anthropic; tokenizer: native via SDK)

Two providers => two tokenizers, so the result isn't an artefact of one
vendor's tokenisation.

## Running it

The harness reads `OPENAI_API_KEY` and `ANTHROPIC_API_KEY` from the
environment. Missing keys cause that provider to be skipped cleanly.

```bash
# from repo root
make agent-bench           # full run with current defaults (n=30)
make agent-bench-quick     # n=3 for smoke-testing the pipeline
make agent-bench-fixtures  # regenerate raw_json/ and acl/ from scratch
```

Or directly:

```bash
python -m benchmark.agent_accuracy.harness \
    --scenarios healthy,degraded,failing \
    --models gpt-4o-mini,claude-3-5-haiku-20241022 \
    --trials 30 \
    --out benchmark/agent_accuracy/results/$(date +%Y-%m-%d)
```

## Reproducibility

- All prompts, scenarios, and questions are checked in
  ([`prompts.py`](./prompts.py), [`questions.yaml`](./questions.yaml),
  [`fixtures/`](./fixtures/)).
- Random seeds for trial ordering are derived from the run timestamp
  and recorded in `summary.json`.
- API responses are cached on disk by `(prompt_sha256, model)` so a
  re-run after a crash resumes from the cache; pass `--no-cache` to
  force fresh calls.

## Output layout

```
results/YYYY-MM-DD/
├── raw.csv         # one row per API call (n_trials × ...)
├── summary.csv     # aggregated by (model, condition, question_kind)
├── summary.md      # paper-ready table + headline numbers
└── meta.json       # git SHA, model versions, seed, total USD spent
```

`raw.csv` and per-run files are gitignored. The aggregate `summary.md`
is intended to be checked in.

## Cost guard

Before any API call, the harness estimates total cost from
`scenarios × questions × models × conditions × trials × max_tokens`
and refuses to start if it exceeds `--max-usd` (default `2.0`). Pass
`--max-usd 10` to override.
