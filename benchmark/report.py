"""Benchmark report generator.

Reads a JSON file produced by ``harness.py`` and emits a markdown summary
suitable for committing to ``results/`` and pasting into the pitch deck.

Usage:

    python3 benchmark/report.py --in results/run-1714666666.json \
        --out results/summary.md
"""

from __future__ import annotations

import argparse
import json
from pathlib import Path


def render(report: dict) -> str:
    method = report.get("method", "unknown")
    init_bytes = report.get("mcp_initialize_response_bytes", 0)

    lines: list[str] = []
    lines.append("# ACP vs MCP - Context-Window Token Cost")
    lines.append("")
    lines.append(f"- Token counter: `{method}`")
    lines.append(f"- MCP `initialize` response size (per server, included in MCP totals): {init_bytes} bytes")
    lines.append("")

    lines.append("## Per-scenario summary")
    lines.append("")
    lines.append(
        "| Scenario | Runs | ACP tokens (mean / p95) | ACP round-trips | "
        "MCP tokens (mean / p95) | MCP round-trips | Token reduction |"
    )
    lines.append(
        "|---|---|---|---|---|---|---|"
    )

    for s in report.get("scenarios", []):
        lines.append(
            f"| {s['scenario_id']} {s['title']} | {s['runs']} | "
            f"{s['acp_tokens_mean']:.0f} / {s['acp_tokens_p95']:.0f} | {s['acp_round_trips']} | "
            f"{s['mcp_tokens_mean']:.0f} / {s['mcp_tokens_p95']:.0f} | {s['mcp_round_trips']} | "
            f"**{s['token_reduction_mean']*100:.1f}%** |"
        )

    lines.append("")
    lines.append("## Headline")
    lines.append("")
    if report.get("scenarios"):
        best = max(report["scenarios"], key=lambda s: s["token_reduction_mean"])
        worst = min(report["scenarios"], key=lambda s: s["token_reduction_mean"])
        lines.append(
            f"Across measured scenarios, ACP reduces tool-context token cost by "
            f"**{worst['token_reduction_mean']*100:.1f}% to {best['token_reduction_mean']*100:.1f}%** "
            f"compared to an MCP `initialize` + `tools/list` baseline using the same tool set."
        )
        lines.append("")
        lines.append(
            f"Round-trip count drops from up to {worst['mcp_round_trips']}-{best['mcp_round_trips']} "
            f"(MCP) to **1** (ACP) before the agent can take its first task action."
        )

    lines.append("")
    lines.append("## Methodology")
    lines.append("")
    lines.append(
        "- ACP measurements come from real `POST /v1/context` calls against the live ACP server."
    )
    lines.append(
        "- MCP measurements come from a faithful reproduction of the MCP "
        "`initialize` + `tools/list` payloads for the same tool set, built per "
        "the MCP 2024-11 spec."
    )
    lines.append(
        "- Tool descriptors used in the MCP baseline are the verbose JSON-Schema "
        "form emitted by real MCP servers (with `description`, examples, "
        "constraints, and `$schema`)."
    )
    lines.append(
        "- Token counts use the same encoder for both paths (apples-to-apples)."
    )
    lines.append("")
    return "\n".join(lines)


def main(argv: list[str] | None = None) -> int:
    p = argparse.ArgumentParser(description="Render the benchmark JSON as markdown")
    p.add_argument("--in", dest="in_path", required=True)
    p.add_argument("--out", dest="out_path", required=True)
    args = p.parse_args(argv)

    report = json.loads(Path(args.in_path).read_text())
    md = render(report)
    out = Path(args.out_path)
    out.parent.mkdir(parents=True, exist_ok=True)
    out.write_text(md)
    print(f"wrote {out}")
    return 0


if __name__ == "__main__":  # pragma: no cover
    raise SystemExit(main())
