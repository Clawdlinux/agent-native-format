"""Tests for ``acp_langgraph.build_graph`` using a stubbed ``langgraph``.

The adapter already gracefully degrades when ``langgraph`` is missing
(StateGraph is None and ``build_graph`` raises ImportError). To exercise the
full graph-construction path without pulling the real heavy dep into CI we
inject a minimal fake ``langgraph.graph`` module before importing the
adapter.
"""

from __future__ import annotations

import importlib
import sys
import types
from typing import Any

import pytest


@pytest.fixture
def langgraph_stub(monkeypatch):
    """Install a fake ``langgraph.graph`` module so ``StateGraph`` is non-None."""

    pkg = types.ModuleType("langgraph")
    sub = types.ModuleType("langgraph.graph")

    sub.START = "__start__"
    sub.END = "__end__"

    class _StateGraph:
        def __init__(self, _state_type: Any) -> None:
            self.nodes: dict[str, Any] = {}
            self.edges: list[tuple[str, str]] = []

        def add_node(self, name: str, fn: Any) -> None:
            self.nodes[name] = fn

        def add_edge(self, src: str, dst: str) -> None:
            self.edges.append((src, dst))

        def compile(self):
            return self

    sub.StateGraph = _StateGraph

    monkeypatch.setitem(sys.modules, "langgraph", pkg)
    monkeypatch.setitem(sys.modules, "langgraph.graph", sub)

    # Force re-import so the adapter picks up the stubbed module.
    sys.modules.pop("acp_langgraph", None)
    yield importlib.import_module("acp_langgraph")
    # Cleanup so other tests get the no-langgraph behavior back.
    sys.modules.pop("acp_langgraph", None)


SAMPLE_MANIFEST_DICT = {
    "manifest_id": "m-test",
    "version": "acp/v1",
    "ttl": "300s",
    "actions": [
        {
            "id": "a1",
            "type": "http",
            "endpoint": "http://upstream/x",
            "method": "POST",
            "schema": {"sql": "string"},
            "auth": "pre-injected",
        },
        {
            "id": "a2",
            "type": "http",
            "endpoint": "http://upstream/y",
            "method": "POST",
            "schema": {"data": "json"},
            "auth": "pre-injected",
            "depends_on": ["a1"],
        },
        {
            "id": "a3",
            "type": "http",
            "endpoint": "http://upstream/z",
            "method": "POST",
            "schema": {"to": "string[]"},
            "auth": "pre-injected",
            "depends_on": ["a2"],
        },
    ],
    "boundaries": {"egress": ["upstream"], "max_tokens_per_action": 15000, "audit_level": "full"},
    "feedback_endpoint": "/v1/feedback",
}


def test_build_graph_creates_node_per_action(langgraph_stub):
    from acp_common import Manifest

    mf = Manifest.from_dict(SAMPLE_MANIFEST_DICT)
    graph = langgraph_stub.build_graph(mf, "http://acp.local")
    assert sorted(graph.nodes) == ["acp_a1", "acp_a2", "acp_a3"]


def test_build_graph_chains_edges_in_dependency_order(langgraph_stub):
    from acp_common import Manifest

    mf = Manifest.from_dict(SAMPLE_MANIFEST_DICT)
    graph = langgraph_stub.build_graph(mf, "http://acp.local")
    assert ("acp_a1", "acp_a2") in graph.edges
    assert ("acp_a2", "acp_a3") in graph.edges
    # Source hangs off START.
    assert ("__start__", "acp_a1") in graph.edges
    # Sink connects to END.
    assert ("acp_a3", "__end__") in graph.edges


def test_build_graph_compile_returns_compiled(langgraph_stub):
    from acp_common import Manifest

    mf = Manifest.from_dict(SAMPLE_MANIFEST_DICT)
    graph = langgraph_stub.build_graph(mf, "http://acp.local").compile()
    assert graph is not None


def test_build_graph_without_langgraph_raises_clearly(monkeypatch):
    """When langgraph is missing, build_graph must raise ImportError."""
    # Force the module to think langgraph is unavailable by clearing both
    # the cached adapter import and the stub (if any).
    monkeypatch.delitem(sys.modules, "langgraph", raising=False)
    monkeypatch.delitem(sys.modules, "langgraph.graph", raising=False)
    sys.modules.pop("acp_langgraph", None)
    adapter = importlib.import_module("acp_langgraph")
    sys.modules.pop("acp_langgraph", None)

    if adapter.StateGraph is not None:
        pytest.skip("langgraph really is installed in this env")
    from acp_common import Manifest

    mf = Manifest.from_dict(SAMPLE_MANIFEST_DICT)
    with pytest.raises(ImportError):
        adapter.build_graph(mf, "http://acp.local")
