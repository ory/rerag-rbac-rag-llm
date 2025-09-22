package api

import (
	"cmp"
	"encoding/json"
	"llm-rag-poc/internal/auth"
	"llm-rag-poc/internal/models"
	"llm-rag-poc/internal/permissions"
	"llm-rag-poc/internal/storage"
	"log"
	"net/http"

	"github.com/ory/herodot"
)

// Interfaces for dependency injection
type EmbedderInterface interface {
	GetEmbedding(text string) ([]float32, error)
}

type LLMInterface interface {
	Generate(question string, documents []models.Document) (string, error)
}

type Server struct {
	mux         *http.ServeMux
	embedder    EmbedderInterface
	vectorStore storage.VectorStore
	llmClient   LLMInterface
	permService permissions.PermissionChecker
	writer      *herodot.JSONWriter
}

func NewServer(embedder EmbedderInterface, vectorStore storage.VectorStore, llmClient LLMInterface, permService permissions.PermissionChecker) *Server {
	s := &Server{
		mux:         http.NewServeMux(),
		embedder:    embedder,
		vectorStore: vectorStore,
		llmClient:   llmClient,
		permService: permService,
		writer:      herodot.NewJSONWriter(nil),
	}

	s.setupRoutes()
	return s
}

func (s *Server) setupRoutes() {
	s.mux.HandleFunc("/documents", s.handleDocuments)
	s.mux.Handle("/query", auth.AuthMiddleware(http.HandlerFunc(s.queryDocuments)))
	s.mux.HandleFunc("/health", s.healthCheck)
	s.mux.Handle("/permissions", auth.AuthMiddleware(http.HandlerFunc(s.handlePermissions)))
}

func (s *Server) Run(addr string) error {
	log.Printf("Server starting on %s", addr)
	handler := loggingMiddleware(s.mux)
	return http.ListenAndServe(addr, handler)
}

func (s *Server) handleDocuments(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		s.addDocument(w, r)
	case http.MethodGet:
		s.listDocuments(w, r)
	default:
		http.Error(w, `{"error": "Method not allowed"}`, http.StatusMethodNotAllowed)
	}
}

func (s *Server) addDocument(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var doc models.Document
	if err := json.NewDecoder(r.Body).Decode(&doc); err != nil {
		s.writer.WriteError(w, r, herodot.ErrBadRequest.WithReason("Invalid request body"))
		return
	}

	embedding, err := s.embedder.GetEmbedding(doc.Content)
	if err != nil {
		s.writer.WriteError(w, r, herodot.ErrInternalServerError.WithReason("Failed to generate embedding"))
		return
	}

	doc.Embedding = embedding
	if err := s.vectorStore.AddDocument(&doc); err != nil {
		s.writer.WriteError(w, r, herodot.ErrInternalServerError.WithReason("Failed to store document"))
		return
	}

	response := &models.DocumentResponse{
		ID:      doc.ID.String(),
		Message: "Document added successfully",
	}
	s.writer.WriteCreated(w, r, "", response)
}

func (s *Server) listDocuments(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	username := auth.GetUserFromContext(r.Context())
	filter := func(doc *models.Document) bool {
		return s.permService.CanAccessDocument(username, doc)
	}

	docs := s.vectorStore.GetFilteredDocuments(filter)
	response := &models.DocumentListResponse{
		Documents: docs,
		Count:     len(docs),
		User:      username,
	}
	s.writer.Write(w, r, response)
}

func (s *Server) queryDocuments(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error": "Method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	var req models.QueryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writer.WriteError(w, r, herodot.ErrBadRequest.WithReason("Invalid request body"))
		return
	}

	req.TopK = cmp.Or(req.TopK, 3)

	questionEmbedding, err := s.embedder.GetEmbedding(req.Question)
	if err != nil {
		s.writer.WriteError(w, r, herodot.ErrInternalServerError.WithReason("Failed to generate question embedding"))
		return
	}

	username := auth.GetUserFromContext(r.Context())
	filter := func(doc *models.Document) bool {
		return s.permService.CanAccessDocument(username, doc)
	}

	relevantDocs, err := s.vectorStore.SearchSimilarWithFilter(questionEmbedding, req.TopK, filter)
	if err != nil {
		s.writer.WriteError(w, r, herodot.ErrInternalServerError.WithReason("Failed to search documents"))
		return
	}

	answer, err := s.llmClient.Generate(req.Question, relevantDocs)
	if err != nil {
		s.writer.WriteError(w, r, herodot.ErrInternalServerError.WithReason("Failed to generate answer"))
		return
	}

	response := &models.QueryResponse{
		Answer:  answer,
		Sources: relevantDocs,
	}
	s.writer.Write(w, r, response)
}

func (s *Server) healthCheck(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, `{"error": "Method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	response := &models.HealthResponse{Status: "healthy"}
	s.writer.Write(w, r, response)
}

func (s *Server) handlePermissions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, `{"error": "Method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	username := auth.GetUserFromContext(r.Context())
	permissions := s.permService.GetUserPermissions(username)
	response := &models.PermissionsResponse{
		User:        username,
		Permissions: permissions,
	}
	s.writer.Write(w, r, response)
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s %s", r.Method, r.RequestURI, r.RemoteAddr)
		next.ServeHTTP(w, r)
	})
}
