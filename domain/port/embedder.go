package port

import (
	"context"

	"com.wyq.01rag/domain/model/document"
	"com.wyq.01rag/domain/model/embedding"
)

// SourceReader 源文件读取器：把一个 SourceFile 解析成若干 Chunk（纯输入端口，不关心 md/pdf/txt 实现）
type SourceReader interface {
	Read(file *document.File) ([]*document.Chunk, error)
}

// Embedder embedding 模型端口：输入一批 Chunk，输出一一对应的 EmbeddingVector
type Embedder interface {
	Embed(ctx context.Context, chunks []*document.Chunk) ([]*embedding.EmbeddingVector, error)
	EmbedOne(ctx context.Context, chunk *document.Chunk) (*embedding.EmbeddingVector, error)
	// ModelName 返回此 Embedder 对应的模型名（用于配置/日志区分）
	ModelName() string
}

// EmbedderRegistry 注册/查找 Embedder（通常由 Bootstrap 组装，避免 Application 层依赖 infra 单例）
type EmbedderRegistry interface {
	FindByName(name string) (Embedder, error)
}
