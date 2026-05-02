.PHONY: check test fmt docs

check: fmt test docs

fmt:
	gofmt -w $$(find . -name '*.go' -not -path './.git/*') 2>/dev/null || true

test:
	go test ./...

docs:
	@python3 scripts/check_docs.py
