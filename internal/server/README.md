# ACP HTTP Server

Hosts `/v1/context`, `/v1/feedback`, and `/healthz`. Bearer-token middleware
uses `crypto/subtle.ConstantTimeCompare`. Collaborators (Resolver, Builder,
FeedbackSink) are consumer-defined interfaces and mocked in tests with
`go.uber.org/mock`.
