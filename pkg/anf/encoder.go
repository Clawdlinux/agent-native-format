// SPDX-License-Identifier: Apache-2.0
package anf

import (
	"io"
	"sort"
	"strings"
)

// Encode writes an ANF document to w.
func Encode(w io.Writer, doc *Document) error {
	var b strings.Builder

	// Headers
	for _, h := range doc.Headers {
		b.WriteByte('@')
		b.WriteString(h.Key)
		b.WriteByte(' ')
		b.WriteString(h.Value)
		b.WriteByte('\n')
	}
	if len(doc.Headers) > 0 {
		b.WriteByte('\n')
	}

	// Entities
	for _, e := range doc.Entities {
		writeEntity(&b, e, 0)
	}

	// Alerts
	if len(doc.Alerts) > 0 {
		b.WriteByte('\n')
		for _, a := range doc.Alerts {
			writeAlert(&b, a)
		}
	}

	// Actions
	if len(doc.Actions) > 0 {
		b.WriteByte('\n')
		for _, a := range doc.Actions {
			writeAction(&b, a)
		}
	}

	_, err := io.WriteString(w, b.String())
	return err
}

// EncodeToString returns the ANF document as a string.
func EncodeToString(doc *Document) string {
	var b strings.Builder
	Encode(&b, doc)
	return b.String()
}

func writeEntity(b *strings.Builder, e Entity, depth int) {
	writeIndent(b, depth)
	b.WriteString(e.Type)
	b.WriteByte(' ')
	b.WriteString(e.Name)

	if e.Status != StatusEmpty {
		b.WriteString(" [")
		b.WriteString(string(e.Status))
		b.WriteByte(']')
	}

	// Inline props (ordered)
	for _, p := range e.InlineProps {
		b.WriteByte(' ')
		b.WriteString(p.Key)
		b.WriteByte(':')
		b.WriteString(p.Value)
	}
	b.WriteByte('\n')

	// Indented props
	for _, p := range e.Props {
		writeIndent(b, depth+1)
		b.WriteString(p.Key)
		b.WriteByte(' ')
		b.WriteString(p.Value)
		b.WriteByte('\n')
	}

	// Children
	for _, child := range e.Children {
		writeEntity(b, child, depth+1)
	}
}

func writeAlert(b *strings.Builder, a Alert) {
	b.WriteByte('!')
	b.WriteString(a.Severity)
	b.WriteByte(' ')
	b.WriteString(a.Type)
	b.WriteByte(' ')
	b.WriteString(a.Name)
	b.WriteByte(' ')
	b.WriteString(a.Message)
	for _, k := range sortedMapKeys(a.Props) {
		b.WriteByte(' ')
		b.WriteString(k)
		b.WriteByte(':')
		b.WriteString(a.Props[k])
	}
	b.WriteByte('\n')
}

func writeAction(b *strings.Builder, a Action) {
	b.WriteByte('?')
	b.WriteString(a.Verb)
	b.WriteByte(' ')
	b.WriteString(a.Type)
	b.WriteByte(' ')
	b.WriteString(a.Name)
	for _, k := range sortedMapKeys(a.Params) {
		b.WriteByte(' ')
		b.WriteString(k)
		b.WriteByte(':')
		b.WriteString(a.Params[k])
	}
	b.WriteByte('\n')
}

func writeIndent(b *strings.Builder, depth int) {
	for i := 0; i < depth; i++ {
		b.WriteString("  ")
	}
}

// sortedMapKeys returns a map's keys in lexicographic order so alert and action
// property output is deterministic.
func sortedMapKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
