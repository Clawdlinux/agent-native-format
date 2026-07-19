// SPDX-License-Identifier: Apache-2.0
package generic

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"github.com/Clawdlinux/agent-native-format/pkg/anf"
)

// fixedTime is a stable timestamp for deterministic header output.
var fixedTime = time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)

// TestTranslateEntities checks the structural mapping of arbitrary JSON into
// ANF entities. Each case asserts on the resulting Document.Entities so the
// deterministic, lossless mapping rules are verified directly.
func TestTranslateEntities(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input any
		scope string
		want  []anf.Entity
	}{
		{
			name:  "flat object with only scalars",
			input: map[string]any{"b": "x", "a": float64(3), "c": true},
			scope: "root",
			want: []anf.Entity{
				{Type: "object", Name: "root", Props: []anf.Property{
					{Key: "a", Value: "3"},
					{Key: "b", Value: "x"},
					{Key: "c", Value: "true"},
				}},
			},
		},
		{
			name:  "nested object keeps name and props",
			input: map[string]any{"deploy": map[string]any{"name": "api", "replicas": float64(3)}},
			scope: "root",
			want: []anf.Entity{
				{Type: "deploy", Name: "api", Props: []anf.Property{
					{Key: "name", Value: "api"},
					{Key: "replicas", Value: "3"},
				}},
			},
		},
		{
			name:  "id used as name and kept as prop",
			input: map[string]any{"svc": map[string]any{"id": "i1", "port": float64(80)}},
			scope: "root",
			want: []anf.Entity{
				{Type: "svc", Name: "i1", Props: []anf.Property{
					{Key: "id", Value: "i1"},
					{Key: "port", Value: "80"},
				}},
			},
		},
		{
			name:  "name preferred over id both kept",
			input: map[string]any{"svc": map[string]any{"id": "i1", "name": "apppp"}},
			scope: "root",
			want: []anf.Entity{
				{Type: "svc", Name: "apppp", Props: []anf.Property{
					{Key: "id", Value: "i1"},
					{Key: "name", Value: "apppp"},
				}},
			},
		},
		{
			name:  "array of objects",
			input: map[string]any{"pods": []any{map[string]any{"name": "p1"}, map[string]any{"name": "p2"}}},
			scope: "root",
			want: []anf.Entity{
				{Type: "pods", Children: []anf.Entity{
					{Type: "pods", Name: "p1", Props: []anf.Property{{Key: "name", Value: "p1"}}},
					{Type: "pods", Name: "p2", Props: []anf.Property{{Key: "name", Value: "p2"}}},
				}},
			},
		},
		{
			name:  "array of scalars",
			input: map[string]any{"tags": []any{"a", float64(2)}},
			scope: "root",
			want: []anf.Entity{
				{Type: "tags", Children: []anf.Entity{
					{Type: "tags", Props: []anf.Property{{Key: "value", Value: "a"}}},
					{Type: "tags", Props: []anf.Property{{Key: "value", Value: "2"}}},
				}},
			},
		},
		{
			name:  "root array",
			input: []any{map[string]any{"name": "x"}, "y"},
			scope: "root",
			want: []anf.Entity{
				{Type: "array", Name: "root", Children: []anf.Entity{
					{Type: "item", Name: "x", Props: []anf.Property{{Key: "name", Value: "x"}}},
					{Type: "item", Props: []anf.Property{{Key: "value", Value: "y"}}},
				}},
			},
		},
		{
			name:  "root scalar string",
			input: "hello",
			scope: "root",
			want: []anf.Entity{
				{Type: "value", Name: "root", Props: []anf.Property{{Key: "value", Value: "hello"}}},
			},
		},
		{
			name:  "root scalar number",
			input: float64(3),
			scope: "root",
			want: []anf.Entity{
				{Type: "value", Name: "root", Props: []anf.Property{{Key: "value", Value: "3"}}},
			},
		},
		{
			name:  "mixed scalar and container top level",
			input: map[string]any{"z": "scal", "a": map[string]any{"k": "v"}, "m": float64(2)},
			scope: "root",
			want: []anf.Entity{
				{Type: "object", Name: "root", Props: []anf.Property{
					{Key: "m", Value: "2"},
					{Key: "z", Value: "scal"},
				}},
				{Type: "a", Props: []anf.Property{{Key: "k", Value: "v"}}},
			},
		},
		{
			name:  "float bool and null formatting",
			input: map[string]any{"f": float64(3.5), "g": float64(3.0), "b": false, "n": nil},
			scope: "root",
			want: []anf.Entity{
				{Type: "object", Name: "root", Props: []anf.Property{
					{Key: "b", Value: "false"},
					{Key: "f", Value: "3.5"},
					{Key: "g", Value: "3"},
					{Key: "n", Value: "null"},
				}},
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			doc, err := Translate(tt.input, "test", tt.scope, fixedTime)
			if err != nil {
				t.Fatalf("Translate returned error: %v", err)
			}
			if !reflect.DeepEqual(doc.Entities, tt.want) {
				t.Errorf("entities mismatch\n got: %#v\nwant: %#v", doc.Entities, tt.want)
			}
		})
	}
}

// TestTranslateSetsTranslatorHeader verifies the translator name header is set.
func TestTranslateSetsTranslatorHeader(t *testing.T) {
	t.Parallel()

	doc, err := Translate(map[string]any{"a": "b"}, "test", "root", fixedTime)
	if err != nil {
		t.Fatalf("Translate returned error: %v", err)
	}

	var found bool
	for _, h := range doc.Headers {
		if h.Key == "translator" {
			found = true
			if h.Value != "clawdlinux/generic-translator" {
				t.Errorf("translator header = %q, want clawdlinux/generic-translator", h.Value)
			}
		}
	}
	if !found {
		t.Errorf("translator header not set; headers: %#v", doc.Headers)
	}
}

// TestTranslateNoSemanticInvention ensures the generic translator never emits
// status, alerts, or actions.
func TestTranslateNoSemanticInvention(t *testing.T) {
	t.Parallel()

	input := map[string]any{
		"status":  "failing",
		"healthy": false,
		"pods":    []any{map[string]any{"name": "p1", "phase": "Running"}},
	}
	doc, err := Translate(input, "test", "root", fixedTime)
	if err != nil {
		t.Fatalf("Translate returned error: %v", err)
	}
	if len(doc.Alerts) != 0 {
		t.Errorf("expected no alerts, got %d", len(doc.Alerts))
	}
	if len(doc.Actions) != 0 {
		t.Errorf("expected no actions, got %d", len(doc.Actions))
	}
	var walk func(e anf.Entity)
	walk = func(e anf.Entity) {
		if e.Status != anf.StatusEmpty {
			t.Errorf("entity %q has non-empty status %q", e.Type, e.Status)
		}
		for _, c := range e.Children {
			walk(c)
		}
	}
	for _, e := range doc.Entities {
		walk(e)
	}
}

// TestTranslateEncodeGolden asserts the full ANF text for a nested object.
func TestTranslateEncodeGolden(t *testing.T) {
	t.Parallel()

	input := map[string]any{"deploy": map[string]any{"name": "api", "replicas": float64(3)}}
	doc, err := Translate(input, "test", "root", fixedTime)
	if err != nil {
		t.Fatalf("Translate returned error: %v", err)
	}

	want := "@source test\n" +
		"@scope root\n" +
		"@time 2026-01-02T03:04:05Z\n" +
		"@translator clawdlinux/generic-translator\n" +
		"\n" +
		"deploy api\n" +
		"  name api\n" +
		"  replicas 3\n"

	if got := anf.EncodeToString(doc); got != want {
		t.Errorf("encoded output mismatch\n got:\n%s\nwant:\n%s", got, want)
	}
}

// TestTranslateDeterministic checks that the same input yields identical output
// across repeated calls, including from real json.Unmarshal decoding.
func TestTranslateDeterministic(t *testing.T) {
	t.Parallel()

	raw := []byte(`{"z":"last","a":1,"m":{"k":"v","j":2},"b":[3,2,1]}`)

	var in1, in2 any
	if err := json.Unmarshal(raw, &in1); err != nil {
		t.Fatalf("unmarshal in1: %v", err)
	}
	if err := json.Unmarshal(raw, &in2); err != nil {
		t.Fatalf("unmarshal in2: %v", err)
	}

	doc1, err := Translate(in1, "test", "root", fixedTime)
	if err != nil {
		t.Fatalf("Translate in1: %v", err)
	}
	doc2, err := Translate(in2, "test", "root", fixedTime)
	if err != nil {
		t.Fatalf("Translate in2: %v", err)
	}

	if anf.EncodeToString(doc1) != anf.EncodeToString(doc2) {
		t.Errorf("output not deterministic\nfirst:\n%s\nsecond:\n%s",
			anf.EncodeToString(doc1), anf.EncodeToString(doc2))
	}
}
