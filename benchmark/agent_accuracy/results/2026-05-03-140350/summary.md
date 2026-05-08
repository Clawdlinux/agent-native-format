# Agent-accuracy benchmark — summary

- Run date: `2026-05-03-140350`
- Git SHA: `bde5167`
- Trials per cell: `n=1`
- Total API cost: `$0.0000`

## Per-cell results

| Model | Condition | Question kind | n | Accuracy (95% CI) | Mean prompt tok | Median latency (ms) | USD |
|---|---|---|---:|---|---:|---:|---:|
| `ollama:gemma3:1b` | `acl` | decision | 4 | 0.0% (0.0–49.0) | 450 | 6622 | $0.0000 |
| `ollama:gemma3:1b` | `acl` | fact | 5 | 40.0% (11.8–76.9) | 417 | 4486 | $0.0000 |
| `ollama:gemma3:1b` | `raw` | decision | 4 | 0.0% (0.0–49.0) | 4096 | 54745 | $0.0000 |
| `ollama:gemma3:1b` | `raw` | fact | 5 | 0.0% (0.0–43.5) | 4096 | 41544 | $0.0000 |

## Headline lift (ACL vs raw kubectl JSON)

| Model | Question kind | Acc. raw | Acc. ACL | Δ acc | Tok raw | Tok ACL | Tok reduction |
|---|---|---:|---:|---:|---:|---:|---:|
| `ollama:gemma3:1b` | decision | 0.0% | 0.0% | +0.0pp | 4096 | 450 | 89.0% |
| `ollama:gemma3:1b` | fact | 0.0% | 40.0% | +40.0pp | 4096 | 417 | 89.8% |

Tok reduction is *(raw − acl) / raw* on the prompt side; completion tokens are not counted because both conditions must answer in the same format.
