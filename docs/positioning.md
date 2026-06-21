# Positioning: Agent Contract Protocol

> Status: 2026-06-04. Companion to [SPEC.md](../SPEC.md).

## One-line statement

MCP answers **what tools exist**. ACP answers **what execution is allowed**.

Agent Contract Protocol is the governed execution layer for autonomous agents:
any tool source in, signed and auditable execution contracts out.

## What changed

The original ACP framing was Agent Context Protocol. It emphasized compact
schemas, one round trip, and token reduction.

That problem is real, but it is no longer enough. Claude Code Tool Search,
MCP-Zero, Semantic Tool Discovery, mcp2cli, and MCPorter all attack discovery,
tool selection, and schema cost. The market is moving those layers toward
commodity infrastructure.

ACP now targets the open layers above discovery:

| Layer | Question | Status |
|---|---|---|
| 1 Discovery | What tools exist? | MCP owns this |
| 2 Selection | Which tools fit this intent? | Commoditizing |
| 3 Schema cost | How much context does the catalog burn? | Becoming platform default |
| 4 Ordering | What sequence is valid? | ACP already declares it |
| 5 Identity | Who is the agent acting as? | Open |
| 6 Policy | What is allowed? | Open |
| 7 Execution | Where is the call isolated and enforced? | Partial |
| 8 Audit | What happened, and can we prove it? | Open |
| 9 Feedback | Did it work, and can the system improve? | Open |

The thesis is simple: discovery matters once per tool. Governance matters on
every action.

## The execution contract

The core primitive is an Execution Contract. The current v0.1 wire type is
still named `ExecutionManifest` for compatibility, but the product meaning has
shifted.

```text
ExecutionContract {
  intent          // the agent's stated goal
  capabilities   // scoped schemas and endpoints
  identity        // principal and credential alias, never raw secrets
  plan            // ordered actions with depends_on
  policy          // egress, approval, TTL, rate limits, audit level
  signature       // server signature over the contract
}
```

The proxy enforces the contract. The model does not get to self-police.

## The Databricks analogy

Databricks did not win by claiming object storage was dead. It defined a table
contract over commodity storage: metadata, transactions, governance, and
interoperability.

ACP should do the same for agents.

- Commodity sources: MCP servers, REST/OpenAPI endpoints, gRPC services, CLIs,
  Kubernetes Services.
- Commodity compute: the LLM agent loop.
- Missing middle: a portable governed execution contract.

That contract is useful locally and in a regulated cluster. Same abstraction.
Different deployment boundary.

## What ACP does not claim

- ACP does not replace MCP.
- ACP does not own tool discovery.
- ACP does not compete on raw token reduction as a moat.
- ACP does not require teams to rewrite their tools.
- ACP is not a full enterprise agent control plane.

ACP consumes existing tool catalogs and turns intent into governed execution.

## Why this matters

The blocker for production autonomy is trust. Teams hesitate to let agents touch
real systems because they cannot bound blast radius, attribute actions, revoke a
plan, or replay an audit trail.

ACP makes those controls part of the primitive:

- Auth stays out of model context.
- Actions are ordered before execution.
- Egress and approval gates are declared up front.
- Contracts can expire and be revoked.
- Audit records attach to the same contract id.

## Validation posture

The next public step is not a launch claim. It is a question.

Ask agent builders how they handle identity, audit, approval, revocation, and
bounded blast radius today. If they say Tool Search solved their problem, ACP
should narrow scope. If they independently name execution trust as the blocker,
the contract primitive is worth hardening.

See [docs/validation/signals.md](./validation/signals.md).

## References inside this repo

- [SPEC.md](../SPEC.md)
- [docs/architecture.md](./architecture.md)
- [docs/operator-integration.md](./operator-integration.md)
- [docs/pitch-deck-data.md](./pitch-deck-data.md)
- [docs/discovery/2026-06-research-log.md](./discovery/2026-06-research-log.md)