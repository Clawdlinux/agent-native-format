# Examples

## VS Code MCP bridge

Use `acp-bridge` when you already have MCP servers in VS Code and want one
ACP-managed tool surface.

Install the bridge:

```bash
go install github.com/Clawdlinux/ninevigil-acp/cmd/acp-bridge@latest
```

Add one server to VS Code `mcp.json`:

```json
{
	"servers": {
		"acp-bridge": {
			"type": "stdio",
			"command": "acp-bridge",
			"args": ["--import-vscode"]
		}
	}
}
```

The bridge reads your VS Code user `mcp.json`, skips its own `acp-bridge`
entry, starts each downstream stdio MCP server, compacts schemas, and forwards
real `tools/call` requests back to the original server.

If you prefer an explicit bridge config, start from [bridge.json](bridge.json):

```bash
acp-bridge --config ./examples/bridge.json
```

Current limits:

- Servers that use VS Code `${input:...}` prompts are skipped. Set secrets in
	environment variables instead.
- VS Code HTTP MCP entries are skipped for now. Use stdio MCP servers for the
	bridge path.
- The bridge keeps downstream auth in each child process environment.
