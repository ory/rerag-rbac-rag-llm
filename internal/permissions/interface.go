package permissions

import "llm-rag-poc/internal/models"

// PermissionChecker defines the interface for checking document access permissions
type PermissionChecker interface {
	CanAccessDocument(username string, doc *models.Document) bool
	FilterDocuments(username string, docs []*models.Document) []*models.Document
	GetUserPermissions(username string) []string
	AddUserPermission(username string, taxpayer string)
}
