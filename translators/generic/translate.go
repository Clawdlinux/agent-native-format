// SPDX-License-Identifier: Apache-2.0

// Package generic provides a deterministic, structure-preserving translator
// that converts arbitrary decoded JSON into an ANF document. It performs a
// purely structural mapping: no fields are dropped and no semantics (health,
// status, alerts, or actions) are inferred. It only removes the syntactic
// overhead of JSON.
//
// Caveat: ANF is line-oriented and does not escape values. A string value that
// contains a newline is emitted verbatim, so byte-exact round-tripping of such
// values is not guaranteed. The facts are preserved; the exact framing of a
// multi-line value is not.
package generic

import (
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strconv"
	"time"

	"github.com/Clawdlinux/agent-native-format/pkg/anf"
)

// maxDepth bounds recursion so pathologically nested input cannot overflow the
// stack. Inputs nested deeper than this are rejected with an error.
const maxDepth = 512

// translatorName identifies this translator in the document header.
const translatorName = "clawdlinux/generic-translator"

// Translate converts an arbitrary decoded JSON value (the result of
// json.Unmarshal into any) into an ANF Document. It is deterministic and
// structure-preserving: no fields are dropped and no semantics are inferred.
// source and scope populate the document header; now sets the timestamp. It
// returns an error if the input nests deeper than maxDepth.
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
			child, err := entityFromValue(k, v[k], 1)
			if err != nil {
				return nil, err
			}
			doc.AddEntity(child)
		}
	case []any:
		e := anf.Entity{Type: "array", Name: scope, Status: anf.StatusEmpty}
		for _, elem := range v {
			child, err := entityFromValue("item", elem, 1)
			if err != nil {
				return nil, err
			}
			e.Children = append(e.Children, child)
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

// entityFromValue recursively maps a keyed JSON value into an ANF entity. depth
// tracks recursion; it returns an error if the input nests beyond maxDepth.
func entityFromValue(key string, value any, depth int) (anf.Entity, error) {
	if depth > maxDepth {
		return anf.Entity{}, fmt.Errorf("generic: input nesting exceeds %d levels", maxDepth)
	}
	switch v := value.(type) {
	case map[string]any:
		e := anf.Entity{Type: key, Name: objectName(v), Status: anf.StatusEmpty}
		for _, k := range sortedKeys(v) {
			if isContainer(v[k]) {
				child, err := entityFromValue(k, v[k], depth+1)
				if err != nil {
					return anf.Entity{}, err
				}
				e.Children = append(e.Children, child)
			} else {
				e.Props = append(e.Props, anf.Property{Key: k, Value: scalarString(v[k])})
			}
		}
		return e, nil
	case []any:
		e := anf.Entity{Type: key, Status: anf.StatusEmpty}
		for _, elem := range v {
			child, err := entityFromValue(key, elem, depth+1)
			if err != nil {
				return anf.Entity{}, err
			}
			e.Children = append(e.Children, child)
		}
		return e, nil
	default:
		return anf.Entity{
			Type:   key,
			Status: anf.StatusEmpty,
			Props:  []anf.Property{{Key: "value", Value: scalarString(value)}},
		}, nil
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
// else the "id" string if present, else "". The chosen key is still emitted as
// a property, so nothing is dropped.
func objectName(m map[string]any) string {
	if s, ok := m["name"].(string); ok {
		return s
	}
	if s, ok := m["id"].(string); ok {
		return s
	}
	return ""
}

// scalarString renders a JSON scalar as its ANF string form. Numbers decoded
// with json.Number are rendered verbatim so large integers are not corrupted.
func scalarString(v any) string {
	switch s := v.(type) {
	case string:
		return s
	case bool:
		if s {
			return "true"
		}
		return "false"
	case json.Number:
		return s.String()
	case float64:
		if !math.IsInf(s, 0) && !math.IsNaN(s) && s == math.Trunc(s) {
			return strconv.FormatFloat(s, 'f', -1, 64)
		}
		return strconv.FormatFloat(s, 'g', -1, 64)
	case nil:
		return "null"
	default:
		return fmt.Sprintf("%v", v)
	}
}
