package main

import (
	"llm-rag-poc/internal/api"
	"llm-rag-poc/internal/embeddings"
	"llm-rag-poc/internal/llm"
	"llm-rag-poc/internal/permissions"
	"llm-rag-poc/internal/storage"
	"log"
)

func main() {
	log.Println("Starting LLM RAG POC...")

	embedder := embeddings.NewEmbedder()
	vectorStore := storage.NewMemoryVectorStore()
	ollama := llm.NewOllamaClient("http://localhost:11434", "llama3")
	// Use Keto-based permissions service
	permService := permissions.NewKetoPermissionService(
		"http://127.0.0.1:4466", // Keto Read API
		"http://127.0.0.1:4467", // Keto Write API
	)

	server := api.NewServer(embedder, vectorStore, ollama, permService)

	if err := server.Run(":8080"); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}
