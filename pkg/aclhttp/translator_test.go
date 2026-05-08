// Copyright 2026 NineVigil / Clawdlinux.
//
// Licensed under the Apache License, Version 2.0 (the "License").

package aclhttp

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Clawdlinux/ninevigil-acp/pkg/acl"
)

func mustRead(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("read %s: %v", name, err)
	}
	return b
}

func TestTranslatePetstore(t *testing.T) {
	t.Parallel()
	out, err := Encode(mustRead(t, "petstore.json"))
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}

	// Output must round-trip through the canonical decoder. This is the
	// central conformance check: any ACL the translator emits must be
	// re-readable by acl.Decode.
	if _, err := acl.Decode(out); err != nil {
		t.Fatalf("Decode(Encode(spec)): %v\noutput:\n%s", err, out)
	}

	// Headline directives must be present.
	for _, want := range []string{
		"@api Swagger", // sanitised; spaces -> "" or kept — see sanitize
		"@version 1.0.20",
		"@source openapi/v0.1",
	} {
		if !strings.Contains(string(out), want) {
			t.Errorf("missing directive fragment %q in output:\n%s", want, out)
		}
	}

	// Endpoint count and the canonical row form.
	if !strings.Contains(string(out), "endpoints 4") {
		t.Errorf("expected 'endpoints 4' in output:\n%s", out)
	}
	if !strings.Contains(string(out), "GET/pet/findByStatus op=findPetsByStatus") {
		t.Errorf("expected findPetsByStatus row in output:\n%s", out)
	}
	if !strings.Contains(string(out), "POST/pet op=addPet") {
		t.Errorf("expected addPet row in output:\n%s", out)
	}

	// Auth schemes captured.
	if !strings.Contains(string(out), "auth 2") {
		t.Errorf("expected 'auth 2' in output:\n%s", out)
	}
	if !strings.Contains(string(out), "api_key type=apiKey") {
		t.Errorf("expected api_key row in output:\n%s", out)
	}

	// Schemas captured with field counts.
	if !strings.Contains(string(out), "schemas 3") {
		t.Errorf("expected 'schemas 3' in output:\n%s", out)
	}
	if !strings.Contains(string(out), "Pet fields=6 required=2") {
		t.Errorf("expected Pet schema row in output:\n%s", out)
	}

	// Affordances row sorted, pipe-joined.
	if !strings.Contains(string(out), "actions\n  delete|get|post") {
		t.Errorf("expected sorted actions row in output:\n%s", out)
	}
}

func TestTranslateRejectsNonV3(t *testing.T) {
	t.Parallel()
	for _, src := range []string{
		`{"swagger":"2.0","info":{"title":"x","version":"1"}}`,
		`{"info":{"title":"x"}}`,
	} {
		_, err := Translate([]byte(src))
		if err == nil {
			t.Fatalf("expected error for %q", src)
		}
	}
}

func TestDeterministic(t *testing.T) {
	t.Parallel()
	spec := mustRead(t, "petstore.json")
	a, _ := Encode(spec)
	b, _ := Encode(spec)
	if string(a) != string(b) {
		t.Fatalf("translator output is not deterministic")
	}
}

// TestPetstoreCompressionRatio is the headline benchmark for the HTTP
// translator: how much smaller is the ACL view of an OpenAPI spec
// compared to the spec itself? At least a 4x reduction is required;
// the trimmed Petstore spec is far cleaner than real-world specs
// (Stripe, GitHub) that carry rich descriptions, examples, vendor
// extensions, so the realistic ratio in production will be larger.
func TestPetstoreCompressionRatio(t *testing.T) {
	t.Parallel()
	spec := mustRead(t, "petstore.json")
	out, err := Encode(spec)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	ratio := float64(len(spec)) / float64(len(out))
	t.Logf("OpenAPI bytes: %d (~%d tokens)", len(spec), len(spec)/4)
	t.Logf("ACL bytes:     %d (~%d tokens)", len(out), len(out)/4)
	t.Logf("compression:   %.1fx", ratio)
	const minRatio = 4.0
	if ratio < minRatio {
		t.Fatalf("expected >=%.1fx compression, got %.1fx", minRatio, ratio)
	}
}
