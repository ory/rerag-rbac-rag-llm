package llm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"llm-rag-poc/internal/models"
	"net/http"
	"strings"
)

type OllamaClient struct {
	baseURL string
	model   string
}

func NewOllamaClient(baseURL, model string) *OllamaClient {
	return &OllamaClient{
		baseURL: baseURL,
		model:   model,
	}
}

func (o *OllamaClient) Generate(question string, context []models.Document) (string, error) {
	prompt := o.buildPrompt(question, context)

	reqBody := map[string]interface{}{
		"model":  o.model,
		"prompt": prompt,
		"stream": false,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	resp, err := http.Post(o.baseURL+"/api/generate", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var result struct {
		Response string `json:"response"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}

	return result.Response, nil
}

func (o *OllamaClient) buildPrompt(question string, documents []models.Document) string {
	var contextStr strings.Builder

	contextStr.WriteString("You are a helpful assistant that answers questions based on the provided tax return documents.\n\n")
	contextStr.WriteString("Context Documents:\n")

	for i, doc := range documents {
		contextStr.WriteString(fmt.Sprintf("\nDocument %d: %s\n", i+1, doc.Title))
		contextStr.WriteString(fmt.Sprintf("Content: %s\n", doc.Content))
		if doc.Metadata != nil && len(doc.Metadata) > 0 {
			contextStr.WriteString("Metadata: ")
			for k, v := range doc.Metadata {
				contextStr.WriteString(fmt.Sprintf("%s: %v, ", k, v))
			}
			contextStr.WriteString("\n")
		}
		contextStr.WriteString("---\n")
	}

	contextStr.WriteString(fmt.Sprintf("\nQuestion: %s\n", question))
	contextStr.WriteString("\nPlease answer the question based ONLY on the information provided in the context documents above. If the answer cannot be found in the documents, say so clearly.\n\nAnswer: ")

	return contextStr.String()
}
