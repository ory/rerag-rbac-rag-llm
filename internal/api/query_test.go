package api

import (
	"encoding/json"
	"llm-rag-poc/internal/models"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
)

func TestQueryScenarios(t *testing.T) {
	tests := []struct {
		name     string
		username string
		setup    func(*MockEmbedder, *MockVectorStore, *MockLLMClient, *MockPermissionService)
		query    models.QueryRequest
		wantCode int
		validate func(*testing.T, models.QueryResponse)
	}{
		{
			name:     "John Doe refund amount - Alice (can access)",
			username: "alice",
			setup: func(e *MockEmbedder, v *MockVectorStore, l *MockLLMClient, p *MockPermissionService) {
				johnDoeDoc := &models.Document{
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
				_ = v.AddDocument(johnDoeDoc)

				p.SetUserPermissions("alice", []string{"taxpayer:John Doe"})
				p.SetDocumentAccess("alice", johnDoeDoc.ID.String(), true)

				question := "What was John Doe's refund amount in 2023?"
				e.SetEmbedding(question, []float32{0.1, 0.2, 0.3})
				l.SetResponse(question, "John Doe's refund amount in 2023 was $2,500")
			},
			query: models.QueryRequest{
				Question: "What was John Doe's refund amount in 2023?",
				TopK:     3,
			},
			wantCode: http.StatusOK,
			validate: func(t *testing.T, resp models.QueryResponse) {
				if resp.Answer != "John Doe's refund amount in 2023 was $2,500" {
					t.Errorf("Expected answer about John Doe's refund, got %s", resp.Answer)
				}
				if len(resp.Sources) != 1 {
					t.Errorf("Expected 1 source document, got %d", len(resp.Sources))
				}
			},
		},
		{
			name:     "John Doe refund amount - Bob (cannot access)",
			username: "bob",
			setup: func(e *MockEmbedder, v *MockVectorStore, l *MockLLMClient, p *MockPermissionService) {
				johnDoeDoc := &models.Document{
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
				_ = v.AddDocument(johnDoeDoc)

				p.SetUserPermissions("bob", []string{"taxpayer:ABC Corporation"})
				p.SetDocumentAccess("bob", johnDoeDoc.ID.String(), false)

				question := "What was John Doe's refund amount in 2023?"
				e.SetEmbedding(question, []float32{0.1, 0.2, 0.3})
				l.SetResponse(question, "No information available")
			},
			query: models.QueryRequest{
				Question: "What was John Doe's refund amount in 2023?",
				TopK:     3,
			},
			wantCode: http.StatusOK,
			validate: func(t *testing.T, resp models.QueryResponse) {
				if len(resp.Sources) != 0 {
					t.Errorf("Bob should not have access to any documents, got %d sources", len(resp.Sources))
				}
			},
		},
		{
			name:     "Married filing jointly - Peter (can access all)",
			username: "peter",
			setup: func(e *MockEmbedder, v *MockVectorStore, l *MockLLMClient, p *MockPermissionService) {
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

				_ = v.AddDocument(johnDoeDoc)
				_ = v.AddDocument(smithDoc)

				p.SetUserPermissions("peter", []string{"taxpayer:*"})
				p.SetDocumentAccess("peter", johnDoeDoc.ID.String(), true)
				p.SetDocumentAccess("peter", smithDoc.ID.String(), true)

				question := "Which taxpayers filed as married filing jointly?"
				e.SetEmbedding(question, []float32{0.12, 0.22, 0.32})
				l.SetResponse(question, "John Doe and Smith Family filed as Married Filing Jointly")
			},
			query: models.QueryRequest{
				Question: "Which taxpayers filed as married filing jointly?",
				TopK:     3,
			},
			wantCode: http.StatusOK,
			validate: func(t *testing.T, resp models.QueryResponse) {
				if resp.Answer != "John Doe and Smith Family filed as Married Filing Jointly" {
					t.Errorf("Expected answer about married filing jointly, got %s", resp.Answer)
				}
				if len(resp.Sources) != 2 {
					t.Errorf("Expected 2 source documents, got %d", len(resp.Sources))
				}
			},
		},
		{
			name:     "ABC Corporation gross receipts - Bob (can access)",
			username: "bob",
			setup: func(e *MockEmbedder, v *MockVectorStore, l *MockLLMClient, p *MockPermissionService) {
				abcDoc := &models.Document{
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
				_ = v.AddDocument(abcDoc)

				p.SetUserPermissions("bob", []string{"taxpayer:ABC Corporation"})
				p.SetDocumentAccess("bob", abcDoc.ID.String(), true)

				question := "What was ABC Corporation's gross receipts in 2023?"
				e.SetEmbedding(question, []float32{0.2, 0.3, 0.4})
				l.SetResponse(question, "ABC Corporation's gross receipts in 2023 were $5,234,000")
			},
			query: models.QueryRequest{
				Question: "What was ABC Corporation's gross receipts in 2023?",
				TopK:     3,
			},
			wantCode: http.StatusOK,
			validate: func(t *testing.T, resp models.QueryResponse) {
				if resp.Answer != "ABC Corporation's gross receipts in 2023 were $5,234,000" {
					t.Errorf("Expected answer about ABC Corp gross receipts, got %s", resp.Answer)
				}
				if len(resp.Sources) != 1 {
					t.Errorf("Expected 1 source document, got %d", len(resp.Sources))
				}
			},
		},
		{
			name:     "Child tax credit - Peter (can access all)",
			username: "peter",
			setup: func(e *MockEmbedder, v *MockVectorStore, l *MockLLMClient, p *MockPermissionService) {
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

				_ = v.AddDocument(johnDoeDoc)
				_ = v.AddDocument(smithDoc)

				p.SetUserPermissions("peter", []string{"taxpayer:*"})
				p.SetDocumentAccess("peter", johnDoeDoc.ID.String(), true)
				p.SetDocumentAccess("peter", smithDoc.ID.String(), true)

				question := "Which taxpayers received child tax credit and how much?"
				e.SetEmbedding(question, []float32{0.32, 0.42, 0.52})
				l.SetResponse(question, "John Doe received $2,000 in child tax credit for 1 child. Smith Family received $6,000 for 3 children.")
			},
			query: models.QueryRequest{
				Question: "Which taxpayers received child tax credit and how much?",
				TopK:     3,
			},
			wantCode: http.StatusOK,
			validate: func(t *testing.T, resp models.QueryResponse) {
				if resp.Answer != "John Doe received $2,000 in child tax credit for 1 child. Smith Family received $6,000 for 3 children." {
					t.Errorf("Unexpected answer about child tax credit: %s", resp.Answer)
				}
				if len(resp.Sources) != 2 {
					t.Errorf("Expected 2 source documents, got %d", len(resp.Sources))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server, embedder, vectorStore, llmClient, permService := createTestServer()

			tt.setup(embedder, vectorStore, llmClient, permService)

			body, _ := json.Marshal(tt.query)
			req := createAuthenticatedRequest(http.MethodPost, "/query", body, tt.username)
			w := httptest.NewRecorder()

			server.queryDocuments(w, req)

			if w.Code != tt.wantCode {
				t.Errorf("Expected status %d, got %d", tt.wantCode, w.Code)
			}

			if tt.validate != nil && w.Code == http.StatusOK {
				var response models.QueryResponse
				if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
					t.Fatalf("Failed to unmarshal response: %v", err)
				}
				tt.validate(t, response)
			}
		})
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
