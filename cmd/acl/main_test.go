// Copyright 2026 NineVigil / Clawdlinux.
//
// Licensed under the Apache License, Version 2.0 (the "License").

package main
// Copyright 2026 NineVigil / Clawdlinux.
//
// Licensed under the Apache License, Version 2.0 (the "License").

package main

import (
	"strings"
	"testing"
)

func TestEncodeSourceOpenAPI(t *testing.T) {
	t.Parallel()
	spec := `{
		"openapi": "3.0.0",
		"info": {"title": "T", "version": "1"},
		"paths": {
			"/foo": {"get": {"operationId": "getFoo"}}
		}
	}`
	out, err := encodeSource("openapi", []byte(spec))
	if err != nil {
		t.Fatalf("encodeSource: %v", err)
	}
	if !strings.Contains(string(out), "GET/foo") {
		t.Fatalf("expected GET/foo row in output:\n%s", out)
	}
}

func TestEncodeSourcePG(t *testing.T) {
	t.Parallel()
	in := `{
		"Database": "x",
		"SchemaName": "public",
		"Tables": [
			{"Name": "t", "Columns": [{"Name": "id", "Type": "bigint"}], "PrimaryKey": ["id"]}
		]
	}`
	out, err := encodeSource("pg", []byte(in))
	if err != nil {
		t.Fatalf("encodeSource: %v", err)
	}
	if !strings.Contains(string(out), "tables 1") {
		t.Fatalf("expected 'tables 1' in output:\n%s", out)
	}
}

func TestEncodeSourceK8sIsRoutedAway(t *testing.T) {
	t.Parallel()
	_, err := encodeSource("k8s", []byte("{}"))
	if err == nil || !strings.Contains(err.Error(), "agentic-operator") {
		t.Fatalf("expected k8s to be routed away, got %v", err)
	}
}

func TestEncodeSourceUnknown(t *testing.T) {
	t.Parallel()
	_, err := encodeSource("nope", []byte("{}"))
	if err == nil || !strings.Contains(err.Error(), "unknown source") {
		t.Fatalf("expected unknown-source error, got %v", err)
	}
}
