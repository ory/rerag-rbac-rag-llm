package permissions

import (
	"encoding/json"
	"fmt"
	"io"
	"llm-rag-poc/internal/models"
	"log"
	"net/http"
	"net/url"
	"strings"
)

// KetoPermissionService implements permission checking using Ory Keto
type KetoPermissionService struct {
	readURL  string
	writeURL string
}

// NewKetoPermissionService creates a new Keto-based permission service
func NewKetoPermissionService(readURL, writeURL string) *KetoPermissionService {
	return &KetoPermissionService{
		readURL:  readURL,
		writeURL: writeURL,
	}
}

// CanAccessDocument checks if a user can access a specific document
func (k *KetoPermissionService) CanAccessDocument(username string, doc *models.Document) bool {
	// Map document to Keto object based on taxpayer and year
	objectID := k.documentToKetoObject(doc)
	if objectID == "" {
		log.Printf("Warning: Could not map document %s to Keto object", doc.Title)
		return false
	}

	// Build the check URL
	checkURL := fmt.Sprintf("%s/relation-tuples/check/openapi", k.readURL)

	// Create query parameters
	params := url.Values{}
	params.Add("namespace", "documents")
	params.Add("object", objectID)
	params.Add("relation", "viewer")
	params.Add("subject_id", username)

	fullURL := fmt.Sprintf("%s?%s", checkURL, params.Encode())

	// Validate URL before making request
	if _, err := url.Parse(fullURL); err != nil {
		log.Printf("Invalid URL for permission check: %v", err)
		return false
	}

	resp, err := http.Get(fullURL) // #nosec G107 - URL is validated above
	if err != nil {
		log.Printf("Error checking permission for user %s on object %s: %v", username, objectID, err)
		return false
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusOK {
		var result struct {
			Allowed bool `json:"allowed"`
		}
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Printf("Error reading response body: %v", err)
			return false
		}
		if err := json.Unmarshal(body, &result); err != nil {
			log.Printf("Error unmarshaling response: %v", err)
			return false
		}
		return result.Allowed
	}

	log.Printf("Keto permission check returned status %d for user %s on object %s", resp.StatusCode, username, objectID)
	return false
}

// FilterDocuments returns only documents the user has permission to access
func (k *KetoPermissionService) FilterDocuments(username string, docs []*models.Document) []*models.Document {
	filtered := make([]*models.Document, 0)
	for _, doc := range docs {
		if k.CanAccessDocument(username, doc) {
			filtered = append(filtered, doc)
		}
	}
	return filtered
}

// GetUserPermissions retrieves all permissions for a given user
func (k *KetoPermissionService) GetUserPermissions(username string) []string {
	// Build the list URL
	listURL := fmt.Sprintf("%s/relation-tuples", k.readURL)

	params := url.Values{}
	params.Add("namespace", "documents")
	params.Add("subject_id", username)

	fullURL := fmt.Sprintf("%s?%s", listURL, params.Encode())

	// Validate URL before making request
	if _, err := url.Parse(fullURL); err != nil {
		log.Printf("Invalid URL for listing permissions: %v", err)
		return []string{}
	}

	resp, err := http.Get(fullURL) // #nosec G107 - URL is validated above
	if err != nil {
		log.Printf("Error getting permissions for user %s: %v", username, err)
		return []string{}
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		log.Printf("Keto list relation tuples returned status %d for user %s", resp.StatusCode, username)
		return []string{}
	}

	var result struct {
		RelationTuples []struct {
			Object string `json:"object"`
		} `json:"relation_tuples"`
	}

	permissions := make([]string, 0)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Error reading response body: %v", err)
		return permissions
	}
	if err := json.Unmarshal(body, &result); err != nil {
		log.Printf("Error unmarshaling response: %v", err)
		return permissions
	}
	for _, tuple := range result.RelationTuples {
		permissions = append(permissions, tuple.Object)
	}

	return permissions
}

// AddUserPermission grants a user permission to access documents for a taxpayer
func (k *KetoPermissionService) AddUserPermission(_ string, _ string) {
	// Convert taxpayer to potential Keto objects and create relation tuples
	// This is a simplified implementation
	log.Printf("AddUserPermission not fully implemented for Keto service")
}

// documentToKetoObject maps a document to a Keto object identifier
func (k *KetoPermissionService) documentToKetoObject(doc *models.Document) string {
	if taxpayer, ok := doc.Metadata["taxpayer"].(string); ok {
		year := "unknown"
		if y, ok := doc.Metadata["year"].(float64); ok {
			year = fmt.Sprintf("%.0f", y)
		} else if y, ok := doc.Metadata["year"].(int); ok {
			year = fmt.Sprintf("%d", y)
		}

		// Normalize taxpayer name to kebab-case
		taxpayerKey := strings.ToLower(strings.ReplaceAll(taxpayer, " ", "-"))

		return fmt.Sprintf("%s:%s", taxpayerKey, year)
	}

	return ""
}
