// Package permissions provides interfaces and implementations for access control.
package permissions

import (
	"rerag-rbac-rag-llm/internal/models"
)

// PermissionChecker defines the interface for checking document access permissions
type PermissionChecker interface {
	CanAccessDocument(username string, doc *models.Document) bool
	GetUserPermissions(username string) []string
}
