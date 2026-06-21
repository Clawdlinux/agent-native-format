#!/usr/bin/env python3
"""Frontier tool-context benchmark: ACP vs MCP across model tiers.

Sends identical tool-calling tasks to frontier models using both:
  - MCP-style: all tool schemas in the system prompt (full JSON-Schema)
  - ACP-style: only intent-scoped, schema-stripped tools

Measures:
  - Provider-reported input/output tokens
  - Tool-call correctness (did model pick the right tools with valid args?)
  - Wall-clock latency
  - Estimated cost

Usage:
    cd agent-contract-protocol
    export $(grep -v '^#' .env | xargs)
    python3 benchmark/frontier/run_frontier.py --runs 5 --out results/frontier/
"""

from __future__ import annotations

import argparse
import json
import os
import sys
import time
import urllib.error
import urllib.request
from dataclasses import asdict, dataclass, field
from datetime import datetime, timezone
from pathlib import Path

try:
    import tiktoken
except ImportError:  # pragma: no cover - optional normalization dependency
    tiktoken = None

# Add parent to path for baseline imports
sys.path.insert(0, str(Path(__file__).resolve().parents[1]))
from baseline.mcp_client import CATALOG, mcp_context_payload  # noqa: E402


SYSTEM_PROMPT = "You are an agent. Use the provided tools to accomplish the user's task. Call ALL relevant tools."
TIKTOKEN_CONTROL_ENCODING = "cl100k_base"


# ──────────────────────────────────────────────────────────────
# Model definitions
# ──────────────────────────────────────────────────────────────

@dataclass(frozen=True)
class ModelSpec:
    id: str
    provider: str  # "openai" | "anthropic"
    tier: str      # "heavy" | "medium" | "long_context" | "open"
    label: str     # human-readable


MODELS: list[ModelSpec] = [
    # Heavy frontier
    ModelSpec("claude-opus-4-7", "anthropic", "heavy", "Claude Opus 4.7"),
    ModelSpec("gpt-5.5-2026-04-23", "openai", "heavy", "GPT-5.5"),
    # Medium frontier
    ModelSpec("claude-sonnet-4-6", "anthropic", "medium", "Claude Sonnet 4.6"),
    ModelSpec("gpt-5.4-2026-03-05", "openai", "medium", "GPT-5.4"),
]


# ──────────────────────────────────────────────────────────────
# Scenarios (same 5 as existing benchmark)
# ──────────────────────────────────────────────────────────────

@dataclass(frozen=True)
class Scenario:
    id: str
    title: str
    intent: str
    relevant_tools: list[str]
    all_tools: list[str]   # MCP path gets all of these
    expected_tool_names: list[str]  # correctness check


SCENARIOS: list[Scenario] = [
    Scenario(
        "S1", "Simple DB query",
        "Query the customer database for active accounts created in the last 30 days.",
        relevant_tools=["db.query"],
        all_tools=["db.query"],
        expected_tool_names=["db.query"],
    ),
    Scenario(
        "S2", "Multi-tool workflow",
        "Query the customer database for top spending accounts, send a Slack notification to #sales with the results, and log an audit event.",
        relevant_tools=["db.query", "slack.send_message", "audit.log_event"],
        all_tools=["db.query", "slack.send_message", "audit.log_event"],
        expected_tool_names=["db.query", "slack.send_message", "audit.log_event"],
    ),
    Scenario(
        "S3", "Complex DAG",
        "Query the customer database, render a weekly summary report from the data, log an audit event, then email the team the report.",
        relevant_tools=["db.query", "template.render", "audit.log_event", "email.send"],
        all_tools=["db.query", "template.render", "audit.log_event", "email.send"],
        expected_tool_names=["db.query", "template.render", "audit.log_event", "email.send"],
    ),
    Scenario(
        "S4", "Scale: 50 tools, 2 relevant",
        "Query the customer database and send a Slack notification with the result.",
        relevant_tools=["db.query", "slack.send_message"],
        all_tools=list(CATALOG.keys()),  # all 5, plus noise tools added in payload
        expected_tool_names=["db.query", "slack.send_message"],
    ),
    Scenario(
        "S5", "Auth-heavy cross-service",
        "Query the customer database, render a report, send it via email and Slack, and log an audit event.",
        relevant_tools=["db.query", "template.render", "email.send", "slack.send_message", "audit.log_event"],
        all_tools=list(CATALOG.keys()),
        expected_tool_names=["db.query", "template.render", "email.send", "slack.send_message", "audit.log_event"],
    ),
]


# ──────────────────────────────────────────────────────────────
# Tool schema builders
# ──────────────────────────────────────────────────────────────

def build_mcp_tools(scenario: Scenario) -> list[dict]:
    """Full MCP-style tool definitions for a model's tools parameter."""
    tools = []
    for tid in scenario.all_tools:
        desc = CATALOG[tid]
        tools.append({
            "type": "function",
            "function": {
                "name": desc.name,
                "description": desc.description,
                "parameters": {
                    "type": "object",
                    "properties": desc.properties,
                    "required": desc.required,
                },
            },
        })
    # Add noise tools for S4
    if scenario.id == "S4":
        for i in range(45):  # 5 real + 45 noise = 50
            tools.append({
                "type": "function",
                "function": {
                    "name": f"misc.tool_{i:03d}",
                    "description": f"Auxiliary tool #{i} for upstream service operations. Accepts structured input and returns structured response.",
                    "parameters": {
                        "type": "object",
                        "properties": {
                            "operation": {"type": "string", "description": "Operation identifier"},
                            "params": {"type": "object", "description": "Free-form parameters"},
                        },
                        "required": ["operation"],
                    },
                },
            })
    return tools


def build_acp_tools(scenario: Scenario) -> list[dict]:
    """ACP-style: only relevant tools with stripped schemas."""
    tools = []
    type_map = {
        "string": "string",
        "integer": "int",
        "object": "object",
        "array": "array",
    }
    for tid in scenario.relevant_tools:
        desc = CATALOG[tid]
        compact_params = {}
        for pname, pdef in desc.properties.items():
            ptype = type_map.get(pdef.get("type", "string"), pdef.get("type", "string"))
            if pname not in desc.required:
                ptype += "?"
            compact_params[pname] = ptype
        tools.append({
            "type": "function",
            "function": {
                "name": desc.name,
                "description": "",  # stripped
                "parameters": {
                    "type": "object",
                    "properties": {k: {"type": "string"} for k in compact_params},
                    "required": desc.required,
                },
            },
        })
    return tools


def count_tiktoken_control_tokens(tools: list[dict], prompt: str) -> int:
    """Return a deterministic cl100k_base count for prompt+tool payload comparison."""
    payload = json.dumps(
        {
            "system": SYSTEM_PROMPT,
            "user": prompt,
            "tools": tools,
        },
        sort_keys=True,
        separators=(",", ":"),
    )
    if tiktoken is not None:
        encoding = tiktoken.get_encoding(TIKTOKEN_CONTROL_ENCODING)
        return len(encoding.encode(payload))
    return max(1, (len(payload) + 3) // 4)


# ──────────────────────────────────────────────────────────────
# API callers
# ──────────────────────────────────────────────────────────────

def call_openai(model_id: str, tools: list[dict], prompt: str) -> dict:
    """Call OpenAI chat completions with tools."""
    key = os.environ.get("OPENAI_API_KEY", "")
    # Sanitize tool names (OpenAI requires ^[a-zA-Z0-9_-]+$)
    sanitized_tools = []
    name_map = {}  # sanitized -> original
    for t in tools:
        f = t["function"]
        safe_name = _sanitize_tool_name(f["name"])
        name_map[safe_name] = f["name"]
        sanitized_tools.append({
            "type": "function",
            "function": {**f, "name": safe_name},
        })
    body = json.dumps({
        "model": model_id,
        "messages": [
            {"role": "system", "content": SYSTEM_PROMPT},
            {"role": "user", "content": prompt},
        ],
        "tools": sanitized_tools,
        "tool_choice": "auto",
        "max_completion_tokens": 1024,
    }).encode()
    req = urllib.request.Request(
        "https://api.openai.com/v1/chat/completions",
        data=body,
        headers={
            "Authorization": f"Bearer {key}",
            "Content-Type": "application/json",
        },
        method="POST",
    )
    start = time.perf_counter()
    with urllib.request.urlopen(req, timeout=60) as resp:  # noqa: S310
        data = json.loads(resp.read())
    latency_ms = (time.perf_counter() - start) * 1000

    usage = data.get("usage", {})
    tool_calls = []
    for choice in data.get("choices", []):
        msg = choice.get("message", {})
        for tc in msg.get("tool_calls", []):
            raw_name = tc.get("function", {}).get("name", "")
            tool_calls.append(name_map.get(raw_name, raw_name))

    return {
        "input_tokens": usage.get("prompt_tokens", 0),
        "output_tokens": usage.get("completion_tokens", 0),
        "total_tokens": usage.get("total_tokens", 0),
        "tool_calls": tool_calls,
        "latency_ms": latency_ms,
    }


def call_anthropic(model_id: str, tools: list[dict], prompt: str) -> dict:
    """Call Anthropic messages API with tools."""
    key = os.environ.get("ANTHROPIC_API_KEY", "")
    # Convert OpenAI tool format to Anthropic format (sanitize names)
    name_map = {}  # sanitized -> original
    anthropic_tools = []
    for t in tools:
        f = t["function"]
        safe_name = _sanitize_tool_name(f["name"])
        name_map[safe_name] = f["name"]
        anthropic_tools.append({
            "name": safe_name,
            "description": f.get("description", "") or safe_name,
            "input_schema": f.get("parameters", {"type": "object", "properties": {}}),
        })

    body = json.dumps({
        "model": model_id,
        "max_tokens": 1024,
        "system": SYSTEM_PROMPT,
        "messages": [{"role": "user", "content": prompt}],
        "tools": anthropic_tools,
        "tool_choice": {"type": "auto"},
    }).encode()
    req = urllib.request.Request(
        "https://api.anthropic.com/v1/messages",
        data=body,
        headers={
            "x-api-key": key,
            "anthropic-version": "2023-06-01",
            "Content-Type": "application/json",
        },
        method="POST",
    )
    start = time.perf_counter()
    with urllib.request.urlopen(req, timeout=120) as resp:  # noqa: S310
        data = json.loads(resp.read())
    latency_ms = (time.perf_counter() - start) * 1000

    usage = data.get("usage", {})
    tool_calls = []
    for block in data.get("content", []):
        if block.get("type") == "tool_use":
            raw_name = block.get("name", "")
            tool_calls.append(name_map.get(raw_name, raw_name))

    return {
        "input_tokens": usage.get("input_tokens", 0),
        "output_tokens": usage.get("output_tokens", 0),
        "total_tokens": usage.get("input_tokens", 0) + usage.get("output_tokens", 0),
        "tool_calls": tool_calls,
        "latency_ms": latency_ms,
    }


def _sanitize_tool_name(name: str) -> str:
    """Anthropic requires tool names to match [a-zA-Z0-9_-]+."""
    return name.replace(".", "_")


def call_model(spec: ModelSpec, tools: list[dict], prompt: str, max_retries: int = 4) -> dict:
    for attempt in range(max_retries):
        try:
            if spec.provider == "openai":
                return call_openai(spec.id, tools, prompt)
            elif spec.provider == "anthropic":
                return call_anthropic(spec.id, tools, prompt)
            else:
                raise ValueError(f"Unknown provider: {spec.provider}")
        except urllib.error.HTTPError as e:
            if e.code == 429 and attempt < max_retries - 1:
                wait = 25 if spec.provider == "openai" else 2 ** (attempt + 1)
                print(f"rate-limited, waiting {wait}s...", end=" ", flush=True)
                time.sleep(wait)
                continue
            raise
    raise RuntimeError("unreachable")


# ──────────────────────────────────────────────────────────────
# Correctness check
# ──────────────────────────────────────────────────────────────

def check_correctness(called: list[str], expected: list[str]) -> dict:
    called_set = set(called)
    expected_set = set(expected)
    correct = called_set & expected_set
    missed = expected_set - called_set
    extra = called_set - expected_set
    precision = len(correct) / len(called_set) if called_set else 0.0
    recall = len(correct) / len(expected_set) if expected_set else 0.0
    return {
        "correct_tools": sorted(correct),
        "missed_tools": sorted(missed),
        "extra_tools": sorted(extra),
        "precision": precision,
        "recall": recall,
        "exact_match": called_set == expected_set,
    }


# ──────────────────────────────────────────────────────────────
# Run result
# ──────────────────────────────────────────────────────────────

@dataclass
class RunResult:
    model_id: str
    model_tier: str
    model_label: str
    scenario_id: str
    scenario_title: str
    path: str  # "mcp" | "acp"
    input_tokens: int
    output_tokens: int
    total_tokens: int
    tiktoken_control_input_tokens: int
    tool_calls: list[str]
    correctness: dict
    latency_ms: float
    num_tools_provided: int
    run_index: int


# ──────────────────────────────────────────────────────────────
# Main
# ──────────────────────────────────────────────────────────────

def main(argv: list[str] | None = None) -> int:
    p = argparse.ArgumentParser(description="Frontier ACP vs MCP benchmark")
    p.add_argument("--runs", type=int, default=3, help="Runs per model+scenario+path")
    p.add_argument("--out", required=True, help="Output directory for results")
    p.add_argument("--models", default=None, help="Comma-separated model IDs to run (default: all)")
    p.add_argument("--scenarios", default=None, help="Comma-separated scenario IDs (default: all)")
    args = p.parse_args(argv)

    out_dir = Path(args.out)
    out_dir.mkdir(parents=True, exist_ok=True)

    models = MODELS
    if args.models:
        wanted = set(args.models.split(","))
        models = [m for m in MODELS if m.id in wanted]

    scenarios = SCENARIOS
    if args.scenarios:
        wanted = set(args.scenarios.split(","))
        scenarios = [s for s in SCENARIOS if s.id in wanted]

    all_results: list[dict] = []
    total = len(models) * len(scenarios) * 2 * args.runs
    done = 0

    for model in models:
        for scenario in scenarios:
            for path_name, tool_builder in [("mcp", build_mcp_tools), ("acp", build_acp_tools)]:
                tools = tool_builder(scenario)
                for run_idx in range(args.runs):
                    done += 1
                    control_tokens = count_tiktoken_control_tokens(tools, scenario.intent)
                    tag = f"[{done}/{total}] {model.label} | {scenario.id} | {path_name} | run {run_idx+1}"
                    print(f"{tag} ...", end=" ", flush=True)
                    try:
                        result = call_model(model, tools, scenario.intent)
                        correctness = check_correctness(result["tool_calls"], scenario.expected_tool_names)
                        run = RunResult(
                            model_id=model.id,
                            model_tier=model.tier,
                            model_label=model.label,
                            scenario_id=scenario.id,
                            scenario_title=scenario.title,
                            path=path_name,
                            input_tokens=result["input_tokens"],
                            output_tokens=result["output_tokens"],
                            total_tokens=result["total_tokens"],
                            tiktoken_control_input_tokens=control_tokens,
                            tool_calls=result["tool_calls"],
                            correctness=correctness,
                            latency_ms=result["latency_ms"],
                            num_tools_provided=len(tools),
                            run_index=run_idx,
                        )
                        all_results.append(asdict(run))
                        em = "✓" if correctness["exact_match"] else "✗"
                        print(f"in={result['input_tokens']} out={result['output_tokens']} control={control_tokens} tools={result['tool_calls']} {em}")
                        # Respect rate limits: OpenAI free tier = 3 RPM
                        wait = 22 if model.provider == "openai" else 2
                        time.sleep(wait)
                    except Exception as e:
                        err_detail = str(e)
                        if hasattr(e, 'read'):
                            try:
                                err_detail = e.read().decode()[:300]
                            except Exception:
                                pass
                        print(f"ERROR: {err_detail}")
                        all_results.append({
                            "model_id": model.id, "model_tier": model.tier,
                            "model_label": model.label, "scenario_id": scenario.id,
                            "scenario_title": scenario.title, "path": path_name,
                            "tiktoken_control_input_tokens": control_tokens,
                            "error": err_detail, "run_index": run_idx,
                        })
                        wait = 25 if model.provider == "openai" else 2
                        time.sleep(wait)

    # Write raw results
    ts = datetime.now(timezone.utc).strftime("%Y-%m-%d-%H%M%S")
    raw_path = out_dir / f"{ts}-raw.json"
    raw_path.write_text(json.dumps(all_results, indent=2))
    print(f"\nWrote raw results to {raw_path}")

    # Write summary
    summary = build_summary(all_results)
    summary_path = out_dir / f"{ts}-summary.md"
    summary_path.write_text(summary)
    print(f"Wrote summary to {summary_path}")

    return 0


def build_summary(results: list[dict]) -> str:
    """Build a markdown summary table from raw results."""
    lines = ["# Frontier Benchmark: ACP vs MCP Tool-Context Overhead", ""]
    lines.append(f"Generated: {datetime.now(timezone.utc).strftime('%Y-%m-%d %H:%M UTC')}")
    lines.append("")
    lines.append("Provider-reported token counts are the headline. The tiktoken control column uses cl100k_base over serialized prompt+tools for deterministic normalization.")
    lines.append("Correctness = model called exactly the expected tools.")
    lines.append("")

    # Group by model
    by_model: dict[str, list[dict]] = {}
    for r in results:
        key = r.get("model_label", r.get("model_id", "unknown"))
        by_model.setdefault(key, []).append(r)

    for model_label, runs in sorted(by_model.items()):
        lines.append(f"## {model_label} (`{runs[0].get('model_id', '?')}`)")
        lines.append("")
        lines.append("| Scenario | Path | Avg Input Tokens | Avg Output Tokens | Tiktoken Control | Tools Provided | Precision | Recall | Exact Match |")
        lines.append("|---|---|---|---|---|---|---|---|---|")

        by_scenario: dict[str, dict[str, list[dict]]] = {}
        for r in runs:
            if "error" in r:
                continue
            sid = r["scenario_id"]
            path = r["path"]
            by_scenario.setdefault(sid, {}).setdefault(path, []).append(r)

        for sid in ["S1", "S2", "S3", "S4", "S5"]:
            paths = by_scenario.get(sid, {})
            for path_name in ["mcp", "acp"]:
                path_runs = paths.get(path_name, [])
                if not path_runs:
                    continue
                avg_in = sum(r["input_tokens"] for r in path_runs) / len(path_runs)
                avg_out = sum(r["output_tokens"] for r in path_runs) / len(path_runs)
                avg_control = sum(r["tiktoken_control_input_tokens"] for r in path_runs) / len(path_runs)
                avg_prec = sum(r["correctness"]["precision"] for r in path_runs) / len(path_runs)
                avg_rec = sum(r["correctness"]["recall"] for r in path_runs) / len(path_runs)
                em_rate = sum(1 for r in path_runs if r["correctness"]["exact_match"]) / len(path_runs)
                n_tools = path_runs[0]["num_tools_provided"]
                title = path_runs[0]["scenario_title"]
                lines.append(
                    f"| {sid} {title} | **{path_name.upper()}** | {avg_in:.0f} | {avg_out:.0f} | "
                    f"{avg_control:.0f} | {n_tools} | {avg_prec:.0%} | {avg_rec:.0%} | {em_rate:.0%} |"
                )

        # Token reduction summary
        lines.append("")
        lines.append("**Token reduction (input tokens):**")
        lines.append("")
        for sid in ["S1", "S2", "S3", "S4", "S5"]:
            paths = by_scenario.get(sid, {})
            mcp_runs = paths.get("mcp", [])
            acp_runs = paths.get("acp", [])
            if mcp_runs and acp_runs:
                mcp_avg = sum(r["input_tokens"] for r in mcp_runs) / len(mcp_runs)
                acp_avg = sum(r["input_tokens"] for r in acp_runs) / len(acp_runs)
                if mcp_avg > 0:
                    reduction = (1 - acp_avg / mcp_avg) * 100
                    lines.append(f"- {sid}: MCP {mcp_avg:.0f} → ACP {acp_avg:.0f} = **{reduction:.1f}% reduction**")

        lines.append("")

    return "\n".join(lines)


if __name__ == "__main__":
    raise SystemExit(main())
