// SPDX-License-Identifier: Apache-2.0
// Package anf provides types and encoding for the Agent Native Format.
package anf

import "time"

// Document is a complete ANF view.
type Document struct {
	Headers  []Header
	Entities []Entity
	Alerts   []Alert
	Actions  []Action
}

// Header is an @-prefixed metadata line.
type Header struct {
	Key   string // e.g. "source", "scope", "time", "ttl"
	Value string
}

// Entity is a typed, named object with status and properties.
type Entity struct {
	Type        string // e.g. "deployment", "pod", "service"
	Name        string
	Status      Status
	InlineProps []Property // key:value pairs on the entity line (ordered)
	Props       []Property // indented property lines
	Children    []Entity   // nested child entities
}

// Status is a bracketed health marker.
type Status string

const (
	StatusHealthy    Status = "healthy"
	StatusDegraded   Status = "degraded"
	StatusFailing    Status = "failing"
	StatusPending    Status = "pending"
	StatusTerminated Status = "terminated"
	StatusUnknown    Status = "unknown"
	StatusRunning    Status = "running"   // pod-specific
	StatusCompleted  Status = "completed" // job-specific
	StatusEmpty      Status = ""          // no status
)

// Property is a key-value pair on an indented line.
type Property struct {
	Key   string
	Value string
}

// Alert is a !-prefixed warning or issue.
type Alert struct {
	Severity string // "critical", "warning", "info"
	Type     string // entity type it references
	Name     string // entity name it references
	Message  string
	Props    map[string]string
}

// Action is a ?-prefixed available operation.
type Action struct {
	Verb   string // e.g. "scale", "rollout", "restart", "logs"
	Type   string // target entity type
	Name   string // target entity name
	Params map[string]string
}

// Helpers for building documents.

func NewDocument(source, scope string, t time.Time) *Document {
	d := &Document{}
	d.Headers = append(d.Headers, Header{Key: "source", Value: source})
	if scope != "" {
		d.Headers = append(d.Headers, Header{Key: "scope", Value: scope})
	}
	d.Headers = append(d.Headers, Header{Key: "time", Value: t.UTC().Format(time.RFC3339)})
	return d
}

func (d *Document) SetTTL(seconds int) {
	d.Headers = append(d.Headers, Header{Key: "ttl", Value: formatDuration(seconds)})
}

func (d *Document) SetTranslator(name string) {
	d.Headers = append(d.Headers, Header{Key: "translator", Value: name})
}

func (d *Document) AddEntity(e Entity) {
	d.Entities = append(d.Entities, e)
}

func (d *Document) AddAlert(a Alert) {
	d.Alerts = append(d.Alerts, a)
}

func (d *Document) AddAction(a Action) {
	d.Actions = append(d.Actions, a)
}

func formatDuration(seconds int) string {
	if seconds >= 86400 && seconds%86400 == 0 {
		return itoa(seconds/86400) + "d"
	}
	if seconds >= 3600 && seconds%3600 == 0 {
		return itoa(seconds/3600) + "h"
	}
	if seconds >= 60 && seconds%60 == 0 {
		return itoa(seconds/60) + "m"
	}
	return itoa(seconds) + "s"
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	buf := [20]byte{}
	pos := len(buf)
	neg := n < 0
	if neg {
		n = -n
	}
	for n > 0 {
		pos--
		buf[pos] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		pos--
		buf[pos] = '-'
	}
	return string(buf[pos:])
}
