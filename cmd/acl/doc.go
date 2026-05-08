// Copyright 2026 NineVigil / Clawdlinux.
//
// Licensed under the Apache License, Version 2.0 (the "License").

// Command acl is the standalone CLI for the Agent Context Language.
//
// It exposes the in-tree translators (OpenAPI, Postgres-schema) plus
// the canonical encoder/decoder as a single static binary. The
// intended use is in CI pipelines, scripts, and agent-side
// preprocessors that want to convert a human-format source into ACL
// on the fly.
//
// The Kubernetes translator (aclk8s) lives in the agentic-operator
// repository because it depends on the kubernetes/client-go module;
// use `agentctl acl` from that repo for K8s sources.
//
// Usage:
//
//	acl encode openapi spec.json
//	acl encode pg      schema.json  // JSON form of aclpg.Schema
//	acl decode         doc.acl
//	acl tokens         file
//	acl version
//
// All commands read from stdin when the file argument is "-".
package main
