# Auth Proxy

The proxy enforces manifest boundaries and injects credentials into tool calls
without exposing secrets to the agent context window.

Week 2 scope:

- Per-action upstream routing
- Header/API-key injection from server-side config
- Egress allow-list enforcement
- Approval gate blocking for marked actions
