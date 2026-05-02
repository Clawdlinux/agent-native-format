package manifest

// ExecutionManifest is the ACP response returned by POST /v1/context.
type ExecutionManifest struct {
	ManifestID       string   `json:"manifest_id"`
	Version          string   `json:"version"`
	TTL              string   `json:"ttl"`
	Actions          []Action `json:"actions"`
	Boundaries       Boundary `json:"boundaries"`
	FeedbackEndpoint string   `json:"feedback_endpoint"`
}

// Action describes one executable step in an ACP manifest.
type Action struct {
	ID        string            `json:"id"`
	Type      string            `json:"type"`
	Endpoint  string            `json:"endpoint"`
	Method    string            `json:"method,omitempty"`
	Schema    map[string]string `json:"schema"`
	Auth      string            `json:"auth"`
	Timeout   string            `json:"timeout,omitempty"`
	DependsOn []string          `json:"depends_on,omitempty"`
}

// Boundary captures execution constraints the agent and proxy must honor.
type Boundary struct {
	Egress             []string `json:"egress"`
	MaxTokensPerAction int      `json:"max_tokens_per_action"`
	RequireApproval    []string `json:"require_approval,omitempty"`
	AuditLevel         string   `json:"audit_level"`
}
