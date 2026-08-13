package embedding

import (
	"com.wyq.01rag/domain/model/document"
)

// 嵌入的模型设计
type EmbeddingVector struct {
	Verctor []float64
	Chunk   *document.Chunk
	ID      string
}
type EmbeddingContext struct {
	// 需要打开的文件数据
	File []*document.File

	// 读取文件的数据
	FileReader *document.IFileReader

	// Chunks
	Chunks []*document.Chunk

	// 生成的嵌入式向量
	Vectors []*EmbeddingVector

	// 嵌入式模型名称
	EmbeddingModelName string
}
