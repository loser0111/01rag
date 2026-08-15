package embedding

import (
	"com.wyq.01rag/domain/model/document"
	"com.wyq.01rag/domain/model/embedding"
)

// Embedding

type Embedding interface {
	Embedding([]*document.Chunk) ([]*embedding.EmbeddingVector, error)
	EmbeddingOneChunk(chunk *document.Chunk) (*embedding.EmbeddingVector, error)
}
