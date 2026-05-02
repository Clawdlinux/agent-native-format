# Go ACP Client

Planned public SDK:

```go
client := acp.NewClient("http://localhost:8080", acp.WithToken(token))
manifest, err := client.Context(ctx, acp.ContextRequest{Intent: "query db"})
```

No runtime logic in Phase 1.
