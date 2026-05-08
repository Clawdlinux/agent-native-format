# Agent-accuracy benchmark — summary

- Run date: `2026-05-03-135034`
- Git SHA: `bde5167`
- Trials per cell: `n=1`
- Total API cost: `$0.0000`

## Per-cell results

| Model | Condition | Question kind | n | Accuracy (95% CI) | Mean prompt tok | Median latency (ms) | USD |
|---|---|---|---:|---|---:|---:|---:|
| `claude-haiku-4-5-20251001` | `acl` | decision | 3 | 33.3% (6.2–79.2) | 456 | 831 | $0.0012 |
| `claude-haiku-4-5-20251001` | `acl` | fact | 5 | 80.0% (37.5–96.4) | 426 | 798 | $0.0019 |
| `claude-haiku-4-5-20251001` | `raw` | decision | 3 | 66.7% (20.8–93.8) | 4567 | 921 | $0.0110 |
| `claude-haiku-4-5-20251001` | `raw` | fact | 5 | 100.0% (56.5–100.0) | 4537 | 1028 | $0.0183 |

## Headline lift (ACL vs raw kubectl JSON)

| Model | Question kind | Acc. raw | Acc. ACL | Δ acc | Tok raw | Tok ACL | Tok reduction |
|---|---|---:|---:|---:|---:|---:|---:|
| `claude-haiku-4-5-20251001` | decision | 66.7% | 33.3% | -33.3pp | 4567 | 456 | 90.0% |
| `claude-haiku-4-5-20251001` | fact | 100.0% | 80.0% | -20.0pp | 4537 | 426 | 90.6% |

Tok reduction is *(raw − acl) / raw* on the prompt side; completion tokens are not counted because both conditions must answer in the same format.
