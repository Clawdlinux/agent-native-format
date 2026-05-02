# ACP Protocol Notes

The normative protocol draft lives in [../SPEC.md](../SPEC.md).

This file tracks implementation notes, deltas, and open questions as the PoC
moves from spec to code.

## v0.1 surface

- `POST /v1/context`
- `POST /v1/feedback`
- Execution Manifest with `actions`, `boundaries`, `ttl`, and
  `feedback_endpoint`
- Compact schema mini-language (`string`, `int?`, `json`, `string[]`, etc.)

## Open questions

| Topic | Question | Initial direction |
|---|---|---|
| Manifest id | UUID v4, ULID, or short hash? | ULID for sortability |
| Capability matching | Keyword matcher vs embedding search | Keyword Week 1, embeddings Week 3 |
| Schema compression | Fixed mini-language vs generated type aliases | Start fixed; benchmark aliasing later |
| Approval gates | ACP server or proxy owns blocking? | Proxy enforces; server declares |
| Feedback | Per-action required or best-effort? | Best-effort in v0.1 |
