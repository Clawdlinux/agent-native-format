// SPDX-License-Identifier: Apache-2.0
package anftools

import (
	"context"
	"strings"
	"testing"

	"github.com/Clawdlinux/agent-native-format/pkg/anfmcp"
)

func TestEncodeHandler(t *testing.T) {
	t.Parallel()

	out, err := encodeHandler(context.Background(), map[string]any{
		"data":  map[string]any{"name": "api", "replicas": float64(3)},
		"scope": "svc",
	})
	if err != nil {
		t.Fatalf("encodeHandler: %v", err)
	}
	if !strings.Contains(out, "replicas 3") {
		t.Errorf("missing replicas property in output:\n%s", out)
	}
	if !strings.Contains(out, "@scope svc") {
		t.Errorf("missing @scope header in output:\n%s", out)
	}
	if !strings.Contains(out, "@source agent-input") {
		t.Errorf("missing default @source header in output:\n%s", out)
	}
}

func TestEncodeHandlerMissingData(t *testing.T) {
	t.Parallel()

	if _, err := encodeHandler(context.Background(), map[string]any{}); err == nil {
		t.Error("expected error when data argument is missing")
	}
}

func TestKubernetesHandler(t *testing.T) {
	t.Parallel()

	view := map[string]any{
		"cluster":   "prod",
		"namespace": "payments",
		"deployments": []any{
			map[string]any{
				"name":          "api",
				"replicas":      float64(3),
				"readyReplicas": float64(3),
			},
		},
	}
	out, err := kubernetesHandler(context.Background(), map[string]any{"view": view})
	if err != nil {
		t.Fatalf("kubernetesHandler: %v", err)
	}
	if !strings.Contains(out, "deployment api") {
		t.Errorf("missing deployment entity in output:\n%s", out)
	}
}

func TestKubernetesHandlerMissingView(t *testing.T) {
	t.Parallel()

	if _, err := kubernetesHandler(context.Background(), map[string]any{}); err == nil {
		t.Error("expected error when view argument is missing")
	}
}

func TestRegisterWiresTools(t *testing.T) {
	t.Parallel()

	s := anfmcp.NewServer("anf-mcp", "test")
	if err := Register(s); err != nil {
		t.Fatalf("Register: %v", err)
	}
	// Registering again must fail because the tool names are already taken,
	// which confirms both tools were registered the first time.
	if err := Register(s); err == nil {
		t.Error("expected duplicate registration to error")
	}
}
