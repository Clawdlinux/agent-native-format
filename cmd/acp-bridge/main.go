/*
Copyright 2026 Clawdlinux / NineVigil.

Licensed under the Business Source License 1.1.
See LICENSE in the repository root.
*/

// Command acp-bridge runs an MCP stdio server that wraps multiple downstream
// MCP servers behind ACP's deferred-intent resolver with schema compaction.
//
// Usage in VS Code mcp.json:
//
//	{
//	  "servers": {
//	    "acp-bridge": {
//	      "type": "stdio",
//	      "command": "acp-bridge",
//	      "args": ["--config", "/path/to/bridge.json"]
//	    }
//	  }
//	}
//
// The bridge reads downstream MCP server configurations from a JSON config
// file, imports and compacts their tools, then serves a dynamic tools/list
// via JSON-RPC over stdin/stdout.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"sort"
	"time"

	"github.com/Clawdlinux/ninevigil-acp/internal/bridge"
	"github.com/Clawdlinux/ninevigil-acp/internal/registry"
	"github.com/Clawdlinux/ninevigil-acp/internal/resolver"
	mcpsource "github.com/Clawdlinux/ninevigil-acp/internal/sources/mcp"
)

var version = "0.2.0-dev"

// BridgeConfig is the JSON config file format for the bridge.
type BridgeConfig struct {
	// Sources lists downstream MCP servers to import tools from.
	Sources []SourceConfig `json:"sources"`

	// NarrowThreshold is the number of tool-call observations before
	// narrowing. Default: 3.
	NarrowThreshold int `json:"narrow_threshold,omitempty"`

	// WindowSize is the sliding window of recent observations. Default: 10.
	WindowSize int `json:"window_size,omitempty"`
}

// SourceConfig describes one downstream MCP server.
type SourceConfig struct {
	// Name is a stable label for this source (e.g. "github", "flyctl").
	Name string `json:"name"`

	// URL is the HTTP base URL where tools/list is served.
	URL string `json:"url"`

	// Auth is sent as the Authorization header. Use env var references.
	Auth string `json:"auth,omitempty"`

	// Capabilities are extra capability tags applied to all tools from
	// this source.
	Capabilities []string `json:"capabilities,omitempty"`
}

func main() {
	configPath := flag.String("config", "", "path to bridge config JSON file")
	versionFlag := flag.Bool("version", false, "print version and exit")
	flag.Parse()

	if *versionFlag {
		fmt.Println(version)
		os.Exit(0)
	}

	// Logger writes to stderr so stdout stays clean for JSON-RPC.
	logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))

	var cfg BridgeConfig
	if *configPath != "" {
		data, err := os.ReadFile(*configPath)
		if err != nil {
			logger.Error("read config", "error", err)
			os.Exit(1)
		}
		if err := json.Unmarshal(data, &cfg); err != nil {
			logger.Error("parse config", "error", err)
			os.Exit(1)
		}
	}

	if len(cfg.Sources) == 0 {
		logger.Info("no sources configured; bridge will serve an empty tool surface")
	}

	// Import tools from all downstream MCP servers.
	reg := registry.NewMemoryRegistry()
	httpClient := &http.Client{Timeout: 30 * time.Second}
	importer := mcpsource.NewImporter(reg, httpClient)

	for _, src := range cfg.Sources {
		auth := expandEnv(src.Auth)
		n, err := importer.ImportSource(mcpsource.Source{
			Name:              src.Name,
			BaseURL:           src.URL,
			Auth:              auth,
			ExtraCapabilities: src.Capabilities,
		})
		if err != nil {
			logger.Error("import source", "source", src.Name, "error", err)
			continue
		}
		logger.Info("imported source", "source", src.Name, "tools", n)
	}

	// Collect all capability tags from imported tools.
	allTools := reg.All()
	capSet := make(map[string]struct{})
	for _, t := range allTools {
		for _, c := range t.Capabilities {
			capSet[c] = struct{}{}
		}
	}
	allCaps := make([]string, 0, len(capSet))
	for c := range capSet {
		allCaps = append(allCaps, c)
	}
	sort.Strings(allCaps)

	// Build deferred resolver.
	dr := resolver.NewDeferredResolver(resolver.DeferredOptions{
		AllCapabilities: allCaps,
		NarrowThreshold: cfg.NarrowThreshold,
		WindowSize:      cfg.WindowSize,
	})

	// Build bridge and register downstream tool mappings.
	b := bridge.New(bridge.Config{
		Registry: reg,
		Resolver: dr,
		Logger:   logger,
		Out:      os.Stdout,
	})

	for _, t := range allTools {
		// Tool ID format: "sourceName.toolName"
		source := t.ID
		toolName := t.ID
		for i, c := range t.ID {
			if c == '.' {
				source = t.ID[:i]
				toolName = t.ID[i+1:]
				break
			}
		}
		b.RegisterTool(source, toolName, t)
	}

	logger.Info("acp-bridge starting",
		"version", version,
		"tools", len(allTools),
		"capabilities", len(allCaps),
	)

	// Serve JSON-RPC over stdin/stdout.
	if err := b.Serve(os.Stdin); err != nil {
		logger.Error("serve", "error", err)
		os.Exit(1)
	}
}

// expandEnv replaces ${VAR} and $VAR references in s with environment
// variable values. This lets bridge.json reference secrets without
// hardcoding them.
func expandEnv(s string) string {
	return os.ExpandEnv(s)
}
