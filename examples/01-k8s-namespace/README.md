# Example 01 — Kubernetes namespace → ACL

Reads a kubectl-shaped JSON dump of a 5-pod, 2-deploy, 2-svc namespace
and converts it to the ACL view an SRE agent would actually consume.

## Run it

```bash
bash examples/01-k8s-namespace/run.sh
```

## What you'll see

```
Source:  kubectl get pods,deploy,svc -o json (5 pods, 2 deploys, 2 svcs)
Bytes:   14506   Tokens: 3671 (cl100k_base)

ACL view (the same data, agent-shaped):
@cluster prod-east
@ns payments
...
Bytes:   572     Tokens: 260

Reduction: 25.4x bytes, 14.1x tokens
```

## What's happening

The K8s translator ([`pkg/aclk8s`](https://github.com/Clawdlinux/agentic-operator-core/tree/main/pkg/aclk8s) — lives in the
agentic-operator repo because it depends on the kubernetes/client-go
module) extracts only the fields an SRE agent needs to choose an action:

- pod name, replica count, restart count, age, warning/critical flags
- deploy name, replicas ready/desired, strategy, image tag
- service name, type, port mapping
- the closed set of actions the agent may invoke

Everything an agent doesn't need (`managedFields`, `resourceVersion`,
service-account volume mounts, container SHA256 digests, default
scheduler tolerations) is dropped. That's where the 14× token reduction
comes from.

## Files used

- `benchmark/agent_accuracy/fixtures/healthy/raw.json` — the kubectl-shaped source
- `benchmark/agent_accuracy/fixtures/healthy/state.acl` — the pre-generated ACL view

The pre-generated file is what an actual translator would produce live.
This example uses the static fixture so you don't need a kind cluster
to see the result.
