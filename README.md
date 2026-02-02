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
- Reputation scoring with weighted calculations
- Portfolio valuation in XLM
- Relationship graph visualization
- Dark/light theme with responsive design

### Blockchain Social Network

The MTL ecosystem implements a social network layer on Stellar using ManageData operations:

- **Identity**: Name, About, Websites stored as base64-encoded ManageData
- **Relationships**: 27 relationship types (Employee, Employer, Partner, Family, etc.) linking accounts
- **Delegation**: Council voting power delegation chains
- **Association Tags**: Programs and Factions for categorizing members

## Architecture

```
cmd/lore/           - CLI entry point (serve, sync commands)
internal/
├── config/         - Configuration constants (tokens, issuer)
├── database/       - PostgreSQL connection + goose migrations
├── handler/        - HTTP handlers (Home, Account, Search, Init)
├── logger/         - Structured logging (slog/JSON)
├── model/          - Data models
├── repository/     - Data access layer (Squirrel query builder)
├── reputation/     - Weighted reputation scoring system
├── service/        - Stellar Horizon API client + XDR generation
├── sync/           - Data synchronization from Horizon to PostgreSQL
└── template/       - Embedded HTML templates
```

## Prerequisites

- Go 1.24+
- PostgreSQL 16+
- Docker & Docker Compose (recommended)

## Development

### Docker Development (Recommended)

Use Docker Compose for local development with fast iteration:

```bash
# Quick start: build, start services, sync data
make dev

# After making code changes, rebuild and restart:
make dev-restart

# View logs
make dev-logs

# Stop everything
make dev-down

# Reset database and re-sync
make db-reset && make dev
```

**Important**: Docker containers run Linux. The Makefile handles cross-compilation automatically. Never run `go build` manually before `make dev-restart` — always use Makefile targets.

### Local Development (without Docker for app)

```bash
# Start only PostgreSQL
make db

# Build
go build -o lore ./cmd/lore

# Sync data from Stellar
./lore --database-url "postgres://lore:lore@localhost:5432/lore?sslmode=disable" sync

# Start server
./lore --database-url "postgres://lore:lore@localhost:5432/lore?sslmode=disable" serve
```

Open http://localhost:8080

### Commands

```bash
# Run tests
go test ./...

# Format and lint
go fmt ./...
go vet ./...

# Generate mocks
mockery
```

## Configuration

| Flag | Env Var | Default | Description |
|------|---------|---------|-------------|
| `--database-url` | `DATABASE_URL` | (required) | PostgreSQL connection URL |
| `--port` | `PORT` | `8080` | HTTP server port |
| `--horizon-url` | `HORIZON_URL` | `https://horizon.stellar.org` | Stellar Horizon API URL |
| `--log-level` | `LOG_LEVEL` | `info` | Log level (debug, info, warn, error) |

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines on how to contribute to this project.

## License

MIT
