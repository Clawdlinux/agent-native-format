// SPDX-License-Identifier: Apache-2.0

// Package anfmcp is a small, dependency-free MCP server that speaks JSON-RPC
// 2.0 over stdio. It exists to expose ANF encoding as MCP tools so any
// MCP-capable agent (Claude, Cursor, Codex, VS Code) can turn verbose system
// state into token-minimal ANF.
//
// The server is stateless by construction. Tools and resources are registered
// once at startup and never mutate per client, and no session state is retained
// between requests. This matches the stateless-first direction of MCP SEP-2575:
// the server implements server/discover for version and capability discovery,
// and every request is handled in isolation. It also keeps the older initialize
// handshake so current clients (Claude, Cursor, VS Code), which still send it,
// interoperate. SEP-2575 explicitly permits a server to support both.
//
// The server is intentionally minimal. Register tools and resources, then call
// Serve. It handles server/discover, initialize, tools/list, tools/call,
// resources/list, resources/read, and ping. It has no third-party dependencies
// and is licensed Apache-2.0 so it is free to embed and ship.
package anfmcp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"sync"
)

// protocolVersion is the MCP protocol version this server implements. The
// server echoes the client's requested version when the client sends one.
const protocolVersion = "2025-06-18"

// ToolHandler runs a tool call. args is the decoded "arguments" object from the
// request. The returned string is delivered as text content. A non-nil error is
// reported to the caller as an MCP tool error (isError: true), not a transport
// error, so the agent can read and react to it.
type ToolHandler func(ctx context.Context, args map[string]any) (string, error)

// Tool is an MCP tool exposed by the server.
type Tool struct {
	Name        string
	Description string
	// InputSchema is a JSON Schema object describing the tool arguments. When
	// nil, an empty object schema is advertised.
	InputSchema map[string]any
	Handler     ToolHandler
}

// ResourceReader returns the body of a resource.
type ResourceReader func(ctx context.Context) (string, error)

// Resource is an MCP resource exposed by the server.
type Resource struct {
	URI         string
	Name        string
	Description string
	MimeType    string
	Read        ResourceReader
}

// Server is a minimal MCP stdio server.
type Server struct {
	name         string
	version      string
	instructions string
	logger       *slog.Logger

	mu        sync.RWMutex
	tools     map[string]Tool
	toolOrder []string
	resources map[string]Resource
	resOrder  []string
}

// Option configures a Server.
type Option func(*Server)

// WithLogger sets the logger used for diagnostics. Logs go to the provided
// handler, never to stdout, so they cannot corrupt the JSON-RPC stream.
func WithLogger(l *slog.Logger) Option {
	return func(s *Server) {
		if l != nil {
			s.logger = l
		}
	}
}

// WithInstructions sets natural-language usage guidance returned by
// server/discover. A client may add it to a system prompt to help an LLM use
// the server's tools well.
func WithInstructions(text string) Option {
	return func(s *Server) { s.instructions = text }
}

// NewServer creates a Server that identifies itself with name and version.
func NewServer(name, version string, opts ...Option) *Server {
	s := &Server{
		name:      name,
		version:   version,
		logger:    slog.New(slog.NewTextHandler(io.Discard, nil)),
		tools:     make(map[string]Tool),
		resources: make(map[string]Resource),
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// RegisterTool adds a tool. It returns an error if the tool name is empty, the
// handler is nil, or a tool with the same name is already registered.
func (s *Server) RegisterTool(t Tool) error {
	if t.Name == "" {
		return fmt.Errorf("anfmcp: tool name is empty")
	}
	if t.Handler == nil {
		return fmt.Errorf("anfmcp: tool %q has nil handler", t.Name)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.tools[t.Name]; exists {
		return fmt.Errorf("anfmcp: tool %q already registered", t.Name)
	}
	s.tools[t.Name] = t
	s.toolOrder = append(s.toolOrder, t.Name)
	return nil
}

// RegisterResource adds a resource. It returns an error if the URI is empty,
// the reader is nil, or a resource with the same URI is already registered.
func (s *Server) RegisterResource(r Resource) error {
	if r.URI == "" {
		return fmt.Errorf("anfmcp: resource URI is empty")
	}
	if r.Read == nil {
		return fmt.Errorf("anfmcp: resource %q has nil reader", r.URI)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.resources[r.URI]; exists {
		return fmt.Errorf("anfmcp: resource %q already registered", r.URI)
	}
	s.resources[r.URI] = r
	s.resOrder = append(s.resOrder, r.URI)
	return nil
}

// JSON-RPC 2.0 wire types.

type rpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  any             `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// JSON-RPC standard error codes.
const (
	codeParseError     = -32700
	codeInvalidRequest = -32600
	codeMethodNotFound = -32601
	codeInvalidParams  = -32602
	codeInternalError  = -32603
)

// Serve reads newline-delimited JSON-RPC requests from in and writes responses
// to out until in reaches EOF or ctx is cancelled. Notifications (requests with
// no id) receive no response. Serve is single-threaded: one request is handled
// at a time, which matches the stdio transport and keeps output ordered.
func (s *Server) Serve(ctx context.Context, in io.Reader, out io.Writer) error {
	scanner := bufio.NewScanner(in)
	// Allow large payloads (system state can be big). 16 MiB line cap.
	scanner.Buffer(make([]byte, 0, 64*1024), 16*1024*1024)

	enc := json.NewEncoder(out)

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var req rpcRequest
		if err := json.Unmarshal(line, &req); err != nil {
			// Cannot recover an id from an unparseable message; reply with a
			// null-id parse error per JSON-RPC.
			s.write(enc, rpcResponse{
				JSONRPC: "2.0",
				ID:      json.RawMessage("null"),
				Error:   &rpcError{Code: codeParseError, Message: "parse error"},
			})
			continue
		}

		resp, respond := s.handle(ctx, req)
		if respond {
			s.write(enc, resp)
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("anfmcp: read: %w", err)
	}
	return nil
}

// handle dispatches one request. The second return reports whether a response
// should be written (false for notifications).
func (s *Server) handle(ctx context.Context, req rpcRequest) (rpcResponse, bool) {
	// A request without an id is a notification. JSON-RPC 2.0 forbids replying
	// to notifications. Every method here is side-effect free, so dropping a
	// notification loses nothing.
	if len(req.ID) == 0 {
		return rpcResponse{}, false
	}

	switch req.Method {
	case "server/discover":
		// SEP-2575 stateless discovery. Preferred over initialize.
		return s.ok(req.ID, s.discoverResult()), true
	case "initialize":
		// Legacy handshake, retained for current clients that still send it.
		return s.ok(req.ID, s.initializeResult(req.Params)), true
	case "ping":
		return s.ok(req.ID, map[string]any{}), true
	case "tools/list":
		return s.ok(req.ID, s.toolsList()), true
	case "tools/call":
		return s.toolsCall(ctx, req), true
	case "resources/list":
		return s.ok(req.ID, s.resourcesList()), true
	case "resources/read":
		return s.resourcesRead(ctx, req), true
	default:
		return s.fail(req.ID, codeMethodNotFound, "method not found: "+req.Method), true
	}
}

// discoverResult builds the server/discover response defined by SEP-2575:
// supported protocol versions, capabilities, server identity, and optional
// usage instructions.
func (s *Server) discoverResult() map[string]any {
	res := map[string]any{
		"supportedVersions": []string{protocolVersion},
		"capabilities": map[string]any{
			"tools":     map[string]any{},
			"resources": map[string]any{},
		},
		"serverInfo": map[string]any{
			"name":    s.name,
			"version": s.version,
		},
	}
	if s.instructions != "" {
		res["instructions"] = s.instructions
	}
	return res
}

func (s *Server) initializeResult(params json.RawMessage) map[string]any {
	version := protocolVersion
	if len(params) > 0 {
		var p struct {
			ProtocolVersion string `json:"protocolVersion"`
		}
		if err := json.Unmarshal(params, &p); err == nil && p.ProtocolVersion != "" {
			version = p.ProtocolVersion
		}
	}
	return map[string]any{
		"protocolVersion": version,
		"capabilities": map[string]any{
			"tools":     map[string]any{},
			"resources": map[string]any{},
		},
		"serverInfo": map[string]any{
			"name":    s.name,
			"version": s.version,
		},
	}
}

func (s *Server) toolsList() map[string]any {
	s.mu.RLock()
	defer s.mu.RUnlock()

	list := make([]map[string]any, 0, len(s.toolOrder))
	for _, name := range s.toolOrder {
		t := s.tools[name]
		schema := t.InputSchema
		if schema == nil {
			schema = map[string]any{"type": "object"}
		}
		list = append(list, map[string]any{
			"name":        t.Name,
			"description": t.Description,
			"inputSchema": schema,
		})
	}
	return map[string]any{"tools": list}
}

type toolCallParams struct {
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments"`
}

func (s *Server) toolsCall(ctx context.Context, req rpcRequest) rpcResponse {
	var p toolCallParams
	// Decode with UseNumber so large integers survive as json.Number rather
	// than being rounded through float64.
	dec := json.NewDecoder(bytes.NewReader(req.Params))
	dec.UseNumber()
	if err := dec.Decode(&p); err != nil {
		return s.fail(req.ID, codeInvalidParams, "invalid params: "+err.Error())
	}
	if p.Name == "" {
		return s.fail(req.ID, codeInvalidParams, "missing tool name")
	}

	s.mu.RLock()
	tool, ok := s.tools[p.Name]
	s.mu.RUnlock()
	if !ok {
		return s.fail(req.ID, codeInvalidParams, "unknown tool: "+p.Name)
	}

	text, err := tool.Handler(ctx, p.Arguments)
	if err != nil {
		s.logger.Warn("tool call failed", "tool", p.Name, "error", err)
		return s.ok(req.ID, toolResult(err.Error(), true))
	}
	return s.ok(req.ID, toolResult(text, false))
}

// toolResult builds an MCP tools/call result payload.
func toolResult(text string, isError bool) map[string]any {
	return map[string]any{
		"content": []map[string]any{
			{"type": "text", "text": text},
		},
		"isError": isError,
	}
}

func (s *Server) resourcesList() map[string]any {
	s.mu.RLock()
	defer s.mu.RUnlock()

	list := make([]map[string]any, 0, len(s.resOrder))
	for _, uri := range s.resOrder {
		r := s.resources[uri]
		list = append(list, map[string]any{
			"uri":         r.URI,
			"name":        r.Name,
			"description": r.Description,
			"mimeType":    r.MimeType,
		})
	}
	return map[string]any{"resources": list}
}

type resourceReadParams struct {
	URI string `json:"uri"`
}

func (s *Server) resourcesRead(ctx context.Context, req rpcRequest) rpcResponse {
	var p resourceReadParams
	if err := json.Unmarshal(req.Params, &p); err != nil {
		return s.fail(req.ID, codeInvalidParams, "invalid params: "+err.Error())
	}
	if p.URI == "" {
		return s.fail(req.ID, codeInvalidParams, "missing resource uri")
	}

	s.mu.RLock()
	res, ok := s.resources[p.URI]
	s.mu.RUnlock()
	if !ok {
		return s.fail(req.ID, codeInvalidParams, "unknown resource: "+p.URI)
	}

	body, err := res.Read(ctx)
	if err != nil {
		s.logger.Warn("resource read failed", "uri", p.URI, "error", err)
		return s.fail(req.ID, codeInternalError, "read resource: "+err.Error())
	}
	return s.ok(req.ID, map[string]any{
		"contents": []map[string]any{
			{"uri": res.URI, "mimeType": res.MimeType, "text": body},
		},
	})
}

// ok builds a success response.
func (s *Server) ok(id json.RawMessage, result any) rpcResponse {
	return rpcResponse{JSONRPC: "2.0", ID: normalizeID(id), Result: result}
}

// fail builds an error response.
func (s *Server) fail(id json.RawMessage, code int, message string) rpcResponse {
	return rpcResponse{JSONRPC: "2.0", ID: normalizeID(id), Error: &rpcError{Code: code, Message: message}}
}

// normalizeID returns the id as-is, or JSON null when absent, so the response
// always carries an id field per JSON-RPC 2.0.
func normalizeID(id json.RawMessage) json.RawMessage {
	if len(id) == 0 {
		return json.RawMessage("null")
	}
	return id
}

// write encodes a response. Encode appends a newline, giving newline-delimited
// framing. Errors are logged, not returned, so one bad write does not kill the
// serve loop.
func (s *Server) write(enc *json.Encoder, resp rpcResponse) {
	if err := enc.Encode(resp); err != nil {
		s.logger.Error("write response", "error", err)
	}
}
