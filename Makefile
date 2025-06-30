# NeuroMesh Makefile

.PHONY: build test clean proto-gen help

# Default target
help:
	@echo "Available targets:"
	@echo "  build      - Build the main server"
	@echo "  build-ui   - Build the chat UI"
	@echo "  test       - Run all tests"
	@echo "  proto-gen  - Regenerate protobuf files"
	@echo "  clean      - Clean build artifacts"
	@echo "  help       - Show this help message"

# Build targets
build:
	@echo "Building NeuroMesh server..."
	go build -o bin/neuromesh ./cmd/server

# Build chat UI (separate module)
build-ui:
	@echo "Building chat UI..."
	cd cmd/chat-ui && go build -o ../../bin/chat-ui .

# Test target
test:
	@echo "Running tests..."
	go test ./...

# Generate protobuf files
proto-gen:
	@echo "Generating protobuf files..."
	protoc --go_out=. --go-grpc_out=. api/proto/orchestration.proto
	@echo "Protobuf files generated in internal/api/grpc/orchestration/"

# Clean artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -rf bin/
	go clean

# Build agent (separate module)
build-agent:
	@echo "Building text-processor agent..."
	cd agents/text-processor && go build -o bin/text-processor .
