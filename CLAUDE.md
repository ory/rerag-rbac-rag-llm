# Claude Integration Guide

## Project Overview

This is a secure RAG (Retrieval-Augmented Generation) system that combines LLMs
with ReBAC (Relationship-Based Access Control) using Ory Keto. The system
demonstrates enterprise-grade document management with fine-grained permissions,
vector search, and LLM-powered query answering.

## Key Commands & Workflows

### Testing & Quality Assurance

```bash
make test        # Run all tests
make lint        # Run golangci-lint
make format      # Format code with gofmt, goimports, and prettier for markdown
```

### Development Workflow

```bash
make dev         # Start Keto and app in tmux
make setup       # Setup permissions and load sample documents
make demo        # Run interactive demo showing permission-aware queries
make reset       # Full reset (clean + remove all data)
make quick-start # One-liner setup and demo (install + dev + demo)
```

## Code Standards

### Go Conventions

- **Go version**: 1.24
- **Module name**: `rerag-rbac-rag-llm`
- **Package structure**: All internal packages under `/internal/`
- **Testing**: Comprehensive unit tests with mocks, E2E tests
- **Error handling**: Use Ory Herodot for HTTP responses
- **Interfaces**: Define interfaces for all external dependencies (embedder,
  LLM, storage, permissions)

### Testing Approach

- Mock all external dependencies (Ollama, Keto)
- Test permission scenarios with Alice (limited), Bob (limited), Peter (admin)
- Use table-driven tests for comprehensive coverage
- Always test error paths and edge cases

### Code Style

- Use `gofmt` and `goimports` for formatting
- Follow standard Go project layout
- Keep functions focused and testable
- Document exported types and functions

## Important Context

### Architecture Components

- **API Server** (`/internal/api/`): RESTful endpoints with auth middleware
- **Embeddings** (`/internal/embeddings/`): Ollama with nomic-embed-text model
- **LLM Client** (`/internal/llm/`): Ollama with llama3.2:1b model
  (temperature=0 for deterministic output)
- **Permissions** (`/internal/permissions/`): Ory Keto ReBAC integration
- **Storage** (`/internal/storage/`): SQLite-based persistent vector store with
  sqlite-vec KNN search and adaptive recursive filtering

### Vector Search Architecture

The storage layer uses a sophisticated approach for permission-aware similarity
search:

**Key Implementation Details:**

- **sqlite-vec Integration**: Uses `vec0` virtual table for native vector
  operations in SQLite
- **Dual-Table Design**: Separates document metadata (`documents`) from vectors
  (`vec_documents`)
- **Adaptive Search**: `SearchSimilarWithFilter()` recursively expands candidate
  pool
  - Starts with `topK × 2` candidates
  - Doubles pool size on each attempt (growth factor: 2.0)
  - Max 10 attempts to prevent infinite recursion
  - Returns best-effort results if max attempts reached
- **Performance**: Optimized for sparse permission scenarios without loading all
  vectors

**Implementation Location:** `internal/storage/sqlite_vector_store.go:217-277`

**Key Functions:**

- `SearchSimilarWithFilter()`: Main entry point for permission-aware search
- `searchWithFilterRecursive()`: Recursive logic for adaptive candidate fetching
- `applyFilter()`: Applies permission filter to candidate documents
- `searchWithSqliteVec()`: Executes KNN query via sqlite-vec

### Permission Model

The system uses ReBAC with three test users:

- **alice**: Can only access John Doe's documents
- **bob**: Can only access ABC Corporation's documents
- **peter**: Admin with access to all documents

### API Endpoints

- `POST /documents` - Add document (no auth required for demo)
- `GET /documents` - List accessible documents (auth required)
- `POST /query` - RAG query with permission filtering (auth required)
- `GET /permissions` - View user permissions (auth required)
- `GET /health` - Health check (no auth)

### External Services

- **Ollama** (localhost:11434): LLM and embeddings (runs via Docker as
  `rerag-ollama`)
- **Ory Keto** (localhost:4466/4467): Permission management

## Common Tasks & Prompts

### Adding New Features

"Add a new endpoint for [feature]. Follow the existing patterns in
/internal/api/server.go and include comprehensive tests."

### Refactoring

"Refactor [component] to improve [aspect]. Maintain the existing interface
contracts and ensure all tests pass."

### Testing

"Create comprehensive tests for [feature] following the patterns in
server_test.go. Include unit tests with mocks and permission-based scenarios."

### Documentation

"Update documentation for [feature]. Include inline comments for complex logic
and update the README if needed."

## Key Files & Directories

```
/internal/api/          # API handlers and tests
  server.go            # Main server implementation
  server_test.go       # Unit tests with mocks (655 lines)
  e2e_test.go         # End-to-end tests (503 lines)
  query_test.go       # Query scenario tests (309 lines)

/internal/permissions/ # ReBAC integration
  keto.go             # Ory Keto client
  service.go          # Permission service interface

/internal/storage/     # Vector storage
  sqlite_vector_store.go  # SQLite-based implementation with sqlite-vec
  vector_store.go         # Storage interface
  recursive_search_test.go # Tests for adaptive recursive search

/keto/                # Keto configuration
  config.yml         # Server config
  definitions.opl    # Permission model
```

## Testing Strategies

### Unit Testing Pattern

```go
func TestFeature(t *testing.T) {
    server, embedder, vectorStore, llmClient, permService := createTestServer()

    // Setup mocks
    embedder.SetEmbedding("query", []float32{0.1, 0.2, 0.3})

    // Execute test
    req := createAuthenticatedRequest(method, path, body, "username")
    w := httptest.NewRecorder()
    server.handler(w, req)

    // Verify
    assert.Equal(t, http.StatusOK, w.Code)
}
```

### Permission Testing

Always test with different user contexts:

- alice: Limited access (John Doe only)
- bob: Limited access (ABC Corp only)
- peter: Full access (admin)

## Gotchas & Important Notes

1. **Authentication**: Uses simple Bearer token for demo (not production-ready)
2. **Storage**: SQLite-based vector store with sqlite-vec - data persists across
   restarts
3. **Embedding Model**: Requires Ollama with nomic-embed-text model pulled
4. **LLM Model**: Requires Ollama with llama3.2:1b model pulled (uses
   temperature=0 for deterministic output)
5. **CGO Required**: sqlite-vec requires CGO_ENABLED=1 and a C compiler
6. **Vector Search**: Uses adaptive recursive search that dynamically adjusts
   candidate pool size based on permission filtering
7. **Error Handling**: All errors return proper HTTP status codes via Herodot

## Useful Resources

- [Ory Keto Documentation](https://www.ory.sh/docs/keto)
- [Ollama API Reference](https://github.com/ollama/ollama/blob/main/docs/api.md)
- [sqlite-vec Documentation](https://github.com/asg017/sqlite-vec)
- [Google Zanzibar Paper](https://research.google/pubs/pub48190/) (ReBAC
  foundation)
- [Project README](./README.md) for setup instructions

## Development Tips

1. Run `make dev` to start all services in tmux
2. Use `make test` before committing changes
3. Check `make lint` output for code quality issues
4. Run `make format` to auto-fix formatting
5. Use the existing mock infrastructure for testing new features
6. Follow the established patterns in server_test.go for consistency

## CI/CD Optimizations

The GitHub Actions workflow uses Docker for Ollama and caching for Keto:

- **Ollama via Docker**: Uses `ollama/ollama:latest` as a service container
- **Keto Installation Caching**: Binary cached in `./.bin/keto` (v0.14.0)
- **No apt-get update**: Skip package index updates for faster dependency
  installation
- **Conditional Installation**: Only download/install Keto if not cached
- **Performance**: ~3-4 minutes for first run with model pulls, ~1-2 minutes for
  cached runs

## Common Issues & Solutions

| Issue                     | Solution                                                                |
| ------------------------- | ----------------------------------------------------------------------- |
| Ollama connection refused | Ensure Docker container is running: `docker start rerag-ollama`         |
| Keto permission denied    | Check Keto is running: `make start-keto`                                |
| Tests failing             | Run `make deps` to ensure dependencies are updated                      |
| Embedding errors          | Pull the model: `docker exec rerag-ollama ollama pull nomic-embed-text` |
| LLM errors                | Pull the model: `docker exec rerag-ollama ollama pull llama3.2:1b`      |
| Docker not found          | Install Docker: https://www.docker.com/get-started                      |

## When Working with Claude

1. **Always provide file paths** when discussing code changes
2. **Reference existing patterns** in the codebase for consistency
3. **Request comprehensive tests** for any new functionality
4. **Ask for permission scenario tests** when modifying access control
5. **Specify the user context** (alice/bob/peter) when testing permissions
