.PHONY: help install deps clean build run dev start-keto start-app setup test reset dev-tools

# Default target
help:
	@echo "Available commands:"
	@echo "  install     - Install all dependencies (Go, Ollama, Keto)"
	@echo "  deps        - Download and tidy Go modules"
	@echo "  build       - Build the application"
	@echo "  run         - Build and run the server"
	@echo "  dev         - Start both Keto and app in tmux"
	@echo "  start-keto  - Start Keto server"
	@echo "  start-app   - Start the application server"
	@echo "  setup       - Setup permissions and load documents"
	@echo "  test        - Run all tests"
	@echo "  test-unit   - Run unit tests"
	@echo "  lint        - Run code linter"
	@echo "  swagger-gen - Generate API documentation"
	@echo "  benchmark   - Run benchmarks"
	@echo "  clean       - Clean build artifacts"
	@echo "  reset       - Full reset (clean + remove binaries)"
	@echo "  dev-tools   - Install development tools"

# Install all dependencies
install: install-ollama install-keto deps
	@echo "All dependencies installed successfully!"

# Install Ollama and models
install-ollama:
	@echo "Installing Ollama..."
	@if ! command -v ollama >/dev/null 2>&1; then \
		curl -fsSL https://ollama.ai/install.sh | sh; \
	else \
		echo "Ollama already installed"; \
	fi
	@echo "Pulling required models..."
	ollama pull llama3
	ollama pull nomic-embed-text

# Install Keto binary
install-keto:
	@echo "Installing Keto..."
	@mkdir -p .bin
	@if [ ! -f .bin/keto ]; then \
		curl -L https://github.com/ory/keto/releases/latest/download/keto_linux_amd64.tar.gz | tar -xzC .bin keto; \
	else \
		echo "Keto already installed"; \
	fi
	@chmod +x .bin/keto

# Download and tidy Go modules
deps:
	go mod download
	go mod tidy

# Build the application
build: deps
	@mkdir -p bin
	go build -o bin/server .

# Run the application
run: build
	./bin/server

# Start development environment with tmux
dev:
	@if ! command -v tmux >/dev/null 2>&1; then \
		echo "tmux not found. Install tmux or use 'make start-keto' and 'make start-app' in separate terminals"; \
		exit 1; \
	fi
	tmux new-session -d -s rag-demo -n main
	tmux split-window -h -t rag-demo:main
	tmux send-keys -t rag-demo:main.left 'make start-keto' Enter
	tmux send-keys -t rag-demo:main.right 'sleep 5 && make start-app' Enter
	tmux attach-session -t rag-demo

# Start Keto server
start-keto:
	@mkdir -p data
	./.bin/keto serve all --config keto/config.yml

# Start the application server
start-app: build
	./bin/server

# Setup permissions and load documents
setup:
	@echo "Setting up permissions..."
	./scripts/setup_keto_permissions.sh
	@echo "Loading sample documents..."
	./scripts/load_documents.sh

# Run tests
test:
	@echo "Running query tests..."
	./scripts/test_queries.sh

# Clean build artifacts
clean:
	go clean
	rm -rf bin/

# Full reset
reset: clean
	rm -rf .bin/ data/
	@echo "Full reset complete"

# Install development tools
dev-tools:
	go install golang.org/x/tools/cmd/goimports@latest
	go install golang.org/x/lint/golint@latest

# Generate API documentation
swagger-gen:
	@echo "Generating Swagger documentation..."
	@if command -v swagger >/dev/null 2>&1; then \
		swagger generate spec -o ./docs/swagger.json --scan-models; \
		echo "Swagger spec generated at ./docs/swagger.json"; \
	else \
		echo "Swagger not installed. Run 'make install-swagger' first"; \
	fi

# Install swagger tool
install-swagger:
	@echo "Installing swagger..."
	go install github.com/go-swagger/go-swagger/cmd/swagger@v0.30.5

# Lint code
lint:
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./...; \
	else \
		echo "golangci-lint not installed. Please install it first"; \
	fi

# Run tests
test-unit:
	go test ./... -v

# Run benchmarks
benchmark:
	go test ./... -bench=. -benchmem