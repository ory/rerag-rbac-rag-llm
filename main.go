// LLM RAG ReBAC OSS is a secure RAG system with relationship-based access control.
package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"rerag-rbac-rag-llm/internal/api"
	"rerag-rbac-rag-llm/internal/config"
	"rerag-rbac-rag-llm/internal/embeddings"
	"rerag-rbac-rag-llm/internal/llm"
	"rerag-rbac-rag-llm/internal/permissions"
	"rerag-rbac-rag-llm/internal/storage"
)

func main() {
	log.Println("Starting LLM RAG ReBAC OSS...")

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	logConfig(cfg)

	// Initialize components
	vectorStore, server := initializeComponents(cfg)
	defer func() {
		if err := vectorStore.Close(); err != nil {
			log.Printf("Error closing vector store: %v", err)
		}
	}()

	// Create and start HTTP server
	httpServer := createHTTPServer(cfg, server)
	startHTTPServer(cfg, httpServer)

	log.Println("Server started successfully")

	// Wait for shutdown signal
	waitForShutdown(server)
}

func logConfig(cfg *config.Config) {
	log.Printf("Environment: %s", cfg.App.Environment)
	log.Printf("Log Level: %s", cfg.App.LogLevel)
	log.Printf("TLS Enabled: %v", cfg.Server.TLS.Enabled)
	log.Printf("Database Encryption: %v", cfg.Database.Encryption.Enabled)
}

func initializeComponents(cfg *config.Config) (*storage.SQLiteVectorStore, *api.Server) {
	// Initialize embeddings client
	embedder := embeddings.NewEmbedder()

	// Initialize SQLite vector store with encryption support
	dsn := cfg.GetDatabaseDSN()
	log.Printf("Initializing database: %s", cfg.Database.Path)
	if cfg.Database.Encryption.Enabled {
		log.Println("Database encryption enabled")
	}

	vectorStore, err := storage.NewSQLiteVectorStore(dsn)
	if err != nil {
		log.Fatalf("Failed to initialize vector store: %v", err)
	}

	// Initialize LLM client
	ollama := llm.NewOllamaClient(cfg.Services.Ollama.BaseURL, cfg.Services.Ollama.LLMModel)

	// Initialize permissions service
	permService := permissions.NewKetoPermissionService(
		cfg.Services.Keto.ReadURL,
		cfg.Services.Keto.WriteURL,
	)

	// Initialize API server
	server := api.NewServer(embedder, vectorStore, ollama, permService)

	return vectorStore, server
}

func createHTTPServer(cfg *config.Config, server *api.Server) *http.Server {
	return &http.Server{
		Addr:         fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port),
		Handler:      server.GetHandler(),
		ReadTimeout:  time.Duration(cfg.Server.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(cfg.Server.WriteTimeout) * time.Second,
		TLSConfig:    cfg.GetTLSConfig(),
	}
}

func startHTTPServer(cfg *config.Config, httpServer *http.Server) {
	if cfg.Server.TLS.Enabled {
		log.Printf("Starting HTTPS server on %s", httpServer.Addr)
		log.Printf("TLS Cert: %s", cfg.Server.TLS.CertFile)
		log.Printf("TLS Key: %s", cfg.Server.TLS.KeyFile)
		log.Printf("Min TLS Version: %s", cfg.Server.TLS.MinTLS)

		go func() {
			if err := httpServer.ListenAndServeTLS(cfg.Server.TLS.CertFile, cfg.Server.TLS.KeyFile); err != nil && err != http.ErrServerClosed {
				log.Fatalf("Failed to start HTTPS server: %v", err)
			}
		}()
	} else {
		log.Printf("Starting HTTP server on %s", httpServer.Addr)
		if cfg.IsProduction() {
			log.Println("WARNING: Running HTTP in production. Consider enabling TLS.")
		}

		go func() {
			if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				log.Fatalf("Failed to start HTTP server: %v", err)
			}
		}()
	}
}

func waitForShutdown(server *api.Server) {
	// Wait for interrupt signal to gracefully shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Graceful shutdown with timeout
	shutdownTimeout := 30 * time.Second
	if err := server.Shutdown(shutdownTimeout); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	}

	log.Println("Server shutdown complete")
}
