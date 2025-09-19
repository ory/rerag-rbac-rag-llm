package main

import (
	"log"
	"llm-rag-poc/internal/api"
	"llm-rag-poc/internal/embeddings"
	"llm-rag-poc/internal/llm"
	"llm-rag-poc/internal/storage"
)

func main() {
	log.Println("Starting LLM RAG POC...")

	embedder := embeddings.NewEmbedder()
	vectorStore := storage.NewMemoryVectorStore()
	ollama := llm.NewOllamaClient("http://localhost:11434", "llama3")

	server := api.NewServer(embedder, vectorStore, ollama)

	log.Println("Server starting on :8080")
	if err := server.Run(":8080"); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}