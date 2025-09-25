package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"rerag-rbac-rag-llm/internal/models"
	"testing"

	"github.com/google/uuid"
)

func TestQuery_JohnDoeRefund_AliceCanAccess(t *testing.T) {
	server, embedder, vectorStore, llmClient, permService := createTestServer()

	johnDoeDoc := setupJohnDoeDocument(vectorStore)
	setupAlicePermissions(permService, johnDoeDoc.ID.String())

	question := "What was John Doe's refund amount in 2023?"
	embedder.SetEmbedding(question, []float32{0.1, 0.2, 0.3})
	llmClient.SetResponse(question, "John Doe's refund amount in 2023 was $2,500")

	response := executeQuery(t, server, question, "alice")
	validateJohnDoeRefundResponse(t, response)
}

func TestQuery_JohnDoeRefund_BobCannotAccess(t *testing.T) {
	server, embedder, vectorStore, llmClient, permService := createTestServer()

	johnDoeDoc := setupJohnDoeDocument(vectorStore)
	setupBobPermissions(permService, johnDoeDoc.ID.String())

	question := "What was John Doe's refund amount in 2023?"
	embedder.SetEmbedding(question, []float32{0.1, 0.2, 0.3})
	llmClient.SetResponse(question, "No information available")

	response := executeQuery(t, server, question, "bob")
	if len(response.Sources) != 0 {
		t.Errorf("Bob should not have access to any documents, got %d sources", len(response.Sources))
	}
}

func TestQuery_MarriedFilingJointly_PeterCanAccessAll(t *testing.T) {
	server, embedder, vectorStore, llmClient, permService := createTestServer()

	johnDoeDoc, smithDoc := setupMarriedFilingJointlyDocuments(vectorStore)
	setupPeterPermissions(permService, johnDoeDoc.ID.String(), smithDoc.ID.String())

	question := "Which taxpayers filed as married filing jointly?"
	embedder.SetEmbedding(question, []float32{0.12, 0.22, 0.32})
	llmClient.SetResponse(question, "John Doe and Smith Family filed as Married Filing Jointly")

	response := executeQuery(t, server, question, "peter")
	if response.Answer != "John Doe and Smith Family filed as Married Filing Jointly" {
		t.Errorf("Expected answer about married filing jointly, got %s", response.Answer)
	}
	if len(response.Sources) != 2 {
		t.Errorf("Expected 2 source documents, got %d", len(response.Sources))
	}
}

func TestQuery_ABCCorporationGrossReceipts_BobCanAccess(t *testing.T) {
	server, embedder, vectorStore, llmClient, permService := createTestServer()

	abcDoc := setupABCCorporationDocument(vectorStore)
	setupBobCorporationPermissions(permService, abcDoc.ID.String())

	question := "What was ABC Corporation's gross receipts in 2023?"
	embedder.SetEmbedding(question, []float32{0.2, 0.3, 0.4})
	llmClient.SetResponse(question, "ABC Corporation's gross receipts in 2023 were $5,234,000")

	response := executeQuery(t, server, question, "bob")
	if response.Answer != "ABC Corporation's gross receipts in 2023 were $5,234,000" {
		t.Errorf("Expected answer about ABC Corp gross receipts, got %s", response.Answer)
	}
	if len(response.Sources) != 1 {
		t.Errorf("Expected 1 source document, got %d", len(response.Sources))
	}
}

func TestQuery_ChildTaxCredit_PeterCanAccessAll(t *testing.T) {
	server, embedder, vectorStore, llmClient, permService := createTestServer()

	johnDoeDoc, smithDoc := setupChildTaxCreditDocuments(vectorStore)
	setupPeterPermissions(permService, johnDoeDoc.ID.String(), smithDoc.ID.String())

	question := "Which taxpayers received child tax credit and how much?"
	embedder.SetEmbedding(question, []float32{0.32, 0.42, 0.52})
	expectedAnswer := "John Doe received $2,000 in child tax credit for 1 child. Smith Family received $6,000 for 3 children."
	llmClient.SetResponse(question, expectedAnswer)

	response := executeQuery(t, server, question, "peter")
	if response.Answer != expectedAnswer {
		t.Errorf("Unexpected answer about child tax credit: %s", response.Answer)
	}
	if len(response.Sources) != 2 {
		t.Errorf("Expected 2 source documents, got %d", len(response.Sources))
	}
}

func setupJohnDoeDocument(vectorStore *MockVectorStore) *models.Document {
	doc := &models.Document{
		ID:      uuid.New(),
		Title:   "Tax Return - John Doe",
		Content: "John Doe's 2023 tax return shows AGI of $85,000, refund amount of $2,500",
		Metadata: map[string]interface{}{
			"taxpayer": "John Doe",
			"year":     2023,
			"type":     "1040",
		},
		Embedding: []float32{0.1, 0.2, 0.3},
	}
	_ = vectorStore.AddDocument(doc)
	return doc
}

func setupMarriedFilingJointlyDocuments(vectorStore *MockVectorStore) (*models.Document, *models.Document) {
	johnDoeDoc := &models.Document{
		ID:      uuid.New(),
		Title:   "Tax Return - John Doe",
		Content: "Filing Status: Married Filing Jointly, Spouse: Jane Doe",
		Metadata: map[string]interface{}{
			"taxpayer":      "John Doe",
			"year":          2023,
			"type":          "1040",
			"filing_status": "Married Filing Jointly",
		},
		Embedding: []float32{0.1, 0.2, 0.3},
	}

	smithDoc := &models.Document{
		ID:      uuid.New(),
		Title:   "Tax Return - Smith Family",
		Content: "Filing Status: Married Filing Jointly, Robert and Mary Smith",
		Metadata: map[string]interface{}{
			"taxpayer":      "Smith Family",
			"year":          2023,
			"type":          "1040",
			"filing_status": "Married Filing Jointly",
		},
		Embedding: []float32{0.15, 0.25, 0.35},
	}

	_ = vectorStore.AddDocument(johnDoeDoc)
	_ = vectorStore.AddDocument(smithDoc)
	return johnDoeDoc, smithDoc
}

func setupABCCorporationDocument(vectorStore *MockVectorStore) *models.Document {
	doc := &models.Document{
		ID:      uuid.New(),
		Title:   "Tax Return - ABC Corporation",
		Content: "ABC Corporation 2023 Form 1120: Gross Receipts: $5,234,000, Net Income: $892,000",
		Metadata: map[string]interface{}{
			"taxpayer": "ABC Corporation",
			"year":     2023,
			"type":     "1120",
		},
		Embedding: []float32{0.2, 0.3, 0.4},
	}
	_ = vectorStore.AddDocument(doc)
	return doc
}

func setupChildTaxCreditDocuments(vectorStore *MockVectorStore) (*models.Document, *models.Document) {
	johnDoeDoc := &models.Document{
		ID:      uuid.New(),
		Title:   "Tax Return - John Doe",
		Content: "Child Tax Credit claimed: $2,000 for 1 qualifying child",
		Metadata: map[string]interface{}{
			"taxpayer":         "John Doe",
			"year":             2023,
			"type":             "1040",
			"child_tax_credit": 2000,
		},
		Embedding: []float32{0.3, 0.4, 0.5},
	}

	smithDoc := &models.Document{
		ID:      uuid.New(),
		Title:   "Tax Return - Smith Family",
		Content: "Child Tax Credit claimed: $6,000 for 3 qualifying children",
		Metadata: map[string]interface{}{
			"taxpayer":         "Smith Family",
			"year":             2023,
			"type":             "1040",
			"child_tax_credit": 6000,
		},
		Embedding: []float32{0.35, 0.45, 0.55},
	}

	_ = vectorStore.AddDocument(johnDoeDoc)
	_ = vectorStore.AddDocument(smithDoc)
	return johnDoeDoc, smithDoc
}

func setupAlicePermissions(permService *MockPermissionService, docID string) {
	permService.SetUserPermissions("alice", []string{"taxpayer:John Doe"})
	permService.SetDocumentAccess("alice", docID, true)
}

func setupBobPermissions(permService *MockPermissionService, docID string) {
	permService.SetUserPermissions("bob", []string{"taxpayer:ABC Corporation"})
	permService.SetDocumentAccess("bob", docID, false)
}

func setupBobCorporationPermissions(permService *MockPermissionService, docID string) {
	permService.SetUserPermissions("bob", []string{"taxpayer:ABC Corporation"})
	permService.SetDocumentAccess("bob", docID, true)
}

func setupPeterPermissions(permService *MockPermissionService, docIDs ...string) {
	permService.SetUserPermissions("peter", []string{"taxpayer:*"})
	for _, docID := range docIDs {
		permService.SetDocumentAccess("peter", docID, true)
	}
}

func executeQuery(t *testing.T, server *Server, question, username string) models.QueryResponse {
	query := models.QueryRequest{
		Question: question,
		TopK:     3,
	}

	body, _ := json.Marshal(query)
	req := createAuthenticatedRequest(http.MethodPost, "/query", body, username)
	w := httptest.NewRecorder()

	server.queryDocuments(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", w.Code)
	}

	var response models.QueryResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	return response
}

func validateJohnDoeRefundResponse(t *testing.T, response models.QueryResponse) {
	if response.Answer != "John Doe's refund amount in 2023 was $2,500" {
		t.Errorf("Expected answer about John Doe's refund, got %s", response.Answer)
	}
	if len(response.Sources) != 1 {
		t.Errorf("Expected 1 source document, got %d", len(response.Sources))
	}
}

func TestUserPermissions(t *testing.T) {
	tests := []struct {
		name        string
		username    string
		permissions []string
		wantCode    int
	}{
		{
			name:        "Alice permissions - John Doe only",
			username:    "alice",
			permissions: []string{"taxpayer:John Doe"},
			wantCode:    http.StatusOK,
		},
		{
			name:        "Bob permissions - ABC Corporation only",
			username:    "bob",
			permissions: []string{"taxpayer:ABC Corporation"},
			wantCode:    http.StatusOK,
		},
		{
			name:        "Peter permissions - all taxpayers",
			username:    "peter",
			permissions: []string{"taxpayer:*", "documents:view", "documents:query"},
			wantCode:    http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server, _, _, _, permService := createTestServer()

			permService.SetUserPermissions(tt.username, tt.permissions)

			req := createAuthenticatedRequest(http.MethodGet, "/permissions", nil, tt.username)
			w := httptest.NewRecorder()

			server.handlePermissions(w, req)

			if w.Code != tt.wantCode {
				t.Errorf("Expected status %d, got %d", tt.wantCode, w.Code)
			}

			var response map[string]interface{}
			if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
				t.Fatalf("Failed to unmarshal response: %v", err)
			}

			if response["user"] != tt.username {
				t.Errorf("Expected user '%s', got '%v'", tt.username, response["user"])
			}

			returnedPerms := response["permissions"].([]interface{})
			if len(returnedPerms) != len(tt.permissions) {
				t.Errorf("Expected %d permissions, got %d", len(tt.permissions), len(returnedPerms))
			}
		})
	}
}
