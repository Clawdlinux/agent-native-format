"""Shared utilities for ACP Python adapters.

This module is intentionally framework-agnostic. Each adapter
(`acp_openai`, `acp_langgraph`, `acp_crewai`) imports from here so the
manifest fetch / execute flow stays consistent.

License: Apache-2.0
"""

from __future__ import annotations

import json
import urllib.error
import urllib.request
from collections import defaultdict, deque
from dataclasses import dataclass, field
from typing import Any, Callable, Iterable


# --- Wire types (mirror pkg/manifest/types.go) ----------------------------


@dataclass(frozen=True)
class Action:
    id: str
    type: str
    endpoint: str
    method: str
    schema: dict[str, str]
    auth: str
    timeout: str = ""
    depends_on: tuple[str, ...] = ()


@dataclass(frozen=True)
class Boundary:
    egress: tuple[str, ...]
    max_tokens_per_action: int
    audit_level: str
    require_approval: tuple[str, ...] = ()


@dataclass(frozen=True)
class Manifest:
    manifest_id: str
    version: str
    ttl: str
    actions: tuple[Action, ...]
    boundaries: Boundary
    feedback_endpoint: str

    @classmethod
    def from_dict(cls, d: dict[str, Any]) -> "Manifest":
        actions = tuple(
            Action(
                id=a["id"],
                type=a["type"],
                endpoint=a["endpoint"],
                method=a.get("method", ""),
                schema=dict(a.get("schema", {})),
                auth=a.get("auth", ""),
                timeout=a.get("timeout", ""),
                depends_on=tuple(a.get("depends_on", []) or []),
            )
            for a in d.get("actions", [])
        )
        b = d.get("boundaries", {})
        boundaries = Boundary(
            egress=tuple(b.get("egress", []) or []),
            max_tokens_per_action=int(b.get("max_tokens_per_action", 0)),
            audit_level=b.get("audit_level", ""),
            require_approval=tuple(b.get("require_approval", []) or []),
        )
        return cls(
            manifest_id=d["manifest_id"],
            version=d["version"],
            ttl=d.get("ttl", ""),
            actions=actions,
            boundaries=boundaries,
            feedback_endpoint=d.get("feedback_endpoint", ""),
        )


# --- Errors ----------------------------------------------------------------


class ACPError(Exception):
    """Base error for ACP adapter failures."""


class ACPHTTPError(ACPError):
    """Raised when the ACP server returns a non-2xx response."""

    def __init__(self, status: int, message: str) -> None:
        super().__init__(f"acp http {status}: {message}")
        self.status = status
        self.message = message


class CycleError(ACPError):
    """Raised when manifest depends_on declares a cycle."""


# --- Client ----------------------------------------------------------------


@dataclass
class ACPClient:
    """Tiny HTTP client over `POST /v1/context` and `POST /v1/feedback`.

    Adapters use this to fetch a manifest given an intent. Anything more
    sophisticated (retries, circuit breakers) belongs in the user's stack.
    """

    base_url: str
    auth_token: str | None = None
    timeout: float = 10.0
    _opener: Callable[[urllib.request.Request, float], Any] = field(
        default=urllib.request.urlopen, repr=False
    )

    def context(
        self,
        intent: str,
        agent_id: str,
        capabilities: Iterable[str] | None = None,
    ) -> Manifest:
        body = json.dumps(
            {
                "intent": intent,
                "agent_id": agent_id,
                "capabilities": list(capabilities or []),
            }
        ).encode("utf-8")
        req = urllib.request.Request(
            self.base_url.rstrip("/") + "/v1/context",
            data=body,
            headers={"Content-Type": "application/json"},
            method="POST",
        )
        if self.auth_token:
            req.add_header("Authorization", f"Bearer {self.auth_token}")
        try:
            with self._opener(req, self.timeout) as resp:  # type: ignore[arg-type]
                raw = resp.read().decode("utf-8")
        except urllib.error.HTTPError as e:
            payload = e.read().decode("utf-8", "replace") if e.fp else str(e)
            raise ACPHTTPError(e.code, payload) from None
        return Manifest.from_dict(json.loads(raw))

    def feedback(
        self,
        manifest_id: str,
        action_id: str,
        outcome: str,
        latency_ms: int = 0,
        error: str = "",
    ) -> None:
        body = json.dumps(
            {
                "manifest_id": manifest_id,
                "action_id": action_id,
                "outcome": outcome,
                "latency_ms": latency_ms,
                "error": error,
            }
        ).encode("utf-8")
        req = urllib.request.Request(
            self.base_url.rstrip("/") + "/v1/feedback",
            data=body,
            headers={"Content-Type": "application/json"},
            method="POST",
        )
        if self.auth_token:
            req.add_header("Authorization", f"Bearer {self.auth_token}")
        try:
            self._opener(req, self.timeout)  # type: ignore[arg-type]
        except urllib.error.HTTPError as e:
            raise ACPHTTPError(e.code, str(e.reason)) from None


# --- Schema translation ----------------------------------------------------


# Map ACP compact types to JSON-Schema-ish primitives used by OpenAI etc.
_TYPE_BASE = {
    "string": {"type": "string"},
    "int": {"type": "integer"},
    "float": {"type": "number"},
    "bool": {"type": "boolean"},
    "json": {"type": "object", "additionalProperties": True},
    "bytes": {"type": "string", "contentEncoding": "base64"},
}


def expand_schema_field(compact: str) -> dict[str, Any]:
    """Translate the compact ACP schema mini-language into JSON-Schema.

    Supports:
        ``string``, ``int?``, ``string[]``, ``enum:a|b|c``, ``ref:<id>``
    Unknown forms fall back to ``{"type": "string"}`` so callers always get
    a well-formed JSON-Schema fragment.
    """
    s = compact.strip()
    if s.startswith("enum:"):
        return {"type": "string", "enum": s[len("enum:") :].split("|")}
    if s.startswith("ref:"):
        return {"type": "string", "description": f"opaque reference of kind {s[len('ref:') :]}"}
    optional = s.endswith("?")
    if optional:
        s = s[:-1]
    if s.endswith("[]"):
        inner = expand_schema_field(s[:-2])
        return {"type": "array", "items": inner}
    return _TYPE_BASE.get(s, {"type": "string"})


def action_to_jsonschema(action: Action) -> dict[str, Any]:
    """Build a JSON-Schema object for an action's input parameters."""
    properties: dict[str, Any] = {}
    required: list[str] = []
    for name, compact in action.schema.items():
        properties[name] = expand_schema_field(compact)
        if not compact.endswith("?"):
            required.append(name)
    schema: dict[str, Any] = {
        "type": "object",
        "properties": properties,
        "additionalProperties": False,
    }
    if required:
        schema["required"] = required
    return schema


# --- Execution ordering ----------------------------------------------------


def topological_order(actions: Iterable[Action]) -> list[Action]:
    """Return actions in a valid execution order honoring depends_on.

    Raises CycleError if depends_on declares a cycle. Tie-breaks by action id
    so the ordering is deterministic.
    """
    by_id = {a.id: a for a in actions}
    indeg: dict[str, int] = defaultdict(int)
    children: dict[str, list[str]] = defaultdict(list)
    for a in by_id.values():
        indeg[a.id]  # ensure key exists
        for dep in a.depends_on:
            if dep not in by_id:
                # depend on something outside this manifest; treat as already done
                continue
            indeg[a.id] += 1
            children[dep].append(a.id)

    ready: deque[str] = deque(sorted(aid for aid, n in indeg.items() if n == 0))
    out: list[Action] = []
    while ready:
        aid = ready.popleft()
        out.append(by_id[aid])
        for child in sorted(children[aid]):
            indeg[child] -= 1
            if indeg[child] == 0:
                ready.append(child)
    if len(out) != len(by_id):
        raise CycleError(f"depends_on cycle in actions: {sorted(by_id)}")
    return out


def upstream_url_for(action: Action, base_proxy_url: str, manifest_id: str) -> str:
    """Return the proxy URL the agent should call for this action.

    When ACP is paired with the auth-injection proxy, agents do NOT call
    `action.endpoint` directly. They call the proxy at
    ``{base_proxy_url}/v1/exec/{manifest_id}/{action_id}`` and the proxy
    forwards (with credentials injected) to the upstream.
    """
    return f"{base_proxy_url.rstrip('/')}/v1/exec/{manifest_id}/{action.id}"


__all__ = [
    "Action",
    "Boundary",
    "Manifest",
    "ACPClient",
    "ACPError",
    "ACPHTTPError",
    "CycleError",
    "expand_schema_field",
    "action_to_jsonschema",
    "topological_order",
    "upstream_url_for",
]
