SHELL := /bin/bash

PREFIX ?= /usr/local
DESTDIR ?=
BINDIR ?= $(PREFIX)/bin

GO ?= go
GOFLAGS ?=
PROGRAM := flapc
SOURCES := $(wildcard *.go)
MODULE_FILES := go.mod $(wildcard go.sum)

.PHONY: all install test test-go test-flap clean help

.DEFAULT_GOAL := all

all: flapc

help:
	@echo "Available targets:"
	@echo "  all        - Build flapc compiler (default)"
	@echo "  flapc      - Build flapc compiler"
	@echo "  test       - Run all tests (Go unit tests + Flap integration tests)"
	@echo "  test-go    - Run Go unit tests only"
	@echo "  test-flap  - Run Flap integration tests only"
	@echo "  install    - Install flapc to $(BINDIR)"
	@echo "  clean      - Remove compiled binaries and build artifacts"
	@echo "  help       - Show this help message"

flapc: $(SOURCES) $(MODULE_FILES)
	$(GO) build -mod=vendor -v $(GOFLAGS) -o $(PROGRAM) .

test-go:
	@echo "Running Go unit tests..."
	$(GO) test -v ./...

test-flap: flapc
	@echo "Running Flap integration tests..."
	./test.sh

test: test-go test-flap
	@echo "All tests passed!"

install: flapc
	install -d "$(DESTDIR)$(BINDIR)"
	install -m 755 $(PROGRAM) "$(DESTDIR)$(BINDIR)/$(PROGRAM)"

clean:
	rm -rf $(PROGRAM) build/ test_*
