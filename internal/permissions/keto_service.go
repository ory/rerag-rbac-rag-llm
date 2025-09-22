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

type KetoPermissionService struct {
	readURL  string
	writeURL string
}

func NewKetoPermissionService(readURL, writeURL string) *KetoPermissionService {
	return &KetoPermissionService{
		readURL:  readURL,
		writeURL: writeURL,
	}
}

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

	resp, err := http.Get(fullURL)
	if err != nil {
		log.Printf("Error checking permission for user %s on object %s: %v", username, objectID, err)
		return false
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		var result struct {
			Allowed bool `json:"allowed"`
		}
		body, _ := io.ReadAll(resp.Body)
		json.Unmarshal(body, &result)
		return result.Allowed
	}

	log.Printf("Keto permission check returned status %d for user %s on object %s", resp.StatusCode, username, objectID)
	return false
}

func (k *KetoPermissionService) FilterDocuments(username string, docs []*models.Document) []*models.Document {
	filtered := make([]*models.Document, 0)
	for _, doc := range docs {
		if k.CanAccessDocument(username, doc) {
			filtered = append(filtered, doc)
		}
	}
	return filtered
}

func (k *KetoPermissionService) GetUserPermissions(username string) []string {
	// Build the list URL
	listURL := fmt.Sprintf("%s/relation-tuples", k.readURL)

	params := url.Values{}
	params.Add("namespace", "documents")
	params.Add("subject_id", username)

	fullURL := fmt.Sprintf("%s?%s", listURL, params.Encode())

	resp, err := http.Get(fullURL)
	if err != nil {
		log.Printf("Error getting permissions for user %s: %v", username, err)
		return []string{}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("Keto list relation tuples returned status %d for user %s", resp.StatusCode, username)
		return []string{}
	}

	var result struct {
		RelationTuples []struct {
			Object string `json:"object"`
		} `json:"relation_tuples"`
	}

	body, _ := io.ReadAll(resp.Body)
	json.Unmarshal(body, &result)

	permissions := make([]string, 0)
	for _, tuple := range result.RelationTuples {
		permissions = append(permissions, tuple.Object)
	}

	return permissions
}

func (k *KetoPermissionService) AddUserPermission(username string, taxpayer string) {
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
