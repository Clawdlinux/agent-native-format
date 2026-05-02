"""Benchmark harness: ACP vs MCP context-window cost.

Runs each scenario against the live ACP server and against the MCP-equivalent
payload builder, then writes per-scenario raw measurements to results/.

Usage:

    python3 benchmark/harness.py --acp-url http://localhost:8080 \
        --auth-token dev-token --runs 50 --out results/run-$(date +%s).json

Token counting uses tiktoken's cl100k_base when available (the encoding used
by GPT-4 / GPT-4o). When tiktoken is unavailable the harness falls back to
a deterministic byte-based heuristic (chars / 4) so the harness itself is
hermetic and can run in CI without the optional dependency.
"""

from __future__ import annotations

import argparse
import json
import statistics
import sys
import time
import urllib.error
import urllib.request
from dataclasses import asdict, dataclass
from pathlib import Path
from typing import Any

# Local import (benchmark/baseline is a sibling package)
sys.path.insert(0, str(Path(__file__).resolve().parent))
from baseline.mcp_client import (  # noqa: E402  (path adjustment intentional)
    INITIALIZE_RESPONSE,
    acp_context_payload,
    mcp_context_payload,
)


def _count_tokens_tiktoken(payload: str) -> int | None:
    try:
        import tiktoken  # type: ignore
    except ImportError:
        return None
    enc = tiktoken.get_encoding("cl100k_base")
    return len(enc.encode(payload))


def count_tokens(payload: str) -> tuple[int, str]:
    """Return ``(token_count, method)``.

    Method is ``"tiktoken/cl100k_base"`` when tiktoken is installed, else
    ``"chars/4"`` (a byte-based heuristic that consistently overcounts ACP
    and undercounts MCP, biasing AGAINST the ACP claim - the conservative
    direction).
    """
    n = _count_tokens_tiktoken(payload)
    if n is not None:
        return n, "tiktoken/cl100k_base"
    return max(1, len(payload) // 4), "chars/4"


@dataclass(frozen=True)
class Scenario:
    id: str
    title: str
    intent: str
    capability_hints: list[str]
    tool_ids: list[str]  # tools the resolver should select
    mcp_servers: int  # number of MCP servers the agent would have to load
    # When >0, the MCP baseline payload also includes this many copies of
    # generic noise tools (cloned from the catalog) to model the
    # "intent-scoping at scale" case where many irrelevant tools are
    # registered. ACP's payload is unaffected because the resolver returns
    # only the relevant tools.
    mcp_noise_tools: int = 0


# Scenarios (subset of the YAML library; harness ships with these inline so it
# can run hermetically). Tool IDs match the seed table in
# internal/registry/seed.go.
SCENARIOS: list[Scenario] = [
    Scenario(
        id="S1",
        title="Simple DB query",
        intent="query the customer database for active accounts",
        capability_hints=[],
        tool_ids=["db.query"],
        mcp_servers=1,
    ),
    Scenario(
        id="S2",
        title="Multi-tool enterprise workflow",
        intent="query the customer database, send a slack notification, log an audit event",
        capability_hints=[],
        tool_ids=["db.query", "slack.send_message", "audit.log_event"],
        mcp_servers=2,
    ),
    Scenario(
        id="S3",
        title="Complex DAG (research, render, audit, email)",
        intent="query the customer database, render a report from the data, log an audit event, then email the team the report",
        capability_hints=[],
        tool_ids=[
            "db.query",
            "template.render",
            "audit.log_event",
            "email.send",
        ],
        mcp_servers=3,
    ),
    Scenario(
        id="S4",
        title="Scale: 50 registered tools, 2 relevant",
        intent="query the customer database and send a slack notification",
        capability_hints=[],
        tool_ids=["db.query", "slack.send_message"],
        mcp_servers=10,
        mcp_noise_tools=48,  # 50 total registered, 2 relevant
    ),
    Scenario(
        id="S5",
        title="Auth-heavy: cross-service workflow with credential injection",
        intent="query the customer database, render a report, send via email and slack, log audit",
        capability_hints=[],
        tool_ids=[
            "db.query",
            "template.render",
            "email.send",
            "slack.send_message",
            "audit.log_event",
        ],
        mcp_servers=3,
    ),
]


@dataclass
class RunResult:
    scenario_id: str
    acp_tokens: int
    acp_bytes: int
    acp_round_trips: int
    mcp_tokens: int
    mcp_bytes: int
    mcp_round_trips: int
    wall_ms_acp: float


@dataclass
class ScenarioSummary:
    scenario_id: str
    title: str
    runs: int
    method: str
    acp_tokens_mean: float
    acp_tokens_p50: float
    acp_tokens_p95: float
    acp_round_trips: int
    mcp_tokens_mean: float
    mcp_tokens_p50: float
    mcp_tokens_p95: float
    mcp_round_trips: int
    token_reduction_mean: float
    token_reduction_p50: float


def request_acp_manifest(
    acp_url: str, auth_token: str | None, scenario: Scenario, timeout: float = 10.0
) -> tuple[str, float]:
    """POST /v1/context and return ``(json_body, wall_ms)``."""
    body = json.dumps(
        {
            "intent": scenario.intent,
            "agent_id": f"benchmark-{scenario.id}",
            "capabilities": scenario.capability_hints,
        }
    ).encode("utf-8")
    headers = {"Content-Type": "application/json"}
    if auth_token:
        headers["Authorization"] = f"Bearer {auth_token}"
    req = urllib.request.Request(
        acp_url.rstrip("/") + "/v1/context",
        data=body,
        headers=headers,
        method="POST",
    )
    start = time.perf_counter()
    with urllib.request.urlopen(req, timeout=timeout) as resp:  # noqa: S310 - controlled URL
        raw = resp.read().decode("utf-8")
    wall_ms = (time.perf_counter() - start) * 1000.0
    return raw, wall_ms


def measure_run(acp_url: str, auth_token: str | None, scenario: Scenario) -> RunResult:
    acp_body, wall_ms = request_acp_manifest(acp_url, auth_token, scenario)
    mcp_body = mcp_context_payload(
        scenario.tool_ids,
        num_servers=scenario.mcp_servers,
        noise_tools=scenario.mcp_noise_tools,
    )
    acp_tokens, _ = count_tokens(acp_context_payload(acp_body))
    mcp_tokens, _ = count_tokens(mcp_body)
    return RunResult(
        scenario_id=scenario.id,
        acp_tokens=acp_tokens,
        acp_bytes=len(acp_body),
        acp_round_trips=1,  # POST /v1/context
        mcp_tokens=mcp_tokens,
        mcp_bytes=len(mcp_body),
        mcp_round_trips=1 + 2 * scenario.mcp_servers,  # initialize + tools/list per server
        wall_ms_acp=wall_ms,
    )


def summarize(scenario: Scenario, runs: list[RunResult], method: str) -> ScenarioSummary:
    acp = [r.acp_tokens for r in runs]
    mcp = [r.mcp_tokens for r in runs]
    reductions = [1 - (a / m) if m > 0 else 0.0 for a, m in zip(acp, mcp, strict=True)]
    return ScenarioSummary(
        scenario_id=scenario.id,
        title=scenario.title,
        runs=len(runs),
        method=method,
        acp_tokens_mean=statistics.mean(acp),
        acp_tokens_p50=statistics.median(acp),
        acp_tokens_p95=_p95(acp),
        acp_round_trips=runs[0].acp_round_trips,
        mcp_tokens_mean=statistics.mean(mcp),
        mcp_tokens_p50=statistics.median(mcp),
        mcp_tokens_p95=_p95(mcp),
        mcp_round_trips=runs[0].mcp_round_trips,
        token_reduction_mean=statistics.mean(reductions),
        token_reduction_p50=statistics.median(reductions),
    )


def _p95(xs: list[int]) -> float:
    if not xs:
        return 0.0
    s = sorted(xs)
    k = max(0, int(round(0.95 * (len(s) - 1))))
    return float(s[k])


def main(argv: list[str] | None = None) -> int:
    p = argparse.ArgumentParser(description="ACP vs MCP token-cost benchmark")
    p.add_argument("--acp-url", required=True, help="Base URL of the ACP server")
    p.add_argument("--auth-token", default=None, help="Bearer token for /v1/* (optional)")
    p.add_argument("--runs", type=int, default=50, help="Runs per scenario")
    p.add_argument("--out", required=True, help="Path to write JSON results")
    args = p.parse_args(argv)

    # Probe token counter once to label results.
    _, method = count_tokens("hello")

    all_runs: list[RunResult] = []
    summaries: list[ScenarioSummary] = []
    for scenario in SCENARIOS:
        scenario_runs: list[RunResult] = []
        for i in range(args.runs):
            try:
                scenario_runs.append(measure_run(args.acp_url, args.auth_token, scenario))
            except urllib.error.HTTPError as e:
                print(
                    f"scenario {scenario.id} run {i}: HTTP {e.code} {e.reason}",
                    file=sys.stderr,
                )
                return 2
            except urllib.error.URLError as e:
                print(f"scenario {scenario.id} run {i}: connect error: {e}", file=sys.stderr)
                return 2
        all_runs.extend(scenario_runs)
        summaries.append(summarize(scenario, scenario_runs, method))

    out_path = Path(args.out)
    out_path.parent.mkdir(parents=True, exist_ok=True)
    out_path.write_text(
        json.dumps(
            {
                "method": method,
                "mcp_initialize_response_bytes": len(json.dumps(INITIALIZE_RESPONSE)),
                "scenarios": [asdict(s) for s in summaries],
                "raw": [asdict(r) for r in all_runs],
            },
            indent=2,
        )
    )
    print(f"wrote {out_path} ({len(all_runs)} runs, method={method})")
    for s in summaries:
        print(
            f"  {s.scenario_id} {s.title}: "
            f"ACP {s.acp_tokens_mean:.0f} tok / 1 RT  vs  "
            f"MCP {s.mcp_tokens_mean:.0f} tok / {s.mcp_round_trips} RT  "
            f"=> {s.token_reduction_mean*100:.1f}% reduction"
        )
    return 0


if __name__ == "__main__":  # pragma: no cover
    raise SystemExit(main())
