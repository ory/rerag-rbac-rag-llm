package models

import "github.com/google/uuid"

type Document struct {
	ID       string    `json:"id"`
	Title    string    `json:"title"`
	Content  string    `json:"content"`
	Metadata map[string]interface{} `json:"metadata"`
	Embedding []float32 `json:"-"`
}

func NewDocument(title, content string, metadata map[string]interface{}) *Document {
	return &Document{
		ID:       uuid.New().String(),
		Title:    title,
		Content:  content,
		Metadata: metadata,
	}
}

type QueryRequest struct {
	Question string `json:"question" binding:"required"`
	TopK     int    `json:"top_k"`
}

type QueryResponse struct {
	Answer   string      `json:"answer"`
	Sources  []Document  `json:"sources"`
}