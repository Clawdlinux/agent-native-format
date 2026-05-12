// SPDX-License-Identifier: Apache-2.0
package kubernetes

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/Clawdlinux/ninevigil-acp/pkg/anf"
)

// TestTranslatePaymentsNamespace produces the example from FORMAT.md and
// measures token counts against equivalent JSON.
func TestTranslatePaymentsNamespace(t *testing.T) {
	now := time.Date(2026, 5, 3, 10, 30, 0, 0, time.UTC)

	view := NamespaceView{
		Cluster:   "prod-east",
		Namespace: "payments",
		Deployments: []Deployment{
			{
				Name:           "payment-api",
				Replicas:       3,
				ReadyReplicas:  3,
				Image:          "registry.io/payments/api:v2.4.1",
				Strategy:       "rolling",
				MaxSurge:       "1",
				MaxUnavailable: "0",
				AgeDays:        14,
				CPUPercent:     42,
				MemPercent:     61,
				RequestsPerSec: 1200,
				ErrorRate:      0.02,
				LatestImage:    "registry.io/payments/api:v2.5.0",
				Pods: []Pod{
					{Name: "payment-api-7f8d", Phase: "Running", Node: "worker-01", CPUPercent: 45, MemPercent: 63, Restarts: 0, AgeDays: 14},
					{Name: "payment-api-9a2b", Phase: "Running", Node: "worker-02", CPUPercent: 38, MemPercent: 55, Restarts: 0, AgeDays: 14},
					{Name: "payment-api-1c4e", Phase: "Running", Node: "worker-01", CPUPercent: 44, MemPercent: 64, Restarts: 0, AgeDays: 14},
				},
			},
			{
				Name:          "payment-worker",
				Replicas:      2,
				ReadyReplicas: 2,
				Image:         "registry.io/payments/worker:v1.8.0",
				AgeDays:       8,
				CPUPercent:    87,
				MemPercent:    78,
				Pods: []Pod{
					{Name: "payment-worker-a3f1", Phase: "Running", Node: "worker-03", CPUPercent: 87, MemPercent: 78, Restarts: 5, Restarts24h: 3, AgeDays: 8},
					{Name: "payment-worker-b7e2", Phase: "Running", Node: "worker-03", CPUPercent: 82, MemPercent: 74, Restarts: 2, Restarts24h: 1, AgeDays: 8},
				},
			},
		},
		Services: []Service{
			{Name: "payment-api", Type: "ClusterIP", Port: 8080, TargetPort: 8080, Endpoints: 3, TotalPods: 3},
			{Name: "payment-grpc", Type: "ClusterIP", Port: 9090, TargetPort: 9090, Endpoints: 2, TotalPods: 2},
		},
		Jobs: []Job{
			{Name: "daily-reconciliation", Completed: true, LastRun: now.Add(-4*time.Hour - 30*time.Minute), Duration: 4 * time.Minute, Succeeded: true},
		},
		CronJobs: []CronJob{
			{Name: "hourly-sync", Schedule: "0_*_*_*_*", LastRun: now.Add(-30 * time.Minute), NextRun: now.Add(30 * time.Minute)},
		},
		AgentPermissions: Permissions{
			CanScale:    true,
			CanRollout:  true,
			CanRestart:  true,
			CanLogs:     true,
			CanExec:     true,
			CanDescribe: true,
		},
	}

	// Translate to ANF
	doc := Translate(view, now)
	anfOutput := anf.EncodeToString(doc)

	// Also produce equivalent JSON for comparison
	jsonOutput := toJSON(view)

	// Token estimation (chars/4 heuristic — conservative for ANF, generous for JSON)
	anfTokens := len(anfOutput) / 4
	jsonTokens := len(jsonOutput) / 4

	fmt.Println("=== ANF Output ===")
	fmt.Println(anfOutput)
	fmt.Println("=== Token Comparison ===")
	fmt.Printf("ANF:  %d chars, ~%d tokens\n", len(anfOutput), anfTokens)
	fmt.Printf("JSON: %d chars, ~%d tokens\n", len(jsonOutput), jsonTokens)
	fmt.Printf("Reduction: %.1f%%\n", float64(jsonTokens-anfTokens)/float64(jsonTokens)*100)

	// Assertions
	if anfTokens == 0 {
		t.Fatal("ANF output is empty")
	}
	if anfTokens >= jsonTokens {
		t.Fatalf("ANF (%d tokens) should be smaller than JSON (%d tokens)", anfTokens, jsonTokens)
	}

	// ANF should contain key structural elements
	if !strings.Contains(anfOutput, "@source kubernetes/prod-east") {
		t.Error("missing @source header")
	}
	if !strings.Contains(anfOutput, "deployment payment-api [healthy]") {
		t.Error("missing healthy deployment entity")
	}
	if !strings.Contains(anfOutput, "deployment payment-worker [degraded]") {
		t.Error("missing degraded deployment entity")
	}
	if !strings.Contains(anfOutput, "!warning") {
		t.Error("missing warning alerts")
	}
	if !strings.Contains(anfOutput, "?scale") {
		t.Error("missing scale action")
	}
	if !strings.Contains(anfOutput, "?rollout") {
		t.Error("missing rollout action")
	}

	// Reduction should be at least 80%
	reduction := float64(jsonTokens-anfTokens) / float64(jsonTokens) * 100
	if reduction < 80 {
		t.Errorf("token reduction %.1f%% is below 80%% threshold", reduction)
	}
}

// toJSON produces a realistic Kubernetes API JSON representation for
// the same namespace state. This simulates what `kubectl get all -o json`
// returns — including all the metadata, status, and spec fields that
// agents have to parse today.
func toJSON(view NamespaceView) string {
	type k8sPod struct {
		APIVersion string `json:"apiVersion"`
		Kind       string `json:"kind"`
		Metadata   struct {
			Name              string            `json:"name"`
			Namespace         string            `json:"namespace"`
			Labels            map[string]string `json:"labels"`
			CreationTimestamp string            `json:"creationTimestamp"`
			UID               string            `json:"uid"`
			ResourceVersion   string            `json:"resourceVersion"`
		} `json:"metadata"`
		Spec struct {
			NodeName   string `json:"nodeName"`
			Containers []struct {
				Name      string `json:"name"`
				Image     string `json:"image"`
				Resources struct {
					Requests map[string]string `json:"requests"`
					Limits   map[string]string `json:"limits"`
				} `json:"resources"`
				Ports []struct {
					ContainerPort int    `json:"containerPort"`
					Protocol      string `json:"protocol"`
				} `json:"ports"`
			} `json:"containers"`
		} `json:"spec"`
		Status struct {
			Phase      string `json:"phase"`
			Conditions []struct {
				Type               string `json:"type"`
				Status             string `json:"status"`
				LastTransitionTime string `json:"lastTransitionTime"`
			} `json:"conditions"`
			ContainerStatuses []struct {
				Name         string `json:"name"`
				RestartCount int32  `json:"restartCount"`
				Ready        bool   `json:"ready"`
				State        struct {
					Running struct {
						StartedAt string `json:"startedAt"`
					} `json:"running"`
				} `json:"state"`
			} `json:"containerStatuses"`
		} `json:"status"`
	}

	type k8sList struct {
		APIVersion string        `json:"apiVersion"`
		Kind       string        `json:"kind"`
		Items      []interface{} `json:"items"`
	}

	items := []interface{}{}

	for _, dep := range view.Deployments {
		for _, pod := range dep.Pods {
			p := k8sPod{}
			p.APIVersion = "v1"
			p.Kind = "Pod"
			p.Metadata.Name = pod.Name
			p.Metadata.Namespace = view.Namespace
			p.Metadata.Labels = map[string]string{
				"app":                          dep.Name,
				"pod-template-hash":            "7f8d9a2b",
				"app.kubernetes.io/name":       dep.Name,
				"app.kubernetes.io/managed-by": "Helm",
				"app.kubernetes.io/version":    "2.4.1",
			}
			p.Metadata.CreationTimestamp = "2026-04-19T10:30:00Z"
			p.Metadata.UID = "a1b2c3d4-e5f6-7890-abcd-ef1234567890"
			p.Metadata.ResourceVersion = "12345678"
			p.Spec.NodeName = pod.Node
			p.Spec.Containers = []struct {
				Name      string `json:"name"`
				Image     string `json:"image"`
				Resources struct {
					Requests map[string]string `json:"requests"`
					Limits   map[string]string `json:"limits"`
				} `json:"resources"`
				Ports []struct {
					ContainerPort int    `json:"containerPort"`
					Protocol      string `json:"protocol"`
				} `json:"ports"`
			}{
				{
					Name:  dep.Name,
					Image: dep.Image,
					Resources: struct {
						Requests map[string]string `json:"requests"`
						Limits   map[string]string `json:"limits"`
					}{
						Requests: map[string]string{"cpu": "250m", "memory": "256Mi"},
						Limits:   map[string]string{"cpu": "1000m", "memory": "512Mi"},
					},
					Ports: []struct {
						ContainerPort int    `json:"containerPort"`
						Protocol      string `json:"protocol"`
					}{
						{ContainerPort: 8080, Protocol: "TCP"},
					},
				},
			}
			p.Status.Phase = pod.Phase
			p.Status.Conditions = []struct {
				Type               string `json:"type"`
				Status             string `json:"status"`
				LastTransitionTime string `json:"lastTransitionTime"`
			}{
				{Type: "Initialized", Status: "True", LastTransitionTime: "2026-04-19T10:30:00Z"},
				{Type: "Ready", Status: "True", LastTransitionTime: "2026-04-19T10:30:05Z"},
				{Type: "ContainersReady", Status: "True", LastTransitionTime: "2026-04-19T10:30:05Z"},
				{Type: "PodScheduled", Status: "True", LastTransitionTime: "2026-04-19T10:30:00Z"},
			}
			p.Status.ContainerStatuses = []struct {
				Name         string `json:"name"`
				RestartCount int32  `json:"restartCount"`
				Ready        bool   `json:"ready"`
				State        struct {
					Running struct {
						StartedAt string `json:"startedAt"`
					} `json:"running"`
				} `json:"state"`
			}{
				{
					Name:         dep.Name,
					RestartCount: pod.Restarts,
					Ready:        true,
				},
			}
			items = append(items, p)
		}
	}

	list := k8sList{
		APIVersion: "v1",
		Kind:       "List",
		Items:      items,
	}

	data, _ := json.MarshalIndent(list, "", "  ")
	return string(data)
}
