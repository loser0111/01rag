package embedding

import (
	"com.wyq.01rag/domain/model/document"
	"com.wyq.01rag/domain/model/embedding"
)

// Embedding 【Legacy 接口】
// 保留以兼容旧代码 domain/service/embedding/handler/* 与 test/test_embedding.go。
// 新代码/DDD 请使用 domain/port.Embedder。
type Embedding interface {
	Embedding([]*document.Chunk) ([]*embedding.EmbeddingVector, error)
	EmbeddingOneChunk(chunk *document.Chunk) (*embedding.EmbeddingVector, error)
}
