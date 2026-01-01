.PHONY: all build run test clean deps lint

BINARY_NAME=server
MAIN_PATH=./cmd/server

all: build

build:
	@echo "Building..."
	go build -o bin/$(BINARY_NAME) $(MAIN_PATH)

run:
	@echo "Running..."
	go run $(MAIN_PATH)

test:
	@echo "Testing..."
	go test -v ./...

clean:
	@echo "Cleaning..."
	go clean
	rm -rf bin/

deps:
	@echo "Downloading dependencies..."
	go mod download

lint:
	@echo "Linting..."
	go vet ./...
