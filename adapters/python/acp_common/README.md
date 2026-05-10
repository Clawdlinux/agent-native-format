# acp-common

Shared client + types for [ACP (Agent Context Protocol)](https://github.com/Clawdlinux/ninevigil-acp) Python adapters. Pure-stdlib — `urllib`, `json`, `dataclasses`. No third-party runtime dependencies.

Used by `acp-langgraph`, `acp-openai`, and `acp-crewai`. You generally do not install this directly — it is a transitive dependency of the framework adapter you actually want.

## Install

```bash
pip install "git+https://github.com/Clawdlinux/ninevigil-acp.git@v0.1.0-spec#subdirectory=adapters/python/acp_common"
```

## Usage

```python
from acp_common import ACPClient

acp = ACPClient("http://localhost:8080", auth_token="dev-token")
manifest = acp.context(intent="query db then email", agent_id="my-agent")
```

License: Apache-2.0. The protocol spec it implements (`SPEC.md`) is CC BY 4.0.
