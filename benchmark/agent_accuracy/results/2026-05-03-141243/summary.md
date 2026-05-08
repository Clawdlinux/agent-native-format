# Agent-accuracy benchmark — summary

- Run date: `2026-05-03-141243`
- Git SHA: `bde5167`
- Trials per cell: `n=10`
- Total API cost: `$1.0957`

## Per-cell results

| Model | Condition | Question kind | n | Accuracy (95% CI) | Mean prompt tok | Median latency (ms) | USD |
|---|---|---|---:|---|---:|---:|---:|
| `claude-haiku-4-5-20251001` | `acl` | decision | 120 | 75.0% (66.6–81.9) | 464 | 780 | $0.0472 |
| `claude-haiku-4-5-20251001` | `acl` | fact | 150 | 93.3% (88.2–96.3) | 428 | 815 | $0.0570 |
| `claude-haiku-4-5-20251001` | `raw` | decision | 120 | 83.3% (75.6–88.9) | 4572 | 888 | $0.4419 |
| `claude-haiku-4-5-20251001` | `raw` | fact | 150 | 93.3% (88.2–96.3) | 4537 | 897 | $0.5496 |

## Headline lift (ACL vs raw kubectl JSON)

| Model | Question kind | Acc. raw | Acc. ACL | Δ acc | Tok raw | Tok ACL | Tok reduction |
|---|---|---:|---:|---:|---:|---:|---:|
| `claude-haiku-4-5-20251001` | decision | 83.3% | 75.0% | -8.3pp | 4572 | 464 | 89.9% |
| `claude-haiku-4-5-20251001` | fact | 93.3% | 93.3% | +0.0pp | 4537 | 428 | 90.6% |

Tok reduction is *(raw − acl) / raw* on the prompt side; completion tokens are not counted because both conditions must answer in the same format.

## Honest read

- **Fact extraction**: ACL preserves accuracy at 1/10 the tokens. This is the strong, defensible claim of the run.
- **Decision making**: ACL underperforms raw by 8.3pp at n=10 (CIs overlap). Inspection of `d3_recommended_action`
  showed all 20 ACL trials in degraded/failing scenarios chose `scale` while all 20 raw trials chose `restart`.
  Both are defensible per the rubric — ACL surfaces the under-replicated deploy signal prominently
  (`replicas=1/3`), while raw kubectl JSON makes per-pod restart counts more salient
  (`containerStatuses[].restartCount`). The choice of representation steers reasoning, not just cost.
  A future run with a sharpened rubric and a larger model is needed to separate "ACL is worse for
  decisions" from "ACL changes which signal the model fixates on."
- **Limitations**: single model; n=10; three scenarios; OpenAI quota was unavailable on the test account
  so OpenAI numbers are missing; the local Ollama (`gemma3:1b`) model was too small to follow the
  multi-line rubric and is excluded.

## Reproducibility

```bash
git checkout bde5167  # the SHA in this file's header
make agent-bench-fixtures
python -m benchmark.agent_accuracy.harness \
    --models claude-haiku-4-5-20251001 \
    --trials 10 \
    --max-tokens 80 \
    --max-usd 2.0
```

Set `ANTHROPIC_API_KEY` in `.env` first. The harness caches responses on disk, so a re-run after
a crash resumes for free.
