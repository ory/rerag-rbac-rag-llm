package permissions

import (
	"llm-rag-poc/internal/models"
	"strings"
)

// PermissionService manages user permissions for document access
type PermissionService struct {
	userPermissions map[string][]string
}

// CanAccessDocument checks if a user can access a specific document
func (ps *PermissionService) CanAccessDocument(username string, doc *models.Document) bool {
	permissions, exists := ps.userPermissions[strings.ToLower(username)]
	if !exists {
		return false
	}

	for _, perm := range permissions {
		if perm == "*" {
			return true
		}

		if taxpayer, ok := doc.Metadata["taxpayer"].(string); ok {
			if strings.EqualFold(taxpayer, perm) {
				return true
			}
		}
	}

	return false
}

// FilterDocuments returns only documents the user has permission to access
func (ps *PermissionService) FilterDocuments(username string, docs []*models.Document) []*models.Document {
	filtered := make([]*models.Document, 0)
	for _, doc := range docs {
		if ps.CanAccessDocument(username, doc) {
			filtered = append(filtered, doc)
		}
	}
	return filtered
}

// GetUserPermissions retrieves all permissions for a given user
func (ps *PermissionService) GetUserPermissions(username string) []string {
	if perms, exists := ps.userPermissions[strings.ToLower(username)]; exists {
		return perms
	}
	return []string{}
}

// AddUserPermission grants a user permission to access documents for a taxpayer
func (ps *PermissionService) AddUserPermission(username string, taxpayer string) {
	username = strings.ToLower(username)
	if _, exists := ps.userPermissions[username]; !exists {
		ps.userPermissions[username] = []string{}
	}
	ps.userPermissions[username] = append(ps.userPermissions[username], taxpayer)
}
