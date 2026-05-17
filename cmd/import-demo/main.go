/*
Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.

import-demo is a self-contained example that:

  1. Spins up an in-memory ACP registry.
  2. Imports tools from one or more upstream MCP servers via their HTTP
     `tools/list` endpoint (the 2024-11 spec envelope).
  3. Prints a side-by-side summary of what each source contributed.

Use this as the template for "I have an existing MCP server in production,
how do I plug it into ACP?" The same NewImporter+ImportSource calls live
inside `acp-server` once #13 ships the `--mcp-source` flag.

Usage:

	# point at one server
	go run ./cmd/import-demo -source name=files,url=http://127.0.0.1:9090

	# multiple servers, with auth
	go run ./cmd/import-demo \
	  -source name=files,url=http://127.0.0.1:9090 \
	  -source name=github,url=http://gh-mcp.local:9100,auth="bearer ghp_xxx"
*/
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/Clawdlinux/ninevigil-acp/internal/registry"
	mcpsource "github.com/Clawdlinux/ninevigil-acp/internal/sources/mcp"
)

// sourceFlag is a repeatable -source flag whose value is parsed as a comma-
// separated key=value list (e.g. "name=files,url=http://...,auth=bearer xxx").
type sourceFlag []mcpsource.Source

func (s *sourceFlag) String() string { return fmt.Sprintf("%d source(s)", len(*s)) }
func (s *sourceFlag) Set(raw string) error {
	src := mcpsource.Source{}
	for _, kv := range strings.Split(raw, ",") {
		k, v, ok := strings.Cut(strings.TrimSpace(kv), "=")
		if !ok {
			return fmt.Errorf("expected key=value, got %q", kv)
		}
		switch strings.ToLower(strings.TrimSpace(k)) {
		case "name":
			src.Name = strings.TrimSpace(v)
		case "url", "baseurl":
			src.BaseURL = strings.TrimSpace(v)
		case "auth":
			src.Auth = strings.TrimSpace(v)
		case "caps", "capabilities":
			for _, c := range strings.Split(v, "|") {
				if c = strings.TrimSpace(c); c != "" {
					src.ExtraCapabilities = append(src.ExtraCapabilities, c)
				}
			}
		default:
			return fmt.Errorf("unknown source key %q", k)
		}
	}
	if src.Name == "" || src.BaseURL == "" {
		return fmt.Errorf("source needs at least name=... and url=...")
	}
	*s = append(*s, src)
	return nil
}

func main() {
	var sources sourceFlag
	flag.Var(&sources, "source", "Repeatable. name=...,url=...,auth=...,caps=a|b")
	flag.Parse()

	if len(sources) == 0 {
		fmt.Fprintln(os.Stderr, "no -source provided. Run with --help.")
		os.Exit(2)
	}

	reg := registry.NewMemoryRegistry()
	imp := mcpsource.NewImporter(reg, nil)

	fmt.Printf("ACP MCP-source importer\n")
	fmt.Printf("  upstream servers : %d\n\n", len(sources))

	for _, src := range sources {
		fmt.Printf("─── %s (%s) ─────────────────────────────────────────\n", src.Name, src.BaseURL)
		n, err := imp.ImportSource(src)
		if err != nil {
			fmt.Printf("  ❌ import failed: %v\n\n", err)
			continue
		}
		fmt.Printf("  ✅ imported %d tool(s)\n\n", n)
	}

	tools := reg.All()
	fmt.Printf("=== ACP registry now holds %d tool(s) ===\n\n", len(tools))
	for _, t := range tools {
		fmt.Printf("  %-40s  caps=%v  endpoint=%s\n", t.ID, t.Capabilities, t.Endpoint)
	}
}
