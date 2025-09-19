# LLM RAG POC - Tax Document Query System

A simple proof-of-concept for a Retrieval-Augmented Generation (RAG) system that allows users to query tax return documents using Llama3 via Ollama.

## Architecture

```
┌─────────────┐     ┌──────────────┐     ┌─────────────┐
│   Client    │────▶│   REST API   │────▶│  Embeddings │
│  (HTTP)     │     │   (Gin)      │     │  (Ollama)   │
└─────────────┘     └──────────────┘     └─────────────┘
                            │                     │
                            ▼                     ▼
                    ┌──────────────┐     ┌─────────────┐
                    │Vector Store  │     │     LLM     │
                    │  (Memory)    │     │   (Llama3)  │
                    └──────────────┘     └─────────────┘
```

## Prerequisites

1. **Install Ollama**: https://ollama.ai/download
2. **Pull required models**:
   ```bash
   ollama pull llama3
   ollama pull nomic-embed-text
   ```

## Setup

1. **Install dependencies**:
   ```bash
   go mod download
   ```

2. **Start Ollama** (if not running):
   ```bash
   ollama serve
   ```

3. **Run the server**:
   ```bash
   go run main.go
   ```

4. **Load sample documents**:
   ```bash
   chmod +x scripts/load_documents.sh
   ./scripts/load_documents.sh
   ```

## API Endpoints

### Add Document
```bash
POST /documents
{
  "title": "Document Title",
  "content": "Document content...",
  "metadata": {"key": "value"}
}
```

### Query Documents
```bash
POST /query
{
  "question": "What was John Doe's refund amount?",
  "top_k": 3
}
```

### List Documents
```bash
GET /documents
```

### Health Check
```bash
GET /health
```

## Testing

Run the test queries:
```bash
chmod +x scripts/test_queries.sh
./scripts/test_queries.sh
```

## How It Works

1. **Document Ingestion**: Documents are added via the API and converted to vector embeddings using `nomic-embed-text`
2. **Vector Storage**: Embeddings are stored in memory with the document content
3. **Query Processing**: 
   - User questions are converted to embeddings
   - Similar documents are retrieved using cosine similarity
   - Top-K most relevant documents are selected
4. **Response Generation**: Llama3 generates an answer based on the retrieved context

## Components

- **`/internal/models`**: Data structures for documents and queries
- **`/internal/embeddings`**: Embedding generation using Ollama
- **`/internal/storage`**: In-memory vector store with cosine similarity search
- **`/internal/llm`**: Ollama/Llama3 integration for answer generation
- **`/internal/api`**: REST API server using Gin

## Sample Documents

The POC includes 5 sample tax return documents:
- Individual returns (1040)
- Corporate return (1120)
- Various filing statuses and income levels

## Limitations

- In-memory storage (not persistent)
- Simple cosine similarity search
- No authentication/authorization
- Basic error handling

This is a minimal POC designed for simplicity and ease of understanding.