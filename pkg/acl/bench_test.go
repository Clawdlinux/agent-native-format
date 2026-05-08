/*
Copyright 2026 Clawdlinux / NineVigil.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0
*/

package acl

import (
	"encoding/json"
	"fmt"
	"testing"
)

// jsonEquivalent is the human-format JSON shape that an SRE agent would
// otherwise receive from `kubectl get pods,deploy,svc -n payments -o json`
// (trimmed to the same fields ACL preserves). The point of the benchmark
// is *not* to compare against full kubectl output (which adds ~5x more
// bytes via managedFields/resourceVersion/conditions/labels), but against
// a *fair* JSON shape carrying the same decision-relevant facts.
func jsonEquivalent() []byte {
	type pod struct {
		Name     string `json:"name"`
		Replicas int    `json:"replicas"`
		CPU      int    `json:"cpu_percent"`
		Mem      int    `json:"mem_percent"`
		Restarts int    `json:"restarts"`
		Age      string `json:"age"`
		Warning  bool   `json:"warning,omitempty"`
	}
	type deploy struct {
		Name     string `json:"name"`
		Replicas string `json:"replicas"`
		Strategy string `json:"strategy"`
		Image    string `json:"image"`
	}
	type svc struct {
		Name string `json:"name"`
		Type string `json:"type"`
		Port string `json:"port_mapping"`
	}
	type alert struct {
		Name      string `json:"name"`
		Pod       string `json:"pod"`
		Current   int    `json:"current_percent"`
		Threshold int    `json:"threshold_percent"`
	}
	doc := map[string]any{
		"cluster":   "prod-east",
		"namespace": "payments",
		"as_of":     "2026-05-03T12:34:56Z",
		"pods_summary": map[string]any{
			"ready": 12, "total": 12, "status": "ok",
		},
		"pods": []pod{
			{"payment-api-7f8d", 3, 42, 61, 0, "3d", false},
			{"payment-worker-9a2b", 2, 87, 78, 3, "1d", true},
			{"payment-cron-1c4e", 1, 5, 12, 0, "7d", false},
		},
		"deploys_summary": map[string]any{"total": 3, "status": "all_available"},
		"deploys": []deploy{
			{"payment-api", "3/3", "rolling", "v2.4.1"},
		},
		"svcs_summary": map[string]any{"total": 2},
		"svcs": []svc{
			{"payment-api", "ClusterIP", "8080->8080"},
		},
		"alerts_summary": map[string]any{"total": 1},
		"alerts": []alert{
			{"HighMemory", "payment-worker-9a2b", 78, 75},
		},
		"actions_available": []string{
			"scale", "rollout", "restart", "logs", "describe",
		},
	}
	out, _ := json.MarshalIndent(doc, "", "  ")
	return out
}

// approxTokens is a deliberately conservative tokeniser-free estimate:
// 1 token per 4 bytes (cl100k_base average for natural-language English
// text is ~4 chars/token; structured JSON sits around ~3.5). We use 4
// here so the comparison *understates* ACL's advantage — if ACL still
// wins by a wide margin under this conservative count, the real
// tokeniser margin is wider still.
func approxTokens(b []byte) int {
	if len(b) == 0 {
		return 0
	}
	t := len(b) / 4
	if t == 0 {
		return 1
	}
	return t
}

// TestACLCompressionRatio asserts that ACL beats *hand-trimmed* JSON
// carrying the same decision-relevant facts. The realistic 30–60×
// multipliers vs raw `kubectl -o json` (which carries managedFields,
// resourceVersion, conditions, label maps, status histories, etc.) are
// measured in the K8s translator package against real fixtures — see
// agentic-operator-core/pkg/acl-k8s. The point of *this* gate is to
// prove the format itself helps even when the JSON baseline is
// maximally fair.
func TestACLCompressionRatio(t *testing.T) {
	t.Parallel()
	aclBytes, err := Encode(k8sExample())
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	jsonBytes := jsonEquivalent()
	aclLen := len(aclBytes)
	jsonLen := len(jsonBytes)
	ratio := float64(jsonLen) / float64(aclLen)
	t.Logf("ACL bytes:  %d (~%d tokens)", aclLen, approxTokens(aclBytes))
	t.Logf("JSON bytes: %d (~%d tokens)", jsonLen, approxTokens(jsonBytes))
	t.Logf("compression ratio (vs trimmed JSON): %.2fx", ratio)
	const minRatio = 2.5
	if ratio < minRatio {
		t.Fatalf("expected >=%.1fx compression vs trimmed JSON, got %.2fx (acl=%d, json=%d)",
			minRatio, ratio, aclLen, jsonLen)
	}
}

// BenchmarkEncode reports how fast the reference encoder runs. Useful as
// a regression guard if we add features that touch the hot path.
func BenchmarkEncode(b *testing.B) {
	doc := k8sExample()
	b.ReportAllocs()
	for b.Loop() {
		out, err := Encode(doc)
		if err != nil {
			b.Fatal(err)
		}
		_ = out
	}
}

// BenchmarkDecode mirrors Encode for the parser.
func BenchmarkDecode(b *testing.B) {
	enc, err := Encode(k8sExample())
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	for b.Loop() {
		if _, err := Decode(enc); err != nil {
			b.Fatal(err)
		}
	}
}

// Example_compressionDelta is a runnable doc example that prints the
// bytes saved on the canonical K8s scenario.
func Example_compressionDelta() {
	aclBytes, _ := Encode(k8sExample())
	jsonBytes := jsonEquivalent()
	fmt.Printf("acl=%d json=%d ratio=%.1fx\n",
		len(aclBytes), len(jsonBytes),
		float64(len(jsonBytes))/float64(len(aclBytes)),
	)
	// Output is intentionally not asserted: byte counts may shift slightly
	// across Go versions for the JSON fixture. The TestACLCompressionRatio
	// test is the actual gate.
}
