#!/usr/bin/env python3
"""Sanity check that paper/acp.md and paper/acp.tex agree on the headline numbers.

This is a thin guardrail - we hand-mirror the LaTeX from the Markdown so
both versions stay readable. The check enforces that the per-scenario
reduction percentages and round-trip counts in the LaTeX match the
Markdown character-for-character.
"""

from __future__ import annotations

import re
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
MD = ROOT / "paper" / "acp.md"
TEX = ROOT / "paper" / "acp.tex"


def extract_md_numbers(text: str) -> set[tuple[str, str, str, str, str]]:
    out: set[tuple[str, str, str, str, str]] = set()
    pat = re.compile(
        r"^\| (S\d) \| ([\d,]+) / 1 \| ([\d,]+) / (\d+) \| \*\*(\d+\.\d+%)\*\* \|",
        re.MULTILINE,
    )
    for m in pat.finditer(text):
        out.add(m.groups())
    return out


def extract_tex_numbers(text: str) -> set[tuple[str, str, str, str, str]]:
    out: set[tuple[str, str, str, str, str]] = set()
    pat = re.compile(
        r"(S\d)[^&]*&\s*([\d,]+)\s*&\s*([\d,]+)\s*&\s*1\s*&\s*(\d+)\s*&\s*\\textbf\{(\d+\.\d+\\%)\}",
    )
    for m in pat.finditer(text):
        sid, acp, mcp, mcprt, pct = m.groups()
        # Normalize TeX percentage (\\%) to bare %
        pct = pct.replace("\\%", "%")
        out.add((sid, acp, mcp, mcprt, pct))
    return out


def main() -> int:
    if not MD.exists():
        print(f"missing {MD}", file=sys.stderr)
        return 1
    if not TEX.exists():
        print(f"missing {TEX}", file=sys.stderr)
        return 1

    md_nums = extract_md_numbers(MD.read_text(encoding="utf-8"))
    tex_nums = extract_tex_numbers(TEX.read_text(encoding="utf-8"))

    if not md_nums:
        print("no numbers extracted from acp.md - regex broken?", file=sys.stderr)
        return 1
    if not tex_nums:
        print("no numbers extracted from acp.tex - regex broken?", file=sys.stderr)
        return 1

    diff = md_nums.symmetric_difference(tex_nums)
    if diff:
        print("paper/acp.md and paper/acp.tex disagree on these tuples:", file=sys.stderr)
        for d in sorted(diff):
            print(f"  {d}", file=sys.stderr)
        return 1

    print(f"paper ok: {len(md_nums)} scenario rows match across acp.md and acp.tex")
    return 0


if __name__ == "__main__":  # pragma: no cover
    raise SystemExit(main())
