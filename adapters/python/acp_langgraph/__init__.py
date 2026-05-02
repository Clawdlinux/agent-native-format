"""LangGraph adapter for ACP Execution Manifests.

Builds a LangGraph ``StateGraph`` whose nodes execute the ACP manifest's
actions in `depends_on` order via the auth-injection proxy.

License: Apache-2.0

Design
------

A manifest with actions ``a1 -> a2 -> a3`` becomes a graph with three
nodes connected in topological order. Each node:

1. Fetches its action's input from the shared state (key ``input.<aid>``).
2. POSTs to the ACP proxy URL for that action.
3. Writes the response to the shared state (key ``output.<aid>``).

The compiled graph can be invoked or streamed like any other LangGraph.

Example
-------

    from acp_common import ACPClient
    from acp_langgraph import build_graph

    acp = ACPClient("http://localhost:8080", auth_token="dev-token")
    manifest = acp.context(intent="query db then email", agent_id="agent")
    graph = build_graph(manifest, proxy_url="http://localhost:8080").compile()
    final_state = graph.invoke({"input.a1": {"sql": "select 1"}})
"""

from __future__ import annotations

import json
import urllib.request
from typing import Any, Callable, TypedDict

from acp_common import (
    Action,
    Manifest,
    topological_order,
    upstream_url_for,
)

try:  # pragma: no cover - exercised in adapter tests when langgraph is installed
    from langgraph.graph import END, START, StateGraph  # type: ignore
except Exception:  # pragma: no cover
    END = "__end__"  # type: ignore
    START = "__start__"  # type: ignore
    StateGraph = None  # type: ignore


class ACPState(TypedDict, total=False):
    """Shared state for an ACP graph run.

    ``input.<aid>`` carries per-action arguments. ``output.<aid>`` holds the
    response body string returned by the proxy. ``status.<aid>`` carries the
    HTTP status code.
    """


def make_node(
    action: Action,
    manifest_id: str,
    proxy_url: str,
    auth_token: str | None = None,
    opener: Callable[..., Any] = urllib.request.urlopen,
    timeout: float = 10.0,
) -> Callable[[dict[str, Any]], dict[str, Any]]:
    """Build a callable LangGraph node for one ACP action."""

    url = upstream_url_for(action, proxy_url, manifest_id)
    method = action.method or "POST"
    in_key = f"input.{action.id}"
    out_key = f"output.{action.id}"
    status_key = f"status.{action.id}"

    def _node(state: dict[str, Any]) -> dict[str, Any]:
        payload = state.get(in_key, {}) or {}
        body = json.dumps(payload).encode("utf-8")
        req = urllib.request.Request(
            url, data=body, headers={"Content-Type": "application/json"}, method=method
        )
        if auth_token:
            req.add_header("Authorization", f"Bearer {auth_token}")
        with opener(req, timeout) as resp:
            status = resp.getcode()
            text = resp.read().decode("utf-8", "replace")
        return {out_key: text, status_key: status}

    _node.__name__ = f"acp_{action.id}"
    return _node


def build_graph(
    manifest: Manifest,
    proxy_url: str,
    auth_token: str | None = None,
):
    """Construct (but do not compile) a LangGraph for the manifest."""
    if StateGraph is None:  # pragma: no cover
        raise ImportError(
            "langgraph is required for build_graph(); install with `pip install langgraph`"
        )

    graph = StateGraph(dict)
    ordered = topological_order(manifest.actions)
    for action in ordered:
        graph.add_node(
            f"acp_{action.id}",
            make_node(action, manifest.manifest_id, proxy_url, auth_token=auth_token),
        )

    # Edges: every action -> its dependents; sources hang off START; sinks -> END.
    by_id = {a.id: a for a in manifest.actions}
    has_dependents = {a.id: False for a in manifest.actions}
    for a in manifest.actions:
        for dep in a.depends_on:
            if dep in by_id:
                graph.add_edge(f"acp_{dep}", f"acp_{a.id}")
                has_dependents[dep] = True
    for a in ordered:
        if not a.depends_on:
            graph.add_edge(START, f"acp_{a.id}")
    for aid, has in has_dependents.items():
        if not has:
            graph.add_edge(f"acp_{aid}", END)

    return graph


__all__ = ["ACPState", "build_graph", "make_node"]
