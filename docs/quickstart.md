# Quickstart

Three independent paths, each ending with a real number you can verify
on your own machine. Pick the one that matches how you'll consume ACL.

| Path | Time | Output |
|---|---|---|
| [CLI](#path-1--cli-30-seconds) | 30 sec | OpenAPI spec compressed 7.4× live |
| [Go library](#path-2--go-library-2-minutes) | 2 min | Encode + decode an ACL document in code |
| [Python decoder](#path-3--python-decoder-1-minute) | 1 min | Read ACL files from a Python agent |

All numbers below are reproducible from a fresh clone. If you see different
output, please [open an issue](https://github.com/Clawdlinux/ninevigil-acp/issues).

---

## Path 1 — CLI (30 seconds)

The fastest way to see what ACL does. No code, just a binary that reads
files and prints token counts.

### Build the CLI

```bash
git clone https://github.com/Clawdlinux/ninevigil-acp
cd ninevigil-acp
make build-acl
```

You should now have `bin/acl`. Confirm:

```bash
bin/acl version
# 06ee7a7-dirty   (or whatever your git describe says)
```

### Compress an OpenAPI spec

The repo bundles the official Swagger Petstore OpenAPI 3 spec (a real
public spec used by thousands of API tools). Translate it to ACL and
count the tokens both before and after:

```bash
# Token count of the raw spec
bin/acl tokens pkg/aclhttp/testdata/petstore.json
# bytes:    5855
# chars:    5855
# tokens:   1492  (cl100k_base)

# Token count of the ACL view
bin/acl encode openapi pkg/aclhttp/testdata/petstore.json | bin/acl tokens -
# bytes:    594
# chars:    594
# tokens:   201   (cl100k_base)
```

**1,492 → 201 tokens = 7.4× fewer tokens** for the same OpenAPI spec.
The ACL document still lists every endpoint, every required parameter,
the auth schemes, and the response schemas — just without the descriptive
prose that an LLM doesn't need.

### See what the agent sees

If you want to read the compact view yourself:

```bash
bin/acl encode openapi pkg/aclhttp/testdata/petstore.json
```

```
@api Swagger-Petstore---OpenAPI-3.0
@version 1.0.20
@server /api/v3
@source openapi/v0.1

auth 2
  api_key type=apiKey in=header name=api_key
  petstore_auth type=oauth2

endpoints 4
  POST/pet op=addPet body=Pet returns=Pet auth=petstore_auth
  GET/pet/findByStatus op=findPetsByStatus opt=status returns=array auth=petstore_auth
  DELETE/pet/{petId} op=deletePet req=petId opt=api_key auth=petstore_auth
  GET/pet/{petId} op=getPetById req=petId returns=Pet auth=api_key

schemas 3
  Category fields=2 required=0
  Pet fields=6 required=2
  Tag fields=2 required=0

actions
  delete|get|post
```

Every endpoint is listed with HTTP method, operation ID, required
parameters, request body schema, and response type. An agent can plan
any HTTP call against this API from these 594 bytes.

---

## Path 2 — Go library (2 minutes)

Use ACL inside a Go service that already produces structured data.

### Add the module

```bash
go get github.com/Clawdlinux/ninevigil-acp/pkg/acl
go get github.com/Clawdlinux/ninevigil-acp/pkg/aclhttp
```

### Encode an OpenAPI spec

```go
package main

import (
	"fmt"
	"os"

	"github.com/Clawdlinux/ninevigil-acp/pkg/aclhttp"
)

func main() {
	spec, err := os.ReadFile("petstore.json")
	if err != nil {
		panic(err)
	}
	out, err := aclhttp.Encode(spec)
	if err != nil {
		panic(err)
	}
	fmt.Print(string(out))
}
```

Run it:

```bash
go run main.go
```

### Decode an ACL document

```go
package main

import (
	"fmt"
	"os"

	"github.com/Clawdlinux/ninevigil-acp/pkg/acl"
)

func main() {
	data, _ := os.ReadFile("doc.acl")
	doc, err := acl.Decode(data)
	if err != nil {
		panic(err)
	}
	for _, dir := range doc.Directives {
		fmt.Printf("@%s = %s\n", dir.Key, dir.Value)
	}
	for _, sec := range doc.Sections {
		fmt.Printf("section %q: %d rows\n", sec.Name, len(sec.Rows))
	}
}
```

### Round-trip guarantee

`acl.Decode(acl.Encode(doc))` returns a document equal to the input. This
is asserted in `pkg/acl/acl_test.go::TestRoundTrip` and is a hard
invariant of the format spec. Use it to safely cache, hash, or transmit
ACL documents.

### Build your own translator

If your data source isn't K8s, OpenAPI, or Postgres, the encoder is the
public surface. Build an `acl.Document` and call `acl.Encode`:

```go
import "github.com/Clawdlinux/ninevigil-acp/pkg/acl"

doc := acl.Document{
    Directives: []acl.Directive{{Key: "source", Value: "myapp/v1"}},
    Sections: []acl.Section{
        {
            Name: "users", Summary: "5 active",
            Rows: []acl.Row{
                {ID: "alice", Fields: []acl.Field{{Key: "role", Value: "admin"}}},
                {ID: "bob",   Fields: []acl.Field{{Key: "role", Value: "viewer"}}},
            },
        },
    },
}
out, _ := acl.Encode(doc)
```

The full format spec is at [docs/acl-spec.md](./acl-spec.md).

---

## Path 3 — Python decoder (1 minute)

Use ACL inside a Python agent (LangGraph, CrewAI, OpenAI tool-use loop,
anything). The decoder is pure Python with zero runtime dependencies.

### Install

```bash
pip install -e python/                 # from the repo
# or once published:
# pip install acp-acl
```

For token counting, install the optional `tiktoken` extra:

```bash
pip install -e 'python/[tokens]'
```

### Decode an ACL document

```python
import acp_acl

with open("benchmark/agent_accuracy/fixtures/healthy/state.acl") as f:
    doc = acp_acl.decode(f.read())

print(doc.directives["ns"])         # 'payments'
print(doc.directives["source"])     # 'k8s-namespace/v0.1'

pods = doc.section("pods")
print(pods.summary)                 # '5/5 ok'
for row in pods.rows:
    print(row.id, row.count, row.fields, row.flags)
```

### Count tokens

```python
import acp_acl

raw = open("benchmark/agent_accuracy/fixtures/healthy/raw.json").read()
acl = open("benchmark/agent_accuracy/fixtures/healthy/state.acl").read()

print(acp_acl.count_tokens(raw))    # 3671  (kubectl JSON)
print(acp_acl.count_tokens(acl))    # 260   (ACL view of the same data)
```

**3,671 → 260 tokens = 14.1× fewer tokens** for a healthy K8s namespace
with 5 pods, 2 deploys, and 2 services.

### Iterate sections

```python
for section in doc:
    print(section.name, len(section.rows))
```

---

## What just happened?

All three paths showed the same thing from different angles: ACL is a
purpose-built representation of structured data, optimised for how LLMs
tokenise text. The compression isn't gzip — it's [deliberate field
selection](./acl-spec.md#design-goals) plus a wire format with no JSON
ceremony.

| Source | Raw tokens | ACL tokens | Reduction |
|---|---:|---:|---:|
| OpenAPI Petstore (4 endpoints) | 1,492 | 201 | 7.4× |
| K8s namespace (5 pods, 2 deploys) | 3,671 | 260 | 14.1× |
| GitHub OpenAPI spec (1,145 endpoints) | ~3M | ~44K | 68× |
| Live kind cluster (kubectl JSON) | ~19K | 145 | 132× |

The wider the source format's overhead-to-substance ratio, the bigger
the compression. K8s and OpenAPI win biggest. SQL DDL (already terse)
wins less but still ~3.5×.

## Next steps

- Read the [format spec](./acl-spec.md) — 200 lines, defines the wire format
- Browse [examples/](../examples/) — three runnable end-to-end demos
- Build your own translator — any structured source can have one in ~250 lines of Go
- See the [agent-accuracy benchmark](../benchmark/agent_accuracy/README.md) — 540 trials proving ACL preserves accuracy at 1/10 the tokens
