# Naming Decision: Agent Native Format

Date: 2026-07-13. Status: DECIDED. Supersedes the 2026-06-16 record.

## What changed

The repo is now Agent Native Format (ANF). The GitHub slug, Go module path,
README, and About all lead with ANF. The prior record from 2026-06-16 ratified
"Agent Contract Protocol / ACP" and is now superseded.

## Why

ANF is the durable primitive. It is a token-minimal view format that translates
live system state into far fewer tokens for agent consumption. It is specified
in `FORMAT.md`, implemented in `pkg/anf`, and shipped with a Kubernetes
translator.

The execution-contract runtime, formerly ACP, stays in the repo as a secondary
component. It is the governed execution layer that consumes tool discovery and
enforces policy at the boundary. Its code identifiers (`pkg/acp`, `acp-server`,
`ACP_AUTH_TOKEN`) are unchanged for now. Renaming them is a separate breaking
follow-up.

## Acronym collisions (still enforced)

ANF as a data format sits apart from the "ACP" cluster:

- Zed Agent Client Protocol is a transport. ANF is not a transport.
- IBM Agent Communication Protocol is multi-agent messaging. ANF is a view format.
- agent-context-protocol is multi-agent coordination. ANF is a serialization.

Rule: on first mention in the README h1, the paper title, and every launch post,
use the full phrase "Agent Native Format". Disambiguate from the ACP cluster on
purpose.

## Follow-ups (not in the reposition PR)

1. Decide whether to keep, rename, or deprecate the execution runtime and its
   `acp-*` code identifiers.
2. Retitle the paper (`paper/acp.*`) and rebuild the PDF as `anf.pdf` before the
   next arXiv update. Needs a chosen paper title.
3. Cut a fresh ANF-branded release once 1 and 2 land.
