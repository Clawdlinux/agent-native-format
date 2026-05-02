# Agent Context Protocol (ACP) — v0.1 DRAFT

> **Status:** DRAFT — May 2, 2026
> **Editors:** Shreyansh Sancheti (Clawdlinux / NineVigil)
> **License:** CC BY 4.0 (spec only; see [LICENSE](./LICENSE) for runtime)
> **Tagline:** One API call. Complete execution context. Minimal tokens.

---

## 1. Abstract

The **Agent Context Protocol (ACP)** is a wire protocol between an autonomous
agent (or agent runtime) and a context-resolution service that returns a
complete, intent-scoped **Execution Manifest**. The manifest declares the
exact endpoints, schemas, authentication injection, execution dependencies,
and security boundaries required to satisfy the agent's stated intent — in
the fewest tokens possible.

ACP is an **intent-resolution and execution-planning layer that sits on top
of MCP** and other tool sources (REST APIs, gRPC services, Kubernetes APIs).
MCP answers “what tools exist?”; ACP answers “what exactly do I need to do
right now, and how?”. ACP servers consume MCP `tools/list` payloads (see
[§10](#10-relationship-to-mcp)) and emit token-minimal manifests that any
agent runtime can execute with one round trip.

## 2. Motivation

MCP was designed for human-like browsing of tool surfaces. It does that job
well. In production, however, that same human-friendly surface becomes a
dominant cost in agent context windows:

| Failure mode | Observed cost |
|---|---|
| Discovery tax (`tools/list` per server, every session) | 22 K tokens before first prompt (Eckstein) |
| Schema bloat (descriptions, examples, nested docs) | 4–32× token cost vs CLI (Scalekit, n=75) |
| No intent scoping (loads all tools, every time) | 143 K of 200 K tokens consumed (Apideck) |
| No auth injection (agent handles credentials in-context) | Credential-leak risk + extra round trips |
| No execution ordering (flat tool list) | Agent burns tokens reasoning about DAGs |

ACP doesn’t replace any of this — MCP servers stay where they are. ACP
strips the verbosity, scopes by intent, injects credentials at the proxy
boundary, and pre-computes execution order, then hands the agent a single
manifest small enough to fit in any context budget.

Cited references in [docs/references.md](./docs/references.md). See also
[docs/positioning.md](./docs/positioning.md) for the full “ACP on top of
MCP” framing.

## 3. Design Principles

1. **The infrastructure computes the context, not the agent.**
2. **One round trip.** Discovery, schema, auth, and ordering arrive together.
3. **Token-minimal by construction.** Schemas are field-name + type only.
4. **Auth never enters the agent's context window.** Injection happens at
   the proxy boundary.
5. **Execution is declared, not inferred.** `depends_on` is explicit.
6. **Boundaries are part of the contract.** Egress, budgets, approvals, and
   audit level are returned with the manifest.
7. **Every result is a training signal.** Feedback endpoint is mandatory.

## 4. Protocol

### 4.1 Endpoint

```
POST /v1/context
Content-Type: application/json
Authorization: Bearer <agent-identity-token>
```

### 4.2 Request

```json
{
  "intent": "query customer data, generate report, email to team",
  "agent_id": "analytics-agent-01",
  "capabilities": ["sql", "template", "email"],
  "constraints": {
    "max_tokens": 50000,
    "timeout": "120s",
    "output_format": "minimal"
  }
}
```

| Field | Type | Required | Notes |
|---|---|---|---|
| `intent` | string | yes | Natural-language statement of what the agent wants to accomplish. |
| `agent_id` | string | yes | Stable identity. Used for policy + audit. |
| `capabilities` | string[] | no | Optional hints the resolver may use to short-circuit intent parsing. |
| `constraints.max_tokens` | int | no | Token budget the manifest must respect. |
| `constraints.timeout` | duration | no | Wall-clock budget for the entire DAG. |
| `constraints.output_format` | enum | no | `minimal` \| `verbose`. Default `minimal`. |

### 4.3 Response — Execution Manifest

```json
{
  "manifest_id": "m-a7f3b2",
  "version": "acp/v1",
  "ttl": "300s",
  "actions": [
    {
      "id": "a1",
      "type": "http",
      "endpoint": "grpc://db-proxy.svc:50051/query",
      "method": "POST",
      "schema": { "sql": "string", "limit": "int?" },
      "auth": "pre-injected",
      "timeout": "30s"
    },
    {
      "id": "a2",
      "type": "template",
      "endpoint": "http://template-svc.svc:8080/render",
      "method": "POST",
      "schema": { "template_id": "string", "data": "json" },
      "auth": "pre-injected",
      "depends_on": ["a1"]
    },
    {
      "id": "a3",
      "type": "http",
      "endpoint": "https://email-gw.svc:443/send",
      "method": "POST",
      "schema": {
        "to": "string[]",
        "subject": "string",
        "body": "string",
        "attachment_ref": "string?"
      },
      "auth": "pre-injected",
      "depends_on": ["a2"]
    }
  ],
  "boundaries": {
    "egress": ["db-proxy.svc", "template-svc.svc", "email-gw.svc"],
    "max_tokens_per_action": 15000,
    "require_approval": ["a3"],
    "audit_level": "full"
  },
  "feedback_endpoint": "http://ninevigil.svc/v1/feedback"
}
```

### 4.4 Schema mini-language

Field types are declared as compact strings:

| Token | Meaning |
|---|---|
| `string`, `int`, `float`, `bool`, `json`, `bytes` | Scalar types |
| `T?` | Optional |
| `T[]` | Array of `T` |
| `enum:a\|b\|c` | Enum |
| `ref:<id>` | Opaque reference (e.g. attachment handle) |

Rationale: a 6,000-token JSON Schema collapses to ~80 tokens.

### 4.5 Action object

| Field | Required | Notes |
|---|---|---|
| `id` | yes | Stable within manifest. Referenced by `depends_on`. |
| `type` | yes | `http` \| `grpc` \| `template` \| `tool` (extensible). |
| `endpoint` | yes | URI reachable via the auth proxy. |
| `method` | yes (http) | HTTP verb. |
| `schema` | yes | Compact field map (see 4.4). |
| `auth` | yes | `pre-injected` \| `none`. Never raw credentials. |
| `timeout` | no | Per-action wall-clock budget. |
| `depends_on` | no | Array of action `id`s that must complete first. |

### 4.6 Boundaries

| Field | Notes |
|---|---|
| `egress` | Allowed destination hosts. Enforced at the proxy. |
| `max_tokens_per_action` | Server-side ceiling on payload size. |
| `require_approval` | Action ids that require human-in-the-loop approval. |
| `audit_level` | `none` \| `summary` \| `full`. |

### 4.7 Feedback

After each action the agent SHOULD POST:

```
POST /v1/feedback
{
  "manifest_id": "m-a7f3b2",
  "action_id": "a1",
  "outcome": "success" | "error" | "skipped",
  "latency_ms": 124,
  "tokens_in": 312,
  "tokens_out": 88,
  "error": null
}
```

Feedback feeds the optimizer. Compliance is encouraged but not required for
v0.1.

## 5. Conformance

A conforming **client** MUST:
- Send exactly one `POST /v1/context` per intent.
- Treat `auth: pre-injected` as opaque; never log credentials.
- Honor `depends_on` ordering.
- Honor `boundaries.egress` (refuse calls to other hosts).

A conforming **server** MUST:
- Return a manifest whose total schema payload is no larger than the
  equivalent MCP `tools/list` payload would be.
- Inject auth at the proxy boundary.
- Expire manifests at `ttl`.

## 6. Versioning

`version` is `acp/<major>`. Breaking changes bump the major. v0.x is
pre-stable.

## 7. Security Considerations

- Identity tokens MUST be short-lived (≤1h recommended).
- Manifests MUST NOT contain secret material.
- Egress allow-lists MUST be enforced at the proxy, not by the agent.
- `require_approval` actions MUST block at the proxy until an out-of-band
  approval is recorded.

## 8. References

See [docs/references.md](./docs/references.md) for the full citation list
(Perplexity, Apideck, Eckstein, Scalekit, arXiv 2602.14878, Cloudflare
Code Mode, AgentSpec, A2A, Oracle Open Agent Spec).

## 9. Changelog

- **v0.1 (2026-05-02)** — Initial draft extracted from
  `ACP_PoC_Specification_CONFIDENTIAL.docx`.
- **v0.1.1 (2026-05-03)** — Repositioned from "successor to MCP" to
  "intent-resolution layer on top of MCP and other tool sources". No wire
  format changes. Added §10.

## 10. Relationship to MCP

ACP **does not replace MCP**. ACP is a layer that consumes MCP (and other
tool sources) and emits intent-scoped manifests.

```
agent ── POST /v1/context ──> ACP server ──┬── reads MCP tools/list
                                           ├── reads REST/gRPC catalogs
                                           └── reads Kubernetes APIs
       <── ExecutionManifest ──
```

A conforming ACP server SHOULD provide an MCP source adapter. The reference
implementation in this repository ships one at
`internal/sources/mcp.Importer` (Go). The adapter:

1. Issues `GET <mcp_base>/tools/list` against any MCP-compliant server.
2. Infers ACP capability tags from each tool's name, plus optional
   source-level extras.
3. Compacts the verbose JSON-Schema `inputSchema` into the ACP
   mini-language defined in §4.4 (`string`, `int?`, `string[]`,
   `enum:a|b|c`, etc.).
4. Registers each tool in the ACP registry under
   `<source_name>.<tool_name>`.

The compaction step is what produces the token reduction headlined in
[docs/pitch-deck-data.md](./docs/pitch-deck-data.md). The MCP server itself
is unchanged; the agent just talks to ACP instead of to MCP directly.

### Conformance for an MCP-source adapter

An adapter MUST:

- Translate MCP `tools/list` descriptors into ACP `Tool` entries without
  loss of *executable* meaning (endpoint, method, required fields).
- Strip MCP `description`, `examples`, and JSON-Schema metadata that does
  not affect execution.
- Preserve MCP authentication mode by mapping it to `auth: pre-injected` on
  the ACP side and routing actual credentials through the ACP auth proxy.
- Set `egress` allow-list to the MCP server's host.

An adapter SHOULD:

- Tag each imported tool with at least one capability derived from the
  source name (e.g. `github`, `slack`).
- Honor MCP `notifications/tools/list_changed` to refresh the ACP
  registry.

See also [docs/positioning.md](./docs/positioning.md) for the strategic
rationale and ecosystem implications of this layering.
