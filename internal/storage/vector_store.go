package storage

import (
	"llm-rag-poc/internal/models"
	"math"
	"sort"
	"sync"

	"github.com/google/uuid"
)

// VectorStore defines the interface for vector-based document storage
type VectorStore interface {
	AddDocument(doc *models.Document) error
	SearchSimilar(embedding []float32, topK int) ([]models.Document, error)
	SearchSimilarWithFilter(embedding []float32, topK int, filter func(*models.Document) bool) ([]models.Document, error)
	GetAllDocuments() []models.Document
	GetFilteredDocuments(filter func(*models.Document) bool) []models.Document
}

// MemoryVectorStore implements an in-memory vector storage system
type MemoryVectorStore struct {
	documents []models.Document
	mu        sync.RWMutex
}

// NewMemoryVectorStore creates a new in-memory vector store
func NewMemoryVectorStore() *MemoryVectorStore {
	return &MemoryVectorStore{
		documents: make([]models.Document, 0),
	}
}

// AddDocument stores a new document with its embedding in the vector store
func (m *MemoryVectorStore) AddDocument(doc *models.Document) (err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	doc.ID, err = uuid.NewUUID()
	if err != nil {
		return err
	}
	m.documents = append(m.documents, *doc)
	return nil
}

// SearchSimilar finds the top K most similar documents to the given embedding
func (m *MemoryVectorStore) SearchSimilar(embedding []float32, topK int) ([]models.Document, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if len(m.documents) == 0 {
		return []models.Document{}, nil
	}

	type scoredDoc struct {
		doc   models.Document
		score float32
	}

	scores := make([]scoredDoc, 0, len(m.documents))
	for _, doc := range m.documents {
		similarity := cosineSimilarity(embedding, doc.Embedding)
		scores = append(scores, scoredDoc{doc: doc, score: similarity})
	}

	sort.Slice(scores, func(i, j int) bool {
		return scores[i].score > scores[j].score
	})

	if topK > len(scores) {
		topK = len(scores)
	}

	results := make([]models.Document, topK)
	for i := 0; i < topK; i++ {
		results[i] = scores[i].doc
	}

	return results, nil
}

// SearchSimilarWithFilter finds the top K most similar documents with an optional filter
func (m *MemoryVectorStore) SearchSimilarWithFilter(embedding []float32, topK int, filter func(*models.Document) bool) ([]models.Document, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if len(m.documents) == 0 {
		return []models.Document{}, nil
	}

	type scoredDoc struct {
		doc   models.Document
		score float32
	}

	scores := make([]scoredDoc, 0)
	for i := range m.documents {
		doc := m.documents[i]
		if filter != nil && !filter(&doc) {
			continue
		}
		similarity := cosineSimilarity(embedding, doc.Embedding)
		scores = append(scores, scoredDoc{doc: doc, score: similarity})
	}

	sort.Slice(scores, func(i, j int) bool {
		return scores[i].score > scores[j].score
	})

	if topK > len(scores) {
		topK = len(scores)
	}

	results := make([]models.Document, topK)
	for i := 0; i < topK; i++ {
		results[i] = scores[i].doc
	}

	return results, nil
}

// GetAllDocuments returns all documents in the store
func (m *MemoryVectorStore) GetAllDocuments() []models.Document {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.documents
}

// GetFilteredDocuments returns documents that match the given filter
func (m *MemoryVectorStore) GetFilteredDocuments(filter func(*models.Document) bool) []models.Document {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if filter == nil {
		return m.documents
	}

	filtered := make([]models.Document, 0)
	for i := range m.documents {
		doc := m.documents[i]
		if filter(&doc) {
			filtered = append(filtered, doc)
		}
	}
	return filtered
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
