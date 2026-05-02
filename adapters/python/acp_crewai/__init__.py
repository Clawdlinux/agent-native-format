"""CrewAI adapter for ACP Execution Manifests.

Exposes each ACP action as a CrewAI ``BaseTool`` so a Crew can use the
manifest's actions like any other tool. The tool body POSTs through the
ACP auth-injection proxy, never touching credentials directly.

License: Apache-2.0

Example
-------

    from crewai import Agent, Crew, Task
    from acp_common import ACPClient
    from acp_crewai import manifest_to_crewai_tools

    acp = ACPClient("http://localhost:8080", auth_token="dev-token")
    manifest = acp.context(intent="query db then email", agent_id="analyst")
    tools = manifest_to_crewai_tools(manifest, proxy_url="http://localhost:8080")

    analyst = Agent(role="Analyst", goal="Summarize weekly metrics", tools=tools)
    crew = Crew(agents=[analyst], tasks=[Task(description="...", agent=analyst)])
    crew.kickoff()
"""

from __future__ import annotations

import json
import urllib.request
from typing import Any, Callable

from acp_common import (
    Action,
    Manifest,
    upstream_url_for,
)

try:  # pragma: no cover - exercised in adapter tests when crewai is installed
    from crewai.tools import BaseTool  # type: ignore
except Exception:  # pragma: no cover
    BaseTool = None  # type: ignore


def make_tool_callable(
    action: Action,
    manifest_id: str,
    proxy_url: str,
    auth_token: str | None = None,
    opener: Callable[..., Any] = urllib.request.urlopen,
    timeout: float = 10.0,
) -> Callable[..., str]:
    """Return a Python callable that executes the action through the proxy.

    Returned callable accepts kwargs (CrewAI tool argument convention) and
    returns the response body as a string.
    """

    url = upstream_url_for(action, proxy_url, manifest_id)
    method = action.method or "POST"

    def _call(**kwargs: Any) -> str:
        body = json.dumps(kwargs).encode("utf-8")
        req = urllib.request.Request(
            url, data=body, headers={"Content-Type": "application/json"}, method=method
        )
        if auth_token:
            req.add_header("Authorization", f"Bearer {auth_token}")
        with opener(req, timeout) as resp:
            return resp.read().decode("utf-8", "replace")

    _call.__name__ = f"acp_{action.id}"
    _call.__doc__ = (
        f"ACP action {action.id} ({action.endpoint}). "
        f"depends_on={list(action.depends_on)}"
    )
    return _call


def manifest_to_crewai_tools(
    manifest: Manifest,
    proxy_url: str,
    auth_token: str | None = None,
) -> list[Any]:
    """Render the manifest as a list of CrewAI ``BaseTool`` instances.

    When ``crewai`` is not installed, returns plain callables (with the same
    name and docstring) so unit tests can drive the adapter without the
    heavy dep.
    """
    callables = [
        make_tool_callable(a, manifest.manifest_id, proxy_url, auth_token=auth_token)
        for a in manifest.actions
    ]
    if BaseTool is None:  # pragma: no cover
        return callables  # callable fallback for environments without crewai

    tools: list[Any] = []
    for fn, action in zip(callables, manifest.actions, strict=True):
        tools.append(_to_basetool(fn, action))
    return tools


def _to_basetool(fn: Callable[..., str], action: Action) -> Any:  # pragma: no cover
    """Wrap a callable in a minimal CrewAI BaseTool subclass."""

    class _Tool(BaseTool):  # type: ignore[misc, valid-type]
        name: str = f"acp_{action.id}"
        description: str = fn.__doc__ or f"ACP action {action.id}"

        def _run(self, **kwargs: Any) -> str:
            return fn(**kwargs)

    return _Tool()


__all__ = ["make_tool_callable", "manifest_to_crewai_tools"]
