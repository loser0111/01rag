package embedding

import (
	"com.wyq.01rag/domain/model/document"
	"com.wyq.01rag/domain/model/store"
)

// EmbeddingVector 嵌入向量：每一条 chunk 与其 embedding 结果的绑定关系
type EmbeddingVector struct {
	Vector []float64
	Chunk  *document.Chunk
	ID     string
}

// EmbeddingModelConfig 【Legacy 占位】 具体配置迁移到 internal/config.AppConfig。
// 保留此类型仅为兼容外部可能的引用。
type EmbeddingModelConfig struct {
}

// EmbeddingContext 【Legacy】旧 pipeline 传递的上下文。
//
// ⚠️ DDD 新路径（application/ingest / application/query）不再使用该上下文。
// 新代码请直接调用 IngestUseCase / QueryUseCase，每个步骤间通过显式参数传递数据。
type EmbeddingContext struct {

	// 生成的文档名称数据
	DocumentName string

	// 需要打开的文件数据
	File []*document.File

	// 【Deprecated】读取文件的组件；DDD 新路径通过 IngestInput.SourceReader 注入。
	// 旧 handler 为兼容仍读取此字段，新代码不要依赖。
	FileReader document.IFileReader

	// Chunks
	Chunks []*document.Chunk

	// 嵌入式模型名称
	EmbeddingModelName string

	// 生成的嵌入式向量
	Vectors []*EmbeddingVector

	// 保存到向量库当中的记录
	Records []*store.VectorRecord

	// 输入参数
	StoreType string

	// 【Deprecated】向量存储组件；DDD 新路径通过 VectorStoreRegistry + use case 注入。
	// 旧 StoreVectorHandler 仍会设置此字段。
	VectorStore store.VectorStore

	IsStop bool

	Error error
}

func (ctx *EmbeddingContext) Stop(err error) {
	ctx.IsStop = true
	ctx.Error = err
}
