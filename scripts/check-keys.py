#!/usr/bin/env python3
"""Verify .env keys are loadable without echoing them.

Prints OK/MISSING/MALFORMED for each expected key. Never prints the key
itself. Safe to run anywhere (CI, locally).
"""

from __future__ import annotations

import sys
from pathlib import Path


def parse_env(path: Path) -> dict[str, str]:
    out: dict[str, str] = {}
    if not path.exists():
        return out
    for line in path.read_text().splitlines():
        s = line.strip()
        if not s or s.startswith("#") or "=" not in s:
            continue
        k, _, v = s.partition("=")
        out[k.strip()] = v.strip().strip('"').strip("'")
    return out


def status(keys: dict[str, str], name: str, prefix: str) -> tuple[str, bool]:
    v = keys.get(name, "")
    if not v:
        return f"{name:20s} MISSING", False
    if not v.startswith(prefix):
        return f"{name:20s} MALFORMED (does not start with {prefix!r})", False
    return f"{name:20s} OK ({len(v)} chars)", True


def main() -> int:
    env_path = Path(__file__).resolve().parent.parent / ".env"
    keys = parse_env(env_path)
    print(f"Reading: {env_path}")
    rows = [
        status(keys, "OPENAI_API_KEY", "sk-"),
        status(keys, "ANTHROPIC_API_KEY", "sk-ant-"),
    ]
    for line, _ in rows:
        print(f"  {line}")
    return 0 if all(ok for _, ok in rows) else 1


if __name__ == "__main__":
    sys.exit(main())
