"""Aggregate per-call results into a paper-ready summary table.

Reads ``raw.csv`` produced by the harness and writes:

- ``summary.csv`` — one row per (model, condition, question_kind)
- ``summary.md``  — a markdown table for paper / blog / README

The aggregation:

- **accuracy**          mean of `correct`, with 95% Wilson CI
- **mean prompt tok**   mean of prompt_tokens (the cost-side claim)
- **median latency ms** p50 of latency_ms
- **total USD**         sum of usd

The headline lift is computed at the bottom of summary.md as
``acl_accuracy / raw_accuracy`` and ``raw_prompt_tok / acl_prompt_tok``
for each (model, question_kind) cell.
"""

from __future__ import annotations

import csv
import math
from collections import defaultdict
from pathlib import Path


def _wilson_ci(successes: int, n: int, z: float = 1.96) -> tuple[float, float]:
    """Wilson score interval for a binomial proportion. More accurate
    than the normal-approximation interval at small n / extreme p."""
    if n == 0:
        return (0.0, 0.0)
    p = successes / n
    denom = 1.0 + z * z / n
    centre = (p + z * z / (2 * n)) / denom
    half = (z * math.sqrt((p * (1 - p) + z * z / (4 * n)) / n)) / denom
    return (max(0.0, centre - half), min(1.0, centre + half))


def aggregate(raw_csv: Path) -> list[dict]:
    """Group raw rows by (model, condition, question_kind) and compute
    the per-cell summary statistics."""
    buckets: dict[tuple[str, str, str], list[dict]] = defaultdict(list)
    with raw_csv.open() as f:
        for row in csv.DictReader(f):
            key = (row["model"], row["condition"], row["question_kind"])
            buckets[key].append(row)

    summary: list[dict] = []
    for (model, condition, kind), rows in sorted(buckets.items()):
        n = len(rows)
        correct = sum(1 for r in rows if r["correct"] == "1")
        prompt_toks = [int(r["prompt_tokens"]) for r in rows]
        latencies = sorted(float(r["latency_ms"]) for r in rows)
        usd = sum(float(r["usd"]) for r in rows)
        accuracy = correct / n
        lo, hi = _wilson_ci(correct, n)
        median_latency = latencies[len(latencies) // 2] if latencies else 0.0
        summary.append(
            {
                "model": model,
                "condition": condition,
                "question_kind": kind,
                "n": n,
                "correct": correct,
                "accuracy": round(accuracy, 4),
                "ci_low": round(lo, 4),
                "ci_high": round(hi, 4),
                "mean_prompt_tokens": round(sum(prompt_toks) / n, 1),
                "median_latency_ms": round(median_latency, 1),
                "total_usd": round(usd, 4),
            }
        )
    return summary


def write_summary_csv(summary: list[dict], out: Path) -> None:
    fields = [
        "model",
        "condition",
        "question_kind",
        "n",
        "correct",
        "accuracy",
        "ci_low",
        "ci_high",
        "mean_prompt_tokens",
        "median_latency_ms",
        "total_usd",
    ]
    with out.open("w", newline="") as f:
        w = csv.DictWriter(f, fieldnames=fields)
        w.writeheader()
        for row in summary:
            w.writerow(row)


def write_summary_md(summary: list[dict], out: Path, meta: dict) -> None:
    """Write a paper-ready markdown summary."""
    lines: list[str] = []
    lines.append("# Agent-accuracy benchmark — summary")
    lines.append("")
    lines.append(f"- Run date: `{meta.get('date', '?')}`")
    lines.append(f"- Git SHA: `{meta.get('git_sha', '?')}`")
    lines.append(f"- Trials per cell: `n={meta.get('trials', '?')}`")
    lines.append(f"- Total API cost: `${meta.get('total_usd', 0):.4f}`")
    lines.append("")
    lines.append("## Per-cell results")
    lines.append("")
    lines.append("| Model | Condition | Question kind | n | Accuracy (95% CI) | Mean prompt tok | Median latency (ms) | USD |")
    lines.append("|---|---|---|---:|---|---:|---:|---:|")
    for r in summary:
        ci = f"{r['accuracy']*100:.1f}% ({r['ci_low']*100:.1f}–{r['ci_high']*100:.1f})"
        lines.append(
            f"| `{r['model']}` | `{r['condition']}` | {r['question_kind']} | {r['n']} | "
            f"{ci} | {r['mean_prompt_tokens']:.0f} | {r['median_latency_ms']:.0f} | "
            f"${r['total_usd']:.4f} |"
        )
    lines.append("")

    # Headline lift table: ACL vs raw, per (model, question_kind).
    lines.append("## Headline lift (ACL vs raw kubectl JSON)")
    lines.append("")
    lines.append("| Model | Question kind | Acc. raw | Acc. ACL | Δ acc | Tok raw | Tok ACL | Tok reduction |")
    lines.append("|---|---|---:|---:|---:|---:|---:|---:|")
    by_pair: dict[tuple[str, str], dict[str, dict]] = defaultdict(dict)
    for r in summary:
        by_pair[(r["model"], r["question_kind"])][r["condition"]] = r
    for (model, kind), pair in sorted(by_pair.items()):
        raw = pair.get("raw")
        acl = pair.get("acl")
        if not raw or not acl:
            continue
        delta = acl["accuracy"] - raw["accuracy"]
        delta_str = f"{delta*100:+.1f}pp"
        tok_red = (raw["mean_prompt_tokens"] - acl["mean_prompt_tokens"]) / raw["mean_prompt_tokens"]
        lines.append(
            f"| `{model}` | {kind} | {raw['accuracy']*100:.1f}% | {acl['accuracy']*100:.1f}% | "
            f"{delta_str} | {raw['mean_prompt_tokens']:.0f} | {acl['mean_prompt_tokens']:.0f} | "
            f"{tok_red*100:.1f}% |"
        )
    lines.append("")
    lines.append("Tok reduction is *(raw − acl) / raw* on the prompt side; "
                 "completion tokens are not counted because both conditions "
                 "must answer in the same format.")
    out.write_text("\n".join(lines) + "\n")
