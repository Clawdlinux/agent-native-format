# acp-acl

[![PyPI](https://img.shields.io/pypi/v/acp-acl.svg)](https://pypi.org/project/acp-acl/)
[![Spec](https://img.shields.io/badge/spec-v0.1-blue.svg)](https://github.com/clawdlinux/ninevigil-acp/blob/main/docs/acl-spec.md)

Pure-Python decoder for the **Agent Context Language (ACL)** — a compact,
machine-native representation designed to be consumed by LLM agents
rather than humans.

ACL is the payload format that ships with [NineVigil ACP](https://github.com/clawdlinux/ninevigil-acp).
This package lets your Python agent (LangGraph, CrewAI, vanilla
OpenAI tool-use loop, anything) parse ACL without shelling out.

## Install

```bash
pip install acp-acl              # decoder only — zero deps
pip install "acp-acl[tokens]"    # adds tiktoken for count_tokens()
```

## Use

```python
import acp_acl

doc = acp_acl.decode(open("state.acl").read())

# Document-scope directives.
print(doc.directives["ns"])          # 'payments'

# Sections (pods, deploys, svcs, alerts, actions...).
pods = doc.section("pods")
print(pods.summary)                  # '12/12 ok'
for row in pods.rows:
    print(row.id, row.count, row.fields, row.flags)

# Token counting (optional).
print(acp_acl.count_tokens(open("state.acl").read()))
# 145
```

## Why ACL?

A live Kubernetes namespace returns ~76 KB of `kubectl get -o json`.
The same decision-relevant facts in ACL: ~580 bytes — a **132×**
compression that preserves agent task accuracy. Full benchmarks at
[ninevigil-acp/benchmark/agent_accuracy](https://github.com/clawdlinux/ninevigil-acp/tree/main/benchmark/agent_accuracy).

## Spec & licence

- Spec: [docs/acl-spec.md](https://github.com/clawdlinux/ninevigil-acp/blob/main/docs/acl-spec.md) (CC BY 4.0)
- Code: Apache 2.0
- Reference encoder (Go): `github.com/Clawdlinux/ninevigil-acp/pkg/acl`
