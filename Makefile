SHELL := /bin/bash

PREFIX ?= /usr/local
DESTDIR ?=
BINDIR ?= $(PREFIX)/bin

GO ?= go
GOFLAGS ?=
PROGRAM := flapc
SOURCES := $(wildcard *.go)
MODULE_FILES := go.mod $(wildcard go.sum)

.PHONY: all install test clean help

.DEFAULT_GOAL := all

all: flapc

help:
	@echo "Available targets:"
	@echo "  all     - Build flapc compiler (default)"
	@echo "  flapc   - Build flapc compiler"
	@echo "  test    - Run all tests"
	@echo "  install - Install flapc to $(BINDIR)"
	@echo "  clean   - Remove compiled binaries and build artifacts"
	@echo "  help    - Show this help message"

flapc: $(SOURCES) $(MODULE_FILES)
	$(GO) build -mod=vendor -v $(GOFLAGS) -o $(PROGRAM) .

test:
	@echo "Running all tests..."
	$(GO) test -v -timeout 5m ./...

install: flapc
	install -d "$(DESTDIR)$(BINDIR)"
	install -m 755 $(PROGRAM) "$(DESTDIR)$(BINDIR)/$(PROGRAM)"

clean:
	rm -rf $(PROGRAM) build/ test_*
