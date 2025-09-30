// Package storage provides vector storage implementations for document embeddings.
package storage

import (
	"math"
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

func cosineSimilarity(a, b []float32) float32 {
	if len(a) != len(b) {
		return 0
	}

	var dotProduct, normA, normB float32
	for i := range a {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return dotProduct / (float32(math.Sqrt(float64(normA))) * float32(math.Sqrt(float64(normB))))
}
