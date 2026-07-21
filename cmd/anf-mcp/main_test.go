// SPDX-License-Identifier: Apache-2.0
package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"strings"
	"testing"
)

// TestSpecEmbedMatchesRoot fails if the embedded FORMAT.md drifts from the
// canonical spec at the repository root.
func TestSpecEmbedMatchesRoot(t *testing.T) {
	t.Parallel()

	root, err := os.ReadFile("../../FORMAT.md")
	if err != nil {
		t.Fatalf("read canonical FORMAT.md: %v", err)
	}
	if specMarkdown != string(root) {
		t.Errorf("embedded spec drifted from ../../FORMAT.md; re-copy it into cmd/anf-mcp/FORMAT.md")
	}
}

// TestBuildServerSmoke drives the fully wired server end-to-end over stdio: it
// lists tools, reads the spec resource, and encodes JSON via anf_encode.
func TestBuildServerSmoke(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(bytes.NewBuffer(nil), nil))
	s, err := buildServer(logger)
	if err != nil {
		t.Fatalf("buildServer: %v", err)
	}

	requests := strings.Join([]string{
		`{"jsonrpc":"2.0","id":1,"method":"tools/list"}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"anf_encode","arguments":{"data":{"name":"api","replicas":3},"scope":"svc"}}}`,
		`{"jsonrpc":"2.0","id":3,"method":"resources/read","params":{"uri":"anf://spec/format"}}`,
	}, "\n") + "\n"

	var out bytes.Buffer
	if err := s.Serve(context.Background(), strings.NewReader(requests), &out); err != nil {
		t.Fatalf("Serve: %v", err)
	}

	var responses []map[string]any
	sc := bufio.NewScanner(&out)
	sc.Buffer(make([]byte, 0, 64*1024), 16*1024*1024)
	for sc.Scan() {
		if len(sc.Bytes()) == 0 {
			continue
		}
		var m map[string]any
		if err := json.Unmarshal(sc.Bytes(), &m); err != nil {
			t.Fatalf("decode: %v", err)
		}
		responses = append(responses, m)
	}
	if len(responses) != 3 {
		t.Fatalf("want 3 responses, got %d", len(responses))
	}

	// tools/list must advertise both tools.
	toolsResult := responses[0]["result"].(map[string]any)
	tools := toolsResult["tools"].([]any)
	names := map[string]bool{}
	for _, tv := range tools {
		names[tv.(map[string]any)["name"].(string)] = true
	}
	if !names["anf_encode"] || !names["anf_encode_kubernetes"] {
		t.Errorf("missing expected tools, got %v", names)
	}

	// anf_encode must return ANF text containing the replicas property.
	callResult := responses[1]["result"].(map[string]any)
	content := callResult["content"].([]any)
	text := content[0].(map[string]any)["text"].(string)
	if !strings.Contains(text, "replicas 3") {
		t.Errorf("anf_encode output missing replicas:\n%s", text)
	}

	// resources/read must return the spec markdown.
	readResult := responses[2]["result"].(map[string]any)
	contents := readResult["contents"].([]any)
	specText := contents[0].(map[string]any)["text"].(string)
	if !strings.Contains(specText, "Agent Native Format") {
		t.Errorf("spec resource content unexpected:\n%s", specText[:min(80, len(specText))])
	}
}
