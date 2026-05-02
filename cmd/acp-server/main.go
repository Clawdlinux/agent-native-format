/*
Copyright 2026 Clawdlinux / NineVigil.

Licensed under the Business Source License 1.1.
See LICENSE in the repository root.
*/

// Command acp-server runs the Week 1 ACP HTTP server with an in-memory
// registry, the keyword resolver, and the manifest builder. Week 2 adds the
// auth-injection proxy mounted at /v1/exec/.
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	builder "github.com/Clawdlinux/ninevigil-acp/internal/builder"
	"github.com/Clawdlinux/ninevigil-acp/internal/proxy"
	"github.com/Clawdlinux/ninevigil-acp/internal/registry"
	"github.com/Clawdlinux/ninevigil-acp/internal/resolver"
	"github.com/Clawdlinux/ninevigil-acp/internal/server"
)

func main() {
	addr := flag.String("addr", ":8080", "listen address")
	authToken := flag.String("auth-token", os.Getenv("ACP_AUTH_TOKEN"),
		"bearer token required on /v1/* (default: ACP_AUTH_TOKEN env var; empty disables auth)")
	feedbackEndpoint := flag.String("feedback-endpoint", "/v1/feedback",
		"feedback endpoint advertised in manifests")
	enableProxy := flag.Bool("enable-proxy", true, "mount the auth-injection proxy at /v1/exec/")
	autoApprove := flag.Bool("auto-approve", false, "approve every gated action (DEV ONLY)")
	flag.Parse()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	reg := registry.NewMemoryRegistry()
	if err := registry.Seed(reg); err != nil {
		logger.Error("registry seed failed", slog.String("err", err.Error()))
		os.Exit(1)
	}

	bld := builder.New(
		reg,
		server.CryptoIDSource{},
		builder.Options{
			TTL:                "300s",
			FeedbackEndpoint:   *feedbackEndpoint,
			MaxTokensPerAction: 15000,
		},
	)

	cfg := server.Config{
		Resolver:  resolver.NewKeywordResolver(nil),
		Builder:   bld,
		Feedback:  &server.LoggingFeedbackSink{Logger: logger},
		AuthToken: *authToken,
		Logger:    logger,
	}

	if *enableProxy {
		store := proxy.NewMemoryStore()
		var gate proxy.ApprovalGate = proxy.AlwaysDeny{}
		if *autoApprove {
			gate = proxy.AlwaysApprove{}
		}
		cfg.Persister = store
		cfg.Proxy = proxy.New(proxy.Config{
			Store:    store,
			Injector: &proxy.MapInjector{}, // placeholder; production wires to a vault
			Approval: gate,
			Logger:   logger,
		})
	}

	handler := server.New(cfg)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	logger.Info("acp-server starting",
		slog.String("addr", *addr),
		slog.Bool("auth_required", *authToken != ""),
		slog.Bool("proxy_enabled", *enableProxy),
		slog.Bool("auto_approve", *autoApprove),
		slog.Int("seeded_tools", len(reg.All())),
	)

	if err := server.ListenAndServe(ctx, *addr, handler); err != nil {
		fmt.Fprintln(os.Stderr, "acp-server:", err)
		os.Exit(1)
	}
	logger.Info("acp-server stopped")
}
