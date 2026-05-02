"""Tests for ``acp_openai`` (manifest -> OpenAI function-calling tools)."""

from __future__ import annotations

import json

import pytest
from conftest import SAMPLE_MANIFEST, fake_opener, fake_response

from acp_common import Manifest
from acp_openai import (
    DispatchResult,
    dispatch_tool_call,
    find_action,
    manifest_to_openai_tools,
)


def test_manifest_to_openai_tools_shape():
    mf = Manifest.from_dict(SAMPLE_MANIFEST)
    tools = manifest_to_openai_tools(mf)
    assert len(tools) == 2
    assert tools[0]["type"] == "function"
    fn = tools[0]["function"]
    assert fn["name"] == "a1"
    assert fn["parameters"]["type"] == "object"
    assert fn["parameters"]["required"] == ["sql"]
    assert "ACP action a1" in fn["description"]


def test_manifest_to_openai_tools_describes_dependencies():
    mf = Manifest.from_dict(SAMPLE_MANIFEST)
    tools = manifest_to_openai_tools(mf)
    a2 = tools[1]["function"]
    assert "requires a1 first" in a2["description"]


def test_dispatch_tool_call_targets_proxy_url_and_passes_args():
    captured = {}

    def opener(req, timeout):  # noqa: ARG001
        captured["url"] = req.full_url
        captured["body"] = req.data
        captured["method"] = req.get_method()
        captured["auth"] = req.get_header("Authorization")
        return fake_response(200, body={"rows": [{"id": 1}]})

    mf = Manifest.from_dict(SAMPLE_MANIFEST)
    call = {"function": {"name": "a1", "arguments": json.dumps({"sql": "select 1"})}}
    out = dispatch_tool_call(
        call,
        mf,
        proxy_url="http://acp.local",
        auth_token="dev",
        opener=opener,
    )
    assert isinstance(out, DispatchResult)
    assert out.action_id == "a1"
    assert out.status == 200
    assert captured["url"] == "http://acp.local/v1/exec/m-test/a1"
    assert json.loads(captured["body"]) == {"sql": "select 1"}
    assert captured["method"] == "POST"
    assert captured["auth"] == "Bearer dev"


def test_dispatch_tool_call_accepts_dict_arguments():
    mf = Manifest.from_dict(SAMPLE_MANIFEST)
    call = {"function": {"name": "a1", "arguments": {"sql": "select 1"}}}
    out = dispatch_tool_call(
        call,
        mf,
        proxy_url="http://acp.local",
        opener=fake_opener([fake_response(200, body={"ok": True})]),
    )
    assert out.status == 200


def test_dispatch_tool_call_handles_sdk_object():
    """Ensure both dict and the OpenAI SDK object shape work."""

    class FakeFn:
        name = "a1"
        arguments = json.dumps({"sql": "x"})

    class FakeCall:
        function = FakeFn()

    mf = Manifest.from_dict(SAMPLE_MANIFEST)
    out = dispatch_tool_call(
        FakeCall(),
        mf,
        proxy_url="http://acp.local",
        opener=fake_opener([fake_response(200, body={"ok": True})]),
    )
    assert out.action_id == "a1"


def test_find_action_unknown_raises():
    mf = Manifest.from_dict(SAMPLE_MANIFEST)
    with pytest.raises(KeyError):
        find_action(mf, "does-not-exist")
