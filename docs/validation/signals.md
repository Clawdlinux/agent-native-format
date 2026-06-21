# Validation Signals

Status: draft for public discovery.

## Hypothesis

Production agent builders have a real execution-governance problem that tool
discovery, Tool Search, and MCP servers do not fully solve.

ACP should continue toward Agent Contract Protocol only if builders independently
name this pain.

## Question to ask

If you are building agents that touch real systems, how are you handling:

- Identity. Who is the agent acting as?
- Policy. What can it call?
- Approval. What requires a human gate?
- Revocation. Can a plan be stopped mid-flight?
- Audit. Can you prove what happened afterward?
- Blast radius. Can the agent leave the allowed boundary?

## Positive signals

- Practitioners mention identity, audit, revocation, approval, or bounded blast
  radius before ACP suggests those terms.
- People ask for a sidecar, proxy, or contract example they can run locally.
- Teams describe hand-rolled API gateway, OPA, proxy, audit-log, or MCP-server
  policy code that feels repetitive.
- Issues or discussions include concrete systems: Slack, GitHub, Sentry,
  Postgres, Kubernetes, billing, deployment, incident response.

## Warning signals

- Replies only care about token savings.
- People treat ACP as another MCP gateway or CLI converter.
- The discussion centers on Claude Code users only.
- No one mentions production, audit, compliance, credentials, or revocation.

## Kill signals

- Builders say MCP server-level controls already solve this cleanly.
- Teams say they do not want a portable contract because policy is too local.
- No useful replies, issues, or trials after the post window.
- The strongest feedback is that Tool Search already removes the pain.

## Decision rule

If positive signals appear, harden the primitive: TTL, signing, audit, identity
binding, and OpenAPI ingestion.

If warning signals dominate, revise the message before building more.

If kill signals dominate, keep ACP as a thin dev tool and stop the control-plane
push.