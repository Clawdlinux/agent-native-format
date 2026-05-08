# Agent-accuracy benchmark — summary

- Run date: `2026-05-03-135113`
- Git SHA: `bde5167`
- Trials per cell: `n=1`
- Total API cost: `$0.1216`

## Per-cell results

| Model | Condition | Question kind | n | Accuracy (95% CI) | Mean prompt tok | Median latency (ms) | USD |
|---|---|---|---:|---|---:|---:|---:|
| `claude-haiku-4-5-20251001` | `acl` | decision | 15 | 66.7% (41.7–84.8) | 457 | 765 | $0.0058 |
| `claude-haiku-4-5-20251001` | `acl` | fact | 15 | 93.3% (70.2–98.8) | 428 | 768 | $0.0057 |
| `claude-haiku-4-5-20251001` | `raw` | decision | 15 | 73.3% (48.0–89.1) | 4565 | 828 | $0.0551 |
| `claude-haiku-4-5-20251001` | `raw` | fact | 15 | 93.3% (70.2–98.8) | 4537 | 869 | $0.0550 |

## Headline lift (ACL vs raw kubectl JSON)

| Model | Question kind | Acc. raw | Acc. ACL | Δ acc | Tok raw | Tok ACL | Tok reduction |
|---|---|---:|---:|---:|---:|---:|---:|
| `claude-haiku-4-5-20251001` | decision | 73.3% | 66.7% | -6.7pp | 4565 | 457 | 90.0% |
| `claude-haiku-4-5-20251001` | fact | 93.3% | 93.3% | +0.0pp | 4537 | 428 | 90.6% |

Tok reduction is *(raw − acl) / raw* on the prompt side; completion tokens are not counted because both conditions must answer in the same format.
