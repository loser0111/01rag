package embedding

import (
	"com.wyq.01rag/domain/model/document"
	"com.wyq.01rag/domain/model/store"
)

// 嵌入的模型设计

// 嵌入向量

type EmbeddingVector struct {
	Vector []float64
	Chunk  *document.Chunk
	ID     string
}

// 嵌入模型设置

type EmbeddingModelConfig struct {
}

// 执行嵌入流程的上下文

type EmbeddingContext struct {

	// 生成的文档名称数据

	DocumentName string

	// 需要打开的文件数据

	File []*document.File

	// 读取文件的数据
	FileReader document.IFileReader

	// Chunks
	Chunks []*document.Chunk

	// 嵌入式模型名称
	EmbeddingModelName string

	// 生成的嵌入式向量
	Vectors []*EmbeddingVector

	// 保存到chromaDB当中的葫记录数量
	Records []*store.VectorRecord

	// 输入参数
	StoreType string

	// 存储数据类型
	store.VectorStore

	IsStop bool

	Error error
}

func (ctx *EmbeddingContext) Stop(err error) {
	ctx.IsStop = true
	ctx.Error = err
}
