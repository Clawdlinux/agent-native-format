# ACP Architecture

ACP shifts execution-context work from the agent runtime to the NineVigil
control plane.

## Components

| Component | Responsibility | Language |
|---|---|---|
| ACP Server | Receives intent, resolves capabilities, computes manifests | Go |
| Tool Registry | Stores tools, compact schemas, auth configs, capability tags | Go |
| Intent Resolver | Maps natural language intent to required capabilities | Go first, embeddings later |
| Manifest Builder | Selects relevant actions, strips schemas, computes ordering | Go |
| Auth Proxy | Injects credentials at the network boundary | Go / Envoy |
| MCP Baseline | Implements equivalent tasks through MCP discovery | Python |
| Benchmark Harness | Runs ACP vs MCP, records tokens, latency, success rate | Python |
| Agent Adapters | Consume ACP manifests from LangGraph, CrewAI, OpenAI flows | Python |

## Request path

1. Agent sends `POST /v1/context` with `intent`, `agent_id`, optional
   capabilities, and constraints.
2. ACP Server authenticates the agent identity token.
3. Intent Resolver maps intent to capability tags.
4. Tool Registry returns candidate tools.
5. Manifest Builder strips schemas, orders actions, attaches boundaries.
6. Auth Proxy prepares per-action credential injection without exposing
   secrets to the agent.
7. ACP Server returns one Execution Manifest.
8. Agent executes actions in manifest order and reports feedback.

## Deployment modes

| Mode | Description | Target |
|---|---|---|
| Local PoC | Docker Compose with mock services and benchmark harness | Weeks 1-2 |
| Cluster PoC | K8s manifests for ACP server + proxy + demo tool services | Weeks 2-3 |
| Operator deployment | Packaged through agentic-operator for regulated clusters | Post-PoC |

## Security model

- Agents authenticate to ACP with short-lived identity tokens.
- Manifests never contain raw credentials.
- Auth injection happens at the proxy boundary.
- Egress allow-lists are enforced by the proxy.
- Human approval gates are declared in `boundaries.require_approval` and
  enforced out of band.
