package embedding

import (
	"bytes"
	"com.wyq.01rag/domain/model/document"
	"com.wyq.01rag/domain/model/embedding"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

type OpenApiEmbedding struct {
	ModelName string
	Url       string
	ApiKey    string
}

type OpenApiEmbeddingRequest struct {
	Model string   `json:"model"`
	Input []string `json:"input"` // 支持批量文本
}

type OpenApiEmbeddingData struct {
	Embedding []float64 `json:"embedding"`
	Index     int       `json:"index"`
	Object    string    `json:"object"`
}

type OpenApiEmbeddingResp struct {
	Object string                 `json:"object"`
	Data   []OpenApiEmbeddingData `json:"data"`
	Model  string                 `json:"model"`
	Usage  struct {
		PromptTokens int `json:"prompt_tokens"`
		TotalTokens  int `json:"total_tokens"`
	} `json:"usage"`
}

func (e *OpenApiEmbedding) Embedding(chunks []*document.Chunk) ([]*embedding.EmbeddingVector, error) {
	return e.EmbeddingBatchChunk(chunks)
}

func (e *OpenApiEmbedding) EmbeddingBatchChunk(chunks []*document.Chunk) ([]*embedding.EmbeddingVector, error) {

	strData := make([]string, len(chunks))
	for i, val := range chunks {
		strData[i] = val.Data
	}

	request := &OpenApiEmbeddingRequest{
		Model: e.ModelName,
		Input: strData,
	}
	body, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodPost, e.Url, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+e.ApiKey)
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("http status %d, resp:%s", resp.StatusCode, string(respBytes))
	}

	var result OpenApiEmbeddingResp
	if err := json.Unmarshal(respBytes, &result); err != nil {
		return nil, err
	}
	vectors := make([]*embedding.EmbeddingVector, len(result.Data))
	for i, d := range result.Data {
		vectors[i] = &embedding.EmbeddingVector{
			Vector: d.Embedding,
			Chunk: &document.Chunk{
				Data: d.Object,
			},
		}
	}
	return vectors, nil
}

func (e *OpenApiEmbedding) EmbeddingOneChunk(chunk *document.Chunk) (*embedding.EmbeddingVector, error) {
	vectors, err := e.EmbeddingBatchChunk([]*document.Chunk{chunk})
	if err != nil {
		log.Println("fail to EmbeddingOneChunk: " + err.Error())
	}
	if len(vectors) <= 0 {
		return nil, fmt.Errorf("fail to EmbeddingOneChunk")
	}
	return vectors[0], nil
}
