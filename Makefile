.PHONY: help build up down logs clean test

help:
	@echo "EdgeBase - Available commands:"
	@echo "  make build          - Build all services"
	@echo "  make up             - Start all services"
	@echo "  make down           - Stop all services"
	@echo "  make logs           - View service logs"
	@echo "  make clean          - Clean build artifacts"
	@echo "  make test           - Run all tests"
	@echo "  make db-build       - Build database service"
	@echo "  make functions-build - Build functions service"
	@echo "  make platform-build - Build platform service"

build:
	docker-compose build

up:
	docker-compose up -d

down:
	docker-compose down

logs:
	docker-compose logs -f

clean:
	docker-compose down -v
	cd db && cargo clean
	cd functions && cargo clean
	cd platform/control-plane && go clean

test:
	cd db && cargo test
	cd functions && cargo test
	cd platform/control-plane && go test ./...

db-build:
	cd db && cargo build --release

functions-build:
	cd functions && cargo build --release

platform-build:
	cd platform/control-plane && go build -o bin/control-plane ./cmd/server

db-test:
	cd db && cargo test

functions-test:
	cd functions && cargo test

platform-test:
	cd platform/control-plane && go test ./...

ps:
	docker-compose ps

restart:
	docker-compose restart

shell-db:
	docker-compose exec postgres psql -U edgebase -d edgebase

shell-mqtt:
	docker-compose exec mqtt sh
