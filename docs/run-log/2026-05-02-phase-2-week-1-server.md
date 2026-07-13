# 2026-05-02 — Phase 2 Week 1 ACP Server

## Objective

Implement the Week 1 deliverable: a working ACP server with three HTTP endpoints
backed by an in-memory registry, keyword resolver, deterministic manifest
builder, and bearer auth.

## Branch

`feat/week1-acp-server` (off `main`).

## Work completed

### Public types — `pkg/manifest` (Apache 2.0)

- Added `ContextRequest`, `Constraints`, `FeedbackEvent`, `ErrorResponse`.
- Added enums: `AuthMode`, `AuditLevel`, `OutputFormat`, `FeedbackOutcome`.
- Constraints is a pointer field so `omitempty` actually omits when unset.

### Registry — `internal/registry` (BSL 1.1)

- `Tool`, `Registry`, `MemoryRegistry` with `sync.RWMutex`.
- Validation: empty IDs and missing capability tags are rejected at
  `Register`.
- `Lookup` is case-insensitive and returns tools sorted by ID for deterministic
  manifest IDs.
- `Seed()` registers five demo tools covering scenarios S1-S3:
  `db.query`, `template.render`, `email.send`, `slack.send_message`,
  `audit.log_event`.

### Resolver — `internal/resolver` (BSL 1.1)

- `Resolver` interface + `KeywordResolver` impl.
- Tokenizes intent on Unicode word boundaries; merges agent capability hints.
- `DefaultKeywordTable()` exposes the seeded keyword index.
- Returns `ErrEmptyIntent` when neither hints nor matched keywords produce
  capabilities.

### Builder — `internal/manifest` (BSL 1.1)

- Consumer-defined `ToolSource` interface (decouples builder from registry
  implementation, idiomatic Go).
- Injected `IDSource` so tests use deterministic IDs and prod uses
  `server.CryptoIDSource`.
- Computes `depends_on` from each tool's `DependsOnCaps` against the
  capabilities present in this manifest, not from global registry state.
- Aggregates egress allow-lists (sorted, deduped) and `require_approval`
  action IDs.
- Compact schemas pass through verbatim from the registry.

### Server — `internal/server` (BSL 1.1)

- `POST /v1/context`, `POST /v1/feedback`, `GET /healthz`.
- Bearer-token middleware uses `crypto/subtle.ConstantTimeCompare`.
- `LoggingFeedbackSink` is a default `FeedbackSink` impl with `slog`.
- `CryptoIDSource` mints `m-<16 hex>` manifest IDs from `crypto/rand`.
- `ListenAndServe` performs graceful shutdown on context cancel.
- All collaborators (Resolver, Builder, FeedbackSink) are interfaces defined
  in this package — consumer-defined per Go convention.

### Entrypoint — `cmd/acp-server` (BSL 1.1)

- Flags: `--addr`, `--auth-token` (env fallback `ACP_AUTH_TOKEN`),
  `--feedback-endpoint`.
- Wires registry seed, builder, resolver, server, signal handler.
- Structured JSON logs via `slog`.

### Go SDK — `pkg/acp` (Apache 2.0)

- `Client` with `Option` functional config (`WithToken`, `WithDoer`).
- `Context`, `Feedback`, `Healthz` methods.
- `APIError` type + `IsAPIError` helper for typed error handling.
- `HTTPDoer` interface for transport injection (matches the pattern in
  `agentic-operator-private/pkg/mcp`).

### Mocks

- All consumer-defined interfaces (`ToolSource`, `Resolver`, `Builder`,
  `HTTPDoer`) have `//go:generate` directives that emit gomock files into the
  consumer test package.
- Mocks are committed (`mocks_test.go`, `mock_httpdoer_test.go`) so CI runs
  without requiring `mockgen` installation.

### Build / CI / deploy

- `Dockerfile` is a multi-stage build with a distroless `nonroot` runtime.
- `deploy/docker-compose.yaml` now builds the real server and accepts an
  `ACP_AUTH_TOKEN` env var.
- `Makefile` adds `vet`, `test -race`, `cover`, `build`, `generate`, `clean`.
- `.github/workflows/verify.yml` runs `go vet`, `go test -race`, `go build`,
  and the docs validator.

## Validation

```bash
$ go test -race -count=1 ./...
ok  github.com/Clawdlinux/agent-native-format/internal/manifest    0.49s
ok  github.com/Clawdlinux/agent-native-format/internal/registry    0.91s
ok  github.com/Clawdlinux/agent-native-format/internal/resolver    0.51s
ok  github.com/Clawdlinux/agent-native-format/internal/server      0.59s
ok  github.com/Clawdlinux/agent-native-format/pkg/acp              0.57s
ok  github.com/Clawdlinux/agent-native-format/pkg/manifest         0.43s
```

End-to-end smoke test against the running binary:

- `GET /healthz` -> `{"status":"ok"}`
- `POST /v1/context` without token -> 401 with JSON envelope.
- `POST /v1/context` with multi-step intent
  `"query customer data, render report, email the team"` returned a
  3-action manifest (`db.query` -> `template.render` -> `email.send`),
  correct depends_on chain, aggregated egress allow-list, and the email
  action listed under `require_approval`.
- `POST /v1/feedback` -> 202 Accepted; sink logged the event.

## Deviations from plan

- The `internal/manifest` directory holds a package named `builder` (Go
  package vs directory name mismatch). Imported with `builder
  "github.com/.../internal/manifest"` everywhere. Acceptable for now; can
  rename to `internal/builder` later if it causes friction.
- Embedding-based intent resolver is still Week 3 work; the keyword resolver
  is intentionally simple for Week 1 determinism.

## Next phase

Week 2:
1. Auth-injection proxy that fronts tool endpoints declared in manifests.
2. MCP baseline client (`benchmark/baseline/mcp_client.py`) using the
   official SDK.
3. Benchmark harness for S1 and S2 with `tiktoken`-based token counting.
4. First measured ACP-vs-MCP token comparison.
