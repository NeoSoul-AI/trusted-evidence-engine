GO ?= go

.PHONY: fmt test run-cli run-server

fmt:
	gofmt -w ./cmd ./internal

test:
	$(GO) test ./...

run-cli:
	$(GO) run ./cmd/tee resolve --query "Will Team A win the final?" --format pretty

run-server:
	$(GO) run ./cmd/tee-server --addr :8088
