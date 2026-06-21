# ACP <-> agentic-operator integration

> Companion to [SPEC.md §10 Relationship to MCP](../SPEC.md#10-relationship-to-mcp)
> and [docs/positioning.md](./positioning.md). Status: 2026-05-03.

## Why this doc

The agentic-operator (Clawdlinux/agentic-operator-core, public) is the
Kubernetes-native control plane for agent workloads in regulated clusters.
ACP is the runtime layer that returns intent-scoped Execution Contracts.
This doc defines the contract between the two so neither side blocks the
other.

## Layering

```
+-------------------------+
|   Agent (Python/Go)     |
+-------------------------+
            |  POST /v1/context
            v
+-------------------------+
|   ACP server            |   <-- this repo (Clawdlinux/agent-contract-protocol)
| - intent resolver       |
| - contract builder      |
| - auth proxy            |
+-------------------------+
            ^  reads tools from
            |
   +--------+--------+--------------------------------+
   |        |        |                                |
   v        v        v                                v
+------+ +------+ +-----------------------------------+
| MCP  | | REST | | k8s source adapter                |
| svc  | | API  | | (this repo, internal/sources/k8s) |
+------+ +------+ +-----------------------------------+
                              |
                              v
                  +-------------------------+
                  | agentic-operator on K8s |
                  | - AgentCard CRD         |
                  | - AgentWorkload CRD     |
                  | - Service + annotations |
                  +-------------------------+
```

ACP's k8s source adapter (this repo) consumes annotated Kubernetes
Services. The operator (other repo) is one producer of those Services.
Both can be deployed independently; pairing them is value-add, not
required.

## What lives where

| Concern | Owner | Repo |
|---|---|---|
| ACP wire protocol | this repo | `Clawdlinux/agent-contract-protocol` SPEC.md |
| ACP server runtime | this repo | `Clawdlinux/agent-contract-protocol` cmd/, internal/ |
| ACP source adapters (MCP, k8s) | this repo | `Clawdlinux/agent-contract-protocol` internal/sources/ |
| Kubernetes CRDs (AgentWorkload, AgentCard, Tenant) | operator | `Clawdlinux/agentic-operator-core` api/v1alpha1/ |
| Reconcilers | operator | `Clawdlinux/agentic-operator-core` internal/controller/ |
| RuntimeClass / NetworkPolicy / RBAC manifests | operator | `Clawdlinux/agentic-operator-core` charts/ |
| ACP Service annotation contract | this repo | `internal/sources/k8s/source.go` |

## Annotation contract (ACP side)

The operator (or any other producer) opts a Service into ACP discovery
by adding annotations under the `acp.clawdlinux.org/` prefix:

| Annotation | Required | Default | Notes |
|---|---|---|---|
| `expose` | yes | — | Must be `"true"` (case-insensitive) to opt in. |
| `tool-id` | yes | — | The ACP tool ID (e.g. `billing.query`). |
| `capabilities` | yes | — | Comma-separated capability tags. |
| `endpoint-path` | no | `/` | Appended to `http://<host>:<port>`. |
| `method` | no | `POST` | HTTP verb. |
| `schema` | no | `{}` | Compact ACP form (`sql:string,limit:int?`). |
| `timeout` | no | `30s` | Per-action wall-clock timeout. |
| `require-approval` | no | `false` | If `true`, the action lands in `boundaries.require_approval` and the proxy blocks until an out-of-band approval is recorded. |

See `internal/sources/k8s/source.go` for the parser and
`internal/sources/k8s/source_test.go` for canonical examples.

## Operator-side work (out of scope for this repo)

The agentic-operator should:

1. **Annotate Services it manages** when an `AgentWorkload` (or
   `AgentCard`) declares its tools should be exposed via ACP. This is
   a one-line addition to the existing reconcile loop.
2. **Optionally bundle the ACP server** as a Helm sub-chart so an
   operator install yields a working ACP control plane out of the box.
3. **Honor `require-approval`** by mirroring the ACP proxy's 403 +
   `X-ACP-Approval-Required` semantics through whatever
   approval-recording UI the operator already exposes.

A separate PR in `Clawdlinux/agentic-operator-core` will:

- Add `acp.clawdlinux.org/*` annotations to the Services emitted by the
  AgentWorkload reconciler.
- Wire an optional ACP server Deployment + Service into the operator
  Helm chart, gated by `acp.enabled=true`.
- Document the integration in the operator's `docs/` tree with a link
  back to this file.

That work is tracked separately so this repo can ship the source
adapter and contract without waiting on operator changes.

## Air-gapped story

The source spec (§6.1) calls out air-gapped enterprises as a target.
With the k8s source adapter:

- The ACP server runs as a `Deployment` in the same cluster.
- The auth proxy runs as a sidecar or shared `Deployment`.
- The credential store is `Secret`s mounted by the proxy (the agent
  never sees them).
- The intent resolver runs offline (keyword or embedding; both are
  pure-Go and dependency-free).
- The MCP source adapter is optional; in air-gapped environments the
  k8s source adapter alone is sufficient.

No outbound network, no API keys in agent context, no shared
infrastructure. The same operator that runs the agent workloads runs
the ACP control plane.

## Decision log

| Date | Decision |
|---|---|
| 2026-05-03 | Implement the k8s source adapter in this repo. Operator-side annotations land in a follow-up PR in `agentic-operator-core`. |
| 2026-05-03 | Choose Service annotations over a custom CRD because annotations are universal and don't require the operator to be installed. |
| 2026-05-03 | Use `acp.clawdlinux.org/` annotation prefix to match the existing CRD group `agentic.clawdlinux.org`. |
