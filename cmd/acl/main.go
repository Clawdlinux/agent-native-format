// Copyright 2026 NineVigil / Clawdlinux.
//
// Licensed under the Apache License, Version 2.0 (the "License").

package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"

	tiktoken "github.com/pkoukk/tiktoken-go"
	"github.com/spf13/cobra"

	"github.com/Clawdlinux/ninevigil-acp/pkg/acl"
	"github.com/Clawdlinux/ninevigil-acp/pkg/aclhttp"
	"github.com/Clawdlinux/ninevigil-acp/pkg/aclpg"
)

// version, commit, and date are set at link time via -ldflags "-X
// main.version=...". goreleaser populates all three for tagged
// releases; the Makefile populates `version` from `git describe` for
// local builds.
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	root := &cobra.Command{
		Use:           "acl",
		Short:         "Agent Context Language CLI",
		Long:          "Translate human-format sources into ACL, decode ACL documents, and measure token cost.",
		SilenceUsage:  true,
		SilenceErrors: false,
	}
	root.AddCommand(
		newEncodeCmd(),
		newDecodeCmd(),
		newTokensCmd(),
		newVersionCmd(),
	)
	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}

// ─── encode ─────────────────────────────────────────────────────────────────

func newEncodeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "encode <openapi|pg> [file|-]",
		Short: "Translate a source into ACL",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			source, path := args[0], args[1]
			data, err := readInput(path)
			if err != nil {
				return err
			}
			out, err := encodeSource(source, data)
			if err != nil {
				return err
			}
			_, err = os.Stdout.Write(out)
			return err
		},
	}
	cmd.Example = "  acl encode openapi spec.json\n" +
		"  cat spec.json | acl encode openapi -\n" +
		"  acl encode pg schema.json"
	return cmd
}

func encodeSource(source string, data []byte) ([]byte, error) {
	switch source {
	case "openapi":
		return aclhttp.Encode(data)
	case "pg":
		var s aclpg.Schema
		if err := json.Unmarshal(data, &s); err != nil {
			return nil, fmt.Errorf("parse pg schema JSON: %w", err)
		}
		return aclpg.Encode(s)
	case "k8s":
		return nil, fmt.Errorf("k8s source lives in the agentic-operator repo; use `agentctl acl` instead")
	default:
		return nil, fmt.Errorf("unknown source %q (want openapi|pg)", source)
	}
}

// ─── decode ─────────────────────────────────────────────────────────────────

func newDecodeCmd() *cobra.Command {
	var indent int
	cmd := &cobra.Command{
		Use:   "decode [file|-]",
		Short: "Parse an ACL document and print its structure as JSON",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			data, err := readInput(args[0])
			if err != nil {
				return err
			}
			doc, err := acl.Decode(data)
			if err != nil {
				return err
			}
			enc := json.NewEncoder(os.Stdout)
			if indent > 0 {
				enc.SetIndent("", spaces(indent))
			}
			return enc.Encode(doc)
		},
	}
	cmd.Flags().IntVar(&indent, "indent", 2, "spaces of indentation in JSON output (0 for compact)")
	return cmd
}

func spaces(n int) string {
	out := make([]byte, n)
	for i := range out {
		out[i] = ' '
	}
	return string(out)
}

// ─── tokens ─────────────────────────────────────────────────────────────────

func newTokensCmd() *cobra.Command {
	var encoding string
	cmd := &cobra.Command{
		Use:   "tokens [file|-]",
		Short: "Count tokens in a file using a public tokenizer",
		Long: "Reports byte length, character length, and token count for the given file.\n" +
			"Default tokenizer is cl100k_base (GPT-3.5/4 family); pass --encoding o200k_base for GPT-4o.",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			data, err := readInput(args[0])
			if err != nil {
				return err
			}
			enc, err := tiktoken.GetEncoding(encoding)
			if err != nil {
				return fmt.Errorf("unknown encoding %q: %w", encoding, err)
			}
			tokens := enc.Encode(string(data), nil, nil)
			fmt.Fprintf(os.Stdout, "bytes:    %d\n", len(data))
			fmt.Fprintf(os.Stdout, "chars:    %d\n", len([]rune(string(data))))
			fmt.Fprintf(os.Stdout, "tokens:   %d  (%s)\n", len(tokens), encoding)
			return nil
		},
	}
	cmd.Flags().StringVar(&encoding, "encoding", "cl100k_base",
		"tiktoken encoding name (cl100k_base, o200k_base, p50k_base, r50k_base)")
	return cmd
}

// ─── version ────────────────────────────────────────────────────────────────

func newVersionCmd() *cobra.Command {
	var verbose bool
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Print the acl CLI version",
		Run: func(cmd *cobra.Command, _ []string) {
			if verbose {
				fmt.Fprintf(os.Stdout, "acl %s\n  commit: %s\n  built:  %s\n",
					version, commit, date)
				return
			}
			fmt.Fprintln(os.Stdout, version)
		},
	}
	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false,
		"include commit hash and build date")
	return cmd
}

// ─── helpers ────────────────────────────────────────────────────────────────

// readInput reads from the given path, or from stdin when path == "-".
func readInput(path string) ([]byte, error) {
	if path == "-" {
		return io.ReadAll(os.Stdin)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("file not found: %s", path)
		}
		return nil, err
	}
	return data, nil
}
