// Copyright 2026 NineVigil / Clawdlinux.
//
// Licensed under the Apache License, Version 2.0 (the "License").

// Package aclhttp translates OpenAPI 3.x specifications into Agent
// Context Language (ACL) v0.1.
//
// HTTP / REST APIs are the second domain ACL is proven on, after
// Kubernetes. The two are deliberately different shapes: K8s state is
// an inventory ("what exists right now?"), an OpenAPI spec is an
// affordance catalogue ("what can I do?"). If ACL works for both, the
// format claim is no longer K8s-specific.
//
// The translator reads enough of the OpenAPI spec to give an agent
// what it needs to plan an HTTP call:
//
//   - endpoints  one row per (method, path) operation
//   - schemas    one row per top-level component schema
//   - auth       one row per security scheme
//   - actions    the closed verb set the agent may invoke
//
// Everything that exists for human readers (descriptions, examples,
// vendor extensions, info.contact, externalDocs, $ref chains) is
// dropped. What remains is the affordance shape.
package aclhttp
