# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Lore is a Stellar blockchain token explorer for MTLAP (Persons) and MTLAC (Companies) tokens. It provides a web interface to browse token holders and view detailed account information including balances, metadata, and trustlines.

## Build & Development Commands

```bash
# Build the binary
go build -o lore ./cmd/lore

# Run the application (default port 8080)
./lore
./lore --port 3000
./lore --horizon-url https://horizon-testnet.stellar.org

# Run tests
go test ./internal/service

# Format and lint
go fmt ./...
go vet ./...
```

## Architecture

The application follows a 3-layer architecture:

- **Handler Layer** (`internal/handler/`) - HTTP request handling with Go 1.22+ routing. Routes: `GET /` (home) and `GET /accounts/{id}` (detail). Uses `r.PathValue()` for path parameters.

- **Service Layer** (`internal/service/`) - `StellarService` wraps the Stellar Horizon API client. Key methods: `GetAccountsWithAsset()` for paginated token holder lists, `GetAccountDetail()` for single account info. Includes utilities for base64 decoding and numbered metadata field parsing.

- **Template Layer** (`internal/template/`) - Embedded templates using Go's `embed` package. `base.html` provides master layout, extended by `home.html` and `account.html`.

## Key Technical Details

- **Stellar Metadata**: Account data is stored in base64 on Stellar; the service layer decodes transparently
- **Pagination**: Cursor-based pagination via Stellar Horizon's cursors, passed as `persons_cursor` and `companies_cursor` query params
- **Numbered Fields**: Account metadata like websites use numbered keys (Website0, Website1) parsed and sorted by `parseNumberedDataKeys()`
- **Configuration**: Port via `--port`/`PORT`, Horizon URL via `--horizon-url`/`HORIZON_URL`, log level via `--log-level`/`LOG_LEVEL` (debug, info, warn, error)
- **Logging**: Uses `log/slog` with JSON output and source location. Log levels: `info` for lifecycle events, `error` for unexpected failures (not expected errors like 404), `debug` for troubleshooting
- **Token Constants**: Defined in `internal/config/config.go` (MTLAP, MTLAC, issuer address)
- **Template Inheritance**: Each page template must be cloned from base separately (see `template.go`). Using `ParseFS` with multiple templates defining the same block causes overwrites.

## Testing

Tests in `internal/service/stellar_test.go` cover utility functions (`parseNumberedDataKeys`, `decodeBase64`) with edge cases for numbered keys, base64 encoding, and error conditions.
