/*
Copyright 2026 Clawdlinux / NineVigil.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0
*/

package acp

import (
	"context"

	"github.com/Clawdlinux/ninevigil-acp/pkg/acl"
)

//go:generate ../../bin/mockgen -source=translator.go -destination=mock_translator_test.go -package=acp

// Translator converts a human-format source S into an agent-native ACL
// document. Implementations live alongside their source: a Kubernetes
// translator lives in the agentic-operator repo, an MCP translator lives
// here, a web-page translator lives in a future `acl-web` package.
//
// The contract has three guarantees:
//
//  1. Determinism — identical input must produce a byte-identical
//     acl.Document (after canonical Encode). This makes ACL outputs
//     content-addressable and cacheable.
//
//  2. Scope honesty — the returned Document carries an `@source`
//     directive identifying the translator and version. An agent that
//     receives an ACL document MUST be able to identify what produced it.
//
//  3. Affordance preservation — if the source exposes actions an agent
//     can invoke, the translator MUST emit an `actions` section listing
//     those affordances.
//
// The ctx parameter is reserved for future cancellation/deadline
// propagation when translators ingest from network sources (HTTP MCP
// servers, Kubernetes API watches, web fetches). v0.1 in-memory
// translators MAY ignore it, but implementations that perform I/O
// MUST honour ctx.Done().
//
// Translator is generic over the source type S so each implementation
// keeps its native types (e.g. *corev1.PodList) without an awkward
// any-typed surface.
type Translator[S any] interface {
	// Translate consumes a source-shaped value and returns an ACL
	// document representing the decision-relevant subset, plus optional
	// metadata about the translation (token count estimate, fields
	// dropped, etc.) for observability.
	Translate(ctx context.Context, src S) (acl.Document, TranslateInfo, error)

	// Name returns a stable identifier used in the @source directive,
	// e.g. "k8s-namespace/v0.1" or "mcp-tools-list/v0.1".
	Name() string
}

// TranslateInfo is observability metadata returned alongside an ACL
// document. None of these fields affect agent behaviour; they exist so
// the translation layer can be measured and improved over time (the data
// flywheel).
type TranslateInfo struct {
	// SourceBytes is the size of the original human-format payload, if
	// known. Zero means "not measured".
	SourceBytes int

	// EncodedBytes is the size of the resulting ACL document after
	// Encode. Zero means "encode not yet performed".
	EncodedBytes int

	// FieldsDropped is the count of source fields the translator chose
	// to omit. Useful for tuning what to include.
	FieldsDropped int

	// Notes carries free-form translator-specific diagnostics.
	Notes []string
}
