// Package config provides application configuration management using koanf
package config

import (
	"crypto/tls"
	"fmt"
	"log"
	"os"

	"github.com/knadh/koanf/parsers/json"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/env/v2"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
)

// Config holds all configuration for the application
type Config struct {
	// Server configuration
	Server ServerConfig `koanf:"server"`

	// Database configuration
	Database DatabaseConfig `koanf:"database"`

	// External services
	Services ServicesConfig `koanf:"services"`

	// Security settings
	Security SecurityConfig `koanf:"security"`

	// Application settings
	App AppConfig `koanf:"app"`
}

// ServerConfig holds HTTP server configuration
type ServerConfig struct {
	Host         string    `koanf:"host"`
	Port         int       `koanf:"port"`
	ReadTimeout  int       `koanf:"read_timeout"`  // seconds
	WriteTimeout int       `koanf:"write_timeout"` // seconds
	TLS          TLSConfig `koanf:"tls"`
}

// TLSConfig holds TLS/HTTPS configuration
type TLSConfig struct {
	Enabled  bool   `koanf:"enabled"`
	CertFile string `koanf:"cert_file"`
	KeyFile  string `koanf:"key_file"`
	MinTLS   string `koanf:"min_version"` // "1.2" or "1.3"
}

// DatabaseConfig holds database configuration
type DatabaseConfig struct {
	Path       string           `koanf:"path"`
	Encryption EncryptionConfig `koanf:"encryption"`
}

// EncryptionConfig holds database encryption settings
type EncryptionConfig struct {
	Enabled bool   `koanf:"enabled"`
	Key     string `koanf:"key"`
}

// ServicesConfig holds external service configuration
type ServicesConfig struct {
	Ollama OllamaConfig `koanf:"ollama"`
	Keto   KetoConfig   `koanf:"keto"`
}

// OllamaConfig holds Ollama service configuration
type OllamaConfig struct {
	BaseURL        string `koanf:"base_url"`
	EmbeddingModel string `koanf:"embedding_model"`
	LLMModel       string `koanf:"llm_model"`
	Timeout        int    `koanf:"timeout"` // seconds
}

// KetoConfig holds Ory Keto configuration
type KetoConfig struct {
	ReadURL  string `koanf:"read_url"`
	WriteURL string `koanf:"write_url"`
	Timeout  int    `koanf:"timeout"` // seconds
}

// SecurityConfig holds security-related settings
type SecurityConfig struct {
	AuthMode  string `koanf:"auth_mode"` // "mock" or "jwt"
	JWTSecret string `koanf:"jwt_secret"`
	ErrorMode string `koanf:"error_mode"` // "detailed" or "secure"
}

// AppConfig holds general application settings
type AppConfig struct {
	Environment string `koanf:"environment"` // "development", "staging", "production"
	LogLevel    string `koanf:"log_level"`   // "debug", "info", "warn", "error"
	LogFormat   string `koanf:"log_format"`  // "text" or "json"
}

// Load loads configuration from multiple sources with precedence:
// 1. config.yaml (if exists)
// 2. config.json (if exists)
// 3. Environment variables (highest precedence)
func Load() (*Config, error) {
	k := koanf.New(".")

	// Set defaults
	setDefaults(k)

	// Load from config files (optional)
	loadConfigFiles(k)

	// Load from environment variables (highest precedence)
	// Use simple prefix matching for now
	if err := k.Load(env.Provider(".", env.Opt{}), nil); err != nil {
		return nil, fmt.Errorf("error loading environment variables: %w", err)
	}

	// Unmarshal into config struct
	var cfg Config
	if err := k.Unmarshal("", &cfg); err != nil {
		return nil, fmt.Errorf("error unmarshaling config: %w", err)
	}

	// Validate configuration
	if err := validate(&cfg); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &cfg, nil
}

// setDefaults sets default configuration values
func setDefaults(k *koanf.Koanf) {
	defaults := map[string]interface{}{
		// Server defaults
		"server.host":            "localhost",
		"server.port":            8080,
		"server.read_timeout":    30,
		"server.write_timeout":   30,
		"server.tls.enabled":     false,
		"server.tls.min_version": "1.3",

		// Database defaults
		"database.path":               "vector_store.db",
		"database.encryption.enabled": false,

		// Services defaults
		"services.ollama.base_url":        "http://localhost:11434",
		"services.ollama.embedding_model": "nomic-embed-text",
		"services.ollama.llm_model":       "llama3",
		"services.ollama.timeout":         60,
		"services.keto.read_url":          "http://localhost:4466",
		"services.keto.write_url":         "http://localhost:4467",
		"services.keto.timeout":           10,

		// Security defaults
		"security.auth_mode":  "mock",
		"security.error_mode": "detailed",

		// App defaults
		"app.environment": "development",
		"app.log_level":   "info",
		"app.log_format":  "text",
	}

	for key, value := range defaults {
		_ = k.Set(key, value) // Ignore error for setting defaults
	}
}

// loadConfigFiles loads configuration from files
func loadConfigFiles(k *koanf.Koanf) {
	// Try to load YAML config
	if _, err := os.Stat("config.yaml"); err == nil {
		if err := k.Load(file.Provider("config.yaml"), yaml.Parser()); err != nil {
			log.Printf("Warning: failed to load config.yaml: %v", err)
		}
	}

	// Try to load JSON config
	if _, err := os.Stat("config.json"); err == nil {
		if err := k.Load(file.Provider("config.json"), json.Parser()); err != nil {
			log.Printf("Warning: failed to load config.json: %v", err)
		}
	}
}

// validate validates the configuration
func validate(cfg *Config) error {
	// Validate TLS configuration
	if cfg.Server.TLS.Enabled {
		if cfg.Server.TLS.CertFile == "" {
			return fmt.Errorf("TLS cert file is required when TLS is enabled")
		}
		if cfg.Server.TLS.KeyFile == "" {
			return fmt.Errorf("TLS key file is required when TLS is enabled")
		}

		// Check if files exist
		if _, err := os.Stat(cfg.Server.TLS.CertFile); os.IsNotExist(err) {
			return fmt.Errorf("TLS cert file does not exist: %s", cfg.Server.TLS.CertFile)
		}
		if _, err := os.Stat(cfg.Server.TLS.KeyFile); os.IsNotExist(err) {
			return fmt.Errorf("TLS key file does not exist: %s", cfg.Server.TLS.KeyFile)
		}
	}

	// Validate database encryption
	if cfg.Database.Encryption.Enabled && cfg.Database.Encryption.Key == "" {
		return fmt.Errorf("database encryption key is required when encryption is enabled")
	}

	// Validate security settings
	if cfg.Security.AuthMode == "jwt" && cfg.Security.JWTSecret == "" {
		return fmt.Errorf("JWT secret is required when auth mode is jwt")
	}

	return nil
}

// GetTLSConfig returns a TLS configuration based on the config
func (c *Config) GetTLSConfig() *tls.Config {
	if !c.Server.TLS.Enabled {
		return nil
	}

	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS12, // Set default minimum version
		CipherSuites: []uint16{
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
		},
	}

	// Set minimum TLS version
	switch c.Server.TLS.MinTLS {
	case "1.2":
		tlsConfig.MinVersion = tls.VersionTLS12
	case "1.3":
		tlsConfig.MinVersion = tls.VersionTLS13
	default:
		tlsConfig.MinVersion = tls.VersionTLS13
	}

	return tlsConfig
}

// GetDatabaseDSN returns the database connection string with encryption if enabled
func (c *Config) GetDatabaseDSN() string {
	if c.Database.Encryption.Enabled {
		// SQLCipher format
		return fmt.Sprintf("%s?_pragma_key=%s&_pragma_cipher_page_size=4096",
			c.Database.Path, c.Database.Encryption.Key)
	}
	return c.Database.Path
}

// IsProduction returns true if running in production environment
func (c *Config) IsProduction() bool {
	return c.App.Environment == "production"
}

// IsDevelopment returns true if running in development environment
func (c *Config) IsDevelopment() bool {
	return c.App.Environment == "development"
}
