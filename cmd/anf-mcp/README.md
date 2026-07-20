# anf-mcp

A stateless MCP stdio server that exposes the Agent Native Format (ANF) encoder
as tools. Point any MCP client at it to turn verbose JSON state into
token-minimal ANF for context engineering and token reduction.

Apache-2.0. No third-party dependencies.

## Install

```sh
go install github.com/Clawdlinux/agent-native-format/cmd/anf-mcp@latest
```

The binary is `anf-mcp` on your `PATH`. It speaks JSON-RPC 2.0 over stdin and
stdout. It holds no session state, so any request can be handled in isolation.

## Configure your client

Claude Desktop, Cursor, and Codex use the `mcpServers` key:

```json
{
  "mcpServers": {
    "anf": {
      "command": "anf-mcp"
    }
  }
}
```

VS Code uses the `servers` key with an explicit stdio type:

```json
{
  "servers": {
    "anf": { "type": "stdio", "command": "anf-mcp" }
  }
}
```

If `anf-mcp` is not on your `PATH`, use the absolute path from
`go env GOPATH`/bin.

## What it exposes

Tools:

- `anf_encode` takes any JSON value and returns ANF. The mapping is lossless and
  deterministic. Objects become entities, scalars become properties, arrays
  become child entities. Keys are sorted. Nothing is dropped and nothing is
  invented.
- `anf_encode_kubernetes` takes a Kubernetes namespace view and returns ANF via
  the domain translator, which surfaces health, alerts, and available actions
  first.

Resource:

- `anf://spec/format` returns the ANF specification (`FORMAT.md`), embedded in
  the binary so it works offline.

## Where the tokens go, honestly

`anf_encode` strips the syntactic overhead of JSON: braces, quotes, and commas.
Same facts, fewer tokens. It does not infer health, status, or alerts, so it
will not mislead an agent about state.

Most of the large reductions in our benchmarks come from two things, not from
clever notation:

1. Scoping. Feed the model only the state a task needs.
2. Domain translators. `anf_encode_kubernetes` knows what matters in a namespace
   and puts it first.

Recent research on compact agent formats (arXiv 2605.29676, 2606.01326) shows
notation tricks save less than people expect and can hurt model accuracy. So
this server does not sell a magic format. It scopes and compacts. Measure it on
your own data before you trust a number.

## Stateless by design

The server follows the stateless-first direction of MCP SEP-2575. It implements
`server/discover` and keeps `initialize` for clients that still send it.
