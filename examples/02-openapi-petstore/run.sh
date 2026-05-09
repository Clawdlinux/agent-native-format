#!/usr/bin/env bash
# Example 02 — OpenAPI spec -> ACL
#
# Pipes the Swagger Petstore OpenAPI 3 spec through the acl CLI to
# show the live encode-then-measure pipeline an agent runtime would
# use.

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
ACL="${ROOT}/bin/acl"
SPEC="${ROOT}/pkg/aclhttp/testdata/petstore.json"

if [[ ! -x "${ACL}" ]]; then
    echo "Building bin/acl first..."
    make -C "${ROOT}" build-acl
fi

echo
echo "=== Source: pkg/aclhttp/testdata/petstore.json ==="
"${ACL}" tokens "${SPEC}"

echo
echo "=== ACL view ==="
"${ACL}" encode openapi "${SPEC}"

echo
echo "=== ACL token cost ==="
"${ACL}" encode openapi "${SPEC}" | "${ACL}" tokens -

echo
RAW_TOK=$("${ACL}" tokens "${SPEC}" | awk '/^tokens/ {print $2}')
ACL_TOK=$("${ACL}" encode openapi "${SPEC}" | "${ACL}" tokens - | awk '/^tokens/ {print $2}')
RAW_B=$(wc -c < "${SPEC}")
ACL_B=$("${ACL}" encode openapi "${SPEC}" | wc -c)
printf "=== Reduction: %.1fx bytes, %.1fx tokens ===\n" \
    "$(echo "scale=2; ${RAW_B}/${ACL_B}" | bc)" \
    "$(echo "scale=2; ${RAW_TOK}/${ACL_TOK}" | bc)"
