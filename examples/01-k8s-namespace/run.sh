#!/usr/bin/env bash
# Example 01 — Kubernetes namespace -> ACL
#
# Reads a real kubectl-shaped JSON fixture (5 pods, 2 deploys, 2 svcs)
# and shows the same data as the ACL view an SRE agent would consume.

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
ACL="${ROOT}/bin/acl"
RAW="${ROOT}/benchmark/agent_accuracy/fixtures/healthy/raw.json"
VIEW="${ROOT}/benchmark/agent_accuracy/fixtures/healthy/state.acl"

if [[ ! -x "${ACL}" ]]; then
    echo "Building bin/acl first..."
    make -C "${ROOT}" build-acl
fi

echo
echo "=== Source: kubectl get pods,deploy,svc -o json ==="
"${ACL}" tokens "${RAW}"

echo
echo "=== ACL view (the same data, agent-shaped) ==="
cat "${VIEW}"

echo
echo "=== ACL token cost ==="
"${ACL}" tokens "${VIEW}"

echo
RAW_TOK=$("${ACL}" tokens "${RAW}"   | awk '/^tokens/ {print $2}')
ACL_TOK=$("${ACL}" tokens "${VIEW}" | awk '/^tokens/ {print $2}')
RAW_B=$(wc -c < "${RAW}")
ACL_B=$(wc -c < "${VIEW}")
printf "=== Reduction: %.1fx bytes, %.1fx tokens ===\n" \
    "$(echo "scale=2; ${RAW_B}/${ACL_B}"   | bc)" \
    "$(echo "scale=2; ${RAW_TOK}/${ACL_TOK}" | bc)"
