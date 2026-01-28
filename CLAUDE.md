# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Lore is a Stellar blockchain token explorer for MTLAP (Persons) and MTLAC (Companies) tokens. It provides a web interface to browse token holders and view detailed account information including balances, metadata, and trustlines.

## Build & Development Commands

```bash
# Build the binary
go build -o lore ./cmd/lore

# Run the application (requires PostgreSQL)
./lore --database-url "postgres://user:pass@localhost:5432/lore?sslmode=disable"
./lore --database-url "..." --port 3000
./lore --database-url "..." --horizon-url https://horizon-testnet.stellar.org

# Run PostgreSQL locally with Docker
docker run -d --name lore-db -e POSTGRES_PASSWORD=lore -e POSTGRES_DB=lore -p 5432:5432 postgres:16

# Run tests
go test ./internal/service

# Format and lint
go fmt ./...
go vet ./...
```

## Architecture

The application follows a 5-layer architecture:

- **Handler Layer** (`internal/handler/`) - HTTP request handling with Go 1.22+ routing. Routes: `GET /` (home) and `GET /accounts/{id}` (detail). Uses `r.PathValue()` for path parameters.

- **Repository Layer** (`internal/repository/`) - Data access layer for PostgreSQL. `AccountRepository` provides methods for querying accounts, stats, persons (MTLAP holders), and companies (MTLAC holders). Uses Squirrel query builder.

- **Database Layer** (`internal/database/`) - PostgreSQL connection pool management via pgx. Includes embedded goose migrations that run automatically on startup. Pool settings: MaxConns=10, MinConns=2.

- **Service Layer** (`internal/service/`) - `StellarService` wraps the Stellar Horizon API client. Key methods: `GetAccountsWithAsset()` for paginated token holder lists, `GetAccountDetail()` for single account info. Includes utilities for base64 decoding and numbered metadata field parsing.

- **Template Layer** (`internal/template/`) - Embedded templates using Go's `embed` package. `base.html` provides master layout, extended by `home.html` and `account.html`. Custom functions: `add`, `truncate`, `slice`.

## Key Technical Details

- **Stellar Metadata**: Account data is stored in base64 on Stellar; the service layer decodes transparently
- **Pagination**: Offset-based pagination for database queries, passed as `persons_offset` and `companies_offset` query params. Horizon API uses cursor-based pagination for account detail pages.
- **Numbered Fields**: Account metadata like websites use numbered keys (Website0, Website1) parsed and sorted by `parseNumberedDataKeys()`
- **Configuration**: Port via `--port`/`PORT`, Horizon URL via `--horizon-url`/`HORIZON_URL`, log level via `--log-level`/`LOG_LEVEL` (debug, info, warn, error), database URL via `--database-url`/`DATABASE_URL` (required)
- **Logging**: Uses `log/slog` with JSON output and source location. Log levels: `info` for lifecycle events, `error` for unexpected failures (not expected errors like 404), `debug` for troubleshooting
- **Token Constants**: Defined in `internal/config/config.go` (MTLAP, MTLAC, issuer address)
- **Template Inheritance**: Each page template must be cloned from base separately (see `template.go`). Using `ParseFS` with multiple templates defining the same block causes overwrites.

## Testing

Tests in `internal/service/stellar_test.go` cover utility functions (`parseNumberedDataKeys`, `decodeBase64`) with edge cases for numbered keys, base64 encoding, and error conditions.

## Git Conventions

- **Commit messages**: Use [Conventional Commits](https://www.conventionalcommits.org/) format (e.g., `feat:`, `fix:`, `refactor:`, `docs:`, `chore:`)
