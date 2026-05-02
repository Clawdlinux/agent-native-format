"""Tests for benchmark.harness (token counting + summarization)."""

from __future__ import annotations

import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
sys.path.insert(0, str(ROOT / "benchmark"))

from harness import (  # noqa: E402
    RunResult,
    Scenario,
    _p95,
    count_tokens,
    summarize,
)


def test_count_tokens_returns_positive():
    n, method = count_tokens("hello world this is a test")
    assert n > 0
    assert method in {"tiktoken/cl100k_base", "chars/4"}


def test_count_tokens_grows_with_payload():
    short, _ = count_tokens("a")
    long, _ = count_tokens("a" * 4000)
    assert long > short * 100


def test_p95_handles_empty_and_small():
    assert _p95([]) == 0.0
    assert _p95([5]) == 5.0
    assert _p95(list(range(100))) >= 90.0


def test_summarize_computes_reduction():
    scenario = Scenario(
        id="X",
        title="t",
        intent="i",
        capability_hints=[],
        tool_ids=["db.query"],
        mcp_servers=1,
    )
    runs = [
        RunResult(
            scenario_id="X",
            acp_tokens=100,
            acp_bytes=400,
            acp_round_trips=1,
            mcp_tokens=1000,
            mcp_bytes=4000,
            mcp_round_trips=3,
            wall_ms_acp=12.0,
        ),
        RunResult(
            scenario_id="X",
            acp_tokens=100,
            acp_bytes=400,
            acp_round_trips=1,
            mcp_tokens=1000,
            mcp_bytes=4000,
            mcp_round_trips=3,
            wall_ms_acp=15.0,
        ),
    ]
    s = summarize(scenario, runs, "test")
    assert s.runs == 2
    assert s.acp_tokens_mean == 100
    assert s.mcp_tokens_mean == 1000
    assert abs(s.token_reduction_mean - 0.9) < 1e-6
    assert s.acp_round_trips == 1
    assert s.mcp_round_trips == 3
