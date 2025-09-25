package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"rerag-rbac-rag-llm/internal/auth"
	"rerag-rbac-rag-llm/internal/models"
	"testing"

	"github.com/google/uuid"
	"github.com/ory/herodot"
)

// Mock implementations for testing

// Remove duplicate interface declaration - using the one from server.go

type MockEmbedder struct {
	embeddings map[string][]float32
	shouldFail bool
}

func NewMockEmbedder() *MockEmbedder {
	return &MockEmbedder{
		embeddings: make(map[string][]float32),
		shouldFail: false,
	}
}

func (m *MockEmbedder) GetEmbedding(text string) ([]float32, error) {
	if m.shouldFail {
		return nil, &EmbeddingError{Message: "mock embedding error"}
	}

	if embedding, exists := m.embeddings[text]; exists {
		return embedding, nil
	}

	// Return a default embedding
	return []float32{0.1, 0.2, 0.3, 0.4}, nil
}

func (m *MockEmbedder) SetEmbedding(text string, embedding []float32) {
	m.embeddings[text] = embedding
}

func (m *MockEmbedder) SetShouldFail(fail bool) {
	m.shouldFail = fail
}

type EmbeddingError struct {
	Message string
}

func (e *EmbeddingError) Error() string {
	return e.Message
}

type MockVectorStore struct {
	documents   map[uuid.UUID]*models.Document
	shouldFail  bool
	searchError bool
}

func NewMockVectorStore() *MockVectorStore {
	return &MockVectorStore{
		documents:   make(map[uuid.UUID]*models.Document),
		shouldFail:  false,
		searchError: false,
	}
}

func (m *MockVectorStore) AddDocument(doc *models.Document) error {
	if m.shouldFail {
		return &VectorStoreError{Message: "mock vector store error"}
	}
	m.documents[doc.ID] = doc
	return nil
}

func (m *MockVectorStore) UpsertDocument(doc *models.Document) error {
	if m.shouldFail {
		return &VectorStoreError{Message: "mock vector store error"}
	}
	// Upsert: insert or update
	m.documents[doc.ID] = doc
	return nil
}

func (m *MockVectorStore) GetAllDocuments() []models.Document {
	var result []models.Document
	for _, doc := range m.documents {
		result = append(result, *doc)
	}
	return result
}

func (m *MockVectorStore) GetFilteredDocuments(filter func(*models.Document) bool) []models.Document {
	var result []models.Document
	for _, doc := range m.documents {
		if filter(doc) {
			result = append(result, *doc)
		}
	}
	return result
}

func (m *MockVectorStore) SearchSimilar(_ []float32, topK int) ([]models.Document, error) {
	if m.searchError {
		return nil, &VectorStoreError{Message: "mock search error"}
	}

	var result []models.Document
	count := 0
	for _, doc := range m.documents {
		if count < topK {
			result = append(result, *doc)
			count++
		}
	}
	return result, nil
}

func (m *MockVectorStore) SearchSimilarWithFilter(_ []float32, topK int, filter func(*models.Document) bool) ([]models.Document, error) {
	if m.searchError {
		return nil, &VectorStoreError{Message: "mock search error"}
	}

	var result []models.Document
	count := 0
	for _, doc := range m.documents {
		if filter(doc) && count < topK {
			result = append(result, *doc)
			count++
		}
	}
	return result, nil
}

func (m *MockVectorStore) SetShouldFail(fail bool) {
	m.shouldFail = fail
}

func (m *MockVectorStore) SetSearchError(fail bool) {
	m.searchError = fail
}

type VectorStoreError struct {
	Message string
}

func (e *VectorStoreError) Error() string {
	return e.Message
}

type MockLLMClient struct {
	responses  map[string]string
	shouldFail bool
}

func NewMockLLMClient() *MockLLMClient {
	return &MockLLMClient{
		responses:  make(map[string]string),
		shouldFail: false,
	}
}

func (m *MockLLMClient) Generate(question string, _ []models.Document) (string, error) {
	if m.shouldFail {
		return "", &LLMError{Message: "mock LLM error"}
	}

	if response, exists := m.responses[question]; exists {
		return response, nil
	}

	return "Mock LLM response for: " + question, nil
}

func (m *MockLLMClient) SetResponse(question, response string) {
	m.responses[question] = response
}

func (m *MockLLMClient) SetShouldFail(fail bool) {
	m.shouldFail = fail
}

type LLMError struct {
	Message string
}

func (e *LLMError) Error() string {
	return e.Message
}

type MockPermissionService struct {
	permissions map[string][]string
	accessRules map[string]map[string]bool // user -> docID -> canAccess
}

func NewMockPermissionService() *MockPermissionService {
	return &MockPermissionService{
		permissions: make(map[string][]string),
		accessRules: make(map[string]map[string]bool),
	}
}

func (m *MockPermissionService) CanAccessDocument(username string, doc *models.Document) bool {
	if userRules, exists := m.accessRules[username]; exists {
		if canAccess, docExists := userRules[doc.ID.String()]; docExists {
			return canAccess
		}
	}
	// Default: allow access if no specific rule
	return true
}

func (m *MockPermissionService) GetUserPermissions(username string) []string {
	if perms, exists := m.permissions[username]; exists {
		return perms
	}
	return []string{}
}

func (m *MockPermissionService) FilterDocuments(username string, docs []*models.Document) []*models.Document {
	var result []*models.Document
	for _, doc := range docs {
		if m.CanAccessDocument(username, doc) {
			result = append(result, doc)
		}
	}
	return result
}

func (m *MockPermissionService) AddUserPermission(username string, taxpayer string) {
	// Mock implementation - just add a permission string
	if m.permissions[username] == nil {
		m.permissions[username] = []string{}
	}
	m.permissions[username] = append(m.permissions[username], "taxpayer:"+taxpayer)
}

func (m *MockPermissionService) SetUserPermissions(username string, permissions []string) {
	m.permissions[username] = permissions
}

func (m *MockPermissionService) SetDocumentAccess(username, docID string, canAccess bool) {
	if m.accessRules[username] == nil {
		m.accessRules[username] = make(map[string]bool)
	}
	m.accessRules[username][docID] = canAccess
}

// Helper function to create a test server
func createTestServer() (*Server, *MockEmbedder, *MockVectorStore, *MockLLMClient, *MockPermissionService) {
	embedder := NewMockEmbedder()
	vectorStore := NewMockVectorStore()
	llmClient := NewMockLLMClient()
	permService := NewMockPermissionService()

	// Create server with mock interfaces
	server := &Server{
		mux:         http.NewServeMux(),
		embedder:    embedder,
		vectorStore: vectorStore,
		llmClient:   llmClient,
		permService: permService,
		writer:      herodot.NewJSONWriter(nil),
	}

	server.setupRoutes()

	return server, embedder, vectorStore, llmClient, permService
}

// Helper function to create authenticated request
func createAuthenticatedRequest(method, url string, body []byte, username string) *http.Request {
	req := httptest.NewRequest(method, url, bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	// Add user to context (simulating auth middleware)
	ctx := context.WithValue(req.Context(), auth.UserContextKey, username)
	req = req.WithContext(ctx)

	return req
}

// Unit Tests

func TestHealthCheck(t *testing.T) {
	server, _, _, _, _ := createTestServer()

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	server.healthCheck(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response["status"] != "healthy" {
		t.Errorf("Expected status 'healthy', got '%s'", response["status"])
	}
}

func TestHealthCheckInvalidMethod(t *testing.T) {
	server, _, _, _, _ := createTestServer()

	req := httptest.NewRequest(http.MethodPost, "/health", nil)
	w := httptest.NewRecorder()

	server.healthCheck(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status %d, got %d", http.StatusMethodNotAllowed, w.Code)
	}
}

func TestAddDocumentSuccess(t *testing.T) {
	server, embedder, vectorStore, _, _ := createTestServer()

	doc := models.Document{
		Title:   "Test Document",
		Content: "This is test content",
		Metadata: map[string]interface{}{
			"category": "test",
		},
	}

	embedder.SetEmbedding(doc.Content, []float32{0.1, 0.2, 0.3})

	body, _ := json.Marshal(doc)
	req := httptest.NewRequest(http.MethodPost, "/documents", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.addDocument(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status %d, got %d", http.StatusCreated, w.Code)
	}

	var response map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response["message"] != "Document added successfully" {
		t.Errorf("Expected success message, got '%s'", response["message"])
	}

	if len(vectorStore.documents) != 1 {
		t.Errorf("Expected 1 document in store, got %d", len(vectorStore.documents))
	}
}

func TestAddDocumentInvalidJSON(t *testing.T) {
	server, _, _, _, _ := createTestServer()

	req := httptest.NewRequest(http.MethodPost, "/documents", bytes.NewBuffer([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.addDocument(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestAddDocumentEmbeddingError(t *testing.T) {
	server, embedder, _, _, _ := createTestServer()
	embedder.SetShouldFail(true)

	doc := models.Document{
		Title:   "Test Document",
		Content: "This is test content",
	}

	body, _ := json.Marshal(doc)
	req := httptest.NewRequest(http.MethodPost, "/documents", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.addDocument(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status %d, got %d", http.StatusInternalServerError, w.Code)
	}
}

func TestAddDocumentVectorStoreError(t *testing.T) {
	server, _, vectorStore, _, _ := createTestServer()
	vectorStore.SetShouldFail(true)

	doc := models.Document{
		Title:   "Test Document",
		Content: "This is test content",
	}

	body, _ := json.Marshal(doc)
	req := httptest.NewRequest(http.MethodPost, "/documents", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.addDocument(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status %d, got %d", http.StatusInternalServerError, w.Code)
	}
}

func TestListDocuments(t *testing.T) {
	const testUsername = "testuser"
	server, _, vectorStore, _, permService := createTestServer()

	// Add test documents
	doc1 := &models.Document{
		ID:      uuid.New(),
		Title:   "Document 1",
		Content: "Content 1",
	}
	doc2 := &models.Document{
		ID:      uuid.New(),
		Title:   "Document 2",
		Content: "Content 2",
	}

	_ = vectorStore.AddDocument(doc1)
	_ = vectorStore.AddDocument(doc2)

	// Set permissions - user can access doc1 but not doc2
	permService.SetDocumentAccess(testUsername, doc1.ID.String(), true)
	permService.SetDocumentAccess(testUsername, doc2.ID.String(), false)

	req := createAuthenticatedRequest(http.MethodGet, "/documents", nil, testUsername)
	w := httptest.NewRecorder()

	server.listDocuments(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	documents := response["documents"].([]interface{})
	if len(documents) != 1 {
		t.Errorf("Expected 1 accessible document, got %d", len(documents))
	}

	if response["user"] != testUsername {
		t.Errorf("Expected user '%s', got '%s'", testUsername, response["user"])
	}
}

func TestQueryDocuments(t *testing.T) {
	const testUsername = "testuser"
	server, embedder, vectorStore, llmClient, permService := createTestServer()

	// Set up test document
	doc := &models.Document{
		ID:        uuid.New(),
		Title:     "Test Document",
		Content:   "This contains important information",
		Embedding: []float32{0.1, 0.2, 0.3},
	}
	_ = vectorStore.AddDocument(doc)
	permService.SetDocumentAccess(testUsername, doc.ID.String(), true)

	// Set up embeddings and LLM response
	question := "What information is available?"
	embedder.SetEmbedding(question, []float32{0.1, 0.2, 0.3})
	llmClient.SetResponse(question, "The document contains important information")

	queryReq := models.QueryRequest{
		Question: question,
		TopK:     3,
	}

	body, _ := json.Marshal(queryReq)
	req := createAuthenticatedRequest(http.MethodPost, "/query", body, testUsername)
	w := httptest.NewRecorder()

	server.queryDocuments(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response models.QueryResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.Answer != "The document contains important information" {
		t.Errorf("Expected specific answer, got '%s'", response.Answer)
	}

	if len(response.Sources) != 1 {
		t.Errorf("Expected 1 source document, got %d", len(response.Sources))
	}
}

func TestQueryDocumentsInvalidMethod(t *testing.T) {
	const testUsername = "testuser"
	server, _, _, _, _ := createTestServer()

	req := createAuthenticatedRequest(http.MethodGet, "/query", nil, testUsername)
	w := httptest.NewRecorder()

	server.queryDocuments(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status %d, got %d", http.StatusMethodNotAllowed, w.Code)
	}
}

func TestQueryDocumentsInvalidJSON(t *testing.T) {
	const testUsername = "testuser"
	server, _, _, _, _ := createTestServer()

	req := createAuthenticatedRequest(http.MethodPost, "/query", []byte("invalid json"), testUsername)
	w := httptest.NewRecorder()

	server.queryDocuments(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestQueryDocumentsEmbeddingError(t *testing.T) {
	const testUsername = "testuser"
	server, embedder, _, _, _ := createTestServer()
	embedder.SetShouldFail(true)

	queryReq := models.QueryRequest{
		Question: "What information is available?",
		TopK:     3,
	}

	body, _ := json.Marshal(queryReq)
	req := createAuthenticatedRequest(http.MethodPost, "/query", body, testUsername)
	w := httptest.NewRecorder()

	server.queryDocuments(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status %d, got %d", http.StatusInternalServerError, w.Code)
	}
}

func TestQueryDocumentsSearchError(t *testing.T) {
	const testUsername = "testuser"
	server, _, vectorStore, _, _ := createTestServer()
	vectorStore.SetSearchError(true)

	queryReq := models.QueryRequest{
		Question: "What information is available?",
		TopK:     3,
	}

	body, _ := json.Marshal(queryReq)
	req := createAuthenticatedRequest(http.MethodPost, "/query", body, testUsername)
	w := httptest.NewRecorder()

	server.queryDocuments(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status %d, got %d", http.StatusInternalServerError, w.Code)
	}
}

func TestQueryDocumentsLLMError(t *testing.T) {
	const testUsername = "testuser"
	server, _, _, llmClient, _ := createTestServer()
	llmClient.SetShouldFail(true)

	queryReq := models.QueryRequest{
		Question: "What information is available?",
		TopK:     3,
	}

	body, _ := json.Marshal(queryReq)
	req := createAuthenticatedRequest(http.MethodPost, "/query", body, testUsername)
	w := httptest.NewRecorder()

	server.queryDocuments(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status %d, got %d", http.StatusInternalServerError, w.Code)
	}
}

func TestHandlePermissions(t *testing.T) {
	const testUsername = "testuser"
	server, _, _, _, permService := createTestServer()

	permService.SetUserPermissions(testUsername, []string{"documents:view", "documents:query"})

	req := createAuthenticatedRequest(http.MethodGet, "/permissions", nil, testUsername)
	w := httptest.NewRecorder()

	server.handlePermissions(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response["user"] != testUsername {
		t.Errorf("Expected user '%s', got '%s'", testUsername, response["user"])
	}

	permissions := response["permissions"].([]interface{})
	if len(permissions) != 2 {
		t.Errorf("Expected 2 permissions, got %d", len(permissions))
	}
}

func TestHandlePermissionsInvalidMethod(t *testing.T) {
	const testUsername = "testuser"
	server, _, _, _, _ := createTestServer()

	req := createAuthenticatedRequest(http.MethodPost, "/permissions", nil, testUsername)
	w := httptest.NewRecorder()

	server.handlePermissions(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status %d, got %d", http.StatusMethodNotAllowed, w.Code)
	}
}

func TestHandleDocumentsMethodSwitch(t *testing.T) {
	server, _, _, _, _ := createTestServer()

	// Test invalid method
	req := httptest.NewRequest(http.MethodPut, "/documents", nil)
	w := httptest.NewRecorder()

	server.handleDocuments(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status %d, got %d", http.StatusMethodNotAllowed, w.Code)
	}
}

func TestDocumentUpsertBehavior(t *testing.T) {
	server, embedder, vectorStore, _, _ := createTestServer()

	// Add initial document and get its ID
	docID := addInitialDocumentForUpsert(t, server, embedder)

	// Update the same document (upsert)
	updateDocumentForUpsert(t, server, embedder, docID)

	// Verify the upsert worked correctly
	verifyUpsertResult(t, vectorStore, docID)
}

func addInitialDocumentForUpsert(t *testing.T, server *Server, embedder *MockEmbedder) string {
	doc := models.Document{
		Title:    "Initial Title",
		Content:  "Initial content for upsert test",
		Metadata: map[string]interface{}{"version": "1.0"},
	}
	embedder.SetEmbedding(doc.Content, []float32{0.1, 0.2, 0.3, 0.4})

	body, _ := json.Marshal(doc)
	req := httptest.NewRequest(http.MethodPost, "/documents", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	server.addDocument(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("Expected status %d, got %d", http.StatusCreated, w.Code)
	}

	var response map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}
	return response["id"]
}

func updateDocumentForUpsert(t *testing.T, server *Server, embedder *MockEmbedder, docID string) {
	parsedID, err := uuid.Parse(docID)
	if err != nil {
		t.Fatalf("Failed to parse document ID: %v", err)
	}

	updatedDoc := models.Document{
		ID:       parsedID,
		Title:    "Updated Title",
		Content:  "Updated content for upsert test",
		Metadata: map[string]interface{}{"version": "2.0"},
	}
	embedder.SetEmbedding(updatedDoc.Content, []float32{0.2, 0.3, 0.4, 0.5})

	body, _ := json.Marshal(updatedDoc)
	req := httptest.NewRequest(http.MethodPost, "/documents", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	server.addDocument(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("Expected status %d for upsert, got %d", http.StatusCreated, w.Code)
	}
}

func verifyUpsertResult(t *testing.T, vectorStore *MockVectorStore, docID string) {
	documents := vectorStore.GetAllDocuments()

	var finalDoc *models.Document
	docCount := 0
	for _, doc := range documents {
		if doc.ID.String() == docID {
			docCount++
			finalDoc = &doc
		}
	}

	if docCount != 1 {
		t.Errorf("Expected exactly 1 document with ID %s, got %d", docID, docCount)
	}
	if finalDoc == nil {
		t.Fatal("Updated document not found")
	}
	if finalDoc.Title != "Updated Title" {
		t.Errorf("Expected updated title 'Updated Title', got '%s'", finalDoc.Title)
	}
	if version, ok := finalDoc.Metadata["version"].(string); !ok || version != "2.0" {
		t.Errorf("Expected version '2.0', got %v", finalDoc.Metadata["version"])
	}
}
