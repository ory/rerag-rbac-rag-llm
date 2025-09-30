package storage

import (
	"os"
	"rerag-rbac-rag-llm/internal/models"
	"testing"

	"github.com/google/uuid"
)

// TestRecursiveSearchWithFilter tests that the recursive search correctly
// increases the candidate pool when not enough matches are found
func TestRecursiveSearchWithFilter(t *testing.T) {
	dbPath := "./test_recursive_search.db"
	t.Cleanup(func() { _ = os.Remove(dbPath) })

	store, err := NewSQLiteVectorStore(dbPath)
	if err != nil {
		t.Fatalf("Failed to create SQLite vector store: %v", err)
	}
	defer store.Close()

	// Add 10 documents with alternating categories
	for i := 0; i < 10; i++ {
		category := "even"
		if i%2 == 1 {
			category = "odd"
		}

		doc := &models.Document{
			ID:      uuid.New(),
			Title:   category,
			Content: "Content " + category,
			Embedding: []float32{
				float32(i) / 10.0,
				float32(i) / 20.0,
				float32(i) / 30.0,
			},
		}

		if err := store.AddDocument(doc); err != nil {
			t.Fatalf("Failed to add document %d: %v", i, err)
		}
	}

	// Search with a filter that only matches "odd" documents (5 total)
	// Request 4 results, which should work but may require recursion
	queryEmbedding := []float32{0.3, 0.15, 0.1}
	filter := func(doc *models.Document) bool {
		return doc.Title == "odd"
	}

	results, err := store.SearchSimilarWithFilter(queryEmbedding, 4, filter)
	if err != nil {
		t.Fatalf("Failed to search with filter: %v", err)
	}

	// Should get 4 results (all matching "odd" category)
	if len(results) != 4 {
		t.Errorf("Expected 4 results, got %d", len(results))
	}

	// Verify all results are "odd"
	for i, doc := range results {
		if doc.Title != "odd" {
			t.Errorf("Result %d has wrong title: %s", i, doc.Title)
		}
	}
}

// TestRecursiveSearchMaxAttempts verifies that the search stops after max attempts
func TestRecursiveSearchMaxAttempts(t *testing.T) {
	dbPath := "./test_max_attempts.db"
	t.Cleanup(func() { _ = os.Remove(dbPath) })

	store, err := NewSQLiteVectorStore(dbPath)
	if err != nil {
		t.Fatalf("Failed to create SQLite vector store: %v", err)
	}
	defer store.Close()

	// Add only 3 documents, all with title "A"
	for i := 0; i < 3; i++ {
		doc := &models.Document{
			ID:      uuid.New(),
			Title:   "A",
			Content: "Content " + string(rune('A'+i)),
			Embedding: []float32{
				float32(i) / 10.0,
				float32(i) / 20.0,
				float32(i) / 30.0,
			},
		}

		if err := store.AddDocument(doc); err != nil {
			t.Fatalf("Failed to add document %d: %v", i, err)
		}
	}

	// Request 5 results with filter that only matches "B" (none exist)
	// This should hit max attempts and return empty results
	queryEmbedding := []float32{0.1, 0.05, 0.03}
	filter := func(doc *models.Document) bool {
		return doc.Title == "B"
	}

	results, err := store.SearchSimilarWithFilter(queryEmbedding, 5, filter)
	if err != nil {
		t.Fatalf("Failed to search with filter: %v", err)
	}

	// Should get 0 results since no documents match
	if len(results) != 0 {
		t.Errorf("Expected 0 results, got %d", len(results))
	}
}
