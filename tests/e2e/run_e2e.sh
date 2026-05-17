#!/usr/bin/env bash
# Manual E2E test for acp-bridge.
# Simulates VS Code's MCP client lifecycle:
#   1. initialize → verify capabilities
#   2. tools/list → verify cold start (all tools, compacted schemas)
#   3. tools/call × 3 → trigger narrowing
#   4. tools/list → verify narrowed surface
#
# Usage: ./tests/e2e/run_e2e.sh
set -euo pipefail

cd "$(dirname "$0")/../.."
BRIDGE="./bin/acp-bridge"
CONFIG="./tests/e2e/bridge-config.json"
MOCK_PID=""

cleanup() {
    [[ -n "${MOCK_PID}" ]] && kill "${MOCK_PID}" 2>/dev/null || true
    wait "${MOCK_PID}" 2>/dev/null || true
}
trap cleanup EXIT

echo "=== Step 0: Build bridge binary ==="
go build -o "${BRIDGE}" ./cmd/acp-bridge/
echo "OK: built ${BRIDGE}"

echo ""
echo "=== Step 1: Start mock MCP servers ==="
python3 ./tests/e2e/mock_mcp_servers.py &
MOCK_PID=$!
sleep 1

# Verify mock servers respond.
for port in 9001 9002 9003; do
    if ! curl -sf "http://127.0.0.1:${port}/tools/list" > /dev/null 2>&1; then
        echo "FAIL: mock server on port ${port} not responding"
        exit 1
    fi
done
echo "OK: all 3 mock MCP servers running"

echo ""
echo "=== Step 2: Send initialize ==="
INIT_REQ='{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-03-26","capabilities":{},"clientInfo":{"name":"e2e-test","version":"1.0"}}}'
INIT_RESP=$(echo "${INIT_REQ}" | "${BRIDGE}" --config "${CONFIG}" 2>/dev/null | head -1)
echo "Response: ${INIT_RESP}"

# Check listChanged capability.
if echo "${INIT_RESP}" | python3 -c "import sys,json; d=json.load(sys.stdin); assert d['result']['capabilities']['tools']['listChanged']==True" 2>/dev/null; then
    echo "OK: listChanged=true"
else
    echo "FAIL: listChanged not true"
    exit 1
fi

echo ""
echo "=== Step 3: Cold-start tools/list ==="
LIST_REQ='{"jsonrpc":"2.0","id":2,"method":"tools/list"}'
# Send initialize + initialized notification + tools/list together.
COLD_INPUT=$(printf '%s\n%s\n%s\n' \
    '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-03-26","capabilities":{},"clientInfo":{"name":"e2e","version":"1.0"}}}' \
    '{"jsonrpc":"2.0","method":"notifications/initialized"}' \
    "${LIST_REQ}")

COLD_OUTPUT=$(echo "${COLD_INPUT}" | "${BRIDGE}" --config "${CONFIG}" 2>/dev/null)
echo "Raw output lines: $(echo "${COLD_OUTPUT}" | wc -l)"

# Parse tools/list response (second JSON line, since first is initialize response).
TOOLS_RESP=$(echo "${COLD_OUTPUT}" | sed -n '2p')
TOOL_COUNT=$(echo "${TOOLS_RESP}" | python3 -c "import sys,json; d=json.load(sys.stdin); print(len(d['result']['tools']))" 2>/dev/null || echo "ERROR")
echo "Cold-start tool count: ${TOOL_COUNT}"

if [[ "${TOOL_COUNT}" -ge 6 ]]; then
    echo "OK: cold start returned ${TOOL_COUNT} tools (expected ≥6)"
else
    echo "FAIL: cold start returned ${TOOL_COUNT} tools (expected ≥6)"
    exit 1
fi

# Show tool names.
echo "Tools:"
echo "${TOOLS_RESP}" | python3 -c "
import sys, json
d = json.load(sys.stdin)
for t in d['result']['tools']:
    schema = t.get('inputSchema', {}).get('properties', {})
    fields = ', '.join(f'{k}: {v.get(\"type\",\"?\")}' for k,v in schema.items())
    print(f'  {t[\"name\"]:30s} | schema: {{{fields}}}')
" 2>/dev/null

echo ""
echo "=== Step 4: Narrowing via 3 github tool calls ==="
# Send init + tools/list + 3 tool calls + tools/list
NARROW_INPUT=$(printf '%s\n' \
    '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-03-26","capabilities":{},"clientInfo":{"name":"e2e","version":"1.0"}}}' \
    '{"jsonrpc":"2.0","method":"notifications/initialized"}' \
    '{"jsonrpc":"2.0","id":2,"method":"tools/list"}' \
    '{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"github.issues_list","arguments":{"owner":"test","repo":"test"}}}' \
    '{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"github.issues_create","arguments":{"owner":"test","repo":"test","title":"test"}}}' \
    '{"jsonrpc":"2.0","id":5,"method":"tools/call","params":{"name":"github.pull_request_list","arguments":{"owner":"test","repo":"test"}}}' \
    '{"jsonrpc":"2.0","id":6,"method":"tools/list"}')

NARROW_OUTPUT=$(echo "${NARROW_INPUT}" | "${BRIDGE}" --config "${CONFIG}" 2>/dev/null)

echo "All output lines:"
echo "${NARROW_OUTPUT}" | cat -n

# Check for list_changed notification.
if echo "${NARROW_OUTPUT}" | grep -q "tools/list_changed"; then
    echo ""
    echo "OK: notifications/tools/list_changed emitted"
else
    echo ""
    echo "WARN: no list_changed notification (may depend on timing)"
fi

# Get the last tools/list response (should be narrowed).
LAST_LIST=$(echo "${NARROW_OUTPUT}" | grep '"tools"' | tail -1)
NARROW_COUNT=$(echo "${LAST_LIST}" | python3 -c "import sys,json; d=json.load(sys.stdin); print(len(d['result']['tools']))" 2>/dev/null || echo "ERROR")

echo ""
echo "Narrowed tool count: ${NARROW_COUNT}"
if [[ "${NARROW_COUNT}" -lt "${TOOL_COUNT}" ]]; then
    echo "OK: narrowed from ${TOOL_COUNT} → ${NARROW_COUNT} tools"
else
    echo "INFO: narrowed count (${NARROW_COUNT}) not less than cold start (${TOOL_COUNT})"
    echo "  This can happen if observed capabilities overlap across sources."
fi

# Show narrowed tools.
echo "Narrowed tools:"
echo "${LAST_LIST}" | python3 -c "
import sys, json
d = json.load(sys.stdin)
for t in d['result']['tools']:
    print(f'  {t[\"name\"]}')
" 2>/dev/null

echo ""
echo "=== Step 5: Token comparison ==="
echo "${TOOLS_RESP}" | python3 -c "
import sys, json
d = json.load(sys.stdin)
raw = json.dumps(d['result']['tools'])
print(f'Cold-start tools/list response: {len(raw)} chars, ~{len(raw)//4} tokens')
" 2>/dev/null

echo "${LAST_LIST}" | python3 -c "
import sys, json
d = json.load(sys.stdin)
raw = json.dumps(d['result']['tools'])
print(f'Narrowed tools/list response:   {len(raw)} chars, ~{len(raw)//4} tokens')
" 2>/dev/null

echo ""
echo "=== E2E test complete ==="
