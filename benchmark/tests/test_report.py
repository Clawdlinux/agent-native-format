"""Tests for benchmark/report.py renderer."""

from __future__ import annotations

import json
from pathlib import Path

from report import render


SAMPLE = {
    "method": "tiktoken/cl100k_base",
    "mcp_initialize_response_bytes": 257,
    "scenarios": [
        {
            "scenario_id": "S1",
            "title": "Simple DB query",
            "runs": 50,
            "method": "tiktoken/cl100k_base",
            "acp_tokens_mean": 111.0,
            "acp_tokens_p50": 111.0,
            "acp_tokens_p95": 113.0,
            "acp_round_trips": 1,
            "mcp_tokens_mean": 373.0,
            "mcp_tokens_p50": 373.0,
            "mcp_tokens_p95": 373.0,
            "mcp_round_trips": 3,
            "token_reduction_mean": 0.7023,
            "token_reduction_p50": 0.7024,
        },
        {
            "scenario_id": "S4",
            "title": "Scale: 50 tools, 2 relevant",
            "runs": 50,
            "method": "tiktoken/cl100k_base",
            "acp_tokens_mean": 241.0,
            "acp_tokens_p50": 241.0,
            "acp_tokens_p95": 244.0,
            "acp_round_trips": 1,
            "mcp_tokens_mean": 9223.0,
            "mcp_tokens_p50": 9223.0,
            "mcp_tokens_p95": 9223.0,
            "mcp_round_trips": 21,
            "token_reduction_mean": 0.974,
            "token_reduction_p50": 0.974,
        },
    ],
    "raw": [],
}


def test_render_includes_method_and_per_scenario_lines():
    md = render(SAMPLE)
    assert "tiktoken/cl100k_base" in md
    assert "S1" in md and "S4" in md
    assert "**70.2%**" in md  # S1 reduction
    assert "**97.4%**" in md  # S4 reduction


def test_render_picks_min_and_max_for_headline():
    md = render(SAMPLE)
    assert "70.2% to 97.4%" in md or "97.4% to 70.2%" in md
    # Round-trip range mentioned (worst MCP RT to best MCP RT).
    assert "21" in md and "MCP" in md


def test_render_empty_scenarios_does_not_crash():
    minimal = {"method": "x", "mcp_initialize_response_bytes": 0, "scenarios": [], "raw": []}
    md = render(minimal)
    assert "Methodology" in md  # static section always present
    assert "Headline" in md or "headline" in md.lower() or len(md) > 0


def test_render_round_trip_via_main(tmp_path: Path):
    """report.py main() reads JSON and writes markdown."""
    import report as report_mod

    in_path = tmp_path / "in.json"
    out_path = tmp_path / "out.md"
    in_path.write_text(json.dumps(SAMPLE))
    rc = report_mod.main(["--in", str(in_path), "--out", str(out_path)])
    assert rc == 0
    assert out_path.exists()
    body = out_path.read_text()
    assert "tiktoken/cl100k_base" in body
