# metro — build and install (FHS: $(PREFIX)/bin, default /usr/local/bin)

PREFIX  ?= /usr/local
BINDIR  ?= $(PREFIX)/bin
DESTDIR ?=

BINARY  := metro
MAIN    := ./cmd/metro

GO      ?= go
GOFLAGS ?=
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null)
LDFLAGS ?= -s -w \
	-X github.com/dixiedream/metro/internal/version.Version=$(VERSION) \
	-X github.com/dixiedream/metro/internal/version.Commit=$(COMMIT)

.PHONY: all build install uninstall clean test

all: build

build:
	$(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(BINARY) $(MAIN)

install: build
	install -d $(DESTDIR)$(BINDIR)
	install -m 755 $(BINARY) $(DESTDIR)$(BINDIR)/$(BINARY)

uninstall:
	rm -f $(DESTDIR)$(BINDIR)/$(BINARY)

clean:
	rm -f $(BINARY)
	$(GO) clean

test:
	$(GO) test ./...
