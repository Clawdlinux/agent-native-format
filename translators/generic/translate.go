// SPDX-License-Identifier: Apache-2.0

// Package generic provides a deterministic, lossless translator that converts
// arbitrary decoded JSON into an ANF document. It performs a purely structural
// mapping: every key and scalar in the input appears exactly once in the output
// and no semantics (health, status, alerts, or actions) are inferred.
package generic

import (
	"math"
	"sort"
	"strconv"
	"time"

	"github.com/Clawdlinux/agent-native-format/pkg/anf"
)

// translatorName identifies this translator in the document header.
const translatorName = "clawdlinux/generic-translator"

// Translate converts an arbitrary decoded JSON value (the result of
// json.Unmarshal into any) into an ANF Document. It is deterministic and
// lossless: every key and scalar in the input appears exactly once in the
// output, and no semantics are inferred. source and scope populate the document
// header; now sets the timestamp.
func Translate(input any, source, scope string, now time.Time) (*anf.Document, error) {
	doc := anf.NewDocument(source, scope, now)
	doc.SetTranslator(translatorName)

	switch v := input.(type) {
	case map[string]any:
		scalars, containers := partitionKeys(v)
		if len(scalars) > 0 {
			e := anf.Entity{Type: "object", Name: scope, Status: anf.StatusEmpty}
			for _, k := range scalars {
				e.Props = append(e.Props, anf.Property{Key: k, Value: scalarString(v[k])})
			}
			doc.AddEntity(e)
		}
		for _, k := range containers {
			doc.AddEntity(entityFromValue(k, v[k]))
		}
	case []any:
		e := anf.Entity{Type: "array", Name: scope, Status: anf.StatusEmpty}
		for _, elem := range v {
			e.Children = append(e.Children, entityFromValue("item", elem))
		}
		doc.AddEntity(e)
	default:
		doc.AddEntity(anf.Entity{
			Type:   "value",
			Name:   scope,
			Status: anf.StatusEmpty,
			Props:  []anf.Property{{Key: "value", Value: scalarString(input)}},
		})
	}

	return doc, nil
}

// entityFromValue recursively maps a keyed JSON value into an ANF entity.
func entityFromValue(key string, value any) anf.Entity {
	switch v := value.(type) {
	case map[string]any:
		e := anf.Entity{Type: key, Name: objectName(v), Status: anf.StatusEmpty}
		for _, k := range sortedKeys(v) {
			if isContainer(v[k]) {
				e.Children = append(e.Children, entityFromValue(k, v[k]))
			} else {
				e.Props = append(e.Props, anf.Property{Key: k, Value: scalarString(v[k])})
			}
		}
		return e
	case []any:
		e := anf.Entity{Type: key, Status: anf.StatusEmpty}
		for _, elem := range v {
			if isContainer(elem) {
				e.Children = append(e.Children, entityFromValue(key, elem))
			} else {
				e.Children = append(e.Children, anf.Entity{
					Type:   key,
					Status: anf.StatusEmpty,
					Props:  []anf.Property{{Key: "value", Value: scalarString(elem)}},
				})
			}
		}
		return e
	default:
		return anf.Entity{
			Type:   key,
			Status: anf.StatusEmpty,
			Props:  []anf.Property{{Key: "value", Value: scalarString(value)}},
		}
	}
}

// partitionKeys splits an object's keys into scalar-valued and container-valued
// groups, each sorted lexicographically.
func partitionKeys(m map[string]any) (scalars, containers []string) {
	for k := range m {
		if isContainer(m[k]) {
			containers = append(containers, k)
		} else {
			scalars = append(scalars, k)
		}
	}
	sort.Strings(scalars)
	sort.Strings(containers)
	return scalars, containers
}

// sortedKeys returns the object's keys sorted lexicographically.
func sortedKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// isContainer reports whether v is an object or array (as opposed to a scalar).
func isContainer(v any) bool {
	switch v.(type) {
	case map[string]any, []any:
		return true
	default:
		return false
	}
}

// objectName returns a human name for an object: the "name" string if present,
// else the "id" string if present, else "". The chosen key remains a property
// and is not consumed here (losslessness is preserved by the caller).
func objectName(m map[string]any) string {
	if s, ok := m["name"].(string); ok {
		return s
	}
	if s, ok := m["id"].(string); ok {
		return s
	}
	return ""
}

// scalarString renders a JSON scalar as its ANF string form.
func scalarString(v any) string {
	switch s := v.(type) {
	case string:
		return s
	case bool:
		if s {
			return "true"
		}
		return "false"
	case float64:
		if !math.IsInf(s, 0) && !math.IsNaN(s) && s == math.Trunc(s) {
			return strconv.FormatFloat(s, 'f', -1, 64)
		}
		return strconv.FormatFloat(s, 'g', -1, 64)
	case nil:
		return "null"
	default:
		return "null"
	}
}
