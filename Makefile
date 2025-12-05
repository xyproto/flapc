PREFIX ?= /usr
DESTDIR ?=
BINDIR ?= $(PREFIX)/bin

GO ?= go
GOFLAGS ?= -mod=vendor
PROGRAM := c67
SOURCES := $(wildcard *.go)
MODULE_FILES := go.mod $(wildcard go.sum)

.PHONY: all install test clean

.DEFAULT_GOAL := all

all: c67

$(PROGRAM): $(SOURCES) $(MODULE_FILES)
	$(GO) build $(GOFLAGS)

test:
	@echo "Running tests..."
	$(GO) test -failfast -timeout 1m ./...

install: c67
	install -d "$(DESTDIR)$(BINDIR)"
	install -m 755 $(PROGRAM) "$(DESTDIR)$(BINDIR)/$(PROGRAM)"

clean:
	rm -rf $(PROGRAM) build/ test_*
