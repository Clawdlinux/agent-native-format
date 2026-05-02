"""Tests for benchmark/charts.py.

Verifies the chart renderer produces all 6 expected files (3 charts × svg+png),
that the SVG bodies contain the expected scenario labels, and that an empty
report raises ValueError before any plotting work.
"""

from __future__ import annotations

from pathlib import Path

import pytest

from charts import render


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


def test_render_writes_six_files(tmp_path: Path):
    paths = render(SAMPLE, tmp_path)
    names = {p.name for p in paths}
    assert names == {
        "tokens.svg",
        "tokens.png",
        "round_trips.svg",
        "round_trips.png",
        "reduction.svg",
        "reduction.png",
    }
    for p in paths:
        assert p.exists() and p.stat().st_size > 0


def test_render_svg_contains_scenario_labels(tmp_path: Path):
    render(SAMPLE, tmp_path)
    svg = (tmp_path / "tokens.svg").read_text()
    assert "S1" in svg and "S4" in svg


def test_render_rejects_empty_scenarios(tmp_path: Path):
    with pytest.raises(ValueError):
        render({"scenarios": []}, tmp_path)


def test_main_writes_charts(tmp_path: Path):
    """End-to-end: feed the committed Week-3 baseline JSON through main()."""
    import json

    import charts

    in_path = tmp_path / "in.json"
    in_path.write_text(json.dumps(SAMPLE))
    out_dir = tmp_path / "out"
    rc = charts.main(["--in", str(in_path), "--out-dir", str(out_dir)])
    assert rc == 0
    assert (out_dir / "tokens.png").exists()
    assert (out_dir / "round_trips.png").exists()
    assert (out_dir / "reduction.png").exists()
