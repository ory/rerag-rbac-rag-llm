package permissions

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"rerag-rbac-rag-llm/internal/models"

	"github.com/google/uuid"
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
	return k.canAccessDocumentByID(username, doc.ID)
}

// canAccessDocumentByID checks if a user can access a document by its ID
func (k *KetoPermissionService) canAccessDocumentByID(username string, docID uuid.UUID) bool {
	// Build the check URL
	checkURL := fmt.Sprintf("%s/relation-tuples/check/openapi", k.readURL)

	// Create query parameters using document ID as the object
	params := url.Values{}
	params.Add("namespace", "documents")
	params.Add("object", docID.String())
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
		log.Printf("Error checking permission for user %s on document %s: %v", username, docID, err)
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

	log.Printf("Keto permission check returned status %d for user %s on document %s", resp.StatusCode, username, docID)
	return false
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
