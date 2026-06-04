"""OpenAI function-calling adapter for ACP Execution Contracts.

Converts an ACP manifest into the OpenAI Chat Completions ``tools=[...]``
parameter so existing OpenAI agent code can consume ACP manifests with no
behavioural changes beyond a single ``client.context(...)`` call up front.

License: Apache-2.0

Example
-------

    from openai import OpenAI
    from acp_common import ACPClient
    from acp_openai import manifest_to_openai_tools, dispatch_tool_call

    acp = ACPClient("http://localhost:8080", auth_token="dev-token")
    manifest = acp.context(
        intent="query the customer database, render a report, email the team",
        agent_id="analytics-agent",
    )
    tools = manifest_to_openai_tools(manifest)

    openai = OpenAI()
    resp = openai.chat.completions.create(
        model="gpt-4o-mini",
        tools=tools,
        messages=[{"role": "user", "content": "Send last week's report"}],
    )

    # When the model emits a tool call, dispatch through the ACP proxy.
    for choice in resp.choices:
        for call in choice.message.tool_calls or []:
            dispatch_tool_call(call, manifest, proxy_url="http://localhost:8080")
"""

from __future__ import annotations

import json
import urllib.request
from dataclasses import dataclass
from typing import Any

from acp_common import (
    Action,
    Manifest,
    action_to_jsonschema,
    upstream_url_for,
)


def manifest_to_openai_tools(manifest: Manifest) -> list[dict[str, Any]]:
    """Render a manifest as the ``tools`` parameter for the OpenAI API.

    Each ACP action becomes one OpenAI ``function`` tool. The function name
    is the action ID (e.g. ``a1``, ``a2``); the description includes the
    upstream endpoint host so the model has minimal grounding without
    leaking credentials.
    """
    tools: list[dict[str, Any]] = []
    for action in manifest.actions:
        tools.append(
            {
                "type": "function",
                "function": {
                    "name": _safe_function_name(action.id),
                    "description": _describe(action),
                    "parameters": action_to_jsonschema(action),
                },
            }
        )
    return tools


def _safe_function_name(action_id: str) -> str:
    # OpenAI function names must match ^[a-zA-Z0-9_-]+$ (max 64).
    out = "".join(c if c.isalnum() or c in "_-" else "_" for c in action_id)
    return out[:64] or "action"


def _describe(action: Action) -> str:
    parts = [f"ACP action {action.id}"]
    if action.depends_on:
        parts.append(f"requires {','.join(action.depends_on)} first")
    if action.timeout:
        parts.append(f"timeout {action.timeout}")
    return "; ".join(parts)


# --- Dispatch -------------------------------------------------------------


@dataclass(frozen=True)
class DispatchResult:
    action_id: str
    status: int
    body: str


def find_action(manifest: Manifest, action_id: str) -> Action:
    """Find an action by its ACP id (the safe-name conversion is one-way)."""
    for a in manifest.actions:
        if a.id == action_id or _safe_function_name(a.id) == action_id:
            return a
    raise KeyError(f"action {action_id!r} not in manifest {manifest.manifest_id}")


def dispatch_tool_call(
    tool_call: Any,
    manifest: Manifest,
    proxy_url: str,
    auth_token: str | None = None,
    timeout: float = 10.0,
    opener: Any = urllib.request.urlopen,
) -> DispatchResult:
    """Forward an OpenAI tool call through the ACP auth-injection proxy.

    ``tool_call`` accepts both the OpenAI SDK object (``tool_call.function.name``)
    and a plain dict shaped like ``{"function": {"name": ..., "arguments": ...}}``.
    Arguments may be a JSON string (per the OpenAI wire format) or a dict.
    """
    name, raw_args = _extract_call(tool_call)
    action = find_action(manifest, name)
    args: dict[str, Any]
    if isinstance(raw_args, str):
        args = json.loads(raw_args) if raw_args else {}
    else:
        args = dict(raw_args or {})

    body = json.dumps(args).encode("utf-8")
    req = urllib.request.Request(
        upstream_url_for(action, proxy_url, manifest.manifest_id),
        data=body,
        headers={"Content-Type": "application/json"},
        method=action.method or "POST",
    )
    if auth_token:
        req.add_header("Authorization", f"Bearer {auth_token}")
    with opener(req, timeout) as resp:
        status = resp.getcode()
        text = resp.read().decode("utf-8", "replace")
    return DispatchResult(action_id=action.id, status=status, body=text)


def _extract_call(tool_call: Any) -> tuple[str, Any]:
    if isinstance(tool_call, dict):
        fn = tool_call.get("function") or {}
        return fn.get("name", ""), fn.get("arguments")
    fn = getattr(tool_call, "function", None)
    if fn is None:
        raise ValueError("tool_call has no .function attribute")
    return getattr(fn, "name", ""), getattr(fn, "arguments", None)


__all__ = [
    "DispatchResult",
    "dispatch_tool_call",
    "find_action",
    "manifest_to_openai_tools",
]
