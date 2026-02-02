# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Lore is a Stellar blockchain token explorer for MTLAP (Persons) and MTLAC (Companies) tokens. It provides a web interface to browse token holders and view detailed account information including balances, metadata, and trustlines.

## Build & Development Commands

```bash
# Build the binary
go build -o lore ./cmd/lore

# Run tests
go test ./...

# Format and lint
go fmt ./...
go vet ./...
```

## Docker Development (Recommended)

Use Docker Compose for local development with fast iteration:

```bash
# Quick start: build, start services, sync data
make dev

# After making code changes, rebuild and restart (fast - no container rebuild):
make dev-restart

# View logs
make dev-logs

# Stop everything
make dev-down

# Reset database and re-sync
make db-reset && make dev
```

### How it works
- `docker-compose.dev.yml` mounts the local binary into containers
- `make dev` and `make dev-restart` use `build-linux` target which cross-compiles for Linux
- Syncer runs once on startup; restart it manually with `docker compose restart syncer`

### IMPORTANT: Cross-compilation for Docker
Docker containers run Linux. The Makefile handles this automatically:
- `make build` — builds for local macOS (use with `make run`)
- `make build-linux` — builds for Linux/arm64 (used by `make dev` and `make dev-restart`)

**Never run `go build` manually before `make dev-restart`** — it will build a macOS binary that won't work in the Linux container. Always use the Makefile targets.

## Docker Production

```bash
# Build images and start all services
make prod

# View logs
make prod-logs

# Stop
make prod-down
```

## Local Development (without Docker containers for app)

```bash
# Start only PostgreSQL
make db

# Run the application
./lore --database-url "postgres://lore:lore@localhost:5432/lore?sslmode=disable" serve

# Run sync
./lore --database-url "postgres://lore:lore@localhost:5432/lore?sslmode=disable" sync
./lore --database-url "..." sync --full  # Full resync (truncates tables first)
```

## Architecture

The application follows a layered architecture:

- **Handler Layer** (`internal/handler/`) - HTTP request handling with Go 1.22+ routing. Uses `r.PathValue()` for path parameters. Routes: `GET /` (home), `GET /accounts/{id}` (detail), `GET /accounts/{id}/reputation`, `GET /transactions/{hash}`, `GET /search`, `GET /tokens/{issuer}/{code}`, `GET|POST /init/*` (forms). Note: `GET /` matches any path; to test "unregistered routes", use wrong HTTP method.

- **Repository Layer** (`internal/repository/`) - Data access layer for PostgreSQL. `AccountRepository` provides methods for querying accounts, stats, persons (MTLAP holders), and corporate accounts (MTLAC holders). Uses Squirrel query builder.

- **Database Layer** (`internal/database/`) - PostgreSQL connection pool management via pgx. Includes embedded goose migrations that run automatically on startup. Pool settings: MaxConns=10, MinConns=2.

- **Service Layer** (`internal/service/`) - `StellarService` wraps the Stellar Horizon API client. Key methods: `GetAccountsWithAsset()` for paginated token holder lists, `GetAccountDetail()` for single account info. Includes utilities for base64 decoding and numbered metadata field parsing.

- **Sync Layer** (`internal/sync/`) - Data synchronization from Stellar Horizon API to PostgreSQL. Fetches MTLAP/MTLAC holders, parses ManageData, calculates delegations, and fetches token prices from SDEX. Uses semaphore + WaitGroup for concurrent processing with 10-worker limit.

- **Reputation Layer** (`internal/reputation/`) - Weighted reputation scoring system. `Rating` is a constrained type (A/B/C/D) with `IsValid()` and `Value()` methods. `Calculator` computes weighted scores based on rater portfolio and connections. `Graph` builds 2-level visualization of rating relationships.

- **Template Layer** (`internal/template/`) - Embedded templates using Go's `embed` package. `base.html` provides master layout, extended by `home.html` and `account.html`. Custom functions: `add`, `addFloat` (float64 + int), `truncate`, `slice`, `formatNumber` (space-separated thousands), `votePower` (log10-based: 1-10=1, 11-100=2, 101-1000=3), `markdown` (renders Markdown with XSS sanitization), `searchURL` (builds /search URLs with query params).

## Key Technical Details

- **Template Buffering**: Render templates to `bytes.Buffer` first, write to ResponseWriter only on success. Prevents partial HTML on template errors.
- **Stellar Metadata**: Account data is stored in base64 on Stellar; the service layer decodes transparently
- **Stellar SDK Types**: `horizon.Balance` embeds `base.Asset`, so prefer `bal.Code` over `bal.Asset.Code` (staticcheck QF1008). Import `base` package when writing tests.
- **Pagination**: Offset-based pagination for database queries, passed as `persons_offset` and `corporate_offset` query params. Horizon API uses cursor-based pagination for account detail pages.
- **Numbered Fields**: Account metadata like websites use numbered keys (Website0, Website1) parsed and sorted by `parseNumberedDataKeys()`
- **Tag Fields**: Account tags use `Tag*` prefix keys (TagBelgrade, TagProgrammer) parsed by `parseTagKeys()`. Value is account ID (ignored for display).
- **AND Filtering Pattern**: For requiring all values match, use `GROUP BY + HAVING COUNT(DISTINCT column) = N` in subqueries.
- **Configuration**: Port via `--port`/`PORT`, Horizon URL via `--horizon-url`/`HORIZON_URL`, log level via `--log-level`/`LOG_LEVEL` (debug, info, warn, error), database URL via `--database-url`/`DATABASE_URL` (required)
- **Logging**: Uses `log/slog` with JSON output and source location. Log levels: `info` for lifecycle events, `error` for unexpected failures (not expected errors like 404), `debug` for troubleshooting
- **Token Constants**: Defined in `internal/config/config.go` (MTLAP, MTLAC, issuer address)
- **Template Inheritance**: Each page template must be cloned from base separately (see `template.go`). Using `ParseFS` with multiple templates defining the same block causes overwrites.
- **Markdown Rendering**: Uses blackfriday for Markdown→HTML and bluemonday for XSS sanitization. Template function: `{{markdown .Field}}`. Always sanitize untrusted blockchain data.
- **CSS Specificity in Templates**: `.detail-block-content a` sets link color globally. Override with specific selectors like `.detail-block-content .tag-chip` when styling links inside detail blocks.
- **No-JS Design**: Avoid page-load animations (like `slide-up`) on interactive elements. Causes flashing when user clicks links that reload the page.
- **Constructor Pattern**: Constructors return `(*T, error)` with nil validation, not `*T` that silently returns nil on error.
- **Constrained Type Pattern**: For domain values with limited valid states (like `Rating`), use a typed string with `IsValid()` method and constants for valid values. Methods like `Value()` can provide numeric conversions.
- **Transaction Atomicity**: Wrap DELETE + INSERT sequences in transactions to prevent partial state on failure.
- **Database Migrations**: Add new migrations in `internal/database/migrations/` with format `NNN_description.sql`. Migrations run automatically on startup via goose.
- **No Foreign Key Constraints**: Do not use foreign key constraints in database schema. Data integrity is managed at the application level during sync operations.
- **Relation Index Preservation**: Relationship indices are stored as strings, not integers, to preserve leading zeros. `PartOf002` and `PartOf2` are distinct keys in the blockchain that must remain distinct in the database. Converting to int would cause both to become `2`, violating the primary key constraint `(source_account_id, target_account_id, relation_type, relation_index)`.

## Stellar Account Data Keys

- **mtla_delegate**: General delegation target (account ID that receives delegated votes)
- **mtla_c_delegate**: Has dual meaning:
  - `"ready"` → account is council-ready (can receive council delegations)
  - Account ID → delegates council votes to that account
- **Council delegation chains**: Use `council_delegate_to` column, not `delegate_to`, for vote calculations

## Stellar XDR Generation

- **Init Forms** (`internal/service/init.go`) - Generate XDR by comparing original vs current form state, producing only the diff as ManageData operations
- Use `txnbuild.ManageData` with `Value: nil` to delete a key, non-nil to set/update
- Always validate account IDs with `keypair.ParseAddress()` before building transactions

## Montelibero Relationship Types

The Montelibero Blockchainization standard defines relationship types stored as account ManageData entries.

### Complementary Pairs (require both for "confirmed" status)

| Tag A (direction) | Tag B (direction) | Meaning |
|-------------------|-------------------|---------|
| MyPart (org→person) | PartOf (person→org) | Membership |
| Guardian (guardian→ward) | Ward (ward→guardian) | Guardianship |
| OwnershipFull (corp→owner) | Owner (owner→corp) | 95%+ ownership |
| OwnershipMajority (corp→owner) | OwnerMajority (owner→corp) | 25-95% ownership |
| OwnershipMinority (corp→owner) | OwnerMinority (owner→corp) | <25% ownership |
| Employer (employer→employee) | Employee (employee→employer) | Employment |

### Symmetric Types (both set same tag = "mutual")

Only display when BOTH parties have declared. Hide one-way declarations.

- Spouse, OneFamily, Partnership, Collaboration, FactionMember

### Unilateral Tags (no confirmation needed)

- A/B/C/D (credit rating), Sympathy, Love, Divorce, Contractor, Client, WelcomeGuest, RecommendToMTLA

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
- `internal/handler/*_test.go` - HTTP handlers using mockery-generated mocks
- `internal/reputation/*_test.go` - reputation calculator and types
- Use table-driven tests with `t.Run()` for edge cases

### Mocking with mockery
- Config in `.mockery.yaml`, regenerate with `mockery` or `go generate ./internal/handler/`
- When renaming interfaces, update `.mockery.yaml` and delete old mock files before regenerating
- Use `EXPECT()` pattern with specific expectations; avoid `.Maybe()` in favor of isolated sub-tests with fresh mocks

## Git Conventions

- **Commit messages**: Use [Conventional Commits](https://www.conventionalcommits.org/) format (e.g., `feat:`, `fix:`, `refactor:`, `docs:`, `chore:`)
