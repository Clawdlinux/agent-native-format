# ACP v0.1 Public Launch Checklist

> Single source of truth for the go/no-go on flipping
> `Clawdlinux/ninevigil-acp` from private to public, posting to
> arxiv/HN, and announcing the spec.
> Status: 2026-05-03 (pre-launch).

## Owner

Shreyansh Sancheti (`@shreyanshjain7174`)

## Hard prerequisites (block launch)

- [ ] All Week 1-Week 4 phases merged on `main` (Done as of `b5266f6`).
- [ ] Phase 1 framing rewrite landed (Done: `c29eb0e`).
- [ ] arxiv paper builds in CI as a PDF artifact (Done; `paper/pdf` job
  is green on `main`).
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
- [ ] Repo description on GitHub is set; topics include
  `agents`, `mcp`, `protocol`, `golang`.

## Soft prerequisites (strongly recommended)

- [ ] One independent reviewer reads `SPEC.md` end-to-end and at
  least one of `paper/acp.md` and `docs/positioning.md`. Sign-off
  comment in the launch PR.
- [ ] `landing/index.html` deployed somewhere reachable (GitHub
  Pages, Netlify, Cloudflare). At minimum a `https://ninevigil.io/acp`
  redirect to the GitHub repo is acceptable.
- [ ] HN post draft (`blog/launch-post.md`) reviewed; submit on a
  Tuesday or Wednesday morning UTC for best reach.
- [ ] Twitter/Bluesky/LinkedIn announcements drafted referencing the
  arxiv ID.

## Launch-day actions (in order)

1. **Squash-merge** the public-launch PR into `main`.
2. Create release tag `v0.1.0-spec` annotated with the SPEC.md
   shasum and the benchmark headline numbers. Push.
3. Submit `paper/acp.pdf` to arxiv (cs.SE primary, cs.LG cross-list).
   Note the assigned arxiv ID.
4. Add the arxiv ID to:
   - `paper/acp.md` and `paper/acp.tex` footers
   - `README.md` (top of TL;DR)
   - `docs/positioning.md` (References section)
   - `landing/index.html` and `blog/launch-post.md`
5. **Flip the GitHub repo to public** (`gh repo edit Clawdlinux/ninevigil-acp --visibility public`).
6. Post `blog/launch-post.md` to:
   - Hacker News (`Show HN: ACP, the execution-context layer above MCP`)
   - r/LocalLLaMA, r/MachineLearning (with arxiv link)
   - LinkedIn (long form)
   - Twitter/Bluesky thread (5-7 tweets, each grounded in one S1-S5
     measured number)
7. Notify the MCP authors (Anthropic) **before** HN goes live.
   Subject: "ACP: a layer that consumes MCP tools/list — wanted to
   share before going public." Tone: collaborative, not adversarial
   (the framing rewrite was for exactly this).

## Rollback plan (if launch goes poorly)

- The repo can be flipped private again (`gh repo edit ... --visibility
  private`). All forks stay public; no way to recall those.
- If a critical security issue surfaces post-launch, file a P0,
  publish a blog post acknowledging it within 24h, and ship the fix.
- Pull arxiv PDF back? Not really possible. arxiv is forever. This
  is why we run `paper/numbers-match` and `paper/pdf` CI jobs on
  every PR and require a clean baseline before submission.

## Go/no-go review

| Reviewer | Date | Decision | Notes |
|---|---|---|---|
| Shreyansh Sancheti (owner) | TBD | TBD | |
| (optional) external reviewer | TBD | TBD | |

## Post-launch

- [ ] Open follow-up PR in `Clawdlinux/agentic-operator-core` adding
  the `acp.clawdlinux.org/*` Service annotations (per
  `docs/operator-integration.md`).
- [ ] Open follow-up PR in this repo adding any feedback/issues from
  the HN comments worth addressing in v0.1.1.
- [ ] Schedule a v0.2 spec discussion 30 days post-launch with any
  external collaborators who reach out.
