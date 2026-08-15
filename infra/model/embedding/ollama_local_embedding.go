package embedding

import (
	"bytes"
	"com.wyq.01rag/domain/model/document"
	"com.wyq.01rag/domain/model/embedding"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type OllamaEmbedding struct {
	ModelName string
	Url       string
}

type OllamaEmbeddingReqeust struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
}

type OllamaEmbeddingResponse struct {
	Embedding []float64 `json:"embedding"`
}

func (e *OllamaEmbedding) Embedding(chunks []*document.Chunk) ([]*embedding.EmbeddingVector, error) {

	// url := "http://127.0.0.1:11434/api/embeddings"
	vectors := make([]*embedding.EmbeddingVector, 0)
	for _, chunk := range chunks {
		vector, err := e.EmbeddingOneChunk(chunk)
		if err != nil {
			return nil, err
		}
		vectors = append(vectors, vector)
	}
	return vectors, nil
}

func (e *OllamaEmbedding) EmbeddingOneChunk(chunk *document.Chunk) (*embedding.EmbeddingVector, error) {
	request := &OllamaEmbeddingReqeust{
		Model:  e.ModelName,
		Prompt: chunk.Data,
	}
	reqJson, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}

	fmt.Println(string(reqJson))

	resp, err := http.Post(e.Url, "application/json", bytes.NewBuffer(reqJson))
	if err != nil {
		return nil, fmt.Errorf("http post failed: %w", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ollama api error, status:%d, body:%s", resp.StatusCode, string(body))
	}

	var result OllamaEmbeddingResponse
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return nil, fmt.Errorf("parse response failed: %w", err)
	}
	s, _ := json.Marshal(result)
	fmt.Println(string(s))
	return &embedding.EmbeddingVector{
		Vector: result.Embedding,
		Chunk:  chunk,
	}, nil
}
