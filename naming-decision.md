# Naming Decision — ACP protocol

Date: 2026-06-16. Status: DECIDED by PR #20. This doc ratifies it.
Author: ralph loop, Lane A1.

## What was decided

PR #20 ("refactor: reframe ACP as Agent Contract Protocol") already made the
call and shipped the rename. The name is **Agent Contract Protocol**, acronym
**ACP**. The "C" moves from Context to Contract. PR #20 renamed the repo slug
and Go module to `agent-contract-protocol` and rewrote README, SPEC,
positioning, landing, paper, and validation docs across ~70 files.

This doc does not reopen that. It records the collision evidence so the risk is
known and the one mitigation is enforced.

## The bet, stated plainly

Keep ACP. Do not fight for the bare acronym. Own the full phrase "Agent Contract
Protocol" and the category: a signed, identity-bound, ordered, auditable
execution contract for agents. MCP stays the discovery supply chain. That
framing is distinct from every other "ACP" below, so the collision is on the
acronym only, not the product.

## Collision evidence (verified 2026-06-16)

The acronym collides at least three ways. Two confirmed by direct fetch:

- agent-context-protocol — live GitHub org, self-describes as "The first
  protocol for multi-agent communication and coordination." MIT, active 2025.
  https://github.com/agent-context-protocol
- Agent Client Protocol (Zed) — brands itself "ACP", standardizes
  editor<->coding-agent comms, JSON-RPC over stdio, reuses MCP JSON. Shares our
  transport and MCP-reuse story. https://agentclientprotocol.com
- Agent Communication Protocol (IBM / Linux Foundation, BeeAI lineage) — widely
  cited "ACP". Verify the URL before citing it in the paper.

## The mitigation (enforce this, it is the whole risk control)

Because Zed's "ACP" shares our transport story, the bare acronym is contested.
The reframe survives that only if we are disciplined:

1. Always use the full phrase "Agent Contract Protocol" on first mention in the
   README h1, the arXiv paper title, and every launch post. Never lead with bare
   "ACP".
2. Paper title must be the full phrase, e.g. "Agent Contract Protocol: a
   governed execution contract for autonomous agents." Do not title it "ACP".
3. In the README, add one line that says what it is NOT: not Zed's Agent Client
   Protocol, not IBM's Agent Communication Protocol. Disambiguate on purpose.
4. SEO/discovery: register the GitHub topic and any social handle as
   "agent-contract-protocol", the unique string, not "acp".

## Status of related lanes (reconciled to reality)

- Lane A (naming): DONE by PR #20. Human gate is now "review + merge PR #20",
  not "pick a name."
- Lane C1 (bridge): DONE. PR #15 is open and mergeable.
- Lane C3 (PyPI): PR #20 adds pyproject.toml to all four adapters. Packaging
  metadata exists; publish is still the human-gated step.

## Human gate (was A2, now restated)

1. Review and merge PR #20 to land the reframe on main.
2. Then merge PR #15 (bridge) on top, or rebase it on the reframed main.
3. Confirm the paper title uses the full phrase before arXiv submit.
