// Copyright 2026 NineVigil / Clawdlinux.
//
// Licensed under the Apache License, Version 2.0 (the "License").

package aclpg

import (
	"sort"
	"strings"
	"testing"
)

// tinyDDL is the spec's worked example, written by hand the way a
// developer would type it (rather than the way pg_dump emits it).
const tinyDDL = `
CREATE TABLE public.users (
    id bigint NOT NULL DEFAULT nextval('users_id_seq'::regclass),
    email text NOT NULL,
    created_at timestamp with time zone DEFAULT now()
);
ALTER TABLE ONLY public.users ADD CONSTRAINT users_pkey PRIMARY KEY (id);
CREATE UNIQUE INDEX users_email_key ON public.users USING btree (email);

CREATE TABLE public.orders (
    id bigint NOT NULL DEFAULT nextval('orders_id_seq'::regclass),
    user_id bigint NOT NULL,
    total_cents integer NOT NULL,
    status character varying(32) DEFAULT 'pending'::text
);
ALTER TABLE ONLY public.orders ADD CONSTRAINT orders_pkey PRIMARY KEY (id);
CREATE INDEX orders_user_id_idx ON public.orders USING btree (user_id);
ALTER TABLE ONLY public.orders ADD CONSTRAINT orders_user_id_fkey
    FOREIGN KEY (user_id) REFERENCES public.users(id);
`

func TestParseDDLTiny(t *testing.T) {
	t.Parallel()
	got, err := ParseDDL(tinyDDL)
	if err != nil {
		t.Fatalf("ParseDDL: %v", err)
	}
	if got.SchemaName != "public" {
		t.Errorf("schema name: got %q, want public", got.SchemaName)
	}
	if len(got.Tables) != 2 {
		t.Fatalf("tables: got %d, want 2", len(got.Tables))
	}
	// Sort tables by name so assertions are deterministic regardless of
	// map iteration order.
	sort.Slice(got.Tables, func(i, j int) bool {
		return got.Tables[i].Name < got.Tables[j].Name
	})
	users := got.Tables[1]
	if users.Name != "users" {
		t.Errorf("table[1].Name = %q, want users", users.Name)
	}
	if len(users.Columns) != 3 {
		t.Errorf("users columns: got %d, want 3", len(users.Columns))
	}
	if len(users.PrimaryKey) != 1 || users.PrimaryKey[0] != "id" {
		t.Errorf("users PK: got %v, want [id]", users.PrimaryKey)
	}
	// id column: NOT NULL + DEFAULT.
	if users.Columns[0].Nullable {
		t.Errorf("users.id should be NOT NULL")
	}
	if !strings.Contains(users.Columns[0].Default, "nextval") {
		t.Errorf("users.id default lost: %q", users.Columns[0].Default)
	}
	// email column: NOT NULL, no default.
	if users.Columns[1].Nullable {
		t.Errorf("users.email should be NOT NULL")
	}
	if users.Columns[1].Default != "" {
		t.Errorf("users.email should have no default, got %q", users.Columns[1].Default)
	}
	// created_at: nullable (no NOT NULL), DEFAULT now().
	if !users.Columns[2].Nullable {
		t.Errorf("users.created_at should be nullable")
	}
	// Indexes.
	if len(got.Indexes) != 2 {
		t.Fatalf("indexes: got %d, want 2", len(got.Indexes))
	}
	var emailKey *Index
	for i := range got.Indexes {
		if got.Indexes[i].Name == "users_email_key" {
			emailKey = &got.Indexes[i]
		}
	}
	if emailKey == nil || !emailKey.Unique {
		t.Errorf("users_email_key missing or not UNIQUE: %+v", emailKey)
	}
	// Relations.
	if len(got.Relations) != 1 {
		t.Fatalf("relations: got %d, want 1", len(got.Relations))
	}
	r := got.Relations[0]
	if r.FromTable != "orders" || r.FromColumn != "user_id" ||
		r.ToTable != "users" || r.ToColumn != "id" {
		t.Errorf("FK: got %+v", r)
	}
}

// TestParseDDLRealistic round-trips the realistic pg_dump fixture:
// build the typed ecommerceSchema, render it as realistic pg_dump
// output, parse that output back, and check the parser recovered the
// same schema shape (table count, PK list, FK count).
//
// This is the headline test: it proves the parser handles the exact
// shape pg_dump emits in production, including the SET / OWNER /
// SEQUENCE / GRANT noise we deliberately ignore.
func TestParseDDLRealistic(t *testing.T) {
	t.Parallel()
	original := ecommerceSchema()
	dump := realisticPgDump(original)
	parsed, err := ParseDDL(dump)
	if err != nil {
		t.Fatalf("ParseDDL: %v", err)
	}

	if got, want := len(parsed.Tables), len(original.Tables); got != want {
		t.Errorf("table count: got %d, want %d", got, want)
	}
	// Compare table names (set-equal).
	gotNames := tableNames(parsed.Tables)
	wantNames := tableNames(original.Tables)
	for _, name := range wantNames {
		if !contains(gotNames, name) {
			t.Errorf("missing table after parse: %s", name)
		}
	}

	// Every original PK must survive.
	pkOriginal := pkMap(original.Tables)
	pkParsed := pkMap(parsed.Tables)
	for name, want := range pkOriginal {
		if got := pkParsed[name]; !equalCols(got, want) {
			t.Errorf("PK on %s: got %v, want %v", name, got, want)
		}
	}

	// Index count + FK count must match. Per-row content is not
	// asserted because pg_dump output preserves order; the parser
	// preserves order too, but we don't want to over-spec the test.
	if got, want := len(parsed.Indexes), len(original.Indexes); got != want {
		t.Errorf("index count: got %d, want %d", got, want)
	}
	if got, want := len(parsed.Relations), len(original.Relations); got != want {
		t.Errorf("relation count: got %d, want %d", got, want)
	}
}

// TestParseDDLEncodes verifies the parsed schema produces a valid,
// round-trippable ACL document.
func TestParseDDLEncodes(t *testing.T) {
	t.Parallel()
	parsed, err := ParseDDL(tinyDDL)
	if err != nil {
		t.Fatalf("ParseDDL: %v", err)
	}
	out, err := Encode(parsed)
	if err != nil {
		t.Fatalf("Encode parsed schema: %v", err)
	}
	if !strings.Contains(string(out), "tables 2") {
		t.Errorf("expected 'tables 2' in output:\n%s", out)
	}
	if !strings.Contains(string(out), "@source pg-schema/v0.1") {
		t.Errorf("expected @source directive in output:\n%s", out)
	}
}

func TestParseDDLIgnoresNoise(t *testing.T) {
	t.Parallel()
	noisy := `
SET statement_timeout = 0;
SELECT pg_catalog.set_config('search_path', '', false);
-- this is a comment
COMMENT ON TABLE foo IS 'bar';
GRANT ALL ON TABLE foo TO postgres;

CREATE TABLE public.thing (
    id bigint NOT NULL,
    name text
);
ALTER TABLE public.thing OWNER TO postgres;
`
	got, err := ParseDDL(noisy)
	if err != nil {
		t.Fatalf("ParseDDL on noisy input: %v", err)
	}
	if len(got.Tables) != 1 || got.Tables[0].Name != "thing" {
		t.Errorf("expected 1 table 'thing', got %+v", got.Tables)
	}
}

// ─── helpers ────────────────────────────────────────────────────────────────

func tableNames(ts []Table) []string {
	out := make([]string, len(ts))
	for i, t := range ts {
		out[i] = t.Name
	}
	return out
}

func pkMap(ts []Table) map[string][]string {
	m := make(map[string][]string, len(ts))
	for _, t := range ts {
		m[t.Name] = t.PrimaryKey
	}
	return m
}

func contains(s []string, v string) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}

func equalCols(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
