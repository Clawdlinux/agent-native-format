/*
Copyright 2026 Clawdlinux / NineVigil.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0
*/

// Package acl is the Go reference implementation of the
// Agent Context Language (ACL) v0.1 — a line-oriented, machine-native
// representation designed to be consumed by LLM agents rather than humans.
//
// The full specification lives at docs/acl-spec.md in the ninevigil-acp
// repository. This package provides types, an Encode/Decode pair with
// round-trip stability, and a small set of validation helpers.
//
// Example:
//
//	doc := acl.Document{
//	    Directives: []acl.Directive{
//	        {Key: "cluster", Value: "prod-east"},
//	        {Key: "ns", Value: "payments"},
//	    },
//	    Sections: []acl.Section{
//	        {
//	            Name: "pods", Summary: "12/12 ok",
//	            Rows: []acl.Row{
//	                {ID: "payment-api-7f8d", Count: 3, Fields: []acl.Field{
//	                    {Key: "cpu", Value: "42"}, {Key: "mem", Value: "61"},
//	                }},
//	            },
//	        },
//	    },
//	}
//	out, _ := acl.Encode(doc)
package acl

// Document is the top-level ACL container.
type Document struct {
	Directives []Directive
	Sections   []Section
}

// Directive is a document-scope `@key value` line.
type Directive struct {
	Key   string
	Value string
}

// Section is a header line plus zero-or-more indented rows.
type Section struct {
	Name    string // e.g. "pods", "deploys", "actions"
	Summary string // optional aggregate, e.g. "12/12 ok"
	Rows    []Row
}

// Row is an indented record under a Section.
type Row struct {
	ID     string  // primary identifier; "" allowed for unnamed rows (e.g. actions)
	Count  int     // optional instance count; 0 means unset (no `xN` token emitted)
	Fields []Field // ordered key=value pairs
	Flags  []string
}

// Field is a single `key=value` pair within a row.
type Field struct {
	Key   string
	Value string
}
