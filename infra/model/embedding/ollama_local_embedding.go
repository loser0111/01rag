package embedding

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"com.wyq.01rag/domain/model/document"
	"com.wyq.01rag/domain/model/embedding"
)

// OllamaEmbedding 通过 Ollama 本地 HTTP 接口算 embedding。
// 实现 port.Embedder 接口（见 domain/port/embedder.go）
type OllamaEmbedding struct {
	// modelName 保留在内部字段；对外通过 ModelName() 方法满足 port.Embedder 接口
	modelName string
	Url       string
}

func (e *OllamaEmbedding) ModelName() string { return e.modelName }

type OllamaEmbeddingRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
}

type OllamaEmbeddingResponse struct {
	Embedding []float64 `json:"embedding"`
}

// Embed 批量向量化；Ollama /api/embeddings 接口单条，这里串行调用
func (e *OllamaEmbedding) Embed(ctx context.Context, chunks []*document.Chunk) ([]*embedding.EmbeddingVector, error) {
	vectors := make([]*embedding.EmbeddingVector, 0, len(chunks))
	for _, chunk := range chunks {
		v, err := e.EmbedOne(ctx, chunk)
		if err != nil {
			return nil, err
		}
		vectors = append(vectors, v)
	}
	return vectors, nil
}

// EmbedOne 算单个 chunk 的向量
func (e *OllamaEmbedding) EmbedOne(ctx context.Context, chunk *document.Chunk) (*embedding.EmbeddingVector, error) {
	request := &OllamaEmbeddingRequest{
		Model:  e.modelName,
		Prompt: chunk.Data,
	}
	reqJson, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, e.Url, bytes.NewReader(reqJson))
	if err != nil {
		return nil, fmt.Errorf("create ollama req: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http post ollama: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ollama api error, status:%d, body:%s", resp.StatusCode, string(body))
	}

	var result OllamaEmbeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("parse ollama response: %w", err)
	}
	return &embedding.EmbeddingVector{
		Vector: result.Embedding,
		Chunk:  chunk,
	}, nil
}

// ================ 兼容旧 API（给 legacy handler / test 调用） ================
// 新代码请使用带 ctx 的 Embed / EmbedOne。

func (e *OllamaEmbedding) Embedding(chunks []*document.Chunk) ([]*embedding.EmbeddingVector, error) {
	return e.Embed(context.Background(), chunks)
}

func (e *OllamaEmbedding) EmbeddingOneChunk(chunk *document.Chunk) (*embedding.EmbeddingVector, error) {
	return e.EmbedOne(context.Background(), chunk)
}
