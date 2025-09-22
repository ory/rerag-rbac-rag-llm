// LLM RAG ReBAC OSS is a secure RAG system with relationship-based access control.
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

	// Initialize SQLite vector store
	vectorStore, err := storage.NewSQLiteVectorStore("./vector_store.db")
	if err != nil {
		log.Fatal("Failed to initialize vector store:", err)
	}
	defer func() {
		if err := vectorStore.Close(); err != nil {
			log.Printf("Error closing vector store: %v", err)
		}
	}()

	ollama := llm.NewOllamaClient("http://localhost:11434", "llama3")
	// Use Keto-based permissions service
	permService := permissions.NewKetoPermissionService(
		"http://127.0.0.1:4466", // Keto Read API
		"http://127.0.0.1:4467", // Keto Write API
	)

	server := api.NewServer(embedder, vectorStore, ollama, permService)

	if err := server.Run(":8080"); err != nil {
		log.Printf("Failed to start server: %v", err)
		return
	}
}
