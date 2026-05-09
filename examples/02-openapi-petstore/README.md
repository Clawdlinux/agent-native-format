# Example 02 — OpenAPI spec → ACL

Translates the official Swagger Petstore OpenAPI 3 spec to ACL using
a stdin/stdout pipeline — the shape an agent runtime or CI step would
use to translate-on-the-fly.

## Run it

```bash
bash examples/02-openapi-petstore/run.sh
```

## What you'll see

```
Source:  pkg/aclhttp/testdata/petstore.json (4 endpoints, 3 schemas)
Bytes:   5855    Tokens: 1492 (cl100k_base)

ACL view:
@api Swagger-Petstore---OpenAPI-3.0
...
Bytes:   594     Tokens: 201

Reduction: 9.9x bytes, 7.4x tokens
```

## What's happening

The OpenAPI translator ([`pkg/aclhttp`](../../pkg/aclhttp/)) reads the
JSON spec and emits one ACL row per `(method, path)` operation with:

- HTTP method and templated path (e.g. `GET/pet/{petId}`)
- operationId, required + optional parameter names
- request body schema reference, success-response schema reference
- which auth scheme this endpoint uses
- the closed set of HTTP methods the agent may invoke

Discarded: prose `summary` and `description` fields, vendor extensions,
4xx/5xx response bodies (an agent retries on errors, doesn't pre-plan
them), full schema definitions (agent fetches on demand), examples,
external doc links.

## On a real-world spec

The same translator on GitHub's full OpenAPI spec (12 MB, 1,145
endpoints) yields a 175 KB ACL document — **68× compression** — with
every endpoint still callable.

## Files used

- `pkg/aclhttp/testdata/petstore.json` — the canonical Swagger Petstore spec
