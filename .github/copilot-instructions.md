## NineVigil ACP — Copilot Project Brain

> This file is read automatically by GitHub Copilot in every session.
> It defines the project context, quality gates, and the autonomous loop.

### What this project is

NineVigil ACP is the **agent-native context layer** for AI agents.
Two products ship from this repo:

1. **ACL (Agent Context Language)** — a line-oriented machine-native
   representation format. Translators convert human-format sources
   (Kubernetes, OpenAPI, Postgres) into compact agent-shaped documents.
   Measured: 132× on K8s, 68× on OpenAPI, 3.5× on Postgres.

2. **ACP (Agent Context Protocol)** — an HTTP server that resolves
   agent intents into Execution Manifests with pre-injected auth,
   dependency ordering, and security boundaries. Sits on top of MCP.

The K8s translator (`pkg/aclk8s`) lives in `agentic-operator-core`.
Everything else lives here.

### Stack constraints (hard rules)

- **Go 1.25+** for all server, CLI, and translator code
- **Python 3.12+** for benchmarks and the `acp-acl` PyPI decoder only
- **No CGO** in any shipped binary (static builds for distroless)
- **Apache 2.0** for all `pkg/` and `cmd/` code
- **CC BY 4.0** for the ACL spec (`docs/acl-spec.md`)
- **BSL 1.1** for `internal/` (server runtime)
- All commits signed: `git commit -s -m`
- Never echo, log, or commit API keys or secrets. Read from `os.Environ` only.

### Quality gates (must pass before any PR)

```bash
make verify     # fmt + vet + test + staticcheck + govulncheck + docs
```

Expanded:
1. `go vet ./...` — zero warnings
2. `bin/staticcheck ./...` — zero findings
3. `go test ./... -race -count=1` — all pass
4. `bin/govulncheck ./...` — zero vulns
5. Python: `cd python && pytest tests/` — all pass
6. Benchmark: `cd benchmark && pytest` — all pass

### ACL format invariants (never break these)

1. **Round-trip stability**: `Decode(Encode(d)) == d` for any well-formed `d`
2. **Determinism**: identical input → identical bytes
3. **No tabs**: tab characters are rejected by the decoder
4. **Two-space indent**: rows use exactly `  ` (two spaces)
5. **Directive order**: `@key value` lines before any section
6. **Section separator**: blank line between sections
7. **Actions section**: pipe-joined row ID (`scale|rollout|restart`)

### Translator contract

Every translator must:
1. Emit `@source <translator-id>` directive
2. Be deterministic (same input → same bytes)
3. Emit an `actions` section listing available affordances
4. Have a golden-output test + compression-ratio test
5. Round-trip through `acl.Decode` without error

### First-action checklist (run at session start)

1. Read `TASKS.md` — find the highest-priority unfinished task
2. Read `AGENTS.md` — identify which agent role applies
3. Run `make verify` to confirm the tree is clean
4. If dirty: fix before starting new work
5. Start work on the top P0 task

### File conventions

| Pattern | Purpose |
|---|---|
| `pkg/acl*` | Public Go SDK — translators, encoder, decoder |
| `internal/` | ACP server runtime (BSL 1.1) |
| `cmd/acl/` | CLI binary |
| `cmd/acp-server/` | Server binary |
| `benchmark/agent_accuracy/` | Agent benchmark harness (Python) |
| `python/acp_acl/` | PyPI decoder package |
| `docs/` | Specs, architecture, methodology |

### Multi-agent coordination

See `AGENTS.md` for the four specialist roles. The coordination
protocol:

1. **Planner** reads TASKS.md, selects task, writes plan to task file
2. **Coder** implements the plan, creates branch, writes code
3. **Tester** runs `make verify` + domain-specific tests
4. **Reviewer** checks against quality gates, blocks on security

Handoff format: each agent appends a `## <Role> — <timestamp>` block
to the task file with their output and status.
