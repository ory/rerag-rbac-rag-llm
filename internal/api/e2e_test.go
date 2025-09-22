// Package api provides E2E/functional tests for the API endpoints
package api

import (
	"bytes"
	"encoding/json"
	"llm-rag-poc/internal/models"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
)

// E2E/Functional Tests - Test the full API flow with auth middleware

func TestE2E_DocumentWorkflow(t *testing.T) {
	server, _, _, _, _ := createTestServer()

	docID := addTestDocument(t, server)
	testDocumentListingWithoutAuth(t, server)
	testDocumentListingWithAuth(t, server, docID)
}

func addTestDocument(t *testing.T, server *Server) string {
	doc := models.Document{
		Title:   "E2E Test Document",
		Content: "This is comprehensive test content for end-to-end testing",
		Metadata: map[string]interface{}{
			"category":  "test",
			"sensitive": true,
		},
	}

	body, _ := json.Marshal(doc)
	req := httptest.NewRequest(http.MethodPost, "/documents", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.mux.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("Failed to add document: status %d", w.Code)
	}

	var addResponse map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &addResponse); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}
	return addResponse["id"]
}

func testDocumentListingWithoutAuth(t *testing.T, server *Server) {
	req := httptest.NewRequest(http.MethodGet, "/documents", nil)
	w := httptest.NewRecorder()

	server.mux.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected unauthorized without auth header, got %d", w.Code)
	}
}

func testDocumentListingWithAuth(t *testing.T, server *Server, docID string) {
	req := httptest.NewRequest(http.MethodGet, "/documents", nil)
	req.Header.Set("Authorization", "Bearer testuser")
	w := httptest.NewRecorder()

	server.mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected success with auth, got %d", w.Code)
	}

	var listResponse map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &listResponse); err != nil {
		t.Fatalf("Failed to unmarshal list response: %v", err)
	}

	if listResponse["user"] != "testuser" {
		t.Errorf("Expected user 'testuser', got %v", listResponse["user"])
	}

	documents := listResponse["documents"].([]interface{})
	if len(documents) != 1 {
		t.Errorf("Expected 1 document, got %d", len(documents))
	}

	docData := documents[0].(map[string]interface{})
	if docData["id"] != docID {
		t.Errorf("Expected document ID %s, got %v", docID, docData["id"])
	}
}

func TestE2E_QueryWorkflow(t *testing.T) {
	server, embedder, _, llmClient, _ := createTestServer()

	addQueryTestDocuments(t, server)
	setupQueryMocks(embedder, llmClient)
	testQueryWithoutAuth(t, server)
	testQueryWithInvalidAuth(t, server)
	testQueryWithValidAuth(t, server)
}

func addQueryTestDocuments(t *testing.T, server *Server) {
	docs := []models.Document{
		{
			Title:    "Financial Report",
			Content:  "The company made a profit of $100,000 this year",
			Metadata: map[string]interface{}{"type": "financial"},
		},
		{
			Title:    "HR Document",
			Content:  "Employee satisfaction scores are high at 95%",
			Metadata: map[string]interface{}{"type": "hr"},
		},
	}

	for _, doc := range docs {
		body, _ := json.Marshal(doc)
		req := httptest.NewRequest(http.MethodPost, "/documents", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		server.mux.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("Failed to add document: %v", w.Body.String())
		}
	}
}

func setupQueryMocks(embedder *MockEmbedder, llmClient *MockLLMClient) {
	question := "What was the company's profit?"
	embedder.SetEmbedding(question, []float32{0.1, 0.2, 0.3})
	llmClient.SetResponse(question, "Based on the financial report, the company made a profit of $100,000.")
}

func testQueryWithoutAuth(t *testing.T, server *Server) {
	queryReq := models.QueryRequest{
		Question: "What was the company's profit?",
		TopK:     3,
	}

	body, _ := json.Marshal(queryReq)
	req := httptest.NewRequest(http.MethodPost, "/query", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.mux.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected unauthorized without auth, got %d", w.Code)
	}
}

func testQueryWithInvalidAuth(t *testing.T, server *Server) {
	queryReq := models.QueryRequest{
		Question: "What was the company's profit?",
		TopK:     3,
	}

	body, _ := json.Marshal(queryReq)
	req := httptest.NewRequest(http.MethodPost, "/query", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Invalid format")
	w := httptest.NewRecorder()

	server.mux.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected unauthorized with invalid auth, got %d", w.Code)
	}
}

func testQueryWithValidAuth(t *testing.T, server *Server) {
	queryReq := models.QueryRequest{
		Question: "What was the company's profit?",
		TopK:     3,
	}

	body, _ := json.Marshal(queryReq)
	req := httptest.NewRequest(http.MethodPost, "/query", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer alice")
	w := httptest.NewRecorder()

	server.mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected success with valid auth, got %d: %s", w.Code, w.Body.String())
	}

	var queryResponse models.QueryResponse
	if err := json.Unmarshal(w.Body.Bytes(), &queryResponse); err != nil {
		t.Fatalf("Failed to unmarshal query response: %v", err)
	}

	if queryResponse.Answer != "Based on the financial report, the company made a profit of $100,000." {
		t.Errorf("Expected specific answer, got: %s", queryResponse.Answer)
	}

	if len(queryResponse.Sources) == 0 {
		t.Error("Expected at least one source document")
	}
}

func TestE2E_PermissionWorkflow(t *testing.T) {
	server, _, vectorStore, _, permService := createTestServer()

	doc1, doc2, doc3 := setupPermissionTestDocuments(t, vectorStore)
	setupPermissionTestUsers(permService, doc1, doc2, doc3)
	testUserDocumentAccess(t, server)
	testPermissionsEndpoint(t, server)
}

func setupPermissionTestDocuments(t *testing.T, vectorStore *MockVectorStore) (*models.Document, *models.Document, *models.Document) {
	doc1 := &models.Document{
		ID:      uuid.New(),
		Title:   "Public Document",
		Content: "This is public information",
	}
	doc2 := &models.Document{
		ID:      uuid.New(),
		Title:   "Private Document",
		Content: "This is private information",
	}
	doc3 := &models.Document{
		ID:      uuid.New(),
		Title:   "Admin Only Document",
		Content: "This is admin-only information",
	}

	if err := vectorStore.AddDocument(doc1); err != nil {
		t.Fatalf("Failed to add doc1: %v", err)
	}
	if err := vectorStore.AddDocument(doc2); err != nil {
		t.Fatalf("Failed to add doc2: %v", err)
	}
	if err := vectorStore.AddDocument(doc3); err != nil {
		t.Fatalf("Failed to add doc3: %v", err)
	}

	return doc1, doc2, doc3
}

func setupPermissionTestUsers(permService *MockPermissionService, doc1, doc2, doc3 *models.Document) {
	permService.SetDocumentAccess("alice", doc1.ID.String(), true)
	permService.SetDocumentAccess("alice", doc2.ID.String(), true)
	permService.SetDocumentAccess("alice", doc3.ID.String(), false)
	permService.SetUserPermissions("alice", []string{"documents:view", "documents:query"})

	permService.SetDocumentAccess("bob", doc1.ID.String(), true)
	permService.SetDocumentAccess("bob", doc2.ID.String(), false)
	permService.SetDocumentAccess("bob", doc3.ID.String(), false)
	permService.SetUserPermissions("bob", []string{"documents:view"})

	permService.SetDocumentAccess("admin", doc1.ID.String(), true)
	permService.SetDocumentAccess("admin", doc2.ID.String(), true)
	permService.SetDocumentAccess("admin", doc3.ID.String(), true)
	permService.SetUserPermissions("admin", []string{"documents:view", "documents:query", "documents:admin"})
}

func testUserDocumentAccess(t *testing.T, server *Server) {
	testUserAccess(t, server, "alice", 2)
	testUserAccess(t, server, "bob", 1)
	testUserAccess(t, server, "admin", 3)
}

func testUserAccess(t *testing.T, server *Server, user string, expectedDocs int) {
	req := httptest.NewRequest(http.MethodGet, "/documents", nil)
	req.Header.Set("Authorization", "Bearer "+user)
	w := httptest.NewRecorder()

	server.mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("%s request failed: %d", user, w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal %s response: %v", user, err)
	}

	docs := response["documents"].([]interface{})
	if len(docs) != expectedDocs {
		t.Errorf("%s should see %d documents, got %d", user, expectedDocs, len(docs))
	}
}

func testPermissionsEndpoint(t *testing.T, server *Server) {
	req := httptest.NewRequest(http.MethodGet, "/permissions", nil)
	req.Header.Set("Authorization", "Bearer alice")
	w := httptest.NewRecorder()

	server.mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Permissions request failed: %d", w.Code)
	}

	var permResponse map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &permResponse); err != nil {
		t.Fatalf("Failed to unmarshal perm response: %v", err)
	}

	permissions := permResponse["permissions"].([]interface{})
	if len(permissions) != 2 {
		t.Errorf("Alice should have 2 permissions, got %d", len(permissions))
	}
}

func TestE2E_ErrorHandling(t *testing.T) {
	server, embedder, vectorStore, llmClient, _ := createTestServer()

	// Test 1: Service failures during query
	embedder.SetShouldFail(true)

	queryReq := models.QueryRequest{
		Question: "Test question",
		TopK:     3,
	}

	body, _ := json.Marshal(queryReq)
	req := httptest.NewRequest(http.MethodPost, "/query", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer testuser")
	w := httptest.NewRecorder()

	server.mux.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected 500 for embedding failure, got %d", w.Code)
	}

	// Reset and test vector store failure
	embedder.SetShouldFail(false)
	vectorStore.SetSearchError(true)

	body, _ = json.Marshal(queryReq)
	req = httptest.NewRequest(http.MethodPost, "/query", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer testuser")
	w = httptest.NewRecorder()
	server.mux.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected 500 for search failure, got %d", w.Code)
	}

	// Reset and test LLM failure
	vectorStore.SetSearchError(false)
	llmClient.SetShouldFail(true)

	body, _ = json.Marshal(queryReq)
	req = httptest.NewRequest(http.MethodPost, "/query", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer testuser")
	w = httptest.NewRecorder()
	server.mux.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected 500 for LLM failure, got %d", w.Code)
	}
}

func TestE2E_ConcurrentAccess(t *testing.T) {
	server, _, _, _, _ := createTestServer()

	// Add a document first
	doc := models.Document{
		Title:   "Concurrent Test Document",
		Content: "This is for testing concurrent access",
	}

	body, _ := json.Marshal(doc)
	req := httptest.NewRequest(http.MethodPost, "/documents", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.mux.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("Failed to add document for concurrent test: %d", w.Code)
	}

	// Test concurrent requests
	done := make(chan bool, 10)
	errors := make(chan error, 10)

	for i := 0; i < 10; i++ {
		go func(userID int) {
			req := httptest.NewRequest(http.MethodGet, "/documents", nil)
			req.Header.Set("Authorization", "Bearer user"+string(rune(userID)))
			w := httptest.NewRecorder()

			server.mux.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				errors <- &ConcurrentTestError{UserID: userID, StatusCode: w.Code}
				return
			}

			done <- true
		}(i)
	}

	// Wait for all requests to complete
	successCount := 0
	errorCount := 0

	timeout := time.NewTimer(5 * time.Second)
	defer timeout.Stop()

	for i := 0; i < 10; i++ {
		select {
		case <-done:
			successCount++
		case err := <-errors:
			t.Logf("Concurrent request error: %v", err)
			errorCount++
		case <-timeout.C:
			t.Fatal("Timeout waiting for concurrent requests")
		}
	}

	if successCount != 10 {
		t.Errorf("Expected 10 successful requests, got %d (errors: %d)", successCount, errorCount)
	}
}

func TestE2E_HealthEndpoint(t *testing.T) {
	server, _, _, _, _ := createTestServer()

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	server.mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Health check failed: %d", w.Code)
	}

	var response map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response["status"] != "healthy" {
		t.Errorf("Expected healthy status, got %s", response["status"])
	}
}

func TestE2E_InvalidEndpoints(t *testing.T) {
	server, _, _, _, _ := createTestServer()

	// Test non-existent endpoint
	req := httptest.NewRequest(http.MethodGet, "/nonexistent", nil)
	w := httptest.NewRecorder()

	server.mux.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected 404 for non-existent endpoint, got %d", w.Code)
	}

	// Test invalid methods on existing endpoints
	testCases := []struct {
		method   string
		endpoint string
		expected int
	}{
		{http.MethodPatch, "/health", http.StatusMethodNotAllowed},
		{http.MethodDelete, "/documents", http.StatusMethodNotAllowed},
		{http.MethodPut, "/query", http.StatusMethodNotAllowed},
		{http.MethodPost, "/permissions", http.StatusMethodNotAllowed},
	}

	for _, tc := range testCases {
		req := httptest.NewRequest(tc.method, tc.endpoint, nil)
		if tc.endpoint == "/query" || tc.endpoint == "/permissions" {
			req.Header.Set("Authorization", "Bearer testuser")
		}
		w := httptest.NewRecorder()

		server.mux.ServeHTTP(w, req)

		if w.Code != tc.expected {
			t.Errorf("Expected %d for %s %s, got %d", tc.expected, tc.method, tc.endpoint, w.Code)
		}
	}
}

// Helper error type for concurrent testing
type ConcurrentTestError struct {
	UserID     int
	StatusCode int
}

func (e *ConcurrentTestError) Error() string {
	return "Concurrent test error"
}
