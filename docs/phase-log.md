# ACP Phase Log

This file is the repo-local source of truth for implementation phases. The
same milestones are mirrored to the `ACP-PoC/` folder inside the NineVigil
Obsidian vault.

## Phase 0 — Invention Capture

**Date:** 2026-05-02

**Inputs**
- Source spec: `ACP_PoC_Specification_CONFIDENTIAL.docx`
- Workspace: `/Users/sunny/clawdlinux/ninevigil-acp`
- GitHub repo: `Clawdlinux/ninevigil-acp` (private)
- GitHub milestone: `https://github.com/Clawdlinux/ninevigil-acp/milestone/1`

**Decisions**
- Repo name: `ninevigil-acp`
- Local path: `/Users/sunny/clawdlinux/ninevigil-acp`
- Phase 1 scope: repo scaffold + `SPEC.md` + documentation index, no runtime logic yet
- Documentation cadence: end-of-phase verifier/doc pass with Obsidian mirror

**Done**
- Private repo created under `clawdlinux`
- Repo cloned locally
- Directory structure created per the PoC spec
- `SPEC.md` v0.1 extracted and normalized from the source document
- Obsidian notes seeded under `ACP-PoC/`
- Verifier pass completed; findings fixed before first commit

## Phase 1 — Scaffold + Protocol Baseline

**Goal:** Create the private repo foundation for Week 1 ACP server work.

**Acceptance criteria**
- Directory tree matches the source PoC spec section 3.3
- License split is documented
- Architecture, benchmark, pitch, and references docs exist
- Obsidian vault contains ACP PoC index + running log
- Initial commit is pushed to private GitHub repo

**Status:** Complete

## Phase 2 — Week 1 ACP Server

**Goal:** Working ACP server with `POST /v1/context`, `POST /v1/feedback`, and
`GET /healthz` backed by an in-memory registry, keyword resolver, and
deterministic manifest builder.

**Acceptance criteria**
- `pkg/manifest` exposes the full ACP wire types (request, response, feedback).
- `internal/registry` is goroutine-safe, capability-indexed, and seeds five
  demo tools.
- `internal/resolver` deterministically maps intent + hints to capability tags.
- `internal/manifest` builder strips schemas, computes `depends_on`, and
  aggregates egress + approvals.
- `internal/server` enforces bearer auth (constant-time compare) and validates
  payloads.
- `cmd/acp-server` boots, handles SIGINT/SIGTERM, structured JSON logs.
- All packages have unit tests; consumer-defined interfaces are mocked with
  `go.uber.org/mock` (gomock).
- `pkg/acp` Go SDK uses dependency-injected `HTTPDoer` for tests.
- `go test -race ./...` is green.

**Status:** Complete
