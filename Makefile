GO ?= go
BINARY ?= mcp2cli
OUTPUT_DIR ?= bin
VERSION ?= dev
GOPROXY ?= https://goproxy.cn,direct

.PHONY: build test fmt run clean

build:
	mkdir -p $(OUTPUT_DIR)
	$(GO) build -ldflags "-X main.version=$(VERSION)" -o $(OUTPUT_DIR)/$(BINARY) ./cmd/mcp2cli

test:
	GOPROXY=$(GOPROXY) $(GO) test ./...

fmt:
	$(GO) fmt ./...

run:
	$(GO) run ./cmd/mcp2cli --help

clean:
	rm -rf $(OUTPUT_DIR)

