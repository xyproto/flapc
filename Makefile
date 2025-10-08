SHELL := /bin/bash

PREFIX ?= /usr/local
DESTDIR ?=
BINDIR ?= $(PREFIX)/bin

GO ?= go
GOFLAGS ?=
PROGRAM := flapc
SOURCES := $(wildcard *.go)
MODULE_FILES := go.mod $(wildcard go.sum)

.PHONY: all install test clean

all: flapc

flapc: $(SOURCES) $(MODULE_FILES)
	$(GO) build -mod=vendor -v $(GOFLAGS) -o $(PROGRAM) .

test: flapc
	./test_programs.sh

install: flapc
	install -d "$(DESTDIR)$(BINDIR)"
	install -m 755 $(PROGRAM) "$(DESTDIR)$(BINDIR)/$(PROGRAM)"

clean:
	rm -f $(PROGRAM)
	rm -rf build/test_programs
