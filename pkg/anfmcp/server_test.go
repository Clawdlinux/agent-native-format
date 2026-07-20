// SPDX-License-Identifier: Apache-2.0
package anfmcp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

// runServer feeds newline-delimited JSON-RPC request lines through a fresh
// Server (configured by setup) and returns the decoded responses in order.
func runServer(t *testing.T, setup func(*Server), requests ...string) []rpcResponse {
	t.Helper()

	s := NewServer("anf-mcp", "test", WithInstructions("use anf_encode to shrink JSON"))
	if setup != nil {
		setup(s)
	}

	in := strings.NewReader(strings.Join(requests, "\n") + "\n")
	var out bytes.Buffer
	if err := s.Serve(context.Background(), in, &out); err != nil {
		t.Fatalf("Serve: %v", err)
	}

	var responses []rpcResponse
	sc := bufio.NewScanner(&out)
	sc.Buffer(make([]byte, 0, 64*1024), 16*1024*1024)
	for sc.Scan() {
		line := sc.Bytes()
		if len(line) == 0 {
			continue
		}
		var resp rpcResponse
		if err := json.Unmarshal(line, &resp); err != nil {
			t.Fatalf("decode response %q: %v", line, err)
		}
		responses = append(responses, resp)
	}
	if err := sc.Err(); err != nil {
		t.Fatalf("scan output: %v", err)
	}
	return responses
}

// resultMap re-decodes a response's Result into a map for assertions.
func resultMap(t *testing.T, r rpcResponse) map[string]any {
	t.Helper()
	raw, err := json.Marshal(r.Result)
	if err != nil {
		t.Fatalf("marshal result: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	return m
}

func echoTool() Tool {
	return Tool{
		Name:        "echo",
		Description: "echoes its text argument",
		InputSchema: map[string]any{"type": "object"},
		Handler: func(_ context.Context, args map[string]any) (string, error) {
			s, _ := args["text"].(string)
			return s, nil
		},
	}
}

func TestServerDiscover(t *testing.T) {
	t.Parallel()

	resp := runServer(t, func(s *Server) {
		if err := s.RegisterTool(echoTool()); err != nil {
			t.Fatalf("register: %v", err)
		}
	}, `{"jsonrpc":"2.0","id":1,"method":"server/discover"}`)

	if len(resp) != 1 {
		t.Fatalf("want 1 response, got %d", len(resp))
	}
	m := resultMap(t, resp[0])
	versions, ok := m["supportedVersions"].([]any)
	if !ok || len(versions) == 0 || versions[0] != protocolVersion {
		t.Errorf("supportedVersions = %v, want [%s]", m["supportedVersions"], protocolVersion)
	}
	if m["instructions"] != "use anf_encode to shrink JSON" {
		t.Errorf("instructions = %v", m["instructions"])
	}
	info, ok := m["serverInfo"].(map[string]any)
	if !ok || info["name"] != "anf-mcp" {
		t.Errorf("serverInfo = %v", m["serverInfo"])
	}
	if _, ok := m["capabilities"].(map[string]any); !ok {
		t.Errorf("capabilities missing: %v", m)
	}
}

func TestServerInitializeEchoesVersion(t *testing.T) {
	t.Parallel()

	resp := runServer(t, nil,
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05"}}`)

	m := resultMap(t, resp[0])
	if m["protocolVersion"] != "2024-11-05" {
		t.Errorf("protocolVersion = %v, want echoed 2024-11-05", m["protocolVersion"])
	}
}

func TestServerToolsList(t *testing.T) {
	t.Parallel()

	resp := runServer(t, func(s *Server) {
		_ = s.RegisterTool(echoTool())
	}, `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`)

	m := resultMap(t, resp[0])
	tools, ok := m["tools"].([]any)
	if !ok || len(tools) != 1 {
		t.Fatalf("tools = %v", m["tools"])
	}
	tool := tools[0].(map[string]any)
	if tool["name"] != "echo" {
		t.Errorf("tool name = %v", tool["name"])
	}
	if _, ok := tool["inputSchema"].(map[string]any); !ok {
		t.Errorf("inputSchema missing: %v", tool)
	}
}

func TestServerToolsCallSuccess(t *testing.T) {
	t.Parallel()

	resp := runServer(t, func(s *Server) {
		_ = s.RegisterTool(echoTool())
	}, `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"echo","arguments":{"text":"hi"}}}`)

	m := resultMap(t, resp[0])
	if m["isError"] != false {
		t.Errorf("isError = %v, want false", m["isError"])
	}
	content := m["content"].([]any)
	first := content[0].(map[string]any)
	if first["type"] != "text" || first["text"] != "hi" {
		t.Errorf("content = %v", content)
	}
}

func TestServerToolsCallHandlerError(t *testing.T) {
	t.Parallel()

	resp := runServer(t, func(s *Server) {
		_ = s.RegisterTool(Tool{
			Name: "boom",
			Handler: func(_ context.Context, _ map[string]any) (string, error) {
				return "", errors.New("kaboom")
			},
		})
	}, `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"boom","arguments":{}}}`)

	m := resultMap(t, resp[0])
	if m["isError"] != true {
		t.Fatalf("isError = %v, want true", m["isError"])
	}
	content := m["content"].([]any)
	first := content[0].(map[string]any)
	if first["text"] != "kaboom" {
		t.Errorf("error text = %v", first["text"])
	}
	if resp[0].Error != nil {
		t.Errorf("handler error must be a tool error, not a transport error: %v", resp[0].Error)
	}
}

func TestServerToolsCallUnknownTool(t *testing.T) {
	t.Parallel()

	resp := runServer(t, nil,
		`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"nope","arguments":{}}}`)

	if resp[0].Error == nil || resp[0].Error.Code != codeInvalidParams {
		t.Errorf("want invalid params error, got %+v", resp[0].Error)
	}
}

func TestServerResources(t *testing.T) {
	t.Parallel()

	setup := func(s *Server) {
		if err := s.RegisterResource(Resource{
			URI:      "anf://spec",
			Name:     "ANF spec",
			MimeType: "text/markdown",
			Read: func(_ context.Context) (string, error) {
				return "# ANF", nil
			},
		}); err != nil {
			t.Fatalf("register resource: %v", err)
		}
	}

	resp := runServer(t, setup,
		`{"jsonrpc":"2.0","id":1,"method":"resources/list"}`,
		`{"jsonrpc":"2.0","id":2,"method":"resources/read","params":{"uri":"anf://spec"}}`,
		`{"jsonrpc":"2.0","id":3,"method":"resources/read","params":{"uri":"anf://missing"}}`,
	)

	if len(resp) != 3 {
		t.Fatalf("want 3 responses, got %d", len(resp))
	}

	list := resultMap(t, resp[0])
	if res, ok := list["resources"].([]any); !ok || len(res) != 1 {
		t.Errorf("resources/list = %v", list["resources"])
	}

	read := resultMap(t, resp[1])
	contents := read["contents"].([]any)
	first := contents[0].(map[string]any)
	if first["text"] != "# ANF" || first["uri"] != "anf://spec" {
		t.Errorf("resources/read = %v", contents)
	}

	if resp[2].Error == nil || resp[2].Error.Code != codeInvalidParams {
		t.Errorf("unknown resource want invalid params, got %+v", resp[2].Error)
	}
}

func TestServerPingAndUnknownMethod(t *testing.T) {
	t.Parallel()

	resp := runServer(t, nil,
		`{"jsonrpc":"2.0","id":1,"method":"ping"}`,
		`{"jsonrpc":"2.0","id":2,"method":"does/notexist"}`,
	)

	if resp[0].Error != nil {
		t.Errorf("ping error: %v", resp[0].Error)
	}
	if resp[1].Error == nil || resp[1].Error.Code != codeMethodNotFound {
		t.Errorf("unknown method want -32601, got %+v", resp[1].Error)
	}
}

func TestServerNotificationHasNoResponse(t *testing.T) {
	t.Parallel()

	// A request without an id is a notification; it must produce no response.
	resp := runServer(t, nil,
		`{"jsonrpc":"2.0","method":"notifications/initialized"}`,
		`{"jsonrpc":"2.0","id":1,"method":"ping"}`,
	)

	if len(resp) != 1 {
		t.Fatalf("want 1 response (ping only), got %d", len(resp))
	}
	if string(resp[0].ID) != "1" {
		t.Errorf("response id = %s, want 1", resp[0].ID)
	}
}

func TestServerParseError(t *testing.T) {
	t.Parallel()

	resp := runServer(t, nil, `{not valid json`)

	if len(resp) != 1 {
		t.Fatalf("want 1 response, got %d", len(resp))
	}
	if resp[0].Error == nil || resp[0].Error.Code != codeParseError {
		t.Errorf("want parse error -32700, got %+v", resp[0].Error)
	}
	if string(resp[0].ID) != "null" {
		t.Errorf("parse error id = %s, want null", resp[0].ID)
	}
}

func TestRegisterToolValidation(t *testing.T) {
	t.Parallel()

	s := NewServer("t", "0")
	if err := s.RegisterTool(Tool{Name: "", Handler: func(context.Context, map[string]any) (string, error) { return "", nil }}); err == nil {
		t.Error("empty name should error")
	}
	if err := s.RegisterTool(Tool{Name: "x"}); err == nil {
		t.Error("nil handler should error")
	}
	if err := s.RegisterTool(echoTool()); err != nil {
		t.Fatalf("first register: %v", err)
	}
	if err := s.RegisterTool(echoTool()); err == nil {
		t.Error("duplicate name should error")
	}
}

func TestRegisterResourceValidation(t *testing.T) {
	t.Parallel()

	s := NewServer("t", "0")
	if err := s.RegisterResource(Resource{URI: "", Read: func(context.Context) (string, error) { return "", nil }}); err == nil {
		t.Error("empty uri should error")
	}
	if err := s.RegisterResource(Resource{URI: "u"}); err == nil {
		t.Error("nil reader should error")
	}
	ok := Resource{URI: "u", Read: func(context.Context) (string, error) { return "", nil }}
	if err := s.RegisterResource(ok); err != nil {
		t.Fatalf("first register: %v", err)
	}
	if err := s.RegisterResource(ok); err == nil {
		t.Error("duplicate uri should error")
	}
}
