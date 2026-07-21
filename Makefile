.PHONY: check test fmt vet build generate docs cover clean fuzz staticcheck vuln docker-build verify

GO       ?= go
MOCKGEN  ?= ./bin/mockgen
PKG       = ./...

check: fmt vet test docs

verify: fmt vet test staticcheck vuln docs

fmt:
	$(GO) fmt $(PKG)

vet:
	$(GO) vet $(PKG)

test:
	$(GO) test -race -count=1 $(PKG)

cover:
	$(GO) test -race -count=1 -coverprofile=coverage.out $(PKG)
	$(GO) tool cover -func=coverage.out | tail -1

fuzz:
	$(GO) test -run=^$$ -fuzz=FuzzKeywordResolver_Resolve -fuzztime=10s ./internal/resolver/

staticcheck:
	@if [ ! -x ./bin/staticcheck ]; then GOBIN=$(PWD)/bin $(GO) install honnef.co/go/tools/cmd/staticcheck@latest; fi
	./bin/staticcheck $(PKG)

vuln:
	@if [ ! -x ./bin/govulncheck ]; then GOBIN=$(PWD)/bin $(GO) install golang.org/x/vuln/cmd/govulncheck@latest; fi
	./bin/govulncheck $(PKG)

docker-build:
	docker build -t agent-contract-protocol/acp-server:dev .

build:
	$(GO) build -trimpath -ldflags="-s -w" -o bin/acp-server ./cmd/acp-server
	$(GO) build -trimpath -ldflags="-s -w" -o bin/acp-bridge ./cmd/acp-bridge
	$(GO) build -trimpath -ldflags="-s -w" -o bin/anf-mcp ./cmd/anf-mcp

generate:
	@if [ ! -x $(MOCKGEN) ]; then \
		GOBIN=$(PWD)/bin $(GO) install go.uber.org/mock/mockgen@v0.6.0; \
	fi
	PATH=$(PWD)/bin:$$PATH $(GO) generate $(PKG)

docs:
	@python3 scripts/check_docs.py

clean:
	rm -rf bin/acp-server bin/acp-bridge coverage.out
