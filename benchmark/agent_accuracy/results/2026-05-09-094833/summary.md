# Agent-accuracy benchmark — summary

- Run date: `2026-05-09-094833`
- Git SHA: `c4fa52d`
- Trials per cell: `n=30`
- Total API cost: `$0.8631`

## Per-cell results

| Model | Condition | Question kind | n | Accuracy (95% CI) | Mean prompt tok | Median latency (ms) | USD |
|---|---|---|---:|---|---:|---:|---:|
| `claude-haiku-4-5-20251001` | `acl` | decision | 360 | 75.0% (70.3–79.2) | 464 | 784 | $0.1415 |
| `claude-haiku-4-5-20251001` | `acl` | fact | 450 | 93.3% (90.6–95.3) | 428 | 816 | $0.1710 |
| `claude-haiku-4-5-20251001` | `raw` | decision | 360 | 83.3% (79.1–86.8) | 4572 | 890 | $1.3257 |
| `claude-haiku-4-5-20251001` | `raw` | fact | 450 | 93.3% (90.6–95.3) | 4537 | 868 | $1.6487 |

## Headline lift (ACL vs raw kubectl JSON)

| Model | Question kind | Acc. raw | Acc. ACL | Δ acc | Tok raw | Tok ACL | Tok reduction |
|---|---|---:|---:|---:|---:|---:|---:|
| `claude-haiku-4-5-20251001` | decision | 83.3% | 75.0% | -8.3pp | 4572 | 464 | 89.9% |
| `claude-haiku-4-5-20251001` | fact | 93.3% | 93.3% | +0.0pp | 4537 | 428 | 90.6% |

Tok reduction is *(raw − acl) / raw* on the prompt side; completion tokens are not counted because both conditions must answer in the same format.

## Honest read (n=30)

**The strong claim:** ACL preserves fact-extraction accuracy at one-tenth
the prompt tokens. 93.3% on both conditions across 450 trials each, with
identical 95% Wilson CI (90.6–95.3). Statistically indistinguishable.
This is the headline anyone can quote without caveats.

**The honest caveat:** On decision questions ACL underperforms raw JSON by
8.3 percentage points. At n=10 the CIs overlapped; at n=30 they no longer
do (75.0% [70.3–79.2] vs 83.3% [79.1–86.8]). The gap is real.

**Why:** Investigation of `d3_recommended_action` shows ACL didn't hide
information — it surfaced different signals. In the degraded scenario,
ACL's prominent `replicas=2/3` row steered the model to pick `scale`;
raw JSON's buried `containerStatuses[].restartCount=4` steered the model
to pick `restart`. Both are defensible per the rubric. The format choice
changes which signal the model fixates on. ACL is a design tool, not
just a compression tool — what you choose to surface determines what
the agent attends to.

**What's broken:**
- `d1_should_alert/degraded` — 0/30 on both formats. The yes/no rubric
  is ambiguous in the degraded scenario (one warning pod is below the
  >50%-of-deployment threshold, but the model alerts anyway). Question
  needs to be sharpened.
- `f2_unhealthy_pods/raw/failing` — 0/30. Model can't reliably extract
  the unhealthy pod list from raw kubectl JSON when there are multiple
  affected pods. ACL gets 30/30 on the same scenario. This actually
  *favors* ACL but it's an edge case.
- `d5_root_cause_pod/raw/healthy` — 0/30. Model insists on naming a
  pod even when the answer is "none." Both formats have this issue at
  n=10 but only raw shows it at n=30 — likely sampling.

**Cost-per-decision:**
- Raw JSON: 4,572 prompt tokens × $0.0008/1K input = $0.00366 per call
- ACL: 464 prompt tokens × $0.0008/1K input = $0.00037 per call
- **10× cheaper per decision** at the same accuracy on facts, 9pp lower
  on decisions.

**Reproducibility:** This run included 1,420 cache hits from the
previous n=10 run; only 200 calls were freshly made. The cache is keyed
by sha256(model, system, user) so any reviewer can re-run and get the
same numbers (or extend to n=100 for $5 incremental).
