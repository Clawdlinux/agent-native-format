// SPDX-License-Identifier: Apache-2.0

// Package anftools registers the ANF encoding tools on an anfmcp.Server. These
// tools let an MCP client turn verbose system state into token-minimal ANF for
// context engineering and token reduction.
package anftools

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Clawdlinux/agent-native-format/pkg/anf"
	"github.com/Clawdlinux/agent-native-format/pkg/anfmcp"
	"github.com/Clawdlinux/agent-native-format/translators/generic"
	k8s "github.com/Clawdlinux/agent-native-format/translators/kubernetes"
)

// Register adds the anf_encode and anf_encode_kubernetes tools to s.
func Register(s *anfmcp.Server) error {
	if err := s.RegisterTool(encodeTool()); err != nil {
		return err
	}
	if err := s.RegisterTool(kubernetesTool()); err != nil {
		return err
	}
	return nil
}

func encodeTool() anfmcp.Tool {
	return anfmcp.Tool{
		Name: "anf_encode",
		Description: "Encode an arbitrary JSON value as Agent Native Format (ANF), a " +
			"line-oriented, token-minimal representation. The mapping is lossless and " +
			"deterministic: same facts, far fewer tokens. Use it to compress verbose JSON " +
			"state before putting it in the context window.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"data": map[string]any{
					"description": "Any JSON value (object, array, or scalar) to encode as ANF.",
				},
				"source": map[string]any{
					"type":        "string",
					"description": "Optional label for the @source header (default \"agent-input\").",
				},
				"scope": map[string]any{
					"type":        "string",
					"description": "Optional label for the @scope header and root entity name.",
				},
			},
			"required": []any{"data"},
		},
		Handler: encodeHandler,
	}
}

func encodeHandler(_ context.Context, args map[string]any) (string, error) {
	data, ok := args["data"]
	if !ok {
		return "", fmt.Errorf("missing required argument: data")
	}
	source := stringArg(args, "source", "agent-input")
	scope := stringArg(args, "scope", "")

	doc, err := generic.Translate(data, source, scope, time.Now().UTC())
	if err != nil {
		return "", fmt.Errorf("encode: %w", err)
	}
	return anf.EncodeToString(doc), nil
}

func kubernetesTool() anfmcp.Tool {
	return anfmcp.Tool{
		Name: "anf_encode_kubernetes",
		Description: "Encode a Kubernetes namespace view as ANF using the domain " +
			"translator, which surfaces health, alerts, and available actions first. Input " +
			"is a namespace view object with cluster, namespace, and resource lists.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"view": map[string]any{
					"type": "object",
					"description": "A Kubernetes namespace view: cluster, namespace, deployments[], " +
						"services[], jobs[], cronjobs[], events[], agentPermissions{}.",
				},
			},
			"required": []any{"view"},
		},
		Handler: kubernetesHandler,
	}
}

func kubernetesHandler(_ context.Context, args map[string]any) (string, error) {
	raw, ok := args["view"]
	if !ok {
		return "", fmt.Errorf("missing required argument: view")
	}
	// Re-marshal the decoded value, then decode into the typed view. JSON field
	// matching is case-insensitive, so camelCase keys map onto the struct.
	b, err := json.Marshal(raw)
	if err != nil {
		return "", fmt.Errorf("marshal view: %w", err)
	}
	var view k8s.NamespaceView
	if err := json.Unmarshal(b, &view); err != nil {
		return "", fmt.Errorf("decode view: %w", err)
	}
	doc := k8s.Translate(view, time.Now().UTC())
	return anf.EncodeToString(doc), nil
}

func stringArg(args map[string]any, key, def string) string {
	if v, ok := args[key].(string); ok {
		return v
	}
	return def
}
