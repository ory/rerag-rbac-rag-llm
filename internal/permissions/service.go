package permissions

import (
	"llm-rag-poc/internal/models"
	"strings"
)

type PermissionService struct {
	userPermissions map[string][]string
}

func NewPermissionService() *PermissionService {
	return &PermissionService{
		userPermissions: map[string][]string{
			"alice": {"John Doe"},
			"bob":   {"ABC Corporation"},
			"peter": {"*"},
		},
	}
}

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

func (ps *PermissionService) FilterDocuments(username string, docs []*models.Document) []*models.Document {
	filtered := make([]*models.Document, 0)
	for _, doc := range docs {
		if ps.CanAccessDocument(username, doc) {
			filtered = append(filtered, doc)
		}
	}
	return filtered
}

func (ps *PermissionService) GetUserPermissions(username string) []string {
	if perms, exists := ps.userPermissions[strings.ToLower(username)]; exists {
		return perms
	}
	return []string{}
}

func (ps *PermissionService) AddUserPermission(username string, taxpayer string) {
	username = strings.ToLower(username)
	if _, exists := ps.userPermissions[username]; !exists {
		ps.userPermissions[username] = []string{}
	}
	ps.userPermissions[username] = append(ps.userPermissions[username], taxpayer)
}