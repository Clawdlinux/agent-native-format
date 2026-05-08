// Copyright 2026 NineVigil / Clawdlinux.
//
// Licensed under the Apache License, Version 2.0 (the "License").

package aclpg

import (
	"fmt"
	"sort"
	"strings"

	"github.com/Clawdlinux/ninevigil-acp/pkg/acl"
)

// translatorID is the value emitted in the @source directive.
const translatorID = "pg-schema/v0.1"

// TranslatorID returns the @source identifier this translator emits.
func TranslatorID() string { return translatorID }

// Schema is the typed input to Encode. Construct it from
// information_schema queries, pg_dump output parsing, or any other
// source — the translator only cares about the typed view.
type Schema struct {
	Database   string // optional; emitted as @db directive
	SchemaName string // optional; emitted as @schema directive (defaults to "public")
	Tables     []Table
	Indexes    []Index
	Relations  []Relation
}

// Table is one entity in the schema.
type Table struct {
	Name       string
	Columns    []Column
	PrimaryKey []string // column names that form the PK
}

// Column is a single field on a table.
type Column struct {
	Name     string
	Type     string // canonicalised: "int8", "text", "timestamptz", etc.
	Nullable bool
	Default  string // empty if no default; "AUTO" for serial / identity
}

// Index is a non-primary-key index. Primary-key indexes are implied by
// Table.PrimaryKey and not duplicated here.
type Index struct {
	Name    string
	Table   string
	Columns []string
	Unique  bool
}

// Relation is a foreign-key relationship from (FromTable.FromColumn) ->
// (ToTable.ToColumn).
type Relation struct {
	Name       string // FK constraint name (optional for display)
	FromTable  string
	FromColumn string
	ToTable    string
	ToColumn   string
}

// Encode renders an ACL v0.1 document for the given schema. The output
// is deterministic: identical schemas produce byte-identical output.
//
// This is the canonical entry point. It builds the typed acl.Document
// then delegates to the canonical encoder in pkg/acl, so the wire
// format is owned by exactly one package.
func Encode(s Schema) ([]byte, error) {
	return acl.Encode(buildDocument(s))
}

// Document returns the typed acl.Document for s, before encoding.
// Useful for callers that want to merge or post-process sections.
func Document(s Schema) acl.Document { return buildDocument(s) }

// ─── Document assembly ──────────────────────────────────────────────────────

func buildDocument(s Schema) acl.Document {
	d := acl.Document{Directives: directives(s)}
	if sec, ok := tablesSection(s.Tables); ok {
		d.Sections = append(d.Sections, sec)
	}
	if sec, ok := columnsSection(s.Tables); ok {
		d.Sections = append(d.Sections, sec)
	}
	if sec, ok := indexesSection(s.Indexes); ok {
		d.Sections = append(d.Sections, sec)
	}
	if sec, ok := relationsSection(s.Relations); ok {
		d.Sections = append(d.Sections, sec)
	}
	d.Sections = append(d.Sections, defaultActionsSection())
	return d
}

func directives(s Schema) []acl.Directive {
	var out []acl.Directive
	if s.Database != "" {
		out = append(out, acl.Directive{Key: "db", Value: s.Database})
	}
	schema := s.SchemaName
	if schema == "" {
		schema = "public"
	}
	out = append(out, acl.Directive{Key: "schema", Value: schema})
	out = append(out, acl.Directive{Key: "source", Value: translatorID})
	return out
}

// tablesSection emits one row per table with cheap structural facts:
// column count, primary-key column list, and index count for cross-ref
// to the indexes section.
func tablesSection(tables []Table) (acl.Section, bool) {
	if len(tables) == 0 {
		return acl.Section{}, false
	}
	// Stable ordering — tables in alpha order so the output is
	// reproducible even if the caller built Schema in a different order.
	sorted := append([]Table(nil), tables...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Name < sorted[j].Name })
	rows := make([]acl.Row, 0, len(sorted))
	for _, t := range sorted {
		fields := []acl.Field{
			{Key: "cols", Value: fmt.Sprintf("%d", len(t.Columns))},
		}
		if len(t.PrimaryKey) > 0 {
			fields = append(fields, acl.Field{Key: "pk", Value: strings.Join(t.PrimaryKey, ",")})
		}
		rows = append(rows, acl.Row{ID: t.Name, Fields: fields})
	}
	return acl.Section{
		Name:    "tables",
		Summary: fmt.Sprintf("%d", len(rows)),
		Rows:    rows,
	}, true
}

// columnsSection emits one row per (table, column). The compact form
// `table.col type [nul] [default]` keeps each column under ~30 bytes
// even for a wide schema, which is where this translator beats DDL
// hardest. Comments / descriptions / statistics are dropped.
func columnsSection(tables []Table) (acl.Section, bool) {
	if len(tables) == 0 {
		return acl.Section{}, false
	}
	sorted := append([]Table(nil), tables...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Name < sorted[j].Name })

	var rows []acl.Row
	for _, t := range sorted {
		for _, c := range t.Columns {
			id := fmt.Sprintf("%s.%s", t.Name, c.Name)
			fields := []acl.Field{
				{Key: "type", Value: canonicalType(c.Type)},
			}
			if c.Nullable {
				fields = append(fields, acl.Field{Key: "nul", Value: "1"})
			}
			if c.Default != "" {
				fields = append(fields, acl.Field{Key: "def", Value: shortDefault(c.Default)})
			}
			rows = append(rows, acl.Row{ID: id, Fields: fields})
		}
	}
	if len(rows) == 0 {
		return acl.Section{}, false
	}
	return acl.Section{
		Name:    "columns",
		Summary: fmt.Sprintf("%d", len(rows)),
		Rows:    rows,
	}, true
}

// indexesSection emits one row per non-PK index. Sort order is
// (table, name) for stable output.
func indexesSection(indexes []Index) (acl.Section, bool) {
	if len(indexes) == 0 {
		return acl.Section{}, false
	}
	sorted := append([]Index(nil), indexes...)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].Table != sorted[j].Table {
			return sorted[i].Table < sorted[j].Table
		}
		return sorted[i].Name < sorted[j].Name
	})
	rows := make([]acl.Row, 0, len(sorted))
	for _, ix := range sorted {
		fields := []acl.Field{
			{Key: "on", Value: ix.Table},
			{Key: "cols", Value: strings.Join(ix.Columns, ",")},
		}
		if ix.Unique {
			fields = append(fields, acl.Field{Key: "uniq", Value: "1"})
		}
		rows = append(rows, acl.Row{ID: ix.Name, Fields: fields})
	}
	return acl.Section{
		Name:    "indexes",
		Summary: fmt.Sprintf("%d", len(rows)),
		Rows:    rows,
	}, true
}

// relationsSection captures the join graph. The compact form
// `from_table.col -> to_table.col` is the spec's `->` mapping token,
// chosen specifically so an agent reading ACL can pivot from any FK
// to its referenced table without parsing a separate relations doc.
func relationsSection(relations []Relation) (acl.Section, bool) {
	if len(relations) == 0 {
		return acl.Section{}, false
	}
	sorted := append([]Relation(nil), relations...)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].FromTable != sorted[j].FromTable {
			return sorted[i].FromTable < sorted[j].FromTable
		}
		return sorted[i].FromColumn < sorted[j].FromColumn
	})
	rows := make([]acl.Row, 0, len(sorted))
	for _, r := range sorted {
		// Use the `from.col->to.col` form as the row ID directly.
		// `->` is a reserved bareword token (spec §3.2) and `.` plus
		// alphanumerics are in the bare-identifier charset, so this
		// round-trips cleanly without quoting.
		id := fmt.Sprintf("%s.%s->%s.%s", r.FromTable, r.FromColumn, r.ToTable, r.ToColumn)
		row := acl.Row{ID: id}
		if r.Name != "" {
			row.Fields = []acl.Field{{Key: "fk", Value: r.Name}}
		}
		rows = append(rows, row)
	}
	return acl.Section{
		Name:    "relations",
		Summary: fmt.Sprintf("%d", len(rows)),
		Rows:    rows,
	}, true
}

// defaultActionsSection emits the closed verb set the agent may invoke.
// Future versions will compute this from per-table grants instead of
// hardcoding it.
func defaultActionsSection() acl.Section {
	return acl.Section{
		Name: "actions",
		Rows: []acl.Row{{ID: "select|insert|update|delete|describe"}},
	}
}

// ─── helpers ────────────────────────────────────────────────────────────────

// canonicalType collapses the verbose Postgres type names into compact
// agent-friendly forms. The agent doesn't need to know the difference
// between "character varying(255)" and "varchar(255)"; both are "text".
//
// This is intentionally lossy: precision and length are dropped. An
// agent that needs the exact type can ask the database directly via
// the `describe` action.
func canonicalType(t string) string {
	t = strings.ToLower(strings.TrimSpace(t))
	// Strip parenthesised modifiers: "varchar(255)" -> "varchar".
	if i := strings.IndexByte(t, '('); i > 0 {
		t = strings.TrimSpace(t[:i])
	}
	switch t {
	case "character varying", "varchar", "char", "character", "bpchar":
		return "text"
	case "integer", "int", "int4":
		return "int4"
	case "bigint", "int8":
		return "int8"
	case "smallint", "int2":
		return "int2"
	case "double precision", "float8":
		return "float8"
	case "real", "float4":
		return "float4"
	case "timestamp without time zone", "timestamp":
		return "timestamp"
	case "timestamp with time zone", "timestamptz":
		return "timestamptz"
	case "boolean", "bool":
		return "bool"
	case "json", "jsonb":
		return t
	default:
		// Replace whitespace with underscore so the type stays a bare
		// token (e.g. "user defined" -> "user_defined").
		return strings.ReplaceAll(t, " ", "_")
	}
}

// shortDefault replaces verbose Postgres default expressions with
// agent-friendly canonical forms. Anything not recognised is returned
// as-is (with whitespace stripped).
func shortDefault(d string) string {
	s := strings.TrimSpace(d)
	low := strings.ToLower(s)
	switch {
	case strings.HasPrefix(low, "nextval("):
		return "AUTO"
	case low == "current_timestamp" || low == "now()":
		return "now"
	case low == "true" || low == "false":
		return low
	case low == "null":
		return "null"
	}
	// Drop quotes from `'string literal'::text` style defaults so the
	// value stays bare-token-friendly.
	s = strings.TrimPrefix(s, "'")
	if idx := strings.Index(s, "'::"); idx >= 0 {
		s = s[:idx]
	}
	s = strings.TrimSuffix(s, "'")
	// Replace whitespace; if anything reserved remains, the encoder will
	// quote it for us.
	return strings.ReplaceAll(s, " ", "_")
}
