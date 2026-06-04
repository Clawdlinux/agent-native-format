# ACP Public Validation Checklist

> Single source of truth for validating the Agent Contract Protocol thesis in
> public. The goal is to learn how teams handle autonomous-agent execution
> governance before we harden the primitive.
> Status: 2026-06-04 (discovery).

## Owner

Shreyansh Sancheti (`@shreyanshjain7174`)

## Hard prerequisites (block validation)

- [ ] Phase 0 research log exists at `docs/discovery/2026-06-research-log.md`.
- [ ] Closest prior art section confirms no checked source owns the same signed
  execution-contract primitive.
- [ ] `docs/validation/signals.md` defines positive, warning, and kill signals.
- [ ] Repo identity is Agent Contract Protocol everywhere public-facing.
- [ ] Repo description on GitHub is set; topics include
  `agents`, `mcp`, `protocol`, `governance`, `audit`.
- [ ] All five S1-S5 benchmark scenarios reproducible from a clean
  clone with the published `harness.py` + committed
  `results/2026-05-02-week3-baseline.json`.
- [ ] No open `Severity: P0` GitHub issues.
- [ ] `govulncheck ./...` clean.
- [ ] License headers correct across `pkg/`, `internal/`, `cmd/`,
  `adapters/` per `docs/LICENSING.md`.
- [ ] `LICENSE` (BSL 1.1) at repo root unchanged.
- [ ] `SPEC.md` is CC BY 4.0 (per its header).
- [ ] No accidental secrets in git history (`gitleaks` or
  `trufflehog` clean).
## Soft prerequisites (strongly recommended)

- [ ] One independent reviewer reads `SPEC.md` end-to-end and at least one of
  `docs/positioning.md` and `docs/discovery/2026-06-research-log.md`.
- [ ] `landing/index.html` deployed somewhere reachable (GitHub
  Pages, Netlify, Cloudflare). At minimum a redirect to the GitHub repo is
  acceptable.
- [ ] Question-led post draft (`blog/launch-post.md`) reviewed.
- [ ] 3-5 trial issues drafted so people can test the sidecar, ingest an MCP
  server, inspect the contract, and report where the thesis is wrong.

## Validation-day actions (in order)

1. **Squash-merge** the validation PR into `main`.
2. Rename the GitHub repository to `agent-contract-protocol` if not already done.
3. **Flip the GitHub repo to public** if validation will use public issues.
4. Open the trial issues.
5. Post `blog/launch-post.md` as a question:
   - Hacker News: `Ask HN: How are you handling execution governance for autonomous agents?`
   - LinkedIn long form.
   - X / Bluesky thread.
6. Record the first 48 hours of replies in `docs/validation/signals.md` or a
   follow-up issue.

## Rollback plan (if validation goes poorly)

- The repo can be flipped private again (`gh repo edit ... --visibility
  private`). All forks stay public; no way to recall those.
- If a critical security issue surfaces post-validation, file a P0,
  publish a blog post acknowledging it within 24h, and ship the fix.
- If the trust pain does not surface, stop the control-plane push and keep ACP
  scoped as a thin dev tool.

## Go/no-go review

| Reviewer | Date | Decision | Notes |
|---|---|---|---|
| Shreyansh Sancheti (owner) | TBD | TBD | |
| (optional) external reviewer | TBD | TBD | |

## Post-validation

- [ ] Open follow-up PR in `Clawdlinux/agentic-operator-core` adding
  the `acp.clawdlinux.org/*` Service annotations (per
  `docs/operator-integration.md`).
- [ ] Open follow-up PR in this repo for any validated hardening work.
- [ ] Schedule a spec discussion 30 days post-validation with any external
  collaborators who reach out.
