PREFIX ?= /usr/local
DESTDIR ?=
BINDIR ?= $(PREFIX)/bin

GO ?= go
GOFLAGS ?=
PROGRAM := flapc
SOURCES := $(wildcard *.go)
MODULE_FILES := go.mod $(wildcard go.sum)

.PHONY: all install test clean

.DEFAULT_GOAL := all

all: flapc

$(PROGRAM): $(SOURCES) $(MODULE_FILES)
	$(GO) build $(GOFLAGS) -o $(PROGRAM) .

test:
	@echo "Running all tests..."
	$(GO) test -timeout 1m ./...

install: flapc
	install -d "$(DESTDIR)$(BINDIR)"
	install -m 755 $(PROGRAM) "$(DESTDIR)$(BINDIR)/$(PROGRAM)"

clean:
	rm -rf $(PROGRAM) build/ test_*
