"""Shared pytest helpers and fixtures for adapter tests."""

from __future__ import annotations

import json
import sys
from io import BytesIO
from pathlib import Path
from typing import Any
from unittest.mock import MagicMock

# Make `acp_common` and the adapter packages importable.
ROOT = Path(__file__).resolve().parents[2]
ADAPTERS = ROOT / "adapters" / "python"
for p in (ADAPTERS, ADAPTERS / "acp_common", ADAPTERS / "acp_openai", ADAPTERS / "acp_langgraph", ADAPTERS / "acp_crewai"):
    sys.path.insert(0, str(p))


def fake_response(status: int = 200, body: dict[str, Any] | str | None = None) -> MagicMock:
    """Build a mock that mimics ``urllib.request.urlopen``'s context-manager response."""
    payload = body if isinstance(body, str) else json.dumps(body or {})
    raw = payload.encode("utf-8")

    cm = MagicMock()
    cm.__enter__.return_value = cm
    cm.__exit__.return_value = False
    cm.getcode.return_value = status
    cm.read.return_value = raw
    cm.fileobj = BytesIO(raw)
    return cm


def fake_opener(responses):
    """Return an opener that yields the next response per call."""
    iterator = iter(responses)

    def _opener(req, timeout):  # noqa: ARG001
        return next(iterator)

    return _opener


SAMPLE_MANIFEST = {
    "manifest_id": "m-test",
    "version": "acp/v1",
    "ttl": "300s",
    "actions": [
        {
            "id": "a1",
            "type": "http",
            "endpoint": "grpc://db-proxy.svc:50051/query",
            "method": "POST",
            "schema": {"sql": "string", "limit": "int?"},
            "auth": "pre-injected",
            "timeout": "30s",
        },
        {
            "id": "a2",
            "type": "http",
            "endpoint": "https://email-gw.svc:443/send",
            "method": "POST",
            "schema": {
                "to": "string[]",
                "subject": "string",
                "body": "string",
                "attachment_ref": "string?",
            },
            "auth": "pre-injected",
            "depends_on": ["a1"],
        },
    ],
    "boundaries": {
        "egress": ["db-proxy.svc", "email-gw.svc"],
        "max_tokens_per_action": 15000,
        "require_approval": ["a2"],
        "audit_level": "full",
    },
    "feedback_endpoint": "/v1/feedback",
}
