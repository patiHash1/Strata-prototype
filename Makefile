.PHONY: dev build test clean install-tools check-redis testsoc

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

# ── Check Redis connection ──
check-redis:
	@echo "==> Checking Redis connection…"
	@REDIS_ADDR=$${REDIS_ADDR:-localhost:6379}; \
	if command -v redis-cli >/dev/null 2>&1; then \
		if redis-cli -h $$(echo $$REDIS_ADDR | cut -d: -f1) -p $$(echo $$REDIS_ADDR | cut -d: -f2) ping >/dev/null 2>&1; then \
			echo "  ✓ Redis connected at $$REDIS_ADDR"; \
		else \
			echo "  ⚠ Redis not reachable at $$REDIS_ADDR (SSE fan-out disabled)"; \
		fi; \
	else \
		echo "  ⚠ redis-cli not found — skipping Redis check"; \
	fi

# ── Generate Templ files ──
templ-generate:
	@echo "==> Generating Templ files…"
	$(TEMPL) generate

# ── Build the Go binary ──
build:
	@echo "==> Building Go binary…"
	go build -ldflags="-s -w" -o ./tmp/main ./cmd/api

# ── Regenerate Swagger docs ──
swagger:
	@echo "==> Regenerating Swagger docs…"
	/home/shino/go/bin/swag init -g cmd/api/main.go -o docs

# ── Run tests ──
test:
	@echo "==> Running tests…"
	go test ./... -count=1

# ── Clean build artifacts ──
clean:
	@echo "==> Cleaning…"
	rm -rf ./tmp

# ── Development server (hot-reload) ──
# Runs templ generate once, checks Redis, then starts air for hot-reload.
dev: install-tools templ-generate check-redis
	@echo "==> Starting development server with hot-reload…"
	$(AIR)

# ── Publish mock SOC events to Redis ──
testsoc:
	@echo "==> Publishing mock SOC events to Redis…"
	REDIS_ADDR=$${REDIS_ADDR:-localhost:6379}; \
	go run ./cmd/cli/publish-soc-events -count 5 -redis $$REDIS_ADDR