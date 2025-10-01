.PHONY: help install deps clean build run dev start-keto start-app setup test reset format demo quick-start stop-ollama

# Default target
help:
	@echo "╔══════════════════════════════════════════════════════════╗"
	@echo "║   ReBAC-Powered RAG Demo - Available Commands           ║"
	@echo "╚══════════════════════════════════════════════════════════╝"
	@echo ""
	@echo "🚀 Quick Start:"
	@echo "  quick-start - One-liner setup and demo (install + dev + demo)"
	@echo "  demo        - Run interactive demo showing permission-aware queries"
	@echo ""
	@echo "📦 Installation:"
	@echo "  install     - Install all dependencies (Go, Ollama, Keto)"
	@echo "  deps        - Download and tidy Go modules"
	@echo ""
	@echo "🏃 Running:"
	@echo "  dev         - Start both Keto and app in tmux (recommended)"
	@echo "  start-keto  - Start Keto server (manual)"
	@echo "  start-app   - Start the application server (manual)"
	@echo "  setup       - Setup permissions and load sample documents"
	@echo ""
	@echo "🧪 Testing & Quality:"
	@echo "  test        - Run all tests"
	@echo "  lint        - Run code linter (golangci-lint)"
	@echo "  format      - Format Go and Markdown files"
	@echo ""
	@echo "🔨 Build & Clean:"
	@echo "  build       - Build the application"
	@echo "  run         - Build and run the server"
	@echo "  clean       - Clean build artifacts"
	@echo "  reset       - Full reset (clean + remove all data)"
	@echo "  stop-ollama - Stop and remove Ollama Docker container"
	@echo ""

# Install all dependencies
install: install-ollama install-keto deps
	@echo "All dependencies installed successfully!"

# Install Ollama via Docker and pull models
install-ollama:
	@echo "Starting Ollama via Docker..."
	@if ! docker ps | grep -q rerag-ollama; then \
		docker run -d --name rerag-ollama -p 11434:11434 ollama/ollama:latest; \
		echo "Waiting for Ollama to start..."; \
		for i in {1..30}; do \
			if curl -s http://localhost:11434/api/version >/dev/null 2>&1; then \
				echo "Ollama is ready"; \
				break; \
			fi; \
			sleep 1; \
		done; \
	else \
		echo "Ollama container already running"; \
	fi
	@echo "Pulling required models..."
	docker exec rerag-ollama ollama pull llama3.2:1b
	docker exec rerag-ollama ollama pull nomic-embed-text

.bin/keto:
	@echo "Installing Keto..."
	@mkdir -p .bin
	@if [ ! -f .bin/keto ]; then \
		curl https://raw.githubusercontent.com/ory/meta/master/install.sh | bash -s -- -d -b .bin/ keto v0.14.0; \
	else \
		echo "Keto already installed"; \
	fi
	@chmod +x .bin/keto
	@touch .bin/keto

# Install Keto binary
install-keto: .bin/keto

# Download and tidy Go modules
deps:
	go mod download
	go mod tidy

# Build the application (CGO required for sqlite-vec)
build: deps
	@mkdir -p .bin
	CGO_ENABLED=1 go build -o .bin/server .

# Run the application
run: build
	.bin/server

# Start development environment with tmux
dev: build
	@if ! command -v tmux >/dev/null 2>&1; then \
		echo "tmux not found. Install tmux or use 'make start-keto' and 'make start-app' in separate terminals"; \
		exit 1; \
	fi
	rm -rf data/*.db
	@mkdir -p data
	tmux kill-session -t rag-demo || true
	tmux new-session -d -s rag-demo -n main
	tmux split-window -h -t rag-demo:main
	tmux send-keys -t rag-demo:main.left 'make start-keto' Enter
	tmux send-keys -t rag-demo:main.right 'make start-app' Enter
	tmux attach-session -t rag-demo

# Start Keto server
start-keto:
	@mkdir -p data
	.bin/keto migrate up --yes --config keto/config.yml
	.bin/keto serve all --config keto/config.yml

# Start the application server
start-app: build
	@mkdir -p data
	.bin/server

setup:
	@echo "Setting up permissions..."
	./demo/setup_keto_permissions.sh
	@echo "Loading sample documents..."
	./demo/load_documents.sh

# Run interactive demo
demo: setup
	@echo "Starting interactive demo..."
	@./demo/demo.sh

# One-liner quick start
quick-start:
	@echo "🚀 Starting ReBAC-Powered RAG Demo..."
	@echo "This will install dependencies and run a full demo."
	@echo ""
	@$(MAKE) install
	@echo ""
	@echo "Starting services in background..."
	@$(MAKE) start-keto > /dev/null 2>&1 &
	@sleep 5
	@$(MAKE) start-app > /dev/null 2>&1 &
	@sleep 3
	@echo "Services started."
	@echo ""
	@$(MAKE) setup
	@echo ""
	@$(MAKE) demo

# Run tests
test:
	go test ./... -v

# Clean build artifacts
clean:
	go clean
	rm -rf .bin/

# Full reset
reset: clean stop-ollama
	rm -rf .bin/ data/*.db
	@echo "Full reset complete"

# Lint code
lint:
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./...; \
	else \
		echo "golangci-lint not installed. Please install it first"; \
	fi

# Format Go and Markdown files
format:
	@echo "Formatting Go files..."
	@go fmt ./...
	@if command -v goimports >/dev/null 2>&1; then \
		goimports -w .; \
	else \
		echo "Installing goimports..."; \
		go install golang.org/x/tools/cmd/goimports@latest; \
		goimports -w .; \
	fi
	@echo "Formatting Markdown files..."
	@npx prettier --write "**/*.md" 2>/dev/null || echo "Prettier not available, skipping markdown formatting"

# Stop Ollama container
stop-ollama:
	@docker stop rerag-ollama 2>/dev/null || true
	@docker rm rerag-ollama 2>/dev/null || true
	@echo "Ollama container stopped"