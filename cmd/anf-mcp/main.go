// SPDX-License-Identifier: Apache-2.0

// Command anf-mcp runs a stateless MCP stdio server that exposes the Agent
// Native Format (ANF) encoder as tools. Point any MCP client at it to turn
// verbose system state into token-minimal ANF for context engineering.
//
// Tools:
//
//	anf_encode              generic, lossless JSON -> ANF
//	anf_encode_kubernetes   Kubernetes namespace view -> ANF (domain translator)
//
// Resource:
//
//	anf://spec/format       the ANF format specification (FORMAT.md)
//
// Usage in an MCP client config (Claude Desktop, Cursor, VS Code, Codex):
//
//	{
//	  "mcpServers": {
//	    "anf": {
//	      "command": "anf-mcp"
//	    }
//	  }
//	}
//
// VS Code uses the "servers" key with an explicit stdio type:
//
//	{
//	  "servers": {
//	    "anf": { "type": "stdio", "command": "anf-mcp" }
//	  }
//	}
//
// Install with: go install github.com/Clawdlinux/agent-native-format/cmd/anf-mcp@latest
package main

import (
	"context"
	_ "embed"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/Clawdlinux/agent-native-format/pkg/anfmcp"
	"github.com/Clawdlinux/agent-native-format/pkg/anftools"
)

// version is set at build time via -ldflags "-X main.version=...".
var version = "0.1.0-dev"

// specMarkdown is the ANF specification, embedded so the binary is
// self-contained. A test verifies it matches the canonical FORMAT.md.
//
//go:embed FORMAT.md
var specMarkdown string

const instructions = "anf_encode turns any JSON value into Agent Native Format: a line-oriented, " +
	"token-minimal representation that keeps the same facts in far fewer tokens. Use it to " +
	"compress verbose tool output or system state before adding it to context. " +
	"anf_encode_kubernetes does the same for a Kubernetes namespace view and surfaces health first. " +
	"Read the anf://spec/format resource for the exact format."

// buildServer wires the ANF tools and spec resource onto a new server. It is
// separated from main so tests can exercise the same configuration.
func buildServer(logger *slog.Logger) (*anfmcp.Server, error) {
	s := anfmcp.NewServer("anf-mcp", version,
		anfmcp.WithLogger(logger),
		anfmcp.WithInstructions(instructions),
	)
	if err := anftools.Register(s); err != nil {
		return nil, err
	}
	err := s.RegisterResource(anfmcp.Resource{
		URI:         "anf://spec/format",
		Name:        "Agent Native Format specification",
		Description: "The ANF format definition (FORMAT.md).",
		MimeType:    "text/markdown",
		Read: func(context.Context) (string, error) {
			return specMarkdown, nil
		},
	})
	if err != nil {
		return nil, err
	}
	return s, nil
}

func main() {
	// Logs go to stderr so they never corrupt the JSON-RPC stream on stdout.
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	s, err := buildServer(logger)
	if err != nil {
		logger.Error("build server", "error", err)
		os.Exit(1)
	}

	if err := s.Serve(ctx, os.Stdin, os.Stdout); err != nil && ctx.Err() == nil {
		logger.Error("serve", "error", err)
		os.Exit(1)
	}
}
