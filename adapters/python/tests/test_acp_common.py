"""Tests for ``acp_common`` (shared adapter core)."""

from __future__ import annotations

import json
from copy import deepcopy

import pytest
from conftest import SAMPLE_MANIFEST, fake_opener, fake_response

from acp_common import (
    ACPClient,
    ACPHTTPError,
    Action,
    CycleError,
    Manifest,
    action_to_jsonschema,
    expand_schema_field,
    topological_order,
    upstream_url_for,
)


def test_manifest_from_dict_round_trip():
    mf = Manifest.from_dict(SAMPLE_MANIFEST)
    assert mf.manifest_id == "m-test"
    assert mf.version == "acp/v1"
    assert mf.actions[0].id == "a1"
    assert mf.actions[1].depends_on == ("a1",)
    assert mf.boundaries.require_approval == ("a2",)


def test_expand_schema_field_primitives():
    assert expand_schema_field("string") == {"type": "string"}
    assert expand_schema_field("int") == {"type": "integer"}
    assert expand_schema_field("bool") == {"type": "boolean"}


def test_expand_schema_field_optional_array_enum_ref():
    assert expand_schema_field("int?") == {"type": "integer"}
    assert expand_schema_field("string[]") == {"type": "array", "items": {"type": "string"}}
    assert expand_schema_field("enum:read|write") == {"type": "string", "enum": ["read", "write"]}
    out = expand_schema_field("ref:attachment")
    assert out["type"] == "string"
    assert "attachment" in out["description"]


def test_expand_schema_field_unknown_falls_back_to_string():
    assert expand_schema_field("snowflake") == {"type": "string"}


def test_action_to_jsonschema_required_vs_optional():
    mf = Manifest.from_dict(SAMPLE_MANIFEST)
    schema = action_to_jsonschema(mf.actions[0])  # db.query: sql string + limit int?
    assert schema["properties"]["sql"]["type"] == "string"
    assert schema["properties"]["limit"]["type"] == "integer"
    assert schema["required"] == ["sql"]
    assert schema["additionalProperties"] is False


def test_topological_order_simple_chain():
    mf = Manifest.from_dict(SAMPLE_MANIFEST)
    order = [a.id for a in topological_order(mf.actions)]
    assert order == ["a1", "a2"]


def test_topological_order_diamond_is_deterministic():
    actions = [
        Action("a", "http", "u", "POST", {}, "pre-injected", depends_on=()),
        Action("b", "http", "u", "POST", {}, "pre-injected", depends_on=("a",)),
        Action("c", "http", "u", "POST", {}, "pre-injected", depends_on=("a",)),
        Action("d", "http", "u", "POST", {}, "pre-injected", depends_on=("b", "c")),
    ]
    order = [x.id for x in topological_order(actions)]
    assert order == ["a", "b", "c", "d"]


def test_topological_order_detects_cycle():
    actions = [
        Action("x", "http", "u", "POST", {}, "pre-injected", depends_on=("y",)),
        Action("y", "http", "u", "POST", {}, "pre-injected", depends_on=("x",)),
    ]
    with pytest.raises(CycleError):
        topological_order(actions)


def test_topological_order_ignores_external_dependencies():
    actions = [
        Action("a", "http", "u", "POST", {}, "pre-injected", depends_on=("external",)),
        Action("b", "http", "u", "POST", {}, "pre-injected", depends_on=("a",)),
    ]
    order = [x.id for x in topological_order(actions)]
    assert order == ["a", "b"]


def test_upstream_url_for_format():
    mf = Manifest.from_dict(SAMPLE_MANIFEST)
    url = upstream_url_for(mf.actions[1], "http://acp.local", "m-1")
    assert url == "http://acp.local/v1/exec/m-1/a2"


def test_acp_client_context_uses_bearer_token_and_returns_manifest():
    captured = {}

    def opener(req, timeout):  # noqa: ARG001
        captured["url"] = req.full_url
        captured["auth"] = req.get_header("Authorization")
        captured["body"] = json.loads(req.data.decode())
        return fake_response(200, body=SAMPLE_MANIFEST)

    client = ACPClient(base_url="http://acp.local/", auth_token="dev", _opener=opener)
    mf = client.context(intent="x", agent_id="agent", capabilities=["sql"])
    assert mf.manifest_id == "m-test"
    assert captured["url"] == "http://acp.local/v1/context"
    assert captured["auth"] == "Bearer dev"
    assert captured["body"] == {"intent": "x", "agent_id": "agent", "capabilities": ["sql"]}


def test_acp_client_context_propagates_http_error():
    import urllib.error

    def opener(req, timeout):  # noqa: ARG001
        raise urllib.error.HTTPError(
            req.full_url, 422, "no caps", hdrs=None, fp=None
        )

    client = ACPClient(base_url="http://acp.local", _opener=opener)
    with pytest.raises(ACPHTTPError) as ei:
        client.context(intent="", agent_id="a")
    assert ei.value.status == 422


def test_manifest_handles_missing_optional_fields():
    minimal = deepcopy(SAMPLE_MANIFEST)
    minimal["actions"][0].pop("timeout", None)
    minimal["actions"][1].pop("depends_on", None)
    mf = Manifest.from_dict(minimal)
    assert mf.actions[0].timeout == ""
    assert mf.actions[1].depends_on == ()
