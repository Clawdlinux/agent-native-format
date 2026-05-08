# Start Here — NineVigil ACP

> This file tells Copilot exactly what to do in the first 60 seconds
> of a session. Read this, then execute.

## Orientation (10 seconds)

**Repo:** `github.com/Clawdlinux/ninevigil-acp`
**Branch:** check with `git branch --show-current`
**Product:** ACL (Agent Context Language) + ACP (Agent Context Protocol)
**Stack:** Go 1.25+ / Python 3.12+ / No CGO
**Quality gate:** `make verify`

## Files to read (20 seconds)

1. `.github/copilot-instructions.md` — project brain, stack rules, invariants
2. `TASKS.md` — find the top unfinished P0 task
3. `AGENTS.md` — identify which role applies to the current work

## First action (30 seconds)

```bash
# Confirm the tree is clean
make verify
```

If `make verify` fails → fix before doing anything else.
If it passes → start the top P0 task from `TASKS.md`.

## How to launch the autonomous loop

Paste this into Copilot Chat:

```
Read COPILOT_START_HERE.md, then read TASKS.md and AGENTS.md.
Find the highest-priority unfinished task. Act as the Planner
first: analyze the codebase and write a plan. Then switch to
Coder and implement the plan. Then switch to Tester and run
make verify. Then switch to Reviewer and check the output.
Commit when all four agents approve.
```

## Quick reference

| Command | What it does |
|---|---|
| `make verify` | Full quality gate (vet + staticcheck + test + vuln) |
| `make test` | Go tests only (faster) |
| `make build` | Build `cmd/acl` and `cmd/acp-server` |
| `go test ./pkg/acl/... -v` | ACL encoder/decoder tests |
| `go test ./pkg/aclhttp/... -v` | OpenAPI translator tests |
| `go test ./pkg/aclpg/... -v` | Postgres translator tests |
| `cd python && pytest tests/ -v` | Python decoder tests |
| `bin/acl encode openapi pkg/aclhttp/testdata/petstore.json` | Quick ACL demo |
| `bin/acl tokens file.acl` | Token count for any file |

## Current status

Last committed work: `feat/acl-v0` branch with 59 files, 7,527 lines:
- ACL spec + reference encoder/decoder
- Three translators (K8s 132×, OpenAPI 68×, Postgres 3.5×)
- CLI (`cmd/acl`), Dockerfile, GHA workflow
- Agent-accuracy benchmark (540 trials, Claude Haiku 4.5)
- Python decoder package (`acp-acl`)

Next: work through `TASKS.md` P0 queue top-down.
