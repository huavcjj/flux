.DEFAULT_GOAL := help

.PHONY: help build run test clean dev up down migrate-new migrate-up migrate-down migrate-status sqlc-generate

# Show help
help:
	@echo "Available targets:"
	@echo "  make build          - Build the application"
	@echo "  make run            - Run with hot reload (air)"
	@echo "  make test           - Run tests"
	@echo "  make clean          - Clean build artifacts"
	@echo "  make dev            - Start development environment"
	@echo "  make up             - Start all services"
	@echo "  make down           - Stop all services"
	@echo "  make migrate-new    - Create a new migration"
	@echo "  make migrate-up     - Run migrations"
	@echo "  make migrate-down   - Rollback migrations"
	@echo "  make migrate-status - Show migration status"
	@echo "  make sqlc-generate  - Generate Go code from SQL queries"

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
	docker-compose up -d --build

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