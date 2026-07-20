---
name: agent-native-format
description: Use when tool output or system state is large and eating context. Turns verbose JSON (Kubernetes state, API responses, dashboards, config dumps) into Agent Native Format (ANF), a line-oriented, token-minimal representation, via the anf-mcp server. Same facts, fewer tokens. Use before pasting big JSON into context, or when an agent's context window is filling with structured state.
---

# Agent Native Format (ANF)

ANF is a line-oriented, token-minimal way to represent structured state for an
LLM. JSON, YAML, and HTML are built for humans and machines. Agents pay for the
braces, quotes, and commas in tokens. ANF keeps the same facts with far less
syntax.

This skill uses the `anf-mcp` MCP server. Install it once:

```sh
go install github.com/Clawdlinux/agent-native-format/cmd/anf-mcp@latest
```

Then register it in your MCP client (see cmd/anf-mcp/README.md for configs).

## When to use it

- A tool returned a large JSON blob and you only need the facts, not the syntax.
- The context window is filling with structured state.
- You are about to paste a Kubernetes namespace, an API response, or a config
  dump into context.

## How to use it

- `anf_encode` with `{"data": <any JSON>}` returns ANF for arbitrary JSON. It is
  lossless and deterministic. Pass an optional `scope` to label the root.
- `anf_encode_kubernetes` with `{"view": <namespace view>}` returns ANF for a
  Kubernetes namespace and surfaces health first.
- Read the `anf://spec/format` resource to learn the exact format.

## What it will not do

It will not invent status, health, or alerts for generic JSON. It only removes
syntax. The big token savings come from scoping (only encode what the task
needs) and from domain translators, not from the notation itself. Measure on
your own data before trusting a specific reduction number.
