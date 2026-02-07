GO ?= go

.PHONY: test build run

test:
	$(GO) test ./...

build:
	$(GO) build ./cmd/paneld

run:
	SERVER_ADDR=:6365 $(GO) run ./cmd/paneld
