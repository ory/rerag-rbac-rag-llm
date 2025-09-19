package api

import (
	"encoding/json"
	"llm-rag-poc/internal/embeddings"
	"llm-rag-poc/internal/llm"
	"llm-rag-poc/internal/models"
	"llm-rag-poc/internal/storage"
	"log"
	"net/http"
)

type Server struct {
	mux         *http.ServeMux
	embedder    *embeddings.Embedder
	vectorStore storage.VectorStore
	llmClient   *llm.OllamaClient
}

func NewServer(embedder *embeddings.Embedder, vectorStore storage.VectorStore, llmClient *llm.OllamaClient) *Server {
	s := &Server{
		mux:         http.NewServeMux(),
		embedder:    embedder,
		vectorStore: vectorStore,
		llmClient:   llmClient,
	}

	s.setupRoutes()
	return s
}

func (s *Server) setupRoutes() {
	s.mux.HandleFunc("/documents", s.handleDocuments)
	s.mux.HandleFunc("/query", s.queryDocuments)
	s.mux.HandleFunc("/health", s.healthCheck)
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
		http.Error(w, `{"error": "Invalid request body"}`, http.StatusBadRequest)
		return
	}

	embedding, err := s.embedder.GetEmbedding(doc.Content)
	if err != nil {
		http.Error(w, `{"error": "Failed to generate embedding"}`, http.StatusInternalServerError)
		return
	}

	doc.Embedding = embedding
	if err := s.vectorStore.AddDocument(&doc); err != nil {
		http.Error(w, `{"error": "Failed to store document"}`, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{
		"id":      doc.ID,
		"message": "Document added successfully",
	})
}

func (s *Server) listDocuments(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	docs := s.vectorStore.GetAllDocuments()
	json.NewEncoder(w).Encode(map[string]interface{}{
		"documents": docs,
		"count":     len(docs),
	})
}

func (s *Server) queryDocuments(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error": "Method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	
	var req models.QueryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error": "Invalid request body"}`, http.StatusBadRequest)
		return
	}

	if req.TopK == 0 {
		req.TopK = 3
	}

	questionEmbedding, err := s.embedder.GetEmbedding(req.Question)
	if err != nil {
		http.Error(w, `{"error": "Failed to generate question embedding"}`, http.StatusInternalServerError)
		return
	}

	relevantDocs, err := s.vectorStore.SearchSimilar(questionEmbedding, req.TopK)
	if err != nil {
		http.Error(w, `{"error": "Failed to search documents"}`, http.StatusInternalServerError)
		return
	}

	answer, err := s.llmClient.Generate(req.Question, relevantDocs)
	if err != nil {
		http.Error(w, `{"error": "Failed to generate answer"}`, http.StatusInternalServerError)
		return
	}

	response := models.QueryResponse{
		Answer:  answer,
		Sources: make([]models.Document, len(relevantDocs)),
	}

	for i, doc := range relevantDocs {
		response.Sources[i] = *doc
	}

	json.NewEncoder(w).Encode(response)
}

func (s *Server) healthCheck(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, `{"error": "Method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s %s", r.Method, r.RequestURI, r.RemoteAddr)
		next.ServeHTTP(w, r)
	})
}