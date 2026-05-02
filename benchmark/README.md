# Benchmark

The benchmark harness will run identical tasks through ACP and MCP, then compare
setup tokens, round trips, time to first useful action, success rate, and cost.

Implementation targets:

- `baseline/mcp_client.py` — official MCP SDK baseline
- `harness.py` — scenario orchestrator
- `report.py` — markdown + chart + PDF generator
- `scenarios/` — checked-in YAML tasks

See [../docs/benchmark-methodology.md](../docs/benchmark-methodology.md).
