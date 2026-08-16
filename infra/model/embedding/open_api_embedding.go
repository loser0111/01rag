package embedding

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"com.wyq.01rag/domain/model/document"
	"com.wyq.01rag/domain/model/embedding"
)

// OpenApiEmbedding 通过 OpenAI 兼容的 HTTP 接口批量算 embedding。
// 实现 port.Embedder 接口（见 domain/port/embedder.go）
type OpenApiEmbedding struct {
	// modelName 保留在内部字段；对外通过 ModelName() 方法满足 port.Embedder 接口
	modelName string
	Url       string
	ApiKey    string
}

func (e *OpenApiEmbedding) ModelName() string { return e.modelName }

type OpenApiEmbeddingRequest struct {
	Model string   `json:"model"`
	Input []string `json:"input"`
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

func (e *OpenApiEmbedding) Embed(ctx context.Context, chunks []*document.Chunk) ([]*embedding.EmbeddingVector, error) {
	strData := make([]string, len(chunks))
	for i, val := range chunks {
		strData[i] = val.Data
	}

	request := &OpenApiEmbeddingRequest{
		Model: e.modelName,
		Input: strData,
	}
	body, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, e.Url, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+e.ApiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
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

	vectors := make([]*embedding.EmbeddingVector, len(chunks))
	for _, d := range result.Data {
		idx := d.Index
		if idx < 0 || idx >= len(chunks) {
			return nil, fmt.Errorf("embedding index out of range: index=%d chunks=%d", idx, len(chunks))
		}
		vectors[idx] = &embedding.EmbeddingVector{
			Vector: d.Embedding,
			Chunk:  chunks[idx],
		}
	}
	for i, v := range vectors {
		if v == nil {
			return nil, fmt.Errorf("embedding missing for chunk[%d]", i)
		}
	}
	return vectors, nil
}

func (e *OpenApiEmbedding) EmbedOne(ctx context.Context, chunk *document.Chunk) (*embedding.EmbeddingVector, error) {
	vectors, err := e.Embed(ctx, []*document.Chunk{chunk})
	if err != nil {
		return nil, err
	}
	if len(vectors) == 0 {
		return nil, fmt.Errorf("fail to EmbedOne")
	}
	return vectors[0], nil
}

// ================ 兼容旧 API（给 legacy handler / test 调用） ================
// 新代码请使用带 ctx 的 Embed / EmbedOne。

func (e *OpenApiEmbedding) Embedding(chunks []*document.Chunk) ([]*embedding.EmbeddingVector, error) {
	return e.Embed(context.Background(), chunks)
}

func (e *OpenApiEmbedding) EmbeddingOneChunk(chunk *document.Chunk) (*embedding.EmbeddingVector, error) {
	return e.EmbedOne(context.Background(), chunk)
}
