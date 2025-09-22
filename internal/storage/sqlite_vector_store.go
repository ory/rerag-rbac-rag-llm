// Package storage provides vector storage implementations for document embeddings.
package storage

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"llm-rag-poc/internal/models"
	"log"
	"sort"

	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3" // Import sqlite3 driver
)

// SQLiteVectorStore implements a SQLite-based vector storage system
type SQLiteVectorStore struct {
	db *sql.DB
}

// scoredDoc represents a document with its similarity score
type scoredDoc struct {
	doc   models.Document
	score float32
}

// NewSQLiteVectorStore creates a new SQLite-based vector store
func NewSQLiteVectorStore(dbPath string) (*SQLiteVectorStore, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	store := &SQLiteVectorStore{db: db}
	if err := store.initDB(); err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	return store, nil
}

// initDB creates the necessary tables for storing documents and embeddings
func (s *SQLiteVectorStore) initDB() error {
	query := `
	CREATE TABLE IF NOT EXISTS documents (
		id TEXT PRIMARY KEY,
		title TEXT NOT NULL,
		content TEXT NOT NULL,
		metadata TEXT,
		embedding BLOB NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_documents_title ON documents(title);
	CREATE INDEX IF NOT EXISTS idx_documents_created_at ON documents(created_at);
	`

	_, err := s.db.Exec(query)
	return err
}

// Close closes the database connection
func (s *SQLiteVectorStore) Close() error {
	return s.db.Close()
}

// AddDocument stores a new document with its embedding in the vector store
func (s *SQLiteVectorStore) AddDocument(doc *models.Document) error {
	if doc.ID == uuid.Nil {
		newID, err := uuid.NewUUID()
		if err != nil {
			return fmt.Errorf("failed to generate UUID: %w", err)
		}
		doc.ID = newID
	}

	metadataJSON, err := json.Marshal(doc.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	embeddingJSON, err := json.Marshal(doc.Embedding)
	if err != nil {
		return fmt.Errorf("failed to marshal embedding: %w", err)
	}

	query := `
	INSERT INTO documents (id, title, content, metadata, embedding)
	VALUES (?, ?, ?, ?, ?)
	`

	_, err = s.db.Exec(query, doc.ID.String(), doc.Title, doc.Content, string(metadataJSON), embeddingJSON)
	if err != nil {
		return fmt.Errorf("failed to insert document: %w", err)
	}

	return nil
}

// SearchSimilar finds the top K most similar documents to the given embedding
func (s *SQLiteVectorStore) SearchSimilar(embedding []float32, topK int) ([]models.Document, error) {
	return s.SearchSimilarWithFilter(embedding, topK, nil)
}

// SearchSimilarWithFilter finds the top K most similar documents with an optional filter
func (s *SQLiteVectorStore) SearchSimilarWithFilter(embedding []float32, topK int, filter func(*models.Document) bool) ([]models.Document, error) {
	query := `SELECT id, title, content, metadata, embedding FROM documents`
	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query documents: %w", err)
	}
	defer func() { _ = rows.Close() }()

	scores, err := s.calculateSimilarityScores(rows, embedding, filter)
	if err != nil {
		return nil, err
	}

	return s.getTopKResults(scores, topK), nil
}

// calculateSimilarityScores processes query results and calculates similarity scores
func (s *SQLiteVectorStore) calculateSimilarityScores(rows *sql.Rows, embedding []float32, filter func(*models.Document) bool) ([]scoredDoc, error) {
	var scores []scoredDoc

	for rows.Next() {
		doc, err := s.scanRowToDocument(rows)
		if err != nil {
			log.Printf("Error scanning row: %v", err)
			continue
		}

		// Apply filter if provided
		if filter != nil && !filter(&doc) {
			continue
		}

		// Calculate similarity
		similarity := cosineSimilarity(embedding, doc.Embedding)
		scores = append(scores, scoredDoc{doc: doc, score: similarity})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return scores, nil
}

// scanRowToDocument scans a database row into a Document struct
func (s *SQLiteVectorStore) scanRowToDocument(rows *sql.Rows) (models.Document, error) {
	var id, title, content, metadataJSON string
	var embeddingJSON []byte

	err := rows.Scan(&id, &title, &content, &metadataJSON, &embeddingJSON)
	if err != nil {
		return models.Document{}, err
	}

	// Parse UUID
	docID, err := uuid.Parse(id)
	if err != nil {
		return models.Document{}, fmt.Errorf("error parsing UUID %s: %w", id, err)
	}

	// Parse metadata
	var metadata map[string]interface{}
	if err := json.Unmarshal([]byte(metadataJSON), &metadata); err != nil {
		return models.Document{}, fmt.Errorf("error parsing metadata for doc %s: %w", id, err)
	}

	// Parse embedding
	var docEmbedding []float32
	if err := json.Unmarshal(embeddingJSON, &docEmbedding); err != nil {
		return models.Document{}, fmt.Errorf("error parsing embedding for doc %s: %w", id, err)
	}

	return models.Document{
		ID:        docID,
		Title:     title,
		Content:   content,
		Metadata:  metadata,
		Embedding: docEmbedding,
	}, nil
}

// getTopKResults sorts scores and returns top K results
func (s *SQLiteVectorStore) getTopKResults(scores []scoredDoc, topK int) []models.Document {
	// Sort by similarity score (descending)
	sort.Slice(scores, func(i, j int) bool {
		return scores[i].score > scores[j].score
	})

	// Return top K results
	if topK > len(scores) {
		topK = len(scores)
	}

	results := make([]models.Document, topK)
	for i := 0; i < topK; i++ {
		results[i] = scores[i].doc
	}

	return results
}

// GetAllDocuments returns all documents in the store
func (s *SQLiteVectorStore) GetAllDocuments() []models.Document {
	query := `SELECT id, title, content, metadata, embedding FROM documents ORDER BY created_at DESC`
	rows, err := s.db.Query(query)
	if err != nil {
		log.Printf("Error querying all documents: %v", err)
		return []models.Document{}
	}
	defer func() { _ = rows.Close() }()

	var documents []models.Document

	for rows.Next() {
		doc, err := s.scanRowToDocument(rows)
		if err != nil {
			log.Printf("Error scanning row: %v", err)
			continue
		}
		documents = append(documents, doc)
	}

	return documents
}

// GetFilteredDocuments returns documents that match the given filter
func (s *SQLiteVectorStore) GetFilteredDocuments(filter func(*models.Document) bool) []models.Document {
	allDocs := s.GetAllDocuments()
	if filter == nil {
		return allDocs
	}

	var filtered []models.Document
	for i := range allDocs {
		doc := allDocs[i]
		if filter(&doc) {
			filtered = append(filtered, doc)
		}
	}

	return filtered
}
