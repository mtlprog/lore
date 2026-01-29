# Lore

Stellar blockchain token explorer for MTLAP (Persons) and MTLAC (Companies) tokens. Provides a web interface to browse token holders and view detailed account information including balances, metadata, and trustlines.

## Overview

Lore is a specialized explorer for the [MTL Association](https://mtla.me/) ecosystem on the Stellar network. It tracks:

- **MTLAP (Persons)** - Individual membership tokens
- **MTLAC (Companies)** - Company/organization tokens

### Key Features

- Browse all MTLAP and MTLAC token holders
- View detailed account information (name, about, websites, trustlines)
- Council voting status and delegation tracking
- Portfolio valuation in XLM
- Dark/light theme with responsive design

### Blockchain Social Network (BSN)

The MTL ecosystem implements a social network layer on Stellar using ManageData operations:

- **Identity**: Name, About, Websites stored as base64-encoded ManageData
- **Relationships**: 27 relationship types (Employee, Employer, Partner, Family, etc.) linking accounts
- **Delegation**: Council voting power delegation chains
- **Association Tags**: Programs and Factions for categorizing members

## Architecture

```
cmd/lore/           - CLI entry point (serve, sync commands)
internal/
├── config/         - Configuration constants
├── database/       - PostgreSQL connection + migrations
├── handler/        - HTTP handlers (Home, Account)
├── logger/         - Structured logging (slog)
├── model/          - Data models
├── repository/     - Data access layer (Squirrel query builder)
├── service/        - Stellar Horizon API client
├── sync/           - Data synchronization orchestration
└── template/       - HTML templates (embedded)
```

## Prerequisites

- Go 1.24+
- PostgreSQL 16+
- Docker (optional, for local PostgreSQL)

## Quick Start

### 1. Start PostgreSQL

```bash
# Using Docker
docker run -d \
  --name lore-db \
  -e POSTGRES_PASSWORD=lore \
  -e POSTGRES_DB=lore \
  -p 5432:5432 \
  postgres:16
```

### 2. Build

```bash
go build -o lore ./cmd/lore
```

### 3. Sync Data

```bash
# Initial sync (fetches all MTLAP/MTLAC holders from Stellar)
./lore sync --database-url "postgres://postgres:lore@localhost:5432/lore?sslmode=disable"

# Full resync (truncates tables first)
./lore sync --database-url "..." --full
```

### 4. Start Server

```bash
./lore serve --database-url "postgres://postgres:lore@localhost:5432/lore?sslmode=disable"

# Custom port
./lore serve --database-url "..." --port 3000

# Debug logging
./lore serve --database-url "..." --log-level debug
```

Open http://localhost:8080

## Configuration

| Flag | Env Var | Default | Description |
|------|---------|---------|-------------|
| `--database-url` | `DATABASE_URL` | (required) | PostgreSQL connection URL |
| `--port` | `PORT` | `8080` | HTTP server port |
| `--horizon-url` | `HORIZON_URL` | `https://horizon.stellar.org` | Stellar Horizon API URL |
| `--log-level` | `LOG_LEVEL` | `info` | Log level (debug, info, warn, error) |

## Development

### Run Tests

```bash
go test ./...
```

### Generate Mocks

```bash
go generate ./...
# or
mockery
```

### Format Code

```bash
go fmt ./...
go vet ./...
```

## Data Sources

- **Stellar Horizon API** - Account data, balances, ManageData
- **SDEX** - Token prices for portfolio valuation

## License

MIT
