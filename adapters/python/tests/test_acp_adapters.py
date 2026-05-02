"""Tests for ``acp_langgraph`` and ``acp_crewai`` adapters.

These don't require the framework deps to be installed: we exercise the
``make_node`` and ``make_tool_callable`` factories directly because they
wrap a stable transport interface (``urllib.request.urlopen``).
"""

from __future__ import annotations

import json

from conftest import SAMPLE_MANIFEST, fake_response

from acp_common import Manifest
from acp_crewai import make_tool_callable, manifest_to_crewai_tools
from acp_langgraph import make_node


def test_langgraph_make_node_executes_via_proxy():
    mf = Manifest.from_dict(SAMPLE_MANIFEST)
    captured = {}

    def opener(req, timeout):  # noqa: ARG001
        captured["url"] = req.full_url
        captured["body"] = json.loads(req.data.decode())
        return fake_response(200, body={"rows": []})

    node = make_node(mf.actions[0], mf.manifest_id, "http://acp.local", opener=opener)
    out = node({"input.a1": {"sql": "select 1"}})
    assert captured["url"] == "http://acp.local/v1/exec/m-test/a1"
    assert captured["body"] == {"sql": "select 1"}
    assert out["status.a1"] == 200
    assert "rows" in out["output.a1"]


def test_langgraph_make_node_handles_missing_input_state():
    mf = Manifest.from_dict(SAMPLE_MANIFEST)

    def opener(req, timeout):  # noqa: ARG001
        # No input state -> empty body is acceptable.
        assert req.data == b"{}"
        return fake_response(204, body="")

    node = make_node(mf.actions[1], mf.manifest_id, "http://acp.local", opener=opener)
    out = node({})
    assert out["status.a2"] == 204


def test_crewai_make_tool_callable_returns_response_text():
    mf = Manifest.from_dict(SAMPLE_MANIFEST)

    def opener(req, timeout):  # noqa: ARG001
        return fake_response(200, body={"ok": True})

    tool = make_tool_callable(
        mf.actions[0], mf.manifest_id, "http://acp.local", auth_token="dev", opener=opener
    )
    text = tool(sql="select 1")
    assert json.loads(text) == {"ok": True}


def test_crewai_manifest_to_tools_falls_back_to_callables_without_dep():
    mf = Manifest.from_dict(SAMPLE_MANIFEST)
    tools = manifest_to_crewai_tools(mf, proxy_url="http://acp.local")
    # Either real crewai BaseTool instances OR the callable fallback when crewai
    # is not installed.
    assert len(tools) == 2
    if callable(tools[0]):
        assert tools[0].__name__ == "acp_a1"
    else:  # pragma: no cover - exercised only when crewai is installed
        assert tools[0].name == "acp_a1"
