PKGS := $(shell go list ./... | grep -v /vendor)

BIN_DIR := $(GOPATH)/bin

# Try to detect current branch if not provided from environment
BRANCH ?= $(shell git rev-parse --abbrev-ref HEAD)

# Commit hash from git
COMMIT=$(shell git rev-parse --short HEAD)

# Tag on this commit
TAG = $(shell git tag --points-at HEAD)


ifneq ("$(shell which gotestsum)", "")
	TESTEXE := gotestsum --
else
	TESTEXE := go test ./...
endif

BUILD_DATE=$(shell date -u +'%Y-%m-%dT%H:%M:%SZ')
VERSION := $(or $(TAG),$(COMMIT)-$(BRANCH)-$(BUILD_DATE))

LDFLAGS = -X main.Version=$(VERSION) -X main.GitCommit=$(COMMIT) -X main.BuildDate=$(BUILD_DATE)

all: test lint build coverage ## test, lint, build, coverage test run


.PHONY: all deps install grammar antlr build lint test coverage clean
lint: ## Run golangci-lint
	golangci-lint run

coverage: ## Verify the test coverage remains high
	./scripts/check-coverage.sh 80

test: ## Run tests without coverage
	$(TESTEXE)

BINARY := wbnf
PLATFORMS := windows linux darwin
.PHONY: $(PLATFORMS)
$(PLATFORMS): build
	mkdir -p release
	GOOS=$@ GOARCH=amd64 \
		go build -o release/$(BINARY)-$(VERSION)-$@$(shell test $@ = windows && echo .exe) \
		-ldflags="$(LDFLAGS)" \
		-v \
		./cmd/sysl

build: ## Build wbnf into the ./dist folder
	go build -o ./dist/$(BINARY) -ldflags="$(LDFLAGS)" -v .

deps: ## Download the project dependencies with `go get`
	go get -v -d ./...

.PHONY: release
release: $(PLATFORMS) ## Build release binaries for all supported platforms into ./release

install: build ## Install the wbnf binary into $(GOPATH)/bin
	cp ./dist/$(BINARY) $(GOPATH)/bin

clean: ## Clean temp and build files
	rm -rf release dist

.PHONY: help

help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
