# Auth Proxy

The proxy enforces manifest boundaries and injects credentials into tool calls
without exposing secrets to the agent context window.

**Wire surface:** `POST /v1/exec/{manifest_id}/{action_id}` (mounted by
`cmd/acp-server` when `--enable-proxy` is set; on by default).

**Behaviour:**

- Looks up the manifest in the configured `ManifestStore`.
- Strips agent-supplied `Authorization` and `Proxy-Authorization`.
- Enforces `boundaries.egress` against the upstream host.
- Blocks `boundaries.require_approval` actions until `ApprovalGate.IsApproved`
  returns true; on denial returns 403 with `X-ACP-Approval-Required: true`.
- Injects credentials returned by `Injector.Inject` and forwards via
  `httputil.ReverseProxy`.

`MapInjector` and `MemoryStore` are simple in-memory implementations for
development and testing. Production deployments should plug in a vault-backed
`Injector` and a durable `ManifestStore`.
