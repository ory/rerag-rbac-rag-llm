package embeddings

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type Embedder struct {
	ollamaURL string
	model     string
}

func NewEmbedder() *Embedder {
	return &Embedder{
		ollamaURL: "http://localhost:11434",
		model:     "nomic-embed-text",
	}
}

func (e *Embedder) GetEmbedding(text string) ([]float32, error) {
	reqBody := map[string]interface{}{
		"model":  e.model,
		"prompt": text,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	resp, err := http.Post(e.ollamaURL+"/api/embeddings", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result struct {
		Embedding []float32 `json:"embedding"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	if len(result.Embedding) == 0 {
		return nil, fmt.Errorf("no embedding returned")
	}

	return result.Embedding, nil
}
