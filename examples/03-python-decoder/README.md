# Example 03 — Python agent reading ACL

Shows how a Python agent (LangGraph, CrewAI, OpenAI tool-use loop, or
your own) parses an ACL document and uses the structured fields to
make a decision.

## Setup

```bash
pip install -e python/[tokens]   # one-time
```

## Run it

```bash
python3 examples/03-python-decoder/run.py
```

## What you'll see

```
Loaded ACL doc:
  source:    k8s-namespace/v0.1
  cluster:   prod-east
  namespace: payments

Sections:
  pods:    summary='5/5 ok'  (5 rows)
  deploys: summary='2 all-avail'  (2 rows)
  svcs:    summary='2'  (2 rows)
  actions: summary=''  (1 rows)

Agent decision:
  All 5 pods healthy, all deploys at desired replicas.
  No action required.

Token cost comparison:
  raw kubectl JSON:  3671 tokens
  ACL document:       260 tokens
  Reduction:         14.1x
```

## What's happening

`acp_acl.decode()` parses the ACL wire format into typed Python objects:

- `Document.directives` — `dict[str, str]` of `@key value` lines
- `Document.sections` — list of `Section(name, summary, rows)`
- `Section.rows` — list of `Row(id, count, fields, flags)`

The decoder is **pure Python with zero runtime dependencies** (the
optional `tokens` extra installs `tiktoken` for `count_tokens()`).
That makes it safe to drop into any agent runtime — no compiled
extensions, no version conflicts.

## Files used

- `benchmark/agent_accuracy/fixtures/healthy/state.acl` — the ACL fixture
- `benchmark/agent_accuracy/fixtures/healthy/raw.json` — the original kubectl JSON (for token comparison)
