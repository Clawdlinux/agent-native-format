# Task Queue — NineVigil ACP

> Copilot reads this file at the start of every session.
> Work top-down. Mark tasks `[x]` when all 4 agents sign off.

---

## P0 — v0.1.0 (ship ACL as a usable product)

### T01: Fix auto-formatter `package` duplication bug ✅
- [x] **Problem:** VS Code's Go formatter prepends a bare `package X`
  line before the license comment, causing `package X` to appear twice.
  This breaks compilation and has required manual fixes 5+ times.
- [x] **Fix:** Move all package-level doc comments to separate `doc.go`
  files in every package that doesn't have one yet. Verify no file has
  two `package` declarations.
- [x] **Packages to check:** `pkg/aclhttp`, `pkg/aclpg`, `cmd/acl`,
  `pkg/acl`, `pkg/acp`
- [x] **Gate:** `go build ./...` passes after saving each file in VS Code

### T02: Wire `acl` CLI into Makefile + CI ✅
- [x] Add `build-acl` target to Makefile that builds `bin/acl`
- [x] Add `install-acl` target that copies to `$GOPATH/bin`
- [x] Add `acl` binary to `.github/workflows/verify.yml` test matrix
- [x] Verify: `make build-acl && bin/acl version` prints the git tag

### T03: Publish Python `acp-acl` to TestPyPI
- [ ] Add `python/Makefile` with `build`, `test`, `publish-test` targets
- [ ] Add `python/.github/workflows/publish.yml` (on tag `acl-py-v*`)
- [ ] Test: `pip install -i https://test.pypi.org/simple/ acp-acl`
- [ ] Verify: `python -c "import acp_acl; print(acp_acl.__version__)"`

### T04: Add `acl encode pg` from DDL string input
- [ ] Currently `aclpg.Encode` takes a typed `Schema` struct. Add a
  `ParseDDL(sql string) (Schema, error)` function that parses a
  minimal subset of `CREATE TABLE` / `ALTER TABLE ... ADD CONSTRAINT`
  SQL into the typed struct.
- [ ] Scope: Postgres DDL only, not MySQL/SQLite. Use `regexp`, not a
  full SQL parser.
- [ ] Wire into `cmd/acl/main.go` so `acl encode pg dump.sql` works.
- [ ] Test: golden-file test with a 5-table DDL fixture.
- [ ] Gate: `make verify` passes, `acl encode pg testdata/five.sql`
  produces valid ACL that round-trips through `acl.Decode`.

### T05: Quickstart documentation ✅
- [x] Create `docs/quickstart.md` — install → translate → measure
  in under 5 minutes.
- [x] Three paths: Go library, CLI, Python decoder.
- [x] Each path ends with a `acl tokens` call showing token savings.
- [x] Link from README.md.

### T06: Three runnable examples ✅
- [x] `examples/01-k8s-namespace/` — shell script that runs the K8s
  translator on the bundled fixture and prints token savings.
- [x] `examples/02-openapi-petstore/` — downloads the Petstore spec,
  runs `acl encode openapi`, prints before/after token counts.
- [x] `examples/03-python-decoder/` — Python script that decodes an
  ACL file and prints the section names + row counts.
- [x] Each example has a `README.md` and a single runnable script.

### T07: README rewrite — ACL-first framing
- [ ] Lead with ACL (the representation), not ACP (the protocol).
- [ ] Hero section: "Agent Context Language — a compact format for
  LLM agent consumption" + the 132×/68×/3.5× headline numbers.
- [ ] Architecture diagram showing: Source → Translator → ACL → Agent.
- [ ] Quick-start block: `acl encode openapi petstore.json | acl tokens -`
- [ ] Move ACP server docs to a "Protocol" subsection.
- [ ] Keep the existing benchmark table but update framing.

### T08: `goreleaser` config for cross-platform binaries
- [ ] Create `.goreleaser.yml` for `cmd/acl` binary.
- [ ] Targets: darwin/arm64, darwin/amd64, linux/arm64, linux/amd64.
- [ ] GHA workflow: `.github/workflows/release.yml` on tag `acl-v*`.
- [ ] Checksum file + cosign signing.
- [ ] Test: `goreleaser build --snapshot --single-target` produces a
  working binary.

### T09: Run agent-accuracy benchmark at n=30
- [ ] Clear cache: `rm -rf benchmark/agent_accuracy/.cache/responses`
- [ ] Run: `python -m benchmark.agent_accuracy.harness --models
  claude-haiku-4-5-20251001 --trials 30 --max-tokens 80 --max-usd 5.0`
- [ ] Commit `summary.md` to `benchmark/agent_accuracy/results/`
- [ ] Update paper + README headline numbers if they changed.

### T10: Landing page for clawdlinux.org
- [ ] Update `landing/index.html` — NineVigil-first framing (matches
  YC application), with ACL demo above the fold.
- [ ] Sections: problem, operator, ACL translators, benchmark, get started.
- [ ] No signup form, no pricing, no features that don't exist yet.
- [ ] Deploy to GitHub Pages or Vercel.

---

## P1 — v0.2.0 (more translators, broader adoption)

- [ ] T11: Web page translator (`pkg/aclweb`) — HTML/DOM → ACL
- [ ] T12: Slack API translator (`pkg/aclslack`)
- [ ] T13: MCP tools/list translator (consume MCP, emit ACL)
- [ ] T14: `acl diff` command — compare two ACL documents
- [ ] T15: `acl serve` — HTTP endpoint that translates on demand
- [ ] T16: Homebrew tap: `brew install clawdlinux/acl/acl`
- [ ] T17: VS Code extension: ACL syntax highlighting + token count

---

## P2 — v0.3.0 (feedback loop, intelligence)

- [ ] T18: Feedback endpoint — agents report which fields they used
- [ ] T19: Adaptive translator — tune field selection based on feedback
- [ ] T20: Multi-translator merge — combine K8s + OpenAPI into one doc
- [ ] T21: Streaming ACL — emit sections as they're ready (SSE)
- [ ] T22: ACL → structured JSON reverse translator (for non-LLM consumers)
