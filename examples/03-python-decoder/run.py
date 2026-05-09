#!/usr/bin/env python3
# Example 03 — Python agent reading ACL
#
# Shows how a Python agent loads an ACL document, inspects structured
# fields, makes a decision, and reports the token savings vs the
# original kubectl JSON.

from __future__ import annotations

import sys
from pathlib import Path

try:
    import acp_acl
except ImportError:
    sys.exit(
        "acp_acl not installed. Run: pip install -e python/[tokens]\n"
        "(from the ninevigil-acp repo root)"
    )

ROOT = Path(__file__).resolve().parent.parent.parent
ACL_FILE = ROOT / "benchmark/agent_accuracy/fixtures/healthy/state.acl"
RAW_FILE = ROOT / "benchmark/agent_accuracy/fixtures/healthy/raw.json"


def main() -> int:
    doc = acp_acl.decode(ACL_FILE.read_text())

    print("Loaded ACL doc:")
    print(f"  source:    {doc.directives.get('source')}")
    print(f"  cluster:   {doc.directives.get('cluster')}")
    print(f"  namespace: {doc.directives.get('ns')}")
    print()

    print("Sections:")
    for section in doc:
        print(
            f"  {section.name + ':':10} "
            f"summary={section.summary!r:20} "
            f"({len(section.rows)} rows)"
        )
    print()

    # Make a real decision based on the parsed structure.
    pods = doc.section("pods")
    deploys = doc.section("deploys")
    print("Agent decision:")
    if pods and "ok" in pods.summary and deploys and "all-avail" in deploys.summary:
        print(f"  All pods healthy ({pods.summary}), all deploys at desired.")
        print("  No action required.")
    elif pods and any("!" in row.flags or "!!" in row.flags for row in pods.rows):
        critical = [r for r in pods.rows if "!!" in r.flags]
        warning = [r for r in pods.rows if "!" in r.flags]
        if critical:
            print(f"  CRITICAL: {len(critical)} failed pod(s): "
                  f"{', '.join(r.id for r in critical)}")
            print("  Recommended action: restart")
        elif warning:
            print(f"  WARNING: {len(warning)} pod(s) need attention.")
            print("  Recommended action: investigate logs")
    else:
        print("  Unexpected state, gather more context.")
    print()

    # Token savings.
    raw_tokens = acp_acl.count_tokens(RAW_FILE.read_text())
    acl_tokens = acp_acl.count_tokens(ACL_FILE.read_text())
    ratio = raw_tokens / acl_tokens
    print("Token cost comparison:")
    print(f"  raw kubectl JSON: {raw_tokens:>5} tokens")
    print(f"  ACL document:     {acl_tokens:>5} tokens")
    print(f"  Reduction:        {ratio:>5.1f}x")
    return 0


if __name__ == "__main__":
    sys.exit(main())
