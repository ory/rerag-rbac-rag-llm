package storage

import (
	"os"
	"rerag-rbac-rag-llm/internal/models"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestSQLiteVectorStore(t *testing.T) {
	store := setupTestStore(t)
	defer cleanupTestStore(store)

	testAddDocuments(t, store)
	testGetAllDocuments(t, store)
	testSearchSimilar(t, store)
	testSearchSimilarWithFilter(t, store)
	testGetFilteredDocuments(t, store)
}

func setupTestStore(t *testing.T) *SQLiteVectorStore {
	dbPath := "./test_vector_store.db"
	t.Cleanup(func() { _ = os.Remove(dbPath) })

	store, err := NewSQLiteVectorStore(dbPath)
	if err != nil {
		t.Fatalf("Failed to create SQLite vector store: %v", err)
	}
	return store
}

func cleanupTestStore(store *SQLiteVectorStore) {
	_ = store.Close()
}

func testAddDocuments(t *testing.T, store *SQLiteVectorStore) {
	doc1 := createTestDocument("Test Document 1", "This is test content 1", []float32{0.1, 0.2, 0.3}, 1)
	doc2 := createTestDocument("Test Document 2", "This is test content 2", []float32{0.2, 0.3, 0.4}, 2)

	if err := store.AddDocument(doc1); err != nil {
		t.Fatalf("Failed to add document 1: %v", err)
	}
	if err := store.AddDocument(doc2); err != nil {
		t.Fatalf("Failed to add document 2: %v", err)
	}
}

func testGetAllDocuments(t *testing.T, store *SQLiteVectorStore) {
	allDocs := store.GetAllDocuments()
	if len(allDocs) != 2 {
		t.Errorf("Expected 2 documents, got %d", len(allDocs))
	}
}

func testSearchSimilar(t *testing.T, store *SQLiteVectorStore) {
	queryEmbedding := []float32{0.15, 0.25, 0.35}
	results, err := store.SearchSimilar(queryEmbedding, 1)
	if err != nil {
		t.Fatalf("Failed to search similar: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(results))
	}
}

func testSearchSimilarWithFilter(t *testing.T, store *SQLiteVectorStore) {
	queryEmbedding := []float32{0.15, 0.25, 0.35}
	// Filter for documents with "Test" in the title
	filter := func(doc *models.Document) bool {
		return strings.Contains(doc.Title, "Test")
	}

	filteredResults, err := store.SearchSimilarWithFilter(queryEmbedding, 2, filter)
	if err != nil {
		t.Fatalf("Failed to search with filter: %v", err)
	}
	if len(filteredResults) != 2 {
		t.Errorf("Expected 2 filtered results, got %d", len(filteredResults))
	}
}

func testGetFilteredDocuments(t *testing.T, store *SQLiteVectorStore) {
	// Filter for documents with "priority" in the content
	priorityFilter := func(doc *models.Document) bool {
		return strings.Contains(strings.ToLower(doc.Content), "priority")
	}

	priorityDocs := store.GetFilteredDocuments(priorityFilter)
	if len(priorityDocs) != 1 {
		t.Errorf("Expected 1 priority document, got %d", len(priorityDocs))
	}
}

func createTestDocument(title, content string, embedding []float32, priority int) *models.Document {
	// Add priority marker to content if priority is 1
	if priority == 1 {
		content += " (priority document)"
	}
	return &models.Document{
		Title:     title,
		Content:   content,
		Embedding: embedding,
	}
}

func TestSQLiteVectorStoreUUIDGeneration(t *testing.T) {
	// Create temporary database file
	dbPath := "./test_uuid_vector_store.db"
	defer func() {
		_ = os.Remove(dbPath)
	}()

	store, err := NewSQLiteVectorStore(dbPath)
	if err != nil {
		t.Fatalf("Failed to create SQLite vector store: %v", err)
	}
	defer func() {
		_ = store.Close()
	}()

	// Test document with no ID (should generate one)
	doc := &models.Document{
		Title:     "Test Document",
		Content:   "This is test content",
		Embedding: []float32{0.1, 0.2, 0.3},
		Metadata:  map[string]interface{}{},
	}

	originalID := doc.ID
	if err := store.AddDocument(doc); err != nil {
		t.Fatalf("Failed to add document: %v", err)
	}

	// Check that ID was generated
	if doc.ID == originalID {
		t.Error("Expected ID to be generated, but it wasn't changed")
	}

	if doc.ID == uuid.Nil {
		t.Error("Expected valid UUID, got nil UUID")
	}
}

func TestSQLiteVectorStoreWithExistingID(t *testing.T) {
	// Create temporary database file
	dbPath := "./test_existing_id_vector_store.db"
	defer func() {
		_ = os.Remove(dbPath)
	}()

	store, err := NewSQLiteVectorStore(dbPath)
	if err != nil {
		t.Fatalf("Failed to create SQLite vector store: %v", err)
	}
	defer func() {
		_ = store.Close()
	}()

	// Test document with existing ID
	existingID := uuid.New()
	doc := &models.Document{
		ID:        existingID,
		Title:     "Test Document",
		Content:   "This is test content",
		Embedding: []float32{0.1, 0.2, 0.3},
		Metadata:  map[string]interface{}{},
	}

	if err := store.AddDocument(doc); err != nil {
		t.Fatalf("Failed to add document: %v", err)
	}

	// Check that ID was preserved
	if doc.ID != existingID {
		t.Errorf("Expected ID to be preserved, got %v instead of %v", doc.ID, existingID)
	}
}

func TestSQLiteVectorStoreEmptyDB(t *testing.T) {
	// Create temporary database file
	dbPath := "./test_empty_vector_store.db"
	defer func() {
		_ = os.Remove(dbPath)
	}()

	store, err := NewSQLiteVectorStore(dbPath)
	if err != nil {
		t.Fatalf("Failed to create SQLite vector store: %v", err)
	}
	defer func() {
		_ = store.Close()
	}()

	// Test operations on empty database
	allDocs := store.GetAllDocuments()
	if len(allDocs) != 0 {
		t.Errorf("Expected 0 documents in empty store, got %d", len(allDocs))
	}

	queryEmbedding := []float32{0.1, 0.2, 0.3}
	results, err := store.SearchSimilar(queryEmbedding, 5)
	if err != nil {
		t.Fatalf("Failed to search in empty store: %v", err)
	}

	if len(results) != 0 {
		t.Errorf("Expected 0 results from empty store, got %d", len(results))
	}
}
