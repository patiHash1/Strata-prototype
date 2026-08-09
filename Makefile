.PHONY: dev build test clean install-tools

# ── Tool versions / paths ──
AIR       := $(shell command -v air 2>/dev/null || echo "$(shell go env GOPATH)/bin/air")
TEMPL     := $(shell command -v templ 2>/dev/null || echo "$(shell go env GOPATH)/bin/templ")
TAILWIND  := $(shell command -v tailwindcss 2>/dev/null || echo "./node_modules/.bin/tailwindcss")

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
	@# tailwindcss (standalone CLI)
	@if [ ! -x "$(TAILWIND)" ]; then \
		echo "  → installing tailwindcss…"; \
		TAILWIND_VERSION=v4.1.17; \
		OS=$$(uname -s | tr '[:upper:]' '[:lower:]'); \
		ARCH=$$(uname -m); \
		if [ "$$ARCH" = "x86_64" ]; then ARCH="x64"; fi; \
		if [ "$$ARCH" = "aarch64" ]; then ARCH="arm64"; fi; \
		URL="https://github.com/tailwindlabs/tailwindcss/releases/download/$$TAILWIND_VERSION/tailwindcss-$$OS-$$ARCH"; \
		mkdir -p ./node_modules/.bin; \
		curl -sL "$$URL" -o "$(TAILWIND)"; \
		chmod +x "$(TAILWIND)"; \
		echo "  ✓ tailwindcss installed at $(TAILWIND)"; \
	else \
		echo "  ✓ tailwindcss found at $(TAILWIND)"; \
	fi

# ── Build Tailwind CSS ──
tailwind:
	@echo "==> Building Tailwind CSS…"
	$(TAILWIND) -i input.css -o internal/static/css/styles.css --minify

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
# Runs templ generate + tailwindcss once, then starts air for hot-reload.
dev: install-tools templ-generate tailwind
	@echo "==> Starting development server with hot-reload…"
	$(AIR)