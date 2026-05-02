"""Tests for benchmark.baseline.mcp_client (payload builder)."""

from __future__ import annotations

import json
import sys
from pathlib import Path

import pytest

ROOT = Path(__file__).resolve().parents[2]
sys.path.insert(0, str(ROOT / "benchmark"))

from baseline.mcp_client import (  # noqa: E402
    CATALOG,
    INITIALIZE_RESPONSE,
    ToolDescriptor,
    acp_context_payload,
    mcp_context_payload,
    tools_list_payload,
)


def test_catalog_matches_seeded_tools():
    expected = {
        "db.query",
        "template.render",
        "email.send",
        "slack.send_message",
        "audit.log_event",
    }
    assert set(CATALOG.keys()) == expected


def test_tools_list_payload_shape():
    payload = tools_list_payload(["db.query"])
    assert "tools" in payload
    assert payload["tools"][0]["name"] == "db.query"
    schema = payload["tools"][0]["inputSchema"]
    assert schema["type"] == "object"
    assert "$schema" in schema
    assert schema["additionalProperties"] is False
    assert "sql" in schema["properties"]
    assert "sql" in schema["required"]


def test_tools_list_payload_unknown_tool_raises():
    with pytest.raises(KeyError, match="unknown tools"):
        tools_list_payload(["does.not.exist"])


def test_mcp_context_payload_includes_initialize_per_server():
    raw = mcp_context_payload(["db.query", "slack.send_message"], num_servers=2)
    parts = json.loads(raw)
    initialize_count = sum(1 for p in parts if "initialize" in p)
    tools_list_count = sum(1 for p in parts if "tools_list" in p)
    assert initialize_count == 2
    assert tools_list_count == 2
    assert parts[0]["initialize"] == INITIALIZE_RESPONSE


def test_mcp_context_payload_rejects_zero_servers():
    with pytest.raises(ValueError):
        mcp_context_payload(["db.query"], num_servers=0)


def test_mcp_payload_is_significantly_larger_than_acp_for_one_tool():
    # Sanity: the verbose MCP payload for one tool should dwarf a hypothetical
    # compact ACP-style schema for the same tool.
    mcp = mcp_context_payload(["db.query"], num_servers=1)
    fake_acp = json.dumps(
        {
            "manifest_id": "m-x",
            "version": "acp/v1",
            "actions": [
                {
                    "id": "a1",
                    "type": "http",
                    "endpoint": "grpc://db",
                    "method": "POST",
                    "schema": {"sql": "string", "limit": "int?"},
                    "auth": "pre-injected",
                }
            ],
            "boundaries": {"egress": ["db"], "audit_level": "full"},
            "feedback_endpoint": "/v1/feedback",
        }
    )
    assert len(mcp) > 4 * len(fake_acp), (len(mcp), len(fake_acp))


def test_acp_context_payload_passthrough():
    body = '{"manifest_id":"m"}'
    assert acp_context_payload(body) == body


def test_tool_descriptor_to_mcp_omits_examples_when_empty():
    td = ToolDescriptor(name="x", description="d", properties={"a": {"type": "string"}}, required=["a"])
    assert "examples" not in td.to_mcp()["inputSchema"]
