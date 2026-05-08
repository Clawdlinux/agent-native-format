"""Cross-translator agent-accuracy benchmark.

Runs each (scenario, question, condition, model) combination `n_trials`
times against real LLM APIs and reports the accuracy lift of ACL over
raw kubectl JSON. Results are written to
``results/YYYY-MM-DD-<run-id>/{raw.csv,summary.csv,summary.md,meta.json}``.

Usage:

    python -m benchmark.agent_accuracy.harness                 # n=30 default
    python -m benchmark.agent_accuracy.harness --trials 3      # smoke test
    python -m benchmark.agent_accuracy.harness --no-cache      # force fresh

Reads OPENAI_API_KEY and ANTHROPIC_API_KEY from the environment (loaded
from .env at repo root via python-dotenv). Missing keys cause that
provider to be skipped cleanly with a printed notice.

The harness is **opinionated about safety**:

- Pre-flight cost estimate uses real tokenizers and refuses to start
  if it exceeds ``--max-usd`` (default 2.0).
- API responses are cached on disk by ``sha256(model, system, user)``
  so a re-run after a crash resumes from the cache for free.
- Keys never appear in any output file (raw.csv, meta.json, summary).
"""

from __future__ import annotations

import argparse
import csv
import datetime as dt
import hashlib
import json
import os
import subprocess
import sys
import time
from dataclasses import asdict, dataclass
from pathlib import Path

import yaml
from dotenv import load_dotenv

# Allow `python -m benchmark.agent_accuracy.harness` *and*
# `python benchmark/agent_accuracy/harness.py`.
PKG_ROOT = Path(__file__).resolve().parent
REPO_ROOT = PKG_ROOT.parent.parent
sys.path.insert(0, str(REPO_ROOT))

from benchmark.agent_accuracy import clients, graders, prompts, token_counter  # noqa: E402
from benchmark.agent_accuracy.aggregate import (  # noqa: E402
    aggregate,
    write_summary_csv,
    write_summary_md,
)

# Keys live in repo-root .env by convention.
load_dotenv(REPO_ROOT / ".env")

DEFAULT_MODELS = ["gpt-4o-mini", "claude-haiku-4-5-20251001"]
DEFAULT_SCENARIOS = ["healthy", "degraded", "failing"]
DEFAULT_TRIALS = 30
DEFAULT_MAX_TOKENS = 200  # generous; most answers are 1–10 tokens
DEFAULT_TEMPERATURE = 0.0  # determinism over creativity for graders
CACHE_DIR = PKG_ROOT / ".cache" / "responses"


# ─── data ────────────────────────────────────────────────────────────────────


@dataclass(frozen=True)
class Question:
    id: str
    kind: str
    grader: str
    prompt: str


@dataclass(frozen=True)
class Scenario:
    name: str
    raw_payload: str
    acl_payload: str
    expected: dict


@dataclass
class TrialRow:
    model: str
    scenario: str
    question_id: str
    question_kind: str
    condition: str
    trial: int
    correct: int  # 0 or 1
    response: str
    prompt_tokens: int
    completion_tokens: int
    latency_ms: float
    usd: float
    cache_hit: int  # 0 or 1


# ─── loading ─────────────────────────────────────────────────────────────────


def load_questions() -> list[Question]:
    raw = yaml.safe_load((PKG_ROOT / "questions.yaml").read_text())
    return [Question(**q) for q in raw["questions"]]


def load_scenarios(names: list[str]) -> list[Scenario]:
    out: list[Scenario] = []
    for name in names:
        d = PKG_ROOT / "fixtures" / name
        if not d.is_dir():
            raise FileNotFoundError(
                f"missing fixture {d}. Run `make agent-bench-fixtures` to regenerate."
            )
        raw = (d / "raw.json").read_text()
        acl = (d / "state.acl").read_text()
        expected = yaml.safe_load((d / "expected.yaml").read_text())
        # YAML parses bare yes/no as booleans; the yes_no grader
        # explicitly rejects bools so the rubric stays unambiguous.
        # Coerce them back to strings here.
        for k, v in list(expected["answers"].items()):
            if isinstance(v, bool):
                expected["answers"][k] = "yes" if v else "no"
        out.append(
            Scenario(name=name, raw_payload=raw, acl_payload=acl, expected=expected["answers"])
        )
    return out


# ─── caching ────────────────────────────────────────────────────────────────


def _cache_key(model: str, system: str, user: str) -> str:
    h = hashlib.sha256()
    h.update(model.encode())
    h.update(b"\x00")
    h.update(system.encode())
    h.update(b"\x00")
    h.update(user.encode())
    return h.hexdigest()


def _cache_get(key: str) -> dict | None:
    p = CACHE_DIR / f"{key}.json"
    if not p.exists():
        return None
    try:
        return json.loads(p.read_text())
    except (json.JSONDecodeError, OSError):
        return None


def _cache_put(key: str, payload: dict) -> None:
    CACHE_DIR.mkdir(parents=True, exist_ok=True)
    (CACHE_DIR / f"{key}.json").write_text(json.dumps(payload))


# ─── cost estimate ──────────────────────────────────────────────────────────


def estimate_total_usd(
    models: list[str],
    scenarios: list[Scenario],
    questions: list[Question],
    trials: int,
    max_completion_tokens: int,
) -> float:
    """Estimate the worst-case total USD before any call goes out.

    Uses the real tokenizer for prompt tokens and assumes every call
    hits the `max_completion_tokens` ceiling for completion (a deliberate
    over-estimate so the cost cap is conservative).
    """
    total = 0.0
    for model in models:
        for s in scenarios:
            for q in questions:
                for cond, payload in (("raw", s.raw_payload), ("acl", s.acl_payload)):
                    p = prompts.build(cond, payload, q.prompt)
                    full = p.system + "\n" + p.user
                    in_tokens = token_counter.count(model, full)
                    per_call = token_counter.estimate_usd(model, in_tokens, max_completion_tokens)
                    total += per_call * trials
    return total


# ─── main loop ──────────────────────────────────────────────────────────────


def run_trial(
    client,
    model: str,
    scenario: Scenario,
    question: Question,
    condition: str,
    trial: int,
    max_completion_tokens: int,
    temperature: float,
    use_cache: bool,
) -> TrialRow:
    payload = scenario.acl_payload if condition == "acl" else scenario.raw_payload
    prompt = prompts.build(condition, payload, question.prompt)
    msgs = prompt.messages()
    cache_key = _cache_key(model, prompt.system, prompt.user) + f".t{trial}"
    cache_hit = 0
    cached = _cache_get(cache_key) if use_cache else None
    if cached is not None:
        result = clients.CallResult(
            response_text=cached["response"],
            prompt_tokens=cached["prompt_tokens"],
            completion_tokens=cached["completion_tokens"],
            latency_ms=cached["latency_ms"],
        )
        cache_hit = 1
    else:
        result = client.call(msgs, max_tokens=max_completion_tokens, temperature=temperature)
        _cache_put(
            cache_key,
            {
                "response": result.response_text,
                "prompt_tokens": result.prompt_tokens,
                "completion_tokens": result.completion_tokens,
                "latency_ms": result.latency_ms,
            },
        )

    expected_answer = scenario.expected.get(question.id)
    if expected_answer is None:
        raise KeyError(
            f"scenario {scenario.name!r} is missing expected answer for {question.id!r}"
        )
    correct = graders.grade(question.grader, result.response_text, expected_answer)
    usd = token_counter.estimate_usd(model, result.prompt_tokens, result.completion_tokens)

    return TrialRow(
        model=model,
        scenario=scenario.name,
        question_id=question.id,
        question_kind=question.kind,
        condition=condition,
        trial=trial,
        correct=int(correct),
        response=result.response_text,
        prompt_tokens=result.prompt_tokens,
        completion_tokens=result.completion_tokens,
        latency_ms=result.latency_ms,
        usd=usd,
        cache_hit=cache_hit,
    )


def write_raw_csv(rows: list[TrialRow], out: Path) -> None:
    fields = list(asdict(rows[0]).keys()) if rows else []
    with out.open("w", newline="") as f:
        w = csv.DictWriter(f, fieldnames=fields)
        w.writeheader()
        for r in rows:
            d = asdict(r)
            # Truncate the response field to keep raw.csv readable; the
            # full text is in the cache files for per-row debugging.
            d["response"] = d["response"][:200].replace("\n", " ").replace("\r", " ")
            w.writerow(d)


def git_sha() -> str:
    try:
        return subprocess.check_output(
            ["git", "rev-parse", "--short", "HEAD"],
            cwd=REPO_ROOT,
            stderr=subprocess.DEVNULL,
        ).decode().strip()
    except (subprocess.CalledProcessError, FileNotFoundError):
        return "unknown"


def main(argv: list[str] | None = None) -> int:
    p = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    p.add_argument("--scenarios", default=",".join(DEFAULT_SCENARIOS),
                   help=f"comma-separated; default: {','.join(DEFAULT_SCENARIOS)}")
    p.add_argument("--models", default=",".join(DEFAULT_MODELS),
                   help=f"comma-separated; default: {','.join(DEFAULT_MODELS)}")
    p.add_argument("--trials", type=int, default=DEFAULT_TRIALS)
    p.add_argument("--max-tokens", type=int, default=DEFAULT_MAX_TOKENS,
                   dest="max_tokens")
    p.add_argument("--temperature", type=float, default=DEFAULT_TEMPERATURE)
    p.add_argument("--max-usd", type=float, default=2.0,
                   dest="max_usd",
                   help="refuse to start if estimated cost exceeds this")
    p.add_argument("--out", type=Path, default=None,
                   help="output dir; default: results/<date>-<run-id>")
    p.add_argument("--no-cache", action="store_true",
                   help="bypass response cache and force fresh API calls")
    p.add_argument("--dry-run", action="store_true",
                   help="estimate cost and exit without making any API calls")
    args = p.parse_args(argv)

    scenarios = load_scenarios(args.scenarios.split(","))
    models = args.models.split(",")
    questions = load_questions()
    use_cache = not args.no_cache

    print(f"Scenarios: {[s.name for s in scenarios]}")
    print(f"Models:    {models}")
    print(f"Questions: {len(questions)} ({sum(1 for q in questions if q.kind=='fact')} fact, "
          f"{sum(1 for q in questions if q.kind=='decision')} decision)")
    print(f"Trials:    n={args.trials} per cell")
    total_calls = len(scenarios) * len(questions) * 2 * len(models) * args.trials
    print(f"Calls:     {total_calls} total before cache")
    print()
    print("Estimating cost using real tokenizers (this calls Anthropic count_tokens once per unique prompt)...")
    est = estimate_total_usd(models, scenarios, questions, args.trials, args.max_tokens)
    print(f"Estimated worst-case cost: ${est:.4f}  (cap: ${args.max_usd:.2f})")
    if est > args.max_usd:
        print(f"REFUSING: estimated cost ${est:.4f} > --max-usd ${args.max_usd:.2f}", file=sys.stderr)
        return 2
    if args.dry_run:
        print("--dry-run: exiting without making API calls.")
        return 0

    # Build clients lazily; skip providers whose key is missing.
    active_models: list[tuple[str, object]] = []
    for m in models:
        try:
            active_models.append((m, clients.make_client(m)))
        except Exception as e:  # noqa: BLE001
            if clients.is_missing_key(e):
                print(f"  skip {m}: API key not set ({e})")
            else:
                raise
    if not active_models:
        print("ERROR: no usable models. Set OPENAI_API_KEY and/or ANTHROPIC_API_KEY in .env.",
              file=sys.stderr)
        return 1

    # Output dir.
    run_id = dt.datetime.now().strftime("%Y-%m-%d-%H%M%S")
    out_dir = args.out or PKG_ROOT / "results" / run_id
    out_dir.mkdir(parents=True, exist_ok=True)
    print(f"Output: {out_dir}")
    print()

    # Run.
    rows: list[TrialRow] = []
    err_counts: dict[str, int] = {}
    t_start = time.perf_counter()
    for scenario in scenarios:
        for question in questions:
            for condition in ("raw", "acl"):
                for model, client in active_models:
                    cell_errs = 0
                    for trial in range(args.trials):
                        try:
                            row = run_trial(
                                client=client,
                                model=model,
                                scenario=scenario,
                                question=question,
                                condition=condition,
                                trial=trial,
                                max_completion_tokens=args.max_tokens,
                                temperature=args.temperature,
                                use_cache=use_cache,
                            )
                        except Exception as e:  # noqa: BLE001
                            cell_errs += 1
                            etype = type(e).__name__
                            err_counts[etype] = err_counts.get(etype, 0) + 1
                            print(f"  ERROR {model} {scenario.name}/{question.id}/{condition}/t{trial}: {etype}: {e}",
                                  file=sys.stderr)
                            continue
                        rows.append(row)
                    cache_share = sum(
                        r.cache_hit for r in rows
                        if r.scenario == scenario.name and r.question_id == question.id
                        and r.condition == condition and r.model == model
                    )
                    n_cell = sum(
                        1 for r in rows
                        if r.scenario == scenario.name and r.question_id == question.id
                        and r.condition == condition and r.model == model
                    )
                    correct_cell = sum(
                        r.correct for r in rows
                        if r.scenario == scenario.name and r.question_id == question.id
                        and r.condition == condition and r.model == model
                    )
                    print(
                        f"  {scenario.name:9s} {question.id:24s} {condition:3s} "
                        f"{model:30s} {correct_cell}/{n_cell} correct  "
                        f"({cache_share} cached, {cell_errs} err)"
                    )
    wall = time.perf_counter() - t_start

    if err_counts:
        print()
        print("Errors by type:")
        for k, v in sorted(err_counts.items(), key=lambda kv: -kv[1]):
            print(f"  {k:30s} {v}")

    if not rows:
        print("ERROR: no rows recorded.", file=sys.stderr)
        return 1

    raw_csv = out_dir / "raw.csv"
    write_raw_csv(rows, raw_csv)

    summary = aggregate(raw_csv)
    write_summary_csv(summary, out_dir / "summary.csv")

    total_usd = sum(r.usd for r in rows if r.cache_hit == 0)
    meta = {
        "date": run_id,
        "git_sha": git_sha(),
        "trials": args.trials,
        "scenarios": [s.name for s in scenarios],
        "models": [m for m, _ in active_models],
        "n_calls": len(rows),
        "n_cache_hits": sum(r.cache_hit for r in rows),
        "wall_seconds": round(wall, 1),
        "total_usd": round(total_usd, 4),
        "max_completion_tokens": args.max_tokens,
        "temperature": args.temperature,
    }
    (out_dir / "meta.json").write_text(json.dumps(meta, indent=2))
    write_summary_md(summary, out_dir / "summary.md", meta)

    print()
    print(f"Done in {wall:.1f}s. Wrote {len(rows)} rows.")
    print(f"Total spent (excluding cache hits): ${total_usd:.4f}")
    print(f"See: {out_dir}/summary.md")
    return 0


if __name__ == "__main__":
    sys.exit(main())
