#!/usr/bin/env python3
from pathlib import Path
import sys

required = [
    "README.md",
    "SPEC.md",
    "LICENSE",
    "docs/architecture.md",
    "docs/benchmark-methodology.md",
    "docs/pitch-deck-data.md",
    "docs/phase-log.md",
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

print("docs ok")
