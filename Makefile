.PHONY: dev build test clean install-tools

# ── Tool versions / paths ──
AIR       := $(shell command -v air 2>/dev/null || echo "$(shell go env GOPATH)/bin/air")
TEMPL     := $(shell command -v templ 2>/dev/null || echo "$(shell go env GOPATH)/bin/templ")

# ── Default target ──
.DEFAULT_GOAL := dev

# ── Install missing tooling ──
install-tools:
	@echo "==> Checking tooling…"
	@# air
	@if [ ! -x "$(AIR)" ]; then \
		echo "  → installing air…"; \
		go install github.com/air-verse/air@latest; \
	else \
		echo "  ✓ air found at $(AIR)"; \
	fi
	@# templ
	@if [ ! -x "$(TEMPL)" ]; then \
		echo "  → installing templ…"; \
		go install github.com/a-h/templ/cmd/templ@latest; \
	else \
		echo "  ✓ templ found at $(TEMPL)"; \
	fi

# ── Generate Templ files ──
templ-generate:
	@echo "==> Generating Templ files…"
	$(TEMPL) generate

# ── Build the Go binary ──
build:
	@echo "==> Building Go binary…"
	go build -o ./tmp/main ./cmd/api

# ── Run tests ──
test:
	@echo "==> Running tests…"
	go test ./... -count=1

# ── Clean build artifacts ──
clean:
	@echo "==> Cleaning…"
	rm -rf ./tmp

# ── Development server (hot-reload) ──
# Runs templ generate once, then starts air for hot-reload.
dev: install-tools templ-generate
	@echo "==> Starting development server with hot-reload…"
	$(AIR)