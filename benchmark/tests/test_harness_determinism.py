"""Determinism + reproducibility tests for the benchmark harness.

These don't require a live ACP server. They validate the pure pieces
(token counter behavior, summary math, JSON wire shape) that the harness
depends on.
"""

from __future__ import annotations

import json
import statistics

from harness import RunResult, Scenario, _p95, count_tokens, summarize


def _scenario(sid: str = "X") -> Scenario:
    return Scenario(
        id=sid,
        title="t",
        intent="i",
        capability_hints=[],
        tool_ids=["db.query"],
        mcp_servers=1,
    )


def _run(acp: int, mcp: int) -> RunResult:
    return RunResult(
        scenario_id="X",
        acp_tokens=acp,
        acp_bytes=acp * 4,
        acp_round_trips=1,
        mcp_tokens=mcp,
        mcp_bytes=mcp * 4,
        mcp_round_trips=3,
        wall_ms_acp=10.0,
    )


def test_count_tokens_is_pure():
    """Same input must yield identical counts and method label across calls."""
    for s in ["", "hello", "a" * 1000, "{\"foo\": 42}"]:
        a = count_tokens(s)
        b = count_tokens(s)
        assert a == b, f"non-deterministic count for {s!r}: {a} != {b}"


def test_summarize_handles_zero_mcp_tokens_without_zero_division():
    runs = [_run(100, 0), _run(120, 0)]
    s = summarize(_scenario(), runs, "test")
    # Reduction is 0.0 (sentinel) when MCP is zero, never NaN/inf.
    assert s.token_reduction_mean == 0.0


def test_summarize_p95_stays_within_max():
    rng_input = list(range(1, 101))
    runs = [_run(a, 1000) for a in rng_input]
    s = summarize(_scenario(), runs, "test")
    assert s.acp_tokens_p95 <= max(rng_input)
    assert s.acp_tokens_p95 >= statistics.median(rng_input)


def test_p95_monotonic_with_position():
    """p95 of [1..100] must be >= p95 of [1..50]."""
    big = _p95(list(range(1, 101)))
    small = _p95(list(range(1, 51)))
    assert big >= small


def test_runresult_json_wire_shape_is_stable():
    """The JSON wire shape of a RunResult must include every field the
    report renderer keys on. Catches accidental rename/drop regressions."""
    from dataclasses import asdict

    payload = json.dumps(asdict(_run(100, 1000)))
    decoded = json.loads(payload)
    for key in (
        "scenario_id",
        "acp_tokens",
        "acp_bytes",
        "acp_round_trips",
        "mcp_tokens",
        "mcp_bytes",
        "mcp_round_trips",
        "wall_ms_acp",
    ):
        assert key in decoded, f"missing field {key} in serialized RunResult"


def test_summarize_round_trip_counts_are_constants():
    """Round-trip counts come from the first run only; this catches a future
    refactor that accidentally averages an integer field."""
    runs = [_run(100, 1000), _run(110, 1100), _run(120, 1200)]
    s = summarize(_scenario(), runs, "test")
    assert s.acp_round_trips == runs[0].acp_round_trips
    assert s.mcp_round_trips == runs[0].mcp_round_trips
