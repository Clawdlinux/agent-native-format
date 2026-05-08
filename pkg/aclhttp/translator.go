// Copyright 2026 NineVigil / Clawdlinux.
//
// Licensed under the Apache License, Version 2.0 (the "License").

package aclhttp

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/Clawdlinux/ninevigil-acp/pkg/acl"
)

// translatorID is the value emitted in the @source directive.
const translatorID = "openapi/v0.1"

// TranslatorID returns the @source identifier this translator emits.
func TranslatorID() string { return translatorID }

// ─── Minimal OpenAPI subset ─────────────────────────────────────────────────
//
// We deliberately do NOT pull in a full OpenAPI library. The translator
// reads only the fields it needs, treats the rest as opaque, and is
// version-tolerant within the OpenAPI 3.x family.

type openAPI struct {
	OpenAPI    string                     `json:"openapi"`
	Info       openAPIInfo                `json:"info"`
	Paths      map[string]openAPIPathItem `json:"paths"`
	Components openAPIComponents          `json:"components"`
	Security   []map[string][]string      `json:"security"`
	Servers    []openAPIServer            `json:"servers"`
}

type openAPIInfo struct {
	Title   string `json:"title"`
	Version string `json:"version"`
}

type openAPIServer struct {
	URL string `json:"url"`
}

type openAPIPathItem struct {
	Get     *openAPIOperation `json:"get"`
	Put     *openAPIOperation `json:"put"`
	Post    *openAPIOperation `json:"post"`
	Delete  *openAPIOperation `json:"delete"`
	Patch   *openAPIOperation `json:"patch"`
	Head    *openAPIOperation `json:"head"`
	Options *openAPIOperation `json:"options"`
}

type openAPIOperation struct {
	OperationID string                `json:"operationId"`
	Tags        []string              `json:"tags"`
	Parameters  []openAPIParameter    `json:"parameters"`
	RequestBody *openAPIRequestBody   `json:"requestBody"`
	Responses   map[string]openAPIRef `json:"responses"`
	Security    []map[string][]string `json:"security"`
	Deprecated  bool                  `json:"deprecated"`
}

type openAPIParameter struct {
	Name     string        `json:"name"`
	In       string        `json:"in"` // path | query | header | cookie
	Required bool          `json:"required"`
	Schema   openAPISchema `json:"schema"`
}

type openAPIRequestBody struct {
	Required bool                        `json:"required"`
	Content  map[string]openAPIMediaType `json:"content"`
}

type openAPIMediaType struct {
	Schema openAPISchema `json:"schema"`
}

type openAPISchema struct {
	Type string `json:"type"`
	Ref  string `json:"$ref"`
}

type openAPIRef struct {
	Ref     string                      `json:"$ref"`
	Content map[string]openAPIMediaType `json:"content"`
}

type openAPIComponents struct {
	Schemas         map[string]openAPIComponentSchema `json:"schemas"`
	SecuritySchemes map[string]openAPISecurityScheme  `json:"securitySchemes"`
}

type openAPIComponentSchema struct {
	Type       string                   `json:"type"`
	Required   []string                 `json:"required"`
	Properties map[string]openAPISchema `json:"properties"`
}

type openAPISecurityScheme struct {
	Type         string `json:"type"`   // apiKey | http | oauth2 | openIdConnect
	Scheme       string `json:"scheme"` // bearer | basic | digest (when type=http)
	In           string `json:"in"`     // header | query | cookie (when type=apiKey)
	Name         string `json:"name"`
	BearerFormat string `json:"bearerFormat"`
}

// ─── Public API ─────────────────────────────────────────────────────────────

// Translate parses an OpenAPI 3.x JSON specification and returns the
// equivalent ACL document.
//
// The translator is tolerant of unknown / extra fields (it only reads
// what it needs) and version-tolerant within the OpenAPI 3.x family.
// It does NOT follow $ref chains across files; only same-document
// references resolve.
func Translate(spec []byte) (acl.Document, error) {
	var s openAPI
	if err := json.Unmarshal(spec, &s); err != nil {
		return acl.Document{}, fmt.Errorf("aclhttp: parse spec: %w", err)
	}
	if !strings.HasPrefix(s.OpenAPI, "3.") {
		return acl.Document{}, fmt.Errorf("aclhttp: expected OpenAPI 3.x, got %q", s.OpenAPI)
	}

	d := acl.Document{Directives: directives(s)}
	if sec, ok := authSection(s.Components.SecuritySchemes); ok {
		d.Sections = append(d.Sections, sec)
	}
	if sec, ok := endpointsSection(s.Paths); ok {
		d.Sections = append(d.Sections, sec)
	}
	if sec, ok := schemasSection(s.Components.Schemas); ok {
		d.Sections = append(d.Sections, sec)
	}
	if sec, ok := actionsSection(s.Paths); ok {
		d.Sections = append(d.Sections, sec)
	}
	return d, nil
}

// Encode is a one-shot helper that translates and serialises in one
// call. Equivalent to calling acl.Encode(Translate(spec)).
func Encode(spec []byte) ([]byte, error) {
	d, err := Translate(spec)
	if err != nil {
		return nil, err
	}
	return acl.Encode(d)
}

// ─── Section builders ───────────────────────────────────────────────────────

func directives(s openAPI) []acl.Directive {
	var out []acl.Directive
	if s.Info.Title != "" {
		out = append(out, acl.Directive{Key: "api", Value: sanitize(s.Info.Title)})
	}
	if s.Info.Version != "" {
		out = append(out, acl.Directive{Key: "version", Value: sanitize(s.Info.Version)})
	}
	if len(s.Servers) > 0 && s.Servers[0].URL != "" {
		out = append(out, acl.Directive{Key: "server", Value: sanitize(s.Servers[0].URL)})
	}
	out = append(out, acl.Directive{Key: "source", Value: translatorID})
	return out
}

// authSection emits one row per security scheme. Agents need to know
// what auth types are available before they pick an endpoint.
func authSection(schemes map[string]openAPISecurityScheme) (acl.Section, bool) {
	if len(schemes) == 0 {
		return acl.Section{}, false
	}
	names := sortedKeys(schemes)
	rows := make([]acl.Row, 0, len(names))
	for _, name := range names {
		sc := schemes[name]
		fields := []acl.Field{{Key: "type", Value: sanitize(sc.Type)}}
		if sc.Scheme != "" {
			fields = append(fields, acl.Field{Key: "scheme", Value: sanitize(sc.Scheme)})
		}
		if sc.In != "" {
			fields = append(fields, acl.Field{Key: "in", Value: sanitize(sc.In)})
		}
		if sc.Name != "" {
			fields = append(fields, acl.Field{Key: "name", Value: sanitize(sc.Name)})
		}
		rows = append(rows, acl.Row{ID: sanitize(name), Fields: fields})
	}
	return acl.Section{Name: "auth", Summary: fmt.Sprintf("%d", len(rows)), Rows: rows}, true
}

// endpointsSection is the most important one: every operation an agent
// can call. We emit a stable row order (path then method) so the doc
// is byte-deterministic across runs.
func endpointsSection(paths map[string]openAPIPathItem) (acl.Section, bool) {
	if len(paths) == 0 {
		return acl.Section{}, false
	}
	type opEntry struct {
		method string
		path   string
		op     *openAPIOperation
	}
	var ops []opEntry
	for path, item := range paths {
		for _, m := range []struct {
			name string
			op   *openAPIOperation
		}{
			{"GET", item.Get},
			{"POST", item.Post},
			{"PUT", item.Put},
			{"PATCH", item.Patch},
			{"DELETE", item.Delete},
			{"HEAD", item.Head},
			{"OPTIONS", item.Options},
		} {
			if m.op != nil {
				ops = append(ops, opEntry{method: m.name, path: path, op: m.op})
			}
		}
	}
	sort.Slice(ops, func(i, j int) bool {
		if ops[i].path != ops[j].path {
			return ops[i].path < ops[j].path
		}
		return ops[i].method < ops[j].method
	})

	rows := make([]acl.Row, 0, len(ops))
	for _, e := range ops {
		rows = append(rows, endpointRow(e.method, e.path, e.op))
	}
	return acl.Section{Name: "endpoints", Summary: fmt.Sprintf("%d", len(rows)), Rows: rows}, true
}

func endpointRow(method, path string, op *openAPIOperation) acl.Row {
	id := fmt.Sprintf("%s%s", method, path)
	fields := []acl.Field{}
	if op.OperationID != "" {
		fields = append(fields, acl.Field{Key: "op", Value: sanitize(op.OperationID)})
	}
	if reqList := requiredParamList(op.Parameters); reqList != "" {
		fields = append(fields, acl.Field{Key: "req", Value: reqList})
	}
	if optList := optionalParamList(op.Parameters); optList != "" {
		fields = append(fields, acl.Field{Key: "opt", Value: optList})
	}
	if op.RequestBody != nil {
		bodyRef := requestBodyRef(op.RequestBody)
		fields = append(fields, acl.Field{Key: "body", Value: bodyRef})
	}
	if rs := successResponseRef(op.Responses); rs != "" {
		fields = append(fields, acl.Field{Key: "returns", Value: rs})
	}
	if len(op.Security) > 0 {
		fields = append(fields, acl.Field{Key: "auth", Value: securityName(op.Security[0])})
	}
	row := acl.Row{ID: sanitize(id), Fields: fields}
	if op.Deprecated {
		row.Flags = []string{"!"}
	}
	return row
}

// requiredParamList returns a comma-separated list of required param
// names, or "" if none. Order is the order the params appear in the
// spec — preserves the API author's intent.
func requiredParamList(params []openAPIParameter) string {
	var names []string
	for _, p := range params {
		if p.Required {
			names = append(names, p.Name)
		}
	}
	return strings.Join(names, ",")
}

func optionalParamList(params []openAPIParameter) string {
	var names []string
	for _, p := range params {
		if !p.Required {
			names = append(names, p.Name)
		}
	}
	return strings.Join(names, ",")
}

func requestBodyRef(rb *openAPIRequestBody) string {
	for _, mt := range rb.Content {
		if ref := refTail(mt.Schema.Ref); ref != "" {
			return ref
		}
		if mt.Schema.Type != "" {
			return mt.Schema.Type
		}
	}
	return "?"
}

// successResponseRef returns the schema ref of the first 2xx response.
// Agents care about the success-path return shape; non-2xx responses
// are noise for plan-time reasoning.
func successResponseRef(responses map[string]openAPIRef) string {
	codes := sortedKeys(responses)
	for _, code := range codes {
		if !strings.HasPrefix(code, "2") {
			continue
		}
		r := responses[code]
		for _, mt := range r.Content {
			if tail := refTail(mt.Schema.Ref); tail != "" {
				return tail
			}
			if mt.Schema.Type != "" {
				return mt.Schema.Type
			}
		}
	}
	return ""
}

func securityName(sec map[string][]string) string {
	for k := range sec {
		return sanitize(k)
	}
	return "?"
}

// schemasSection emits one row per top-level component schema with its
// required-field list. Property type details are intentionally dropped
// — an agent can fetch a specific schema's full definition on demand,
// it doesn't need the universe of types upfront.
func schemasSection(schemas map[string]openAPIComponentSchema) (acl.Section, bool) {
	if len(schemas) == 0 {
		return acl.Section{}, false
	}
	names := sortedKeys(schemas)
	rows := make([]acl.Row, 0, len(names))
	for _, name := range names {
		s := schemas[name]
		fields := []acl.Field{
			{Key: "fields", Value: fmt.Sprintf("%d", len(s.Properties))},
			{Key: "required", Value: fmt.Sprintf("%d", len(s.Required))},
		}
		rows = append(rows, acl.Row{ID: sanitize(name), Fields: fields})
	}
	return acl.Section{Name: "schemas", Summary: fmt.Sprintf("%d", len(rows)), Rows: rows}, true
}

// actionsSection is the closed verb set the agent may invoke — the set
// of HTTP methods present in the spec. Mirrors the K8s translator's
// affordance contract.
func actionsSection(paths map[string]openAPIPathItem) (acl.Section, bool) {
	verbs := map[string]bool{}
	for _, item := range paths {
		if item.Get != nil {
			verbs["get"] = true
		}
		if item.Post != nil {
			verbs["post"] = true
		}
		if item.Put != nil {
			verbs["put"] = true
		}
		if item.Patch != nil {
			verbs["patch"] = true
		}
		if item.Delete != nil {
			verbs["delete"] = true
		}
	}
	if len(verbs) == 0 {
		return acl.Section{}, false
	}
	keys := make([]string, 0, len(verbs))
	for k := range verbs {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return acl.Section{
		Name: "actions",
		Rows: []acl.Row{{ID: strings.Join(keys, "|")}},
	}, true
}

// ─── helpers ────────────────────────────────────────────────────────────────

// refTail extracts the tail of a JSON-pointer $ref.
// "#/components/schemas/Pet" -> "Pet"
func refTail(ref string) string {
	if ref == "" {
		return ""
	}
	if idx := strings.LastIndex(ref, "/"); idx >= 0 {
		return ref[idx+1:]
	}
	return ref
}

// sanitize replaces characters the ACL identifier charset can't carry
// in a bare value with hyphens. Used for human-authored fields like
// titles and operation IDs that may contain spaces or punctuation.
//
// Note: '{' and '}' are NOT replaced — they are valid in the ACL
// charset specifically so OpenAPI-style path templates round-trip
// unchanged.
func sanitize(s string) string {
	if s == "" {
		return s
	}
	var b strings.Builder
	for _, r := range s {
		switch {
		case (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9'):
			b.WriteRune(r)
		case r == '_' || r == '.' || r == '/' || r == ':' || r == '-' || r == '+' || r == '@' || r == '{' || r == '}' || r == ',':
			b.WriteRune(r)
		default:
			b.WriteByte('-')
		}
	}
	return b.String()
}

// sortedKeys returns the keys of a map in lexicographic order so
// translator output is byte-deterministic.
func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
