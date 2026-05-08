# Agent-accuracy benchmark — summary

- Run date: `2026-05-03-134736`
- Git SHA: `bde5167`
- Trials per cell: `n=1`
- Total API cost: `$0.0000`

## Per-cell results

| Model | Condition | Question kind | n | Accuracy (95% CI) | Mean prompt tok | Median latency (ms) | USD |
|---|---|---|---:|---|---:|---:|---:|
| `claude-haiku-4-5-20251001` | `acl` | decision | 9 | 55.6% (26.7–81.1) | 459 | 807 | $0.0035 |
| `claude-haiku-4-5-20251001` | `acl` | fact | 15 | 93.3% (70.2–98.8) | 428 | 812 | $0.0057 |
| `claude-haiku-4-5-20251001` | `raw` | decision | 9 | 66.7% (35.4–87.9) | 4567 | 921 | $0.0331 |
| `claude-haiku-4-5-20251001` | `raw` | fact | 15 | 93.3% (70.2–98.8) | 4537 | 923 | $0.0550 |

## Headline lift (ACL vs raw kubectl JSON)

| Model | Question kind | Acc. raw | Acc. ACL | Δ acc | Tok raw | Tok ACL | Tok reduction |
|---|---|---:|---:|---:|---:|---:|---:|
| `claude-haiku-4-5-20251001` | decision | 66.7% | 55.6% | -11.1pp | 4567 | 459 | 90.0% |
| `claude-haiku-4-5-20251001` | fact | 93.3% | 93.3% | +0.0pp | 4537 | 428 | 90.6% |

Tok reduction is *(raw − acl) / raw* on the prompt side; completion tokens are not counted because both conditions must answer in the same format.
