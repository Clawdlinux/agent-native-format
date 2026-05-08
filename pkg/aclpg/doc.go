// Copyright 2026 NineVigil / Clawdlinux.
//
// Licensed under the Apache License, Version 2.0 (the "License").

// Package aclpg translates relational database schemas (Postgres /
// information_schema-shaped) into Agent Context Language (ACL) v0.1.
//
// Database schemas are the third domain ACL is proven on, after
// Kubernetes and HTTP/OpenAPI. The shape is different again: a
// schema is a *graph* of typed entities related by foreign keys.
// What an agent needs to plan a query is the entity catalog plus
// the join graph, not column-level descriptions for human DBAs.
//
// The translator emits four sections:
//
//   - tables     one row per table with column / PK / index counts
//   - columns    one row per column (table.column type nullable default)
//   - indexes    one row per non-PK index
//   - relations  one row per foreign-key relationship (a -> b)
//   - actions    the closed verb set the agent may invoke
//
// Comments, descriptions, statistics, ownership, ACLs, and other
// metadata that exists for human DBAs are intentionally dropped.
package aclpg
