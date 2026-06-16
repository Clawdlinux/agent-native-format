# Ralph Backlog — ninevigil-acp

Ordered task queue for the review-execution ralph loop. One task = one signed-off
commit on a feature branch. States: `[ ]` todo, `[~]` in progress, `[x]` done,
`[HUMAN]` needs the human gate (loop stops and flags).

Pre-flight gates before any new branch or issue (do not skip):
- `gh pr list --state open` before branching. Build on an existing branch if scope overlaps.
- `gh issue list --state open --search "<keywords>"` before filing any issue.
- Feature branch + PR, squash merge, delete branch. `git commit -s`. No coauthor. No emojis.

## Lane A — naming (top risk, blocks paper title + packaging names)
- [ ] A1 Acronym-collision matrix. Confirm collisions: agent-context-protocol org (GitHub), Agent Client Protocol (Zed), Agent Communication Protocol (IBM/LF). Score 3-4 candidates on free GitHub org/repo, free PyPI name, paper-title uniqueness, domain. -> naming-decision.md with ONE recommendation. skill: competitive-landscape.
- [HUMAN] A2 Approve the name.
- [ ] A3 Mechanical local rename: README title/body, paper acp.tex title+refs, package metadata. LOCAL only. Branch feat/agent-contract-protocol-reframe already exists — build on it.
- [HUMAN] A4 GitHub repo slug rename (agent-contract-protocol). Go install path note.

## Lane C — adoption + packaging
- [HUMAN] C1 Push feat/acp-bridge-plug-play -> PR -> code-reviewer gate -> squash-merge. Bridge is the most adoption-relevant feature and is currently unpushed.
- [ ] C2 Ship #12 (--mcp-source flag, no-Go onboarding) + #13 (stdio transport for Claude Desktop/Cursor). One PR each, tests. skills: golang-pro, test-automator.
- [ ] C3 PyPI packaging for the 4 Python adapters: pyproject metadata, build, twine check, stage dist/ to _staging/pypi/. Uses A2 name. skill: python-packaging.
- [ ] C4 README "try it in 60s" using stdio + --mcp-source. Reconcile name per A3.

## Lane D — distribution debt (human-gated publish; subagent preps)
- [ ] D1 Stage arXiv package (paper.pdf, source, abstract, category) to _staging/arxiv/. skill: docs-architect.
- [HUMAN] D2 arXiv submit (account + endorsement).
- [HUMAN] D3 twine upload from _staging/pypi/ (token).
- [HUMAN] D4 Create ninevigil Docker org; push off personal goodra007 mirror. Subagent writes push-images.sh.

## Lane E — validation outreach (review item #2; human runs conversations)
- [ ] E1 10-target list (platform/security eng at regulated shops) + cold-message drafts + 5-min attestation-demo runbook from booth script. Map to issue #19. skills: content-marketer + tutorial-engineer.
- [ ] E2 Draft never-published X + Reddit + HN launch posts (no emojis, plain).
- [HUMAN] E3 Run 10 conversations; log into issue #19.

## Notes
- Bridge work lives on feat/acp-bridge-plug-play (committed, not on main).
- Naming reframe branch: feat/agent-contract-protocol-reframe.
