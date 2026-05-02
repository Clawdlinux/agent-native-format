# Changelog

All notable changes to this project. Format: [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).
This project follows [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- `LAUNCH_CHECKLIST.md` — go/no-go checklist for the public v0.1 spec launch.
- `landing/index.html` — single-page summary of the protocol, headline
  benchmark numbers, and code links.
- `blog/launch-post.md` — HN/Reddit/Twitter draft for the public launch.

## [0.1.0-spec] — 2026-05-03 (pre-launch)

This is the v0.1 of the **Agent Context Protocol** specification and the
reference implementation. The repository is still private at this tag; the
public release happens in `v0.1.0-public` once the launch checklist is signed
off.

### Added

#### Protocol

- `SPEC.md` v0.1.1 — defines `POST /v1/context`, `POST /v1/feedback`,
  the Execution Manifest format, the schema mini-language, the
  auth-injection contract, and the boundaries section.
- `SPEC.md` §10 — relationship to MCP (ACP sits **on top of** MCP).
- `docs/positioning.md` — strategic framing.
- `docs/operator-integration.md` — Kubernetes deployment story.
- `docs/protocol.md`, `docs/architecture.md`,
  `docs/benchmark-methodology.md`, `docs/paper-plan.md`,
  `docs/pitch-deck-data.md`, `docs/references.md`,
  `docs/LICENSING.md`, `docs/decision-log.md`.

#### Reference server (BSL 1.1)

- `cmd/acp-server` — Go HTTP server with `--addr`, `--auth-token`,
  `--enable-proxy`, `--auto-approve`, `--resolver` flags.
- `internal/server` — handlers for `/v1/context`, `/v1/feedback`,
  `/healthz`. Bearer auth via `crypto/subtle.ConstantTimeCompare`.
  Graceful shutdown.
- `internal/registry` — goroutine-safe in-memory tool registry with
  capability-indexed Lookup and validating Register.
- `internal/builder` — manifest builder with deterministic ordering,
  schema compaction, egress aggregation, approval-gate collection.
- `internal/proxy` — auth-injection reverse proxy at
  `/v1/exec/{manifest_id}/{action_id}`. Strips agent
  `Authorization` + `Proxy-Authorization`. Enforces `egress`
  allow-list. Approval-gates fail closed by default.
- `internal/resolver` — `KeywordResolver` (default) and
  `EmbeddingResolver` (opt-in, hash TF-IDF, dependency-free).

#### Source adapters (BSL 1.1)

- `internal/sources/mcp` — ingests MCP `tools/list` payloads and
  registers each as an ACP `Tool` with compacted schemas.
- `internal/sources/k8s` — registers annotated Kubernetes Services as
  ACP tools via the `acp.clawdlinux.org/*` annotation contract.

#### SDKs (Apache 2.0)

- `pkg/manifest` — Go wire types.
- `pkg/acp` — Go client SDK with `WithToken`/`WithDoer` options and
  typed `APIError`.
- `adapters/python/acp_common` — shared core (Manifest, ACPClient,
  schema translation, topological ordering).
- `adapters/python/acp_openai` — OpenAI function-calling adapter.
- `adapters/python/acp_langgraph` — LangGraph StateGraph adapter.
- `adapters/python/acp_crewai` — CrewAI BaseTool adapter.

#### Benchmark + paper

- `benchmark/baseline/mcp_client.py` — MCP 2024-11 spec-faithful
  payload reproducer.
- `benchmark/harness.py` — runs N runs against a live ACP server,
  counts tokens with `tiktoken/cl100k_base`.
- `benchmark/charts.py` — matplotlib renderer (3 charts × svg+png).
- `benchmark/report.py` — markdown report renderer.
- `paper/acp.md` + `paper/acp.tex` + `paper/references.bib` +
  `paper/figures/` — arxiv preprint, ~6 pages, single author.
- `paper/Makefile` — local PDF build (TeX Live required).

#### CI

- `.github/workflows/verify.yml` — `go vet`, `go test -race`,
  staticcheck, govulncheck, fuzz smoke (10s), Docker build, Python
  pytest with `--cov-fail-under=85`, docs check.
- `.github/workflows/paper.yml` — chart renderability, paper number
  consistency check, PDF build inside `texlive/texlive` container.

### Headline benchmark (50 runs/scenario, `tiktoken/cl100k_base`)

| Scenario | ACP / MCP tokens | ACP / MCP RT | Reduction |
|---|---|---|---|
| S1 Simple DB query | 111 / 373 | 1 / 3 | **70.2%** |
| S2 Multi-tool workflow | 295 / 837 | 1 / 5 | **64.7%** |
| S3 Complex DAG | 306 / 1,257 | 1 / 7 | **75.6%** |
| **S4 Scale (50 tools, 2 relevant)** | **241 / 9,223** | **1 / 21** | **97.4%** |
| S5 Auth-heavy | 359 / 1,431 | 1 / 7 | **74.9%** |

Mean reduction: **76.6%**.

### License split

- `SPEC.md`, `docs/protocol.md`, `docs/positioning.md`: CC BY 4.0.
- `pkg/`, `adapters/`: Apache 2.0.
- `cmd/`, `internal/`: BSL 1.1, converts to Apache 2.0 on 2029-05-02.

[Unreleased]: https://github.com/Clawdlinux/ninevigil-acp/compare/v0.1.0-spec...HEAD
[0.1.0-spec]: https://github.com/Clawdlinux/ninevigil-acp/releases/tag/v0.1.0-spec
