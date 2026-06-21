# Draft: Question-led ACP validation post

Do not post this until Phase 0 research is checked and the repo rename is live.

## X / Twitter

1/ Honest question for people building agents that touch real systems:

How are you handling execution governance?

2/ Tool discovery is getting better. MCP exists. Claude Code has Tool Search.
Semantic retrieval is improving. That helps agents find tools.

But I still do not see a common answer for execution trust.

3/ When an autonomous agent calls a production system, how do you bind identity,
enforce egress, require approvals, revoke a plan, and audit what happened?

Are you using MCP server controls, API gateways, OPA, custom proxies, audit logs,
or just keeping agents away from prod?

4/ I am testing a small open primitive around this: an execution contract over
existing MCP/OpenAPI-style tools. Intent in. Signed, identity-bound, ordered,
auditable execution out.

If this is not your pain, I want to know.

Repo: https://github.com/Clawdlinux/agent-contract-protocol

## LinkedIn

If you are building agents that touch real systems, how are you handling
execution governance?

Tool discovery is getting better. MCP exists. Claude Code has Tool Search.
Semantic retrieval is improving. The agent can increasingly find the right tool.

The part I still do not see a common answer for is execution trust.

When an autonomous agent calls a production system, how do you bind identity,
enforce egress, require approvals, revoke a plan, and audit what happened?

Are teams using MCP server controls, API gateways, OPA, custom proxies, audit
logs, or just keeping agents away from production?

I am testing a small open primitive around this: an execution contract over
existing MCP/OpenAPI-style tools. Intent in. Signed, identity-bound, ordered,
auditable execution out.

Genuinely looking for counterexamples. If this is already solved in your stack,
I want to learn how.

Repo: https://github.com/Clawdlinux/agent-contract-protocol

## Hacker News / Reddit

Title: Ask HN: How are you handling execution governance for autonomous agents?

I am trying to understand how people are handling a problem that shows up once
agents move past demos.

MCP handles tool discovery. Claude Code now has Tool Search. Semantic retrieval
papers and projects are making large tool catalogs cheaper to search. So the old
"MCP burns too many tokens" pitch feels less interesting by itself.

The problem I still do not see a common answer for is execution governance.

If an autonomous agent touches a real system, how do you handle:

- Identity. Who is it acting as?
- Policy. What can it call, and under what approval rules?
- Revocation. Can you stop a plan mid-flight?
- Audit. Can you prove what happened afterward?
- Blast radius. Can the agent leave the allowed boundary?

Are you solving this with MCP server-level controls, API gateways, OPA, custom
proxies, audit logs, or by keeping agents away from production?

I am testing a small open execution-contract primitive in ACP. It sits over
existing tool sources and emits a signed, identity-bound, ordered, auditable unit
of execution. I am not sure yet if this is the right primitive, so I would rather
ask before building too much.

Repo: https://github.com/Clawdlinux/agent-contract-protocol