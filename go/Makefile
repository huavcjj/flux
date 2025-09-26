.PHONY: build run test clean dev up down migrate-create migrate-up migrate-down migrate-status sqlc-install sqlc-generate

# Build the application
build:
	go build -o bin/server ./cmd/server

# Run with hot reload
run:
	air

# Run tests
test:
	go test ./...

# Clean build artifacts
clean:
	rm -rf bin/ tmp/

# Start development environment
dev:
	docker-compose up -d

# Start all services
up:
	docker-compose up -d

# Stop all services
down:
	docker-compose down

# Create a new migration
migrate-new:
	@read -p "Enter migration name: " name; \
	dbmate --url mysql://root:password@localhost:3306/dbname new $$name

# Run migrations
migrate-up:
	dbmate --url mysql://root:password@localhost:3306/dbname up

# Rollback migrations
migrate-down:
	dbmate --url mysql://root:password@localhost:3306/dbname down

# Show migration status
migrate-status:
	dbmate --url mysql://root:password@localhost:3306/dbname status

# Generate Go code from SQL queries using sqlc
sqlc-generate:
	sqlc generate