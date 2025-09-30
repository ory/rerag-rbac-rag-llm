// Package storage provides vector storage implementations for document embeddings.
package storage

import (
	"database/sql"
	"encoding/binary"
	"fmt"
	"log"
	"math"
	"rerag-rbac-rag-llm/internal/models"

	sqlite_vec "github.com/asg017/sqlite-vec-go-bindings/cgo"
	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3" // Import sqlite3 driver
)

func init() {
	sqlite_vec.Auto()
}

// SQLiteVectorStore implements a SQLite-based vector storage system using sqlite-vec
type SQLiteVectorStore struct {
	db              *sql.DB
	embeddingLength int
}

// NewSQLiteVectorStore creates a new SQLite-based vector store with sqlite-vec support
func NewSQLiteVectorStore(dsn string) (*SQLiteVectorStore, error) {
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Test the connection
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	store := &SQLiteVectorStore{
		db:              db,
		embeddingLength: 768, // Default for nomic-embed-text, will be updated on first insert
	}

	if err := store.initDB(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	return store, nil
}

// initDB creates the necessary tables for storing documents and embeddings using sqlite-vec
func (s *SQLiteVectorStore) initDB() error {
	// Create metadata table for documents
	metadataQuery := `
	CREATE TABLE IF NOT EXISTS documents (
		id TEXT PRIMARY KEY,
		title TEXT NOT NULL,
		content TEXT NOT NULL
	);
	`

	if _, err := s.db.Exec(metadataQuery); err != nil {
		return fmt.Errorf("failed to create documents table: %w", err)
	}

	// vec_documents will be created dynamically on first insert
	// when we know the embedding dimension

	return nil
}

// Close closes the database connection
func (s *SQLiteVectorStore) Close() error {
	return s.db.Close()
}

// serializeFloat32Vector converts a float32 slice to the byte format expected by sqlite-vec
func serializeFloat32Vector(vec []float32) []byte {
	buf := make([]byte, len(vec)*4)
	for i, v := range vec {
		binary.LittleEndian.PutUint32(buf[i*4:(i+1)*4], math.Float32bits(v))
	}
	return buf
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

	// Ensure vec_documents table exists with correct dimensions
	if err := s.ensureVecTableExists(len(doc.Embedding)); err != nil {
		return fmt.Errorf("failed to ensure vec table exists: %w", err)
	}

	// Start transaction
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// Insert metadata
	metadataQuery := `INSERT INTO documents (id, title, content) VALUES (?, ?, ?)`
	if _, err := tx.Exec(metadataQuery, doc.ID.String(), doc.Title, doc.Content); err != nil {
		return fmt.Errorf("failed to insert document metadata: %w", err)
	}

	// Insert vector
	embeddingBytes := serializeFloat32Vector(doc.Embedding)
	vecQuery := `INSERT INTO vec_documents (id, embedding) VALUES (?, ?)`
	if _, err := tx.Exec(vecQuery, doc.ID.String(), embeddingBytes); err != nil {
		return fmt.Errorf("failed to insert document vector: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// ensureVecTableExists creates the vec_documents table if it doesn't exist
func (s *SQLiteVectorStore) ensureVecTableExists(embeddingLen int) error {
	// Check if we need to update the embedding length
	if s.embeddingLength != embeddingLen && s.embeddingLength != 768 {
		// Only allow changing from default, otherwise it's an error
		var count int
		if err := s.db.QueryRow("SELECT COUNT(*) FROM documents").Scan(&count); err == nil && count > 0 {
			return fmt.Errorf("cannot change embedding length from %d to %d with existing documents", s.embeddingLength, embeddingLen)
		}
	}

	// Check if table exists
	var tableExists int
	err := s.db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='vec_documents'").Scan(&tableExists)
	if err != nil {
		return fmt.Errorf("failed to check vec_documents existence: %w", err)
	}

	if tableExists == 0 {
		s.embeddingLength = embeddingLen
		vecQuery := fmt.Sprintf(`
			CREATE VIRTUAL TABLE vec_documents USING vec0(
				id TEXT PRIMARY KEY,
				embedding FLOAT[%d]
			)
		`, s.embeddingLength)

		if _, err := s.db.Exec(vecQuery); err != nil {
			return fmt.Errorf("failed to create vec_documents table: %w", err)
		}
	}

	return nil
}

// UpsertDocument inserts or updates a document with its embedding in the vector store
func (s *SQLiteVectorStore) UpsertDocument(doc *models.Document) error {
	if doc.ID == uuid.Nil {
		newID, err := uuid.NewUUID()
		if err != nil {
			return fmt.Errorf("failed to generate UUID: %w", err)
		}
		doc.ID = newID
	}

	// Ensure vec_documents table exists with correct dimensions
	if err := s.ensureVecTableExists(len(doc.Embedding)); err != nil {
		return fmt.Errorf("failed to ensure vec table exists: %w", err)
	}

	// Start transaction
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// Upsert metadata
	metadataQuery := `
		INSERT INTO documents (id, title, content)
		VALUES (?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			title = excluded.title,
			content = excluded.content
	`
	if _, err := tx.Exec(metadataQuery, doc.ID.String(), doc.Title, doc.Content); err != nil {
		return fmt.Errorf("failed to upsert document metadata: %w", err)
	}

	// Upsert vector (delete and insert since vec0 doesn't support UPDATE)
	if _, err := tx.Exec(`DELETE FROM vec_documents WHERE id = ?`, doc.ID.String()); err != nil {
		return fmt.Errorf("failed to delete old vector: %w", err)
	}

	embeddingBytes := serializeFloat32Vector(doc.Embedding)
	vecQuery := `INSERT INTO vec_documents (id, embedding) VALUES (?, ?)`
	if _, err := tx.Exec(vecQuery, doc.ID.String(), embeddingBytes); err != nil {
		return fmt.Errorf("failed to insert document vector: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

const (
	initialMultiplier = 2
	growthFactor      = 2.0
	maxAttempts       = 10
)

// SearchSimilarWithFilter finds the top K most similar documents with an optional filter
// Uses sqlite-vec's KNN search for efficient vector similarity
// Recursively increases the candidate pool until topK matching documents are found
func (s *SQLiteVectorStore) SearchSimilarWithFilter(embedding []float32, topK int, filter func(*models.Document) bool) ([]models.Document, error) {
	return s.searchWithFilterRecursive(embedding, topK, filter, initialMultiplier, 0)
}

// searchWithFilterRecursive recursively fetches more candidates until topK matching documents are found
func (s *SQLiteVectorStore) searchWithFilterRecursive(embedding []float32, topK int, filter func(*models.Document) bool, multiplier int, attempt int) ([]models.Document, error) {
	// Safety check to prevent infinite recursion
	if attempt >= maxAttempts {
		log.Printf("Warning: Reached max attempts (%d) in recursive search, returning partial results", maxAttempts)
		// Return whatever we can get with the maximum multiplier
		candidates, err := s.searchWithSqliteVec(embedding, topK*multiplier)
		if err != nil {
			return nil, err
		}
		return s.applyFilter(candidates, topK, filter), nil
	}

	// Fetch candidates with current multiplier
	candidateCount := topK * multiplier
	candidates, err := s.searchWithSqliteVec(embedding, candidateCount)
	if err != nil {
		return nil, err
	}

	// Apply filter
	filtered := s.applyFilter(candidates, topK, filter)

	// If we have enough results or no more documents exist, return
	if len(filtered) >= topK || len(candidates) < candidateCount {
		return filtered, nil
	}

	// Not enough results, recurse with increased multiplier
	newMultiplier := int(float64(multiplier) * growthFactor)
	log.Printf("Only found %d/%d matching documents, increasing search from %d to %d candidates (attempt %d/%d)",
		len(filtered), topK, candidateCount, topK*newMultiplier, attempt+1, maxAttempts)
	return s.searchWithFilterRecursive(embedding, topK, filter, newMultiplier, attempt+1)
}

// applyFilter applies the filter function to candidates and returns up to topK results
func (s *SQLiteVectorStore) applyFilter(candidates []models.Document, topK int, filter func(*models.Document) bool) []models.Document {
	var filtered []models.Document
	for i := range candidates {
		if filter(&candidates[i]) {
			filtered = append(filtered, candidates[i])
			if len(filtered) >= topK {
				break
			}
		}
	}
	return filtered
}

// searchWithSqliteVec performs KNN vector search using sqlite-vec
func (s *SQLiteVectorStore) searchWithSqliteVec(embedding []float32, topK int) ([]models.Document, error) {
	embeddingBytes := serializeFloat32Vector(embedding)

	// Use sqlite-vec's KNN search with distance calculation
	// Note: sqlite-vec requires the k parameter to be passed as part of the MATCH expression
	query := `
		SELECT
			d.id,
			d.title,
			d.content,
			v.distance
		FROM vec_documents v
		JOIN documents d ON d.id = v.id
		WHERE v.embedding MATCH ? AND k = ?
		ORDER BY v.distance
	`

	rows, err := s.db.Query(query, embeddingBytes, topK)
	if err != nil {
		return nil, fmt.Errorf("failed to perform vector search: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var results []models.Document
	for rows.Next() {
		var id, title, content string
		var distance float32

		if err := rows.Scan(&id, &title, &content, &distance); err != nil {
			log.Printf("Error scanning row: %v", err)
			continue
		}

		docID, err := uuid.Parse(id)
		if err != nil {
			log.Printf("Error parsing UUID %s: %v", id, err)
			continue
		}

		results = append(results, models.Document{
			ID:      docID,
			Title:   title,
			Content: content,
			// Note: We don't fetch the embedding vector to save memory
			// If needed, it can be fetched separately
		})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating results: %w", err)
	}

	return results, nil
}

// GetAllDocuments returns all documents in the store (without embeddings for efficiency)
func (s *SQLiteVectorStore) GetAllDocuments() []models.Document {
	query := `SELECT id, title, content FROM documents ORDER BY id DESC`
	rows, err := s.db.Query(query)
	if err != nil {
		log.Printf("Error querying all documents: %v", err)
		return []models.Document{}
	}
	defer func() { _ = rows.Close() }()

	var documents []models.Document

	for rows.Next() {
		var id, title, content string
		if err := rows.Scan(&id, &title, &content); err != nil {
			log.Printf("Error scanning row: %v", err)
			continue
		}

		docID, err := uuid.Parse(id)
		if err != nil {
			log.Printf("Error parsing UUID %s: %v", id, err)
			continue
		}

		documents = append(documents, models.Document{
			ID:      docID,
			Title:   title,
			Content: content,
		})
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
