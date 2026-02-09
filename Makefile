.PHONY: build build-linux dev dev-restart dev-logs dev-down db db-reset run sync test fmt vet lint clean mocks swagger

# Build for local macOS
build:
	go build -o lore ./cmd/lore

# Build for Linux (Docker containers)
build-linux:
	GOOS=linux GOARCH=arm64 go build -o lore ./cmd/lore

# Start dev environment (build + docker compose up)
dev: build-linux
	docker compose up -d
	@echo "Server running at http://localhost:8080"

# Restart dev after code changes (rebuild + restart containers)
dev-restart: build-linux
	docker compose restart server syncer

# View dev logs
dev-logs:
	docker compose logs -f

# Stop dev environment
dev-down:
	docker compose down

# Start only PostgreSQL
db:
	docker compose up -d db

# Reset database (stop, remove volume, start fresh)
db-reset:
	docker compose down -v
	docker compose up -d db
	@echo "Waiting for db..."
	@sleep 3

# Run locally (requires PostgreSQL running)
run: build
	./lore --database-url "postgres://lore:lore@localhost:5432/lore?sslmode=disable" serve

# Run sync locally
sync: build
	./lore --database-url "postgres://lore:lore@localhost:5432/lore?sslmode=disable" sync

# Run sync in docker
sync-docker:
	docker compose run --rm syncer

# Run tests
test:
	go test ./... -v

# Run short tests
test-short:
	go test ./... -short

# Format code
fmt:
	go fmt ./...

# Vet code
vet:
	go vet ./...

# Lint (format + vet)
lint: fmt vet

# Clean build artifacts
clean:
	rm -f lore
	docker compose down -v

# Regenerate mocks
mocks:
	mockery

# Regenerate Swagger docs
swagger:
	swag init -g cmd/lore/main.go -o internal/api/docs
