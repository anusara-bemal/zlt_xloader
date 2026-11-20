BINARY     := zlt_xloader
PKG        := github.com/anusara-bemal/zlt_xloader
VERSION    ?= dev
COMMIT     ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
DATE       ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS    := -s -w \
               -X $(PKG)/internal/buildinfo.Version=$(VERSION) \
               -X $(PKG)/internal/buildinfo.Commit=$(COMMIT) \
               -X $(PKG)/internal/buildinfo.Date=$(DATE)
DIST       := dist

.PHONY: all build build-all build-windows build-linux build-macos test cover vet fmt fmt-check check ci release snapshot install clean help

all: build

## Build for the host platform.
build:
	go build -trimpath -ldflags "$(LDFLAGS)" -o $(BINARY) .

## Cross-compile a release matrix into $(DIST)/.
build-all: build-windows build-linux build-macos

build-windows:
	@mkdir -p $(DIST)/windows
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -trimpath -ldflags "$(LDFLAGS)" -o $(DIST)/windows/$(BINARY)-amd64.exe .
	CGO_ENABLED=0 GOOS=windows GOARCH=386   go build -trimpath -ldflags "$(LDFLAGS)" -o $(DIST)/windows/$(BINARY)-386.exe .

build-linux:
	@mkdir -p $(DIST)/linux
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags "$(LDFLAGS)" -o $(DIST)/linux/$(BINARY)-amd64 .
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -trimpath -ldflags "$(LDFLAGS)" -o $(DIST)/linux/$(BINARY)-arm64 .
	CGO_ENABLED=0 GOOS=linux GOARCH=arm   go build -trimpath -ldflags "$(LDFLAGS)" -o $(DIST)/linux/$(BINARY)-armv7 .

build-macos:
	@mkdir -p $(DIST)/macos
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -trimpath -ldflags "$(LDFLAGS)" -o $(DIST)/macos/$(BINARY)-amd64 .
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -trimpath -ldflags "$(LDFLAGS)" -o $(DIST)/macos/$(BINARY)-arm64 .

## Run the full test suite.
test:
	go test ./...

## Run tests with a coverage report.
cover:
	go test -coverprofile=coverage.txt ./...
	go tool cover -func=coverage.txt

## Run go vet across all packages.
vet:
	go vet ./...

## Format all Go source files.
fmt:
	gofmt -w .

## Fail if any Go file is not gofmt-clean.
fmt-check:
	@test -z "$$(gofmt -l .)" || (echo "Run 'make fmt'"; gofmt -l .; exit 1)

## Local pre-push gate: formatting, vet and tests.
check: fmt-check vet test

## Everything CI runs, including the full cross-compile matrix.
ci: check build-all

## Build and publish a tagged release with GoReleaser (requires a git tag).
release:
	goreleaser release --clean

## Build release artifacts locally without publishing them.
snapshot:
	goreleaser release --snapshot --clean

install:
	go install -trimpath -ldflags "$(LDFLAGS)" .

clean:
	rm -rf $(DIST) $(BINARY) $(BINARY).exe coverage.txt

help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-16s %s\n", $$1, $$2}'
