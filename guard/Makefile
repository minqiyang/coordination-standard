GO ?= $(shell command -v go 2>/dev/null || echo /opt/homebrew/bin/go)
BIN := bin/coord-guard
PREFIX ?= $(HOME)/.local

.PHONY: test vet build install clean

test: vet
	$(GO) test ./...

vet:
	$(GO) vet ./...

build:
	$(GO) build -o $(BIN) .

install: build
	install -d $(PREFIX)/bin
	install -m 0755 $(BIN) $(PREFIX)/bin/coord-guard

clean:
	rm -rf bin
