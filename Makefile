BINARY := agent-postmark
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -s -w -X main.version=$(VERSION)
GOCACHE ?= $(CURDIR)/.cache/go-build

build:
	GOCACHE=$(GOCACHE) go build -buildvcs=false -ldflags "$(LDFLAGS)" -o $(BINARY) ./cmd/agent-postmark

build-mock:
	GOCACHE=$(GOCACHE) go build -buildvcs=false -o mockpostmark ./cmd/mockpostmark

mock:
	GOCACHE=$(GOCACHE) go run ./cmd/mockpostmark

mock-dev:
	AGENT_POSTMARK_BASE_URL=http://127.0.0.1:12122 AGENT_POSTMARK_ACCOUNT_TOKEN=account_mock AGENT_POSTMARK_SERVER_TOKEN=server_mock GOCACHE=$(GOCACHE) go run ./cmd/agent-postmark $(ARGS)

test:
	GOCACHE=$(GOCACHE) go test ./... -count=1

test-short:
	GOCACHE=$(GOCACHE) go test ./... -count=1 -short

lint:
	golangci-lint run ./...

fmt:
	gofmt -w .
	@command -v goimports >/dev/null && goimports -w . || echo "goimports not installed (optional; install: go install golang.org/x/tools/cmd/goimports@latest)"

clean:
	rm -f $(BINARY)
	rm -f mockpostmark
	rm -rf dist/

dev:
	GOCACHE=$(GOCACHE) go run ./cmd/agent-postmark $(ARGS)

vet:
	GOCACHE=$(GOCACHE) go vet ./...

.PHONY: build build-mock mock mock-dev test test-short lint fmt clean dev vet

