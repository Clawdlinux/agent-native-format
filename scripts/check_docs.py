#!/usr/bin/env python3
from pathlib import Path
import re
import sys

required = [
    "README.md",
    "SPEC.md",
    "LICENSE",
    "docs/architecture.md",
    "docs/benchmark-methodology.md",
    "docs/pitch-deck-data.md",
    "docs/phase-log.md",
    "docs/positioning.md",
    "docs/protocol.md",
    "docs/references.md",
]

missing = [path for path in required if not Path(path).is_file()]
if missing:
    for path in missing:
        print(f"missing required doc: {path}", file=sys.stderr)
    sys.exit(1)

spec = Path("SPEC.md").read_text(encoding="utf-8")
checks = ["POST /v1/context", "Execution Manifest", "feedback_endpoint"]
for needle in checks:
    if needle not in spec:
        print(f"SPEC.md missing required phrase: {needle}", file=sys.stderr)
        sys.exit(1)

# Guard against the old positioning silently coming back. The phrase is
# explicitly allowed in the changelog and in docs/positioning.md (where we
# explain the deprecation), but nowhere else.
banned = re.compile(r"successor to MCP|protocol that replaces MCP", re.IGNORECASE)
allowed = {"SPEC.md": ["Repositioned from"], "docs/positioning.md": ["earlier draft of ACP framed itself"]}
for md in Path(".").glob("**/*.md"):
    rel = str(md.as_posix())
    # Skip vendored / generated trees.
    if any(part in {".venv", "node_modules", "results", "bin"} for part in md.parts):
        continue
    text = md.read_text(encoding="utf-8", errors="replace")
    for m in banned.finditer(text):
        line_no = text.count("\n", 0, m.start()) + 1
        line = text.splitlines()[line_no - 1]
        marker = allowed.get(rel, [])
        if any(s in line for s in marker):
            continue
        print(f"{rel}:{line_no}: stale 'successor to MCP' framing: {line.strip()}", file=sys.stderr)
        sys.exit(1)

print("docs ok")
