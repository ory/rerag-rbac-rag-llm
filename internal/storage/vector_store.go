package storage

import (
	"llm-rag-poc/internal/models"
	"math"
	"sort"
	"sync"
)

type VectorStore interface {
	AddDocument(doc *models.Document) error
	SearchSimilar(embedding []float32, topK int) ([]*models.Document, error)
	GetAllDocuments() []*models.Document
}

type MemoryVectorStore struct {
	documents []*models.Document
	mu        sync.RWMutex
}

func NewMemoryVectorStore() *MemoryVectorStore {
	return &MemoryVectorStore{
		documents: make([]*models.Document, 0),
	}
}

func (m *MemoryVectorStore) AddDocument(doc *models.Document) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.documents = append(m.documents, doc)
	return nil
}

func (m *MemoryVectorStore) SearchSimilar(embedding []float32, topK int) ([]*models.Document, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if len(m.documents) == 0 {
		return []*models.Document{}, nil
	}

	type scoredDoc struct {
		doc   *models.Document
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

	results := make([]*models.Document, topK)
	for i := 0; i < topK; i++ {
		results[i] = scores[i].doc
	}

	return results, nil
}

func (m *MemoryVectorStore) GetAllDocuments() []*models.Document {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.documents
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