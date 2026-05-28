.PHONY: build run test clean deps test-api

# Build the memory-brain binary
build:
	go build -o bin/memory-brain ./cmd/main.go

# Run the server (default port 12321)
run:
	go run ./cmd/main.go server --port 12321

# Run tests
test:
	go test -v ./...

# Clean build artifacts
clean:
	rm -rf bin/

# Install dependencies
deps:
	go mod download
	go mod tidy

# Run API integration tests
test-api:
	@./scripts/test_api.sh