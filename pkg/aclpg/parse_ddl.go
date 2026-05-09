// Copyright 2026 NineVigil / Clawdlinux.
//
// Licensed under the Apache License, Version 2.0 (the "License").

package aclpg

import (
	"bufio"
	"fmt"
	"regexp"
	"strings"
)

// ParseDDL parses a deliberately narrow subset of Postgres DDL into
// the typed Schema struct. The target dialect is what `pg_dump -s`
// emits: CREATE TABLE, ALTER TABLE ... ADD CONSTRAINT pkey, CREATE
// INDEX, and ALTER TABLE ... ADD CONSTRAINT fkey statements.
//
// What is supported:
//   - CREATE TABLE [IF NOT EXISTS] [schema.]name ( col type [NOT NULL]
//     [DEFAULT expr], ... );
//   - ALTER TABLE [ONLY] [schema.]name ADD CONSTRAINT name_pkey
//     PRIMARY KEY (col, ...);
//   - CREATE [UNIQUE] INDEX name ON [schema.]table [USING btree] (col, ...);
//   - ALTER TABLE [ONLY] [schema.]table ADD CONSTRAINT name FOREIGN KEY
//     (col) REFERENCES [schema.]table(col);
//
// What is NOT supported (intentionally — out of scope for v0.1):
//   - Triggers, functions, views, materialized views
//   - CHECK constraints, EXCLUSION constraints
//   - Inheritance, partitioning, OIDs, tablespaces
//   - GRANT, REVOKE, COMMENT ON, OWNER TO (these are passed through as
//     comments by pg_dump and silently ignored here)
//   - Multi-column FKs (only single-column FKs are parsed; the row
//     types support arbitrary keys but the regex only captures one)
//
// The parser is line-oriented and uses regular expressions — no SQL
// AST. This keeps the scope honest: ACL is for translating
// human-shaped data into agent-shaped data, not for being a SQL
// engine.
func ParseDDL(sql string) (Schema, error) {
	p := &ddlParser{
		schema:  Schema{SchemaName: "public"},
		tables:  map[string]*Table{},
		schemas: map[string]bool{},
	}
	if err := p.run(sql); err != nil {
		return Schema{}, err
	}
	return p.schema, nil
}

// ─── implementation ─────────────────────────────────────────────────────────

type ddlParser struct {
	schema Schema
	// tables indexed by canonical "schema.name" so PK/FK statements that
	// arrive after CREATE TABLE can find their target.
	tables  map[string]*Table
	schemas map[string]bool
}

// Statement-level regexps. These match the entire statement once we've
// stitched continuation lines together (statements terminate at `;`).
var (
	reCreateTableHead = regexp.MustCompile(
		`(?is)^\s*CREATE\s+TABLE(?:\s+IF\s+NOT\s+EXISTS)?\s+` +
			`(?:([a-z_][a-z0-9_]*)\.)?([a-z_][a-z0-9_]*)\s*\(\s*(.*?)\s*\)\s*;?\s*$`)

	reAddPK = regexp.MustCompile(
		`(?is)^\s*ALTER\s+TABLE(?:\s+ONLY)?\s+` +
			`(?:([a-z_][a-z0-9_]*)\.)?([a-z_][a-z0-9_]*)\s+` +
			`ADD\s+CONSTRAINT\s+\S+\s+PRIMARY\s+KEY\s*\(\s*([^)]+?)\s*\)\s*;?\s*$`)

	reCreateIndex = regexp.MustCompile(
		`(?is)^\s*CREATE\s+(UNIQUE\s+)?INDEX\s+(\S+)\s+ON\s+` +
			`(?:([a-z_][a-z0-9_]*)\.)?([a-z_][a-z0-9_]*)\s+` +
			`(?:USING\s+\S+\s+)?\(\s*([^)]+?)\s*\)\s*;?\s*$`)

	reAddFK = regexp.MustCompile(
		`(?is)^\s*ALTER\s+TABLE(?:\s+ONLY)?\s+` +
			`(?:([a-z_][a-z0-9_]*)\.)?([a-z_][a-z0-9_]*)\s+` +
			`ADD\s+CONSTRAINT\s+(\S+)\s+FOREIGN\s+KEY\s*\(\s*([^)]+?)\s*\)\s+` +
			`REFERENCES\s+(?:([a-z_][a-z0-9_]*)\.)?([a-z_][a-z0-9_]*)\s*\(\s*([^)]+?)\s*\)` +
			`(?:\s+[A-Z\s]+)?\s*;?\s*$`)
)

// Column-line regex: matches one column definition inside a CREATE
// TABLE body. The body has been split on commas at the top level
// (parens balanced) before this is applied.
var reColumn = regexp.MustCompile(
	`(?is)^\s*([a-z_][a-z0-9_]*)\s+` + // name
		`(.+?)` + // type (greedy until NOT NULL or DEFAULT)
		`(\s+NOT\s+NULL)?` +
		`(?:\s+DEFAULT\s+(.+))?\s*$`)

func (p *ddlParser) run(sql string) error {
	stmts, err := splitStatements(sql)
	if err != nil {
		return err
	}
	for i, stmt := range stmts {
		if err := p.dispatch(stmt); err != nil {
			return fmt.Errorf("aclpg: statement %d: %w", i+1, err)
		}
	}
	// Move parsed tables into the schema (deterministic order).
	for _, t := range p.tables {
		p.schema.Tables = append(p.schema.Tables, *t)
	}
	// Schema name follows the first table seen (most pg_dump output is
	// consistent within a single dump).
	for s := range p.schemas {
		if s != "" && s != "public" {
			p.schema.SchemaName = s
			break
		}
	}
	return nil
}

// dispatch routes a single SQL statement to the right handler. Lines
// the parser doesn't recognise are silently ignored — this is by
// design, because pg_dump emits dozens of SET / SELECT pg_catalog /
// COMMENT / OWNER / GRANT lines that don't carry schema content.
func (p *ddlParser) dispatch(stmt string) error {
	upper := strings.ToUpper(strings.TrimSpace(stmt))
	switch {
	case strings.HasPrefix(upper, "CREATE TABLE"):
		return p.handleCreateTable(stmt)
	case strings.HasPrefix(upper, "ALTER TABLE"):
		// Could be PK or FK. Try both regexps.
		if reAddPK.MatchString(stmt) {
			return p.handleAddPK(stmt)
		}
		if reAddFK.MatchString(stmt) {
			return p.handleAddFK(stmt)
		}
		// Other ALTER TABLE forms (OWNER, ALTER COLUMN, etc.) — ignore.
		return nil
	case strings.HasPrefix(upper, "CREATE UNIQUE INDEX"),
		strings.HasPrefix(upper, "CREATE INDEX"):
		return p.handleCreateIndex(stmt)
	default:
		return nil
	}
}

func (p *ddlParser) handleCreateTable(stmt string) error {
	m := reCreateTableHead.FindStringSubmatch(stmt)
	if m == nil {
		return fmt.Errorf("CREATE TABLE: unrecognised form: %s", firstLine(stmt))
	}
	schema, name, body := m[1], m[2], m[3]
	p.schemas[schema] = true
	t := &Table{Name: name}
	for _, raw := range splitTopLevelCommas(body) {
		col, ok := parseColumn(raw)
		if !ok {
			// Skip non-column items inside the table body (CONSTRAINT
			// inline definitions are pg_dump output for unique/check
			// constraints — out of scope).
			continue
		}
		t.Columns = append(t.Columns, col)
	}
	key := canon(schema, name)
	p.tables[key] = t
	return nil
}

func (p *ddlParser) handleAddPK(stmt string) error {
	m := reAddPK.FindStringSubmatch(stmt)
	if m == nil {
		return nil
	}
	schema, name, cols := m[1], m[2], m[3]
	t, ok := p.tables[canon(schema, name)]
	if !ok {
		// PK on an unknown table — pg_dump always orders TABLE before
		// its PK so this is a malformed input we silently ignore.
		return nil
	}
	t.PrimaryKey = parseIdentList(cols)
	return nil
}

func (p *ddlParser) handleCreateIndex(stmt string) error {
	m := reCreateIndex.FindStringSubmatch(stmt)
	if m == nil {
		return nil
	}
	uniq := m[1] != ""
	idxName := m[2]
	schema := m[3]
	tableName := m[4]
	cols := parseIdentList(m[5])
	p.schemas[schema] = true
	p.schema.Indexes = append(p.schema.Indexes, Index{
		Name:    idxName,
		Table:   tableName,
		Columns: cols,
		Unique:  uniq,
	})
	return nil
}

func (p *ddlParser) handleAddFK(stmt string) error {
	m := reAddFK.FindStringSubmatch(stmt)
	if m == nil {
		return nil
	}
	fkName := m[3]
	fromCol := strings.TrimSpace(m[4])
	toTable := m[6]
	toCol := strings.TrimSpace(m[7])
	fromSchema := m[1]
	fromTable := m[2]
	p.schemas[fromSchema] = true
	p.schema.Relations = append(p.schema.Relations, Relation{
		Name:       fkName,
		FromTable:  fromTable,
		FromColumn: fromCol,
		ToTable:    toTable,
		ToColumn:   toCol,
	})
	return nil
}

// ─── helpers ────────────────────────────────────────────────────────────────

// splitStatements splits SQL into statements at top-level semicolons.
// It honours single-quoted string literals and parenthesis depth so
// that `DEFAULT 'a;b'` and `CHECK (x;)` (legal) don't get cut early.
// Comment lines (-- ...) are stripped.
func splitStatements(sql string) ([]string, error) {
	var out []string
	var cur strings.Builder
	depth := 0
	inQuote := false
	scanner := bufio.NewScanner(strings.NewReader(sql))
	scanner.Buffer(make([]byte, 0, 64*1024), 16*1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		// Strip line comments at -- but not inside strings.
		if !inQuote {
			if idx := strings.Index(line, "--"); idx >= 0 {
				line = line[:idx]
			}
		}
		for i := 0; i < len(line); i++ {
			c := line[i]
			switch {
			case c == '\'' && !inQuote:
				inQuote = true
				cur.WriteByte(c)
			case c == '\'' && inQuote:
				// Doubled quote escape: ''
				if i+1 < len(line) && line[i+1] == '\'' {
					cur.WriteByte(c)
					cur.WriteByte(line[i+1])
					i++
					continue
				}
				inQuote = false
				cur.WriteByte(c)
			case c == '(' && !inQuote:
				depth++
				cur.WriteByte(c)
			case c == ')' && !inQuote:
				depth--
				cur.WriteByte(c)
			case c == ';' && !inQuote && depth == 0:
				stmt := strings.TrimSpace(cur.String())
				if stmt != "" {
					out = append(out, stmt)
				}
				cur.Reset()
			default:
				cur.WriteByte(c)
			}
		}
		cur.WriteByte('\n')
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan: %w", err)
	}
	// Trailing statement without semicolon.
	if rest := strings.TrimSpace(cur.String()); rest != "" {
		out = append(out, rest)
	}
	return out, nil
}

// splitTopLevelCommas splits a CREATE TABLE body on commas that are
// not inside parentheses (so types like `numeric(10,2)` stay together).
func splitTopLevelCommas(body string) []string {
	var out []string
	var cur strings.Builder
	depth := 0
	for i := 0; i < len(body); i++ {
		c := body[i]
		switch {
		case c == '(':
			depth++
			cur.WriteByte(c)
		case c == ')':
			depth--
			cur.WriteByte(c)
		case c == ',' && depth == 0:
			out = append(out, strings.TrimSpace(cur.String()))
			cur.Reset()
		default:
			cur.WriteByte(c)
		}
	}
	if rest := strings.TrimSpace(cur.String()); rest != "" {
		out = append(out, rest)
	}
	return out
}

// parseColumn applies reColumn and converts the result to a Column.
// Returns ok=false if the line isn't a column (e.g. inline CONSTRAINT).
func parseColumn(s string) (Column, bool) {
	s = strings.TrimSpace(s)
	upper := strings.ToUpper(s)
	if strings.HasPrefix(upper, "CONSTRAINT") ||
		strings.HasPrefix(upper, "PRIMARY KEY") ||
		strings.HasPrefix(upper, "UNIQUE") ||
		strings.HasPrefix(upper, "CHECK") ||
		strings.HasPrefix(upper, "FOREIGN KEY") {
		return Column{}, false
	}
	m := reColumn.FindStringSubmatch(s)
	if m == nil {
		return Column{}, false
	}
	col := Column{
		Name:     strings.TrimSpace(m[1]),
		Type:     strings.TrimSpace(m[2]),
		Nullable: m[3] == "", // present means NOT NULL → not nullable
	}
	if m[4] != "" {
		col.Default = strings.TrimSpace(m[4])
	}
	return col, true
}

// parseIdentList splits "a, b, c" into ["a", "b", "c"], preserving
// the order. Only used for column lists in PK / index / FK clauses.
func parseIdentList(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

// canon returns the lookup key for a (schema, table) pair. Uses
// "public" as the default schema so dump output without an explicit
// schema qualifier still resolves.
func canon(schema, name string) string {
	if schema == "" {
		schema = "public"
	}
	return schema + "." + name
}

// firstLine returns the first non-empty line of s, used in error
// messages so they identify the offending statement without dumping
// the whole CREATE TABLE body.
func firstLine(s string) string {
	for _, line := range strings.Split(s, "\n") {
		if t := strings.TrimSpace(line); t != "" {
			if len(t) > 80 {
				return t[:77] + "..."
			}
			return t
		}
	}
	return ""
}
