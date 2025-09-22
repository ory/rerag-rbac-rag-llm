package models

import "github.com/google/uuid"

type Document struct {
	ID        uuid.UUID              `json:"id"`
	Title     string                 `json:"title"`
	Content   string                 `json:"content"`
	Metadata  map[string]interface{} `json:"metadata"`
	Embedding []float32              `json:"-"`
}

type QueryRequest struct {
	Question string `json:"question" binding:"required"`
	TopK     int    `json:"top_k"`
}

// QueryResponse represents the response from a document query
// swagger:model QueryResponse
type QueryResponse struct {
	// The generated answer based on the query and accessible documents
	// required: true
	Answer string `json:"answer"`

	// The source documents used to generate the answer
	// required: true
	Sources []Document `json:"sources"`
}

// DocumentResponse represents the response when a document is successfully added
// swagger:model DocumentResponse
type DocumentResponse struct {
	// The unique identifier of the added document
	// required: true
	ID string `json:"id"`

	// Success message
	// required: true
	Message string `json:"message"`
}

// DocumentListResponse represents the response when listing documents
// swagger:model DocumentListResponse
type DocumentListResponse struct {
	// List of documents accessible to the user
	// required: true
	Documents []Document `json:"documents"`

	// Total count of accessible documents
	// required: true
	Count int `json:"count"`

	// The authenticated user
	// required: true
	User string `json:"user"`
}

// PermissionsResponse represents the user's permissions
// swagger:model PermissionsResponse
type PermissionsResponse struct {
	// The authenticated user
	// required: true
	User string `json:"user"`

	// List of permissions granted to the user
	// required: true
	Permissions []string `json:"permissions"`
}

// HealthResponse represents the health check response
// swagger:model HealthResponse
type HealthResponse struct {
	// Service status
	// required: true
	Status string `json:"status"`
}

// ErrorResponse represents an API error response
// swagger:model ErrorResponse
type ErrorResponse struct {
	// Error message
	// required: true
	Error string `json:"error"`
}
