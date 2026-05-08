"""Graders for the agent-accuracy benchmark.

Each grader is a pure function: ``(model_response: str, expected) -> bool``.

The graders are deliberately strict-but-fair. They normalise whitespace
and case, accept the canonical answer plus a small number of obvious
synonyms ("none" / "no pods" / "[]"), and reject anything ambiguous.
A response that doesn't match exactly is scored 0; the run logs the
raw response so you can hand-audit edge cases.
"""

from __future__ import annotations

import re
from typing import Callable

# A grader takes the raw text the model returned plus the expected
# value (which may be int / str / list[str]) and returns True when the
# answer is correct.
Grader = Callable[[str, object], bool]


# ─── helpers ────────────────────────────────────────────────────────────────


def _norm(s: str) -> str:
    """Lowercase, strip, collapse whitespace."""
    return re.sub(r"\s+", " ", s.strip().lower())


def _first_token(s: str) -> str:
    """Return the first non-empty word (after stripping punctuation)."""
    for tok in re.split(r"[\s,.;:!?\"\'`]+", s.strip()):
        if tok:
            return tok.lower()
    return ""


def _name_set(s: str) -> set[str]:
    """Parse a multi-line or comma-separated list of pod names into a
    normalised set. The literal token ``none`` (any case) yields an
    empty set. Pod names are lowercase and stripped of common bullet
    prefixes (``- name`` and ``* name``)."""
    s = s.strip()
    if not s or _norm(s) == "none":
        return set()
    out: set[str] = set()
    for line in re.split(r"[\n,]+", s):
        token = re.sub(r"^[\s\-*]+", "", line).strip()
        # Strip trailing punctuation a model may add.
        token = re.sub(r"[\s.,;:!?]+$", "", token)
        if not token:
            continue
        if token.lower() == "none":
            continue
        out.add(token.lower())
    return out


# ─── graders ────────────────────────────────────────────────────────────────


def integer(response: str, expected: object) -> bool:
    if not isinstance(expected, int):
        raise TypeError(f"integer grader expects int, got {type(expected).__name__}")
    nums = re.findall(r"-?\d+", response)
    if not nums:
        return False
    # Take the first integer mentioned. A strict-er variant could
    # require the *only* integer; in practice the prompt asks for a
    # single number and models comply.
    return int(nums[0]) == expected


def yes_no(response: str, expected: object) -> bool:
    if expected not in ("yes", "no"):
        raise ValueError(f"yes_no expects 'yes' or 'no', got {expected!r}")
    tok = _first_token(response)
    return tok == expected


def exact(response: str, expected: object) -> bool:
    if not isinstance(expected, str):
        raise TypeError("exact grader expects a string expected value")
    return _norm(response) == _norm(expected)


def exact_or_none(response: str, expected: object) -> bool:
    """Matches `exact`, but also accepts the bare answer 'none' when
    the expected value is the literal string 'none'."""
    if not isinstance(expected, str):
        raise TypeError("exact_or_none expects a string expected value")
    norm = _norm(response)
    if expected == "none":
        return norm == "none"
    # Accept a leading bullet/dash, but the substantive token must match.
    cleaned = re.sub(r"^[\s\-*]+", "", response.strip()).split("\n")[0].strip()
    return _norm(cleaned) == _norm(expected)


def name_set(response: str, expected: object) -> bool:
    if expected == "none":
        expected_set: set[str] = set()
    elif isinstance(expected, list):
        expected_set = {x.lower() for x in expected}
    else:
        raise TypeError(f"name_set expects list or 'none', got {expected!r}")
    return _name_set(response) == expected_set


_VERB_ALIASES = {
    "restart": "restart",
    "scale": "scale",
    "rollout": "rollout",
    "logs": "logs",
    "log": "logs",
    "none": "none",
    "no-op": "none",
    "noop": "none",
    "wait": "none",
}


def verb(response: str, expected: object) -> bool:
    if not isinstance(expected, str):
        raise TypeError("verb grader expects a string expected value")
    tok = _first_token(response)
    return _VERB_ALIASES.get(tok, tok) == expected


# Lookup table consumed by harness.py. Keep keys aligned with the
# `grader:` field in questions.yaml.
GRADERS: dict[str, Grader] = {
    "integer": integer,
    "yes_no": yes_no,
    "exact": exact,
    "exact_or_none": exact_or_none,
    "name_set": name_set,
    "verb": verb,
}


def grade(grader_name: str, response: str, expected: object) -> bool:
    grader = GRADERS.get(grader_name)
    if grader is None:
        raise KeyError(f"unknown grader: {grader_name!r}")
    return grader(response, expected)
