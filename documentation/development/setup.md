# Development Setup

## Prerequisites

- **Go 1.26+** (check `go.mod` for exact version)
- **Docker** and **Docker Compose** (for PostgreSQL)
- **Air** (optional, for hot reload) — `go install github.com/air-verse/air@latest`
- **swaggo/swag** (optional, for Swagger spec regeneration) — `go install github.com/swaggo/swag/v2/cmd/swag@latest`

## Quick start

### 1. Clone and start PostgreSQL

```bash
git clone https://github.com/patiHash1/Strata-prototype.git
cd Strata-prototype

# Start PostgreSQL (Docker)
docker compose up -d
```

### 2. Create a .env file (optional)

```bash
# .env
PORT=8080
DATABASE_URL=postgres://strata-user:strata-pass@localhost:5432/strata-db-beta?sslmode=disable
JWT_SECRET=change-this-to-a-secure-secret-in-production
JWT_ISSUER=strata
ENABLE_SWAGGER=true
```

### 3. Run the server

```bash
# Standard
go run ./cmd/api

# Or with hot-reload
air
```

### 4. Verify it works

```bash
curl http://localhost:8080/health
```

Expected response:

```json
{"database":"connected","status":"ok"}
```

### 5. Access Swagger UI

Open [http://localhost:8080/swagger/](http://localhost:8080/swagger/) in your browser.

## Smoke test — register and login

```bash
# Register a new organization
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "company_name": "My Company",
    "domain_slug": "my-company",
    "owner_email": "owner@mycompany.com",
    "owner_password": "password123",
    "owner_full_name": "John Doe"
  }'

# Response will include an access_token — save it for subsequent requests
# Example:
# {"access_token":"eyJ...","org_id":"...","user_id":"..."}
```

## Running tests

```bash
# Run all tests
go test ./...

# Run tests with race detection
go test -race ./...

# Run tests for a specific package
go test ./internal/services/...
```

## Database management

### Connect to PostgreSQL directly

```bash
docker exec -it strata-db-test psql -U strata-user -d strata-db-beta
```

### Reset the database

```bash
docker compose down -v
docker compose up -d
```

This destroys the volume and recreates a fresh database. The server will re-run migrations on startup.

## Code quality

```bash
# Format code
go fmt ./...

# Vet for issues
go vet ./...

# Run all checks
go fmt ./... && go vet ./... && go test ./...
```

## Dependency management

```bash
# Add a new dependency
go get github.com/example/library@v1.0.0

# Tidy up go.mod and go.sum
go mod tidy

# Update all dependencies
go get -u ./...
go mod tidy
```

## Project commands

| Command | Description |
|---|---|
| `go run ./cmd/api` | Start the API server |
| `air` | Start with hot-reload |
| `go test ./...` | Run all tests |
| `go vet ./...` | Static analysis |
| `go fmt ./...` | Format all Go files |
| `go mod tidy` | Clean up module dependencies |
| `swag init ...` | Regenerate Swagger spec |
| `docker compose up -d` | Start PostgreSQL |
| `docker compose down` | Stop PostgreSQL |

## Troubleshooting

### Database connection refused

Ensure PostgreSQL is running:

```bash
docker compose ps
```

If the container is not running, start it:

```bash
docker compose up -d
```

### Port already in use

Change the port via the `PORT` environment variable or `.env` file:

```bash
PORT=8081 go run ./cmd/api
```

### Swagger UI not loading

Ensure `ENABLE_SWAGGER=true` is set and the Swagger spec has been generated:

```bash
swag init --dir ./cmd/api,./internal/handlers --output ./docs --parseDependency --parseInternal
```