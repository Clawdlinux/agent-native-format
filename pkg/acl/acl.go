/*
Copyright 2026 Clawdlinux / NineVigil.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0
*/

package acl

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// indent is the exact two-space indentation required by the ACL spec.
const indent = "  "

// idAllowed reports whether r is permitted in a bare identifier or value.
// Matches [A-Za-z0-9_./:%@+-] from spec §6.1, plus '*' commonly seen in
// k8s wildcard contexts. References the spec; bring spec changes here.
func idAllowed(r rune) bool {
	switch {
	case r >= 'a' && r <= 'z':
		return true
	case r >= 'A' && r <= 'Z':
		return true
	case r >= '0' && r <= '9':
		return true
	}
	switch r {
	case '_', '.', '/', ':', '%', '@', '+', '-', '*', '|', '>', '{', '}', ',':
		// '|' lets the canonical `actions` row round-trip as a bare ID.
		// '>' covers the reserved `->` mapping token used in fields like
		// `port=8080->8080` and `ref=deploy->pod` (spec §3.2).
		// '{' '}' permit OpenAPI-style path templates like `/pet/{petId}`
		// to round-trip without quoting.
		// ',' lets parameter lists (`req=id,name`) stay bare.
		return true
	}
	return false
}

// needsQuote reports whether v must be backtick-quoted on the wire.
func needsQuote(v string) bool {
	if v == "" {
		return true
	}
	for _, r := range v {
		if !idAllowed(r) {
			return true
		}
	}
	return false
}

// quote returns the backtick-quoted form of v, escaping inner backticks
// per spec §6.2 (a backtick is escaped by doubling it).
func quote(v string) string {
	return "`" + strings.ReplaceAll(v, "`", "``") + "`"
}

// Encode serialises a Document into the canonical ACL byte form.
//
// Encoding is deterministic: identical inputs produce identical bytes,
// independent of map iteration order or platform. This makes ACL safe
// for content-addressable storage and stable agent caching.
func Encode(d Document) ([]byte, error) {
	var b bytes.Buffer
	for _, dir := range d.Directives {
		if dir.Key == "" {
			return nil, fmt.Errorf("acl: directive with empty key")
		}
		if strings.ContainsAny(dir.Key, " \t\n") {
			return nil, fmt.Errorf("acl: directive key %q contains whitespace", dir.Key)
		}
		if strings.ContainsAny(dir.Value, "\n") {
			return nil, fmt.Errorf("acl: directive %q value contains newline", dir.Key)
		}
		b.WriteByte('@')
		b.WriteString(dir.Key)
		if dir.Value != "" {
			b.WriteByte(' ')
			b.WriteString(dir.Value)
		}
		b.WriteByte('\n')
	}
	for i, s := range d.Sections {
		// Blank line between directives and first section, and between
		// successive sections.
		if i == 0 && len(d.Directives) > 0 {
			b.WriteByte('\n')
		} else if i > 0 {
			b.WriteByte('\n')
		}
		if err := writeSection(&b, s); err != nil {
			return nil, err
		}
	}
	return b.Bytes(), nil
}

func writeSection(b *bytes.Buffer, s Section) error {
	if s.Name == "" {
		return fmt.Errorf("acl: section with empty name")
	}
	if strings.ContainsAny(s.Name, " \t\n") {
		return fmt.Errorf("acl: section name %q contains whitespace", s.Name)
	}
	b.WriteString(s.Name)
	if s.Summary != "" {
		if strings.ContainsAny(s.Summary, "\n") {
			return fmt.Errorf("acl: section %q summary contains newline", s.Name)
		}
		b.WriteByte(' ')
		b.WriteString(s.Summary)
	}
	b.WriteByte('\n')
	for _, r := range s.Rows {
		if err := writeRow(b, r); err != nil {
			return fmt.Errorf("acl: section %q: %w", s.Name, err)
		}
	}
	return nil
}

func writeRow(b *bytes.Buffer, r Row) error {
	b.WriteString(indent)
	first := true
	if r.ID != "" {
		if needsQuote(r.ID) {
			return fmt.Errorf("row ID %q must match identifier charset", r.ID)
		}
		b.WriteString(r.ID)
		first = false
	}
	if r.Count > 0 {
		if !first {
			b.WriteByte(' ')
		}
		b.WriteByte('x')
		b.WriteString(strconv.Itoa(r.Count))
		first = false
	}
	for _, f := range r.Fields {
		if f.Key == "" {
			return fmt.Errorf("field with empty key")
		}
		if needsQuote(f.Key) {
			return fmt.Errorf("field key %q must match identifier charset", f.Key)
		}
		if !first {
			b.WriteByte(' ')
		}
		b.WriteString(f.Key)
		b.WriteByte('=')
		if needsQuote(f.Value) {
			b.WriteString(quote(f.Value))
		} else {
			b.WriteString(f.Value)
		}
		first = false
	}
	for _, fl := range r.Flags {
		if fl == "" {
			return fmt.Errorf("empty flag")
		}
		if strings.ContainsAny(fl, " \t\n=") {
			return fmt.Errorf("flag %q contains reserved character", fl)
		}
		if !first {
			b.WriteByte(' ')
		}
		b.WriteString(fl)
		first = false
	}
	if first {
		// Empty row — drop the indent, don't emit a stray "  \n".
		b.Truncate(b.Len() - len(indent))
		return nil
	}
	b.WriteByte('\n')
	return nil
}

// Decode parses canonical ACL bytes back into a Document. Comment lines
// (`# ...`) are stripped; blank lines act as section separators only.
func Decode(data []byte) (Document, error) {
	var d Document
	sc := bufio.NewScanner(bytes.NewReader(data))
	// Default Scanner buffer (64KB) is too small for big translator output.
	sc.Buffer(make([]byte, 0, 64*1024), 16*1024*1024)
	var cur *Section
	directivesAllowed := true
	lineNo := 0
	for sc.Scan() {
		lineNo++
		raw := sc.Text()
		if strings.ContainsRune(raw, '\t') {
			return Document{}, fmt.Errorf("acl: line %d: tab characters are forbidden", lineNo)
		}
		if strings.HasPrefix(raw, "# ") || raw == "#" {
			continue
		}
		if raw == "" {
			cur = nil
			continue
		}
		if strings.HasPrefix(raw, "@") {
			if !directivesAllowed {
				return Document{}, fmt.Errorf("acl: line %d: directive after section", lineNo)
			}
			dir, err := parseDirective(raw[1:])
			if err != nil {
				return Document{}, fmt.Errorf("acl: line %d: %w", lineNo, err)
			}
			d.Directives = append(d.Directives, dir)
			continue
		}
		directivesAllowed = false
		if strings.HasPrefix(raw, indent) {
			if cur == nil {
				return Document{}, fmt.Errorf("acl: line %d: row outside any section", lineNo)
			}
			row, err := parseRow(raw[len(indent):])
			if err != nil {
				return Document{}, fmt.Errorf("acl: line %d: %w", lineNo, err)
			}
			cur.Rows = append(cur.Rows, row)
			continue
		}
		if raw[0] == ' ' {
			return Document{}, fmt.Errorf("acl: line %d: indentation must be exactly two spaces", lineNo)
		}
		// Section header.
		sec, err := parseSection(raw)
		if err != nil {
			return Document{}, fmt.Errorf("acl: line %d: %w", lineNo, err)
		}
		d.Sections = append(d.Sections, sec)
		cur = &d.Sections[len(d.Sections)-1]
	}
	if err := sc.Err(); err != nil {
		return Document{}, fmt.Errorf("acl: scan: %w", err)
	}
	return d, nil
}

func parseDirective(s string) (Directive, error) {
	if s == "" {
		return Directive{}, errors.New("empty directive")
	}
	sp := strings.IndexByte(s, ' ')
	if sp < 0 {
		return Directive{Key: s}, nil
	}
	return Directive{Key: s[:sp], Value: s[sp+1:]}, nil
}

func parseSection(s string) (Section, error) {
	sp := strings.IndexByte(s, ' ')
	if sp < 0 {
		return Section{Name: s}, nil
	}
	return Section{Name: s[:sp], Summary: s[sp+1:]}, nil
}

// parseRow tokenises a row body (post-indent). Backtick-quoted values
// preserve embedded spaces; doubled backticks decode to a single backtick.
func parseRow(s string) (Row, error) {
	tokens, err := tokeniseRow(s)
	if err != nil {
		return Row{}, err
	}
	if len(tokens) == 0 {
		return Row{}, errors.New("empty row")
	}
	var r Row
	start := 0
	// First token may be the ID. It is the ID iff it does not contain '='
	// and does not match the xN count pattern.
	if !strings.ContainsRune(tokens[0], '=') && !isCountToken(tokens[0]) && !isFlagToken(tokens[0]) {
		r.ID = tokens[0]
		start = 1
	}
	if start < len(tokens) && isCountToken(tokens[start]) {
		n, err := strconv.Atoi(tokens[start][1:])
		if err != nil || n <= 0 {
			return Row{}, fmt.Errorf("invalid count token %q", tokens[start])
		}
		r.Count = n
		start++
	}
	for _, t := range tokens[start:] {
		if eq := strings.IndexByte(t, '='); eq >= 0 {
			key := t[:eq]
			val := t[eq+1:]
			val, err := unquoteValue(val)
			if err != nil {
				return Row{}, err
			}
			r.Fields = append(r.Fields, Field{Key: key, Value: val})
			continue
		}
		r.Flags = append(r.Flags, t)
	}
	return r, nil
}

// isCountToken reports whether t looks like xN where N>=1 (e.g. "x3").
func isCountToken(t string) bool {
	if len(t) < 2 || t[0] != 'x' {
		return false
	}
	for i := 1; i < len(t); i++ {
		if t[i] < '0' || t[i] > '9' {
			return false
		}
	}
	return true
}

// isFlagToken reports whether t is a reserved flag token (avoids treating
// a leading "!" or "?" as the row ID).
func isFlagToken(t string) bool {
	switch t {
	case "!", "!!", "?":
		return true
	}
	return false
}

// tokeniseRow splits a row body on spaces, treating backtick-quoted spans
// as single tokens.
func tokeniseRow(s string) ([]string, error) {
	var out []string
	var cur strings.Builder
	inQuote := false
	for i := 0; i < len(s); i++ {
		c := s[i]
		if inQuote {
			if c == '`' {
				if i+1 < len(s) && s[i+1] == '`' {
					cur.WriteByte('`')
					i++
					continue
				}
				inQuote = false
				cur.WriteByte('`') // keep markers; unquoteValue strips them
				continue
			}
			cur.WriteByte(c)
			continue
		}
		switch c {
		case ' ':
			if cur.Len() > 0 {
				out = append(out, cur.String())
				cur.Reset()
			}
		case '`':
			inQuote = true
			cur.WriteByte('`')
		default:
			cur.WriteByte(c)
		}
	}
	if inQuote {
		return nil, errors.New("unterminated backtick quote")
	}
	if cur.Len() > 0 {
		out = append(out, cur.String())
	}
	return out, nil
}

// unquoteValue unwraps a backtick-quoted value if it is one; otherwise
// returns the value unchanged. Escaped doubled-backticks have already been
// collapsed by tokeniseRow.
func unquoteValue(v string) (string, error) {
	if len(v) >= 2 && v[0] == '`' && v[len(v)-1] == '`' {
		return v[1 : len(v)-1], nil
	}
	if strings.ContainsRune(v, '`') {
		return "", fmt.Errorf("stray backtick in value %q", v)
	}
	return v, nil
}
