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

# Run sync to fetch data from Stellar Horizon
./lore sync --database-url "..."
./lore sync --database-url "..." --full  # Full resync (truncates tables first)

# Run PostgreSQL locally with Docker
docker run -d --name lore-db -e POSTGRES_PASSWORD=lore -e POSTGRES_DB=lore -p 5432:5432 postgres:16

# Run tests
go test ./...

# Format and lint
go fmt ./...
go vet ./...
```

## Architecture

The application follows a layered architecture:

- **Handler Layer** (`internal/handler/`) - HTTP request handling with Go 1.22+ routing. Routes: `GET /` (home) and `GET /accounts/{id}` (detail). Uses `r.PathValue()` for path parameters.

- **Repository Layer** (`internal/repository/`) - Data access layer for PostgreSQL. `AccountRepository` provides methods for querying accounts, stats, persons (MTLAP holders), and companies (MTLAC holders). Uses Squirrel query builder.

- **Database Layer** (`internal/database/`) - PostgreSQL connection pool management via pgx. Includes embedded goose migrations that run automatically on startup. Pool settings: MaxConns=10, MinConns=2.

- **Service Layer** (`internal/service/`) - `StellarService` wraps the Stellar Horizon API client. Key methods: `GetAccountsWithAsset()` for paginated token holder lists, `GetAccountDetail()` for single account info. Includes utilities for base64 decoding and numbered metadata field parsing.

- **Sync Layer** (`internal/sync/`) - Data synchronization from Stellar Horizon API to PostgreSQL. Fetches MTLAP/MTLAC holders, parses ManageData, calculates delegations, and fetches token prices from SDEX. Uses semaphore + WaitGroup for concurrent processing with 10-worker limit.

- **Template Layer** (`internal/template/`) - Embedded templates using Go's `embed` package. `base.html` provides master layout, extended by `home.html` and `account.html`. Custom functions: `add`, `truncate`, `slice`.

## Key Technical Details

- **Stellar Metadata**: Account data is stored in base64 on Stellar; the service layer decodes transparently
- **Stellar SDK Types**: `horizon.Balance.Asset` is type `base.Asset` (from `github.com/stellar/go/protocols/horizon/base`), not `horizon.Asset`. Import the `base` package when writing tests.
- **Pagination**: Offset-based pagination for database queries, passed as `persons_offset` and `companies_offset` query params. Horizon API uses cursor-based pagination for account detail pages.
- **Numbered Fields**: Account metadata like websites use numbered keys (Website0, Website1) parsed and sorted by `parseNumberedDataKeys()`
- **Configuration**: Port via `--port`/`PORT`, Horizon URL via `--horizon-url`/`HORIZON_URL`, log level via `--log-level`/`LOG_LEVEL` (debug, info, warn, error), database URL via `--database-url`/`DATABASE_URL` (required)
- **Logging**: Uses `log/slog` with JSON output and source location. Log levels: `info` for lifecycle events, `error` for unexpected failures (not expected errors like 404), `debug` for troubleshooting
- **Token Constants**: Defined in `internal/config/config.go` (MTLAP, MTLAC, issuer address)
- **Template Inheritance**: Each page template must be cloned from base separately (see `template.go`). Using `ParseFS` with multiple templates defining the same block causes overwrites.
- **Constructor Pattern**: Constructors return `(*T, error)` with nil validation, not `*T` that silently returns nil on error.
- **Transaction Atomicity**: Wrap DELETE + INSERT sequences in transactions to prevent partial state on failure.

## samber/lo - Utility Library

Use `github.com/samber/lo` for readable, type-safe slice/map operations. Prefer `lo` helpers over manual loops.

### Slice Operations
```go
lo.Filter(slice, func(x T, _ int) bool { return condition })  // Filter elements
lo.Map(slice, func(x T, _ int) R { return transform(x) })     // Transform elements
lo.Reduce(slice, func(acc R, x T, _ int) R { ... }, init)     // Reduce to single value
lo.ForEach(slice, func(x T, _ int) { ... })                   // Iterate with side effects
lo.Uniq(slice)                                                 // Remove duplicates
lo.UniqBy(slice, func(x T) K { return key })                  // Remove duplicates by key
lo.Compact(slice)                                              // Remove zero values ("", 0, nil)
lo.Flatten(nested)                                             // Flatten nested slices
lo.Chunk(slice, size)                                          // Split into chunks
lo.GroupBy(slice, func(x T) K { return key })                 // Group by key -> map[K][]T
lo.KeyBy(slice, func(x T) K { return key })                   // Index by key -> map[K]T
lo.Partition(slice, func(x T, _ int) bool { ... })            // Split into [match, nomatch]
```

### Search Operations
```go
lo.Find(slice, func(x T) bool { return condition })           // Returns (value, found)
lo.FindOrElse(slice, fallback, func(x T) bool { ... })        // Returns value or fallback
lo.Contains(slice, value)                                      // Check if exists
lo.IndexOf(slice, value)                                       // Find index (-1 if not found)
lo.Every(slice, func(x T, _ int) bool { ... })                // All match predicate
lo.Some(slice, func(x T, _ int) bool { ... })                 // Any matches predicate
```

### Map Operations
```go
lo.Keys(m)                                                     // Get all keys
lo.Values(m)                                                   // Get all values
lo.PickBy(m, func(k K, v V) bool { ... })                     // Filter map entries
lo.OmitBy(m, func(k K, v V) bool { ... })                     // Exclude map entries
lo.MapKeys(m, func(v V, k K) K2 { return newKey })            // Transform keys
lo.MapValues(m, func(v V, k K) V2 { return newValue })        // Transform values
lo.Invert(m)                                                   // Swap keys and values
lo.Assign(maps...)                                             // Merge maps (later wins)
```

### Safety & Error Handling
```go
lo.Must(val, err)                                              // Panic on error, return val
lo.Must0(err)                                                  // Panic on error (no return)
lo.Must2(v1, v2, err)                                          // Panic on error, return v1, v2
lo.Coalesce(vals...)                                           // First non-zero value
lo.CoalesceOrEmpty(vals...)                                    // First non-zero or zero value
lo.IsEmpty(val)                                                // Check if zero value
lo.FromPtr(ptr)                                                // Dereference or zero value
lo.ToPtr(val)                                                  // Create pointer to value
lo.Ternary(cond, ifTrue, ifFalse)                             // Inline conditional
lo.If(cond, val).Else(other)                                  // Fluent conditional
```

### Parallel Processing
```go
import lop "github.com/samber/lo/parallel"
lop.Map(slice, func(x T, _ int) R { ... })                    // Parallel map
lop.ForEach(slice, func(x T, _ int) { ... })                  // Parallel iteration
lop.Filter(slice, func(x T, _ int) bool { ... })              // Parallel filter
```

## Testing

- `internal/service/stellar_test.go` - utility functions (`parseNumberedDataKeys`, `decodeBase64`)
- `internal/sync/*_test.go` - sync parsing functions (`parseAccountData`, `parseAssociationTags`, `getAssetType`)
- Use table-driven tests with `t.Run()` for edge cases

## Git Conventions

- **Commit messages**: Use [Conventional Commits](https://www.conventionalcommits.org/) format (e.g., `feat:`, `fix:`, `refactor:`, `docs:`, `chore:`)
