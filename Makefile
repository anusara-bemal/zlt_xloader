BINARY     := zlt-unlocker
PKG        := github.com/anusara-bemal/zlt-unlocker
VERSION    ?= dev
LDFLAGS    := -s -w -X $(PKG)/cmd.version=$(VERSION)
DIST       := dist

.PHONY: all build build-all build-windows build-linux build-macos test lint fmt vet clean install help

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

test:
	go test ./...

vet:
	go vet ./...

lint: vet

fmt:
	gofmt -w .

install:
	go install -trimpath -ldflags "$(LDFLAGS)" .

clean:
	rm -rf $(DIST) $(BINARY)

help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-18s %s\n", $$1, $$2}'
