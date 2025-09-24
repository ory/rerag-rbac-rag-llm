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
	"time"

	"github.com/ory/herodot"
)

// EmbedderInterface defines the contract for text embedding services
type EmbedderInterface interface {
	GetEmbedding(text string) ([]float32, error)
}

// LLMInterface defines the contract for Large Language Model services
type LLMInterface interface {
	Generate(question string, documents []models.Document) (string, error)
}

// Server handles HTTP requests for the RAG API
type Server struct {
	mux         *http.ServeMux
	embedder    EmbedderInterface
	vectorStore storage.VectorStore
	llmClient   LLMInterface
	permService permissions.PermissionChecker
	writer      *herodot.JSONWriter
}

// NewServer creates a new API server with the provided dependencies
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
	s.mux.Handle("/query", auth.Middleware(http.HandlerFunc(s.queryDocuments)))
	s.mux.HandleFunc("/health", s.healthCheck)
	s.mux.Handle("/permissions", auth.Middleware(http.HandlerFunc(s.handlePermissions)))
}

// Run starts the HTTP server on the specified address
func (s *Server) Run(addr string) error {
	log.Printf("Server starting on %s", addr)
	handler := loggingMiddleware(s.mux)

	server := &http.Server{
		Addr:           addr,
		Handler:        handler,
		ReadTimeout:    15 * time.Second,
		WriteTimeout:   15 * time.Second,
		IdleTimeout:    60 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	return server.ListenAndServe()
}

func (s *Server) handleDocuments(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		s.addDocument(w, r)
	case http.MethodGet:
		// GET requests require authentication
		auth.Middleware(http.HandlerFunc(s.listDocuments)).ServeHTTP(w, r)
	default:
		http.Error(w, `{"error": "Method not allowed"}`, http.StatusMethodNotAllowed)
	}
}

func (s *Server) addDocument(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var doc models.Document
	if err := json.NewDecoder(r.Body).Decode(&doc); err != nil {
		s.writer.WriteError(w, r, herodot.ErrBadRequest.WithReason("Invalid request body").WithError(err.Error()))
		return
	}

	embedding, err := s.embedder.GetEmbedding(doc.Content)
	if err != nil {
		s.writer.WriteError(w, r, herodot.ErrInternalServerError.WithReason("Failed to generate embedding").WithError(err.Error()))
		return
	}

	doc.Embedding = embedding

	if err := s.vectorStore.UpsertDocument(&doc); err != nil {
		s.writer.WriteError(w, r, herodot.ErrInternalServerError.WithReason("Failed to store document").WithError(err.Error()))
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
		s.writer.WriteError(w, r, herodot.ErrBadRequest.WithReason("Invalid request body").WithError(err.Error()))
		return
	}

	req.TopK = cmp.Or(req.TopK, 3)

	questionEmbedding, err := s.embedder.GetEmbedding(req.Question)
	if err != nil {
		s.writer.WriteError(w, r, herodot.ErrInternalServerError.WithReason("Failed to generate question embedding").WithError(err.Error()))
		return
	}

	username := auth.GetUserFromContext(r.Context())
	filter := func(doc *models.Document) bool {
		return s.permService.CanAccessDocument(username, doc)
	}

	relevantDocs, err := s.vectorStore.SearchSimilarWithFilter(questionEmbedding, req.TopK, filter)
	if err != nil {
		s.writer.WriteError(w, r, herodot.ErrInternalServerError.WithReason("Failed to search documents").WithError(err.Error()))
		return
	}

	answer, err := s.llmClient.Generate(req.Question, relevantDocs)
	if err != nil {
		s.writer.WriteError(w, r, herodot.ErrInternalServerError.WithReason("Failed to generate answer").WithError(err.Error()))
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

// GetHandler returns the HTTP handler for the server
func (s *Server) GetHandler() http.Handler {
	return loggingMiddleware(s.mux)
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(timeout time.Duration) error {
	log.Printf("Server shutdown initiated with timeout: %v", timeout)
	// In a more complex implementation, you might close database connections,
	// stop background workers, etc.
	return nil
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s %s", r.Method, r.RequestURI, r.RemoteAddr)
		next.ServeHTTP(w, r)
	})
}
