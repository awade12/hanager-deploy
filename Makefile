.PHONY: test build-agent build-cli install-cli install-agent install tidy caddy-swap deploy-hello check-go install-go stop-agent setup setup-laptop

VERSION ?= dev
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
LDFLAGS := -s -w -X github.com/awade12/hanager-deploy/cli/internal/version.Version=$(VERSION) -X github.com/awade12/hanager-deploy/cli/internal/version.Commit=$(COMMIT)

# Prefer /usr/local/go (install-go.sh) over Ubuntu's golang-go 1.18
GO := $(firstword $(wildcard /usr/local/go/bin/go) $(shell PATH="/usr/local/go/bin:$$PATH" command -v go 2>/dev/null) go)

export PATH := /usr/local/go/bin:$(PATH)

check-go:
	@$(GO) version 2>/dev/null | grep -qE 'go1\.(2[2-9]|[3-9][0-9])' || \
		( echo "error: need Go 1.22+ at $(GO) (got: $$($(GO) version 2>&1 || echo missing))"; \
		  echo "  run: make install-go"; \
		  echo "  then: export PATH=/usr/local/go/bin:\$$PATH"; \
		  exit 1 )

install-go:
	sudo bash scripts/install-go.sh

setup: install-go build-agent build-cli
	@echo "built bin/hangar-agent and bin/hangar"
	@echo "start agent: ./bin/hangar-agent -config ./agent.dev.json"
	@echo "then: make deploy-hello"

stop-agent:
	bash scripts/stop-agent.sh

test: check-go
	$(GO) test ./...

build-agent: check-go
	$(GO) build -ldflags "$(LDFLAGS)" -o bin/hangar-agent ./agent/cmd/hangar-agent

build-cli: check-go
	$(GO) build -ldflags "$(LDFLAGS)" -o bin/hangar ./cli/cmd/hangar

install: check-go
	$(GO) install -ldflags "$(LDFLAGS)" ./cli/cmd/hangar
	$(GO) install -ldflags "$(LDFLAGS)" ./agent/cmd/hangar-agent
	@echo "installed hangar and hangar-agent to $$(go env GOPATH)/bin"

install-cli: build-cli
	install -m 755 bin/hangar /usr/local/bin/hangar

install-agent: build-agent
	install -m 755 bin/hangar-agent /usr/local/bin/hangar-agent

tidy: check-go
	$(GO) mod tidy

caddy-swap:
	bash scripts/caddy-swap/swap.sh

deploy-hello:
	bash scripts/deploy-hello.sh

setup-laptop:
	bash scripts/setup-laptop.sh
