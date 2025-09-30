// Package storage provides vector storage implementations for document embeddings.
package storage

import (
	"rerag-rbac-rag-llm/internal/models"
)

// VectorStore defines the interface for vector-based document storage
type VectorStore interface {
	AddDocument(doc *models.Document) error
	UpsertDocument(doc *models.Document) error
	SearchSimilarWithFilter(embedding []float32, topK int, filter func(*models.Document) bool) ([]models.Document, error)
	GetAllDocuments() []models.Document
	GetFilteredDocuments(filter func(*models.Document) bool) []models.Document
}
