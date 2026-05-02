"""Render benchmark charts from a results JSON file.

Reads ``results/<dated>-baseline.json`` and writes:

  - ``results/charts/tokens.svg`` (and .png) - bar chart of ACP vs MCP
    tokens per scenario.
  - ``results/charts/round_trips.svg`` (and .png) - bar chart of RTs.
  - ``results/charts/reduction.svg`` (and .png) - reduction percentage by
    scenario.

Usage:

    python3 benchmark/charts.py \\
        --in results/2026-05-02-week3-baseline.json \\
        --out-dir results/charts

License: Apache-2.0
"""

from __future__ import annotations

import argparse
import json
import os
import sys
from pathlib import Path


def load(path: Path) -> dict:
    return json.loads(path.read_text())


def render(report: dict, out_dir: Path) -> list[Path]:
    """Render the three pitch-deck charts. Returns the paths written."""
    # Headless backend so this works in CI without a display.
    os.environ.setdefault("MPLBACKEND", "Agg")
    import matplotlib  # noqa: PLC0415  - intentional late import after env tweak.

    matplotlib.use("Agg", force=True)
    import matplotlib.pyplot as plt  # noqa: PLC0415

    scenarios = report.get("scenarios", [])
    if not scenarios:
        raise ValueError("no scenarios in report")

    out_dir.mkdir(parents=True, exist_ok=True)
    written: list[Path] = []

    labels = [s["scenario_id"] for s in scenarios]
    acp = [s["acp_tokens_mean"] for s in scenarios]
    mcp = [s["mcp_tokens_mean"] for s in scenarios]
    rt_acp = [s["acp_round_trips"] for s in scenarios]
    rt_mcp = [s["mcp_round_trips"] for s in scenarios]
    reductions = [s["token_reduction_mean"] * 100 for s in scenarios]

    # 1) Tokens: log-scale grouped bar.
    fig, ax = plt.subplots(figsize=(8, 4.5))
    width = 0.36
    xs = list(range(len(labels)))
    ax.bar([x - width / 2 for x in xs], mcp, width, label="MCP", color="#9aa0a6")
    ax.bar([x + width / 2 for x in xs], acp, width, label="ACP", color="#1a73e8")
    ax.set_xticks(xs)
    ax.set_xticklabels(labels)
    ax.set_yscale("log")
    ax.set_ylabel("tokens (log scale, lower is better)")
    ax.set_title("Tool-context tokens per request: ACP vs MCP")
    ax.grid(True, axis="y", linestyle="--", alpha=0.4)
    ax.legend()
    fig.tight_layout()
    written.extend(_save(fig, out_dir, "tokens"))
    plt.close(fig)

    # 2) Round trips: linear-scale grouped bar.
    fig, ax = plt.subplots(figsize=(8, 4.5))
    ax.bar([x - width / 2 for x in xs], rt_mcp, width, label="MCP", color="#9aa0a6")
    ax.bar([x + width / 2 for x in xs], rt_acp, width, label="ACP", color="#1a73e8")
    ax.set_xticks(xs)
    ax.set_xticklabels(labels)
    ax.set_ylabel("round trips before first useful action")
    ax.set_title("Round trips per request: ACP vs MCP")
    ax.grid(True, axis="y", linestyle="--", alpha=0.4)
    ax.legend()
    fig.tight_layout()
    written.extend(_save(fig, out_dir, "round_trips"))
    plt.close(fig)

    # 3) Reduction percentage.
    fig, ax = plt.subplots(figsize=(8, 4.5))
    bars = ax.bar(xs, reductions, color="#34a853")
    for bar, pct in zip(bars, reductions, strict=True):
        ax.text(
            bar.get_x() + bar.get_width() / 2,
            bar.get_height() + 0.5,
            f"{pct:.1f}%",
            ha="center",
            va="bottom",
            fontsize=10,
        )
    ax.set_xticks(xs)
    ax.set_xticklabels(labels)
    ax.set_ylabel("token reduction (%)")
    ax.set_title("ACP token reduction over MCP per scenario")
    ax.set_ylim(0, 100)
    ax.grid(True, axis="y", linestyle="--", alpha=0.4)
    fig.tight_layout()
    written.extend(_save(fig, out_dir, "reduction"))
    plt.close(fig)

    return written


def _save(fig, out_dir: Path, name: str) -> list[Path]:
    paths = [out_dir / f"{name}.svg", out_dir / f"{name}.png"]
    for p in paths:
        fig.savefig(p, bbox_inches="tight")
    return paths


def main(argv: list[str] | None = None) -> int:
    p = argparse.ArgumentParser(description="Render benchmark charts from a results JSON")
    p.add_argument("--in", dest="in_path", required=True)
    p.add_argument("--out-dir", dest="out_dir", default="results/charts")
    args = p.parse_args(argv)

    report = load(Path(args.in_path))
    try:
        written = render(report, Path(args.out_dir))
    except ImportError as e:
        print(f"matplotlib is required: {e}", file=sys.stderr)
        return 2
    for w in written:
        print(f"wrote {w}")
    return 0


if __name__ == "__main__":  # pragma: no cover
    raise SystemExit(main())
