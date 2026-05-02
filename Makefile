.PHONY: check test fmt vet build generate docs cover clean

GO       ?= go
MOCKGEN  ?= ./bin/mockgen
PKG       = ./...

check: fmt vet test docs

fmt:
	$(GO) fmt $(PKG)

vet:
	$(GO) vet $(PKG)

test:
	$(GO) test -race -count=1 $(PKG)

cover:
	$(GO) test -race -count=1 -coverprofile=coverage.out $(PKG)
	$(GO) tool cover -func=coverage.out | tail -1

build:
	$(GO) build -trimpath -ldflags="-s -w" -o bin/acp-server ./cmd/acp-server

generate:
	@if [ ! -x $(MOCKGEN) ]; then \
		GOBIN=$(PWD)/bin $(GO) install go.uber.org/mock/mockgen@v0.6.0; \
	fi
	PATH=$(PWD)/bin:$$PATH $(GO) generate $(PKG)

docs:
	@python3 scripts/check_docs.py

clean:
	rm -rf bin/acp-server coverage.out
