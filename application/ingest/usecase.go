// Package ingest 【Application 层 - 知识入库用例】
//
// 职责：编排"读文件 → 向量化 → 入库"的步骤流。
// 本包只依赖 domain 层接口（domain/port），不 import infra。
// 真实的 md 解析/HTTP embedding/chroma 调用均由 infra 通过构造参数注入进来。
package ingest

import (
	"context"
	"errors"
	"fmt"

	"com.wyq.01rag/domain/model/document"
	"com.wyq.01rag/domain/model/embedding"
	"com.wyq.01rag/domain/model/store"
	"com.wyq.01rag/domain/port"
	"com.wyq.01rag/infra/utils"
)

// ========== UseCase 入参 / 出参 DTO（Application 层，只属于本用例） ==========

// IngestInput 一次文档入库的入参
type IngestInput struct {
	DocumentName  string            // 集合/知识库名字（对应 collection name）
	Files         []*document.File  // 要读的文件
	SourceReader  port.SourceReader // 文件解析器（调用方根据扩展名选择 md/txt/pdf）
	EmbedderName  string            // 用哪个 embedding 模型
	VectorStore   string            // 用哪个向量存储实现 ("chroma" / ...)
	MetadataPatch map[string]any    // 每条 chunk 额外附加的 metadata（如 source/publisher）
}

// IngestOutput 入库结果
type IngestOutput struct {
	DocumentName string
	ChunksCount  int
	VectorsCount int
	RecordsCount int
	VectorDim    int
}

// ========== UseCase ==========

// UseCase 知识入库用例
// 字段都是 domain/port 接口 → 可以被任意 mock，单测不需要真的 chroma/ollama
type UseCase struct {
	embedders port.EmbedderRegistry
	stores    port.VectorStoreRegistry
}

func NewIngestUseCase(
	embedders port.EmbedderRegistry,
	stores port.VectorStoreRegistry,
) *UseCase {
	return &UseCase{embedders: embedders, stores: stores}
}

// Execute 执行：读文件 → 切 chunk → 向量化 → 写入向量库
// 依赖方向：Use Case -> Domain Port 接口；外部由 bootstrap 装配真实实现。
func (uc *UseCase) Execute(ctx context.Context, input IngestInput) (*IngestOutput, error) {
	if err := validateInput(input); err != nil {
		return nil, err
	}

	// 步骤1：读文件 + 切 chunk
	chunks, err := stepReadFiles(input.Files, input.SourceReader)
	if err != nil {
		return nil, fmt.Errorf("read files: %w", err)
	}
	if len(chunks) == 0 {
		return nil, errors.New("no valid chunk generated from source files")
	}

	// 步骤2：embedding
	embed, err := uc.embedders.FindByName(input.EmbedderName)
	if err != nil {
		return nil, fmt.Errorf("resolve embedder %q: %w", input.EmbedderName, err)
	}
	vectors, err := stepEmbed(ctx, chunks, embed)
	if err != nil {
		return nil, fmt.Errorf("embedding: %w", err)
	}

	// 步骤3：把 vectors 转成 records（metadata 合并 + 类型转换）
	records := stepBuildRecords(vectors, input.DocumentName, input.MetadataPatch)

	// 步骤4：写入向量库
	writer, err := uc.stores.GetWriter(input.VectorStore)
	if err != nil {
		return nil, fmt.Errorf("resolve vector store %q: %w", input.VectorStore, err)
	}
	if err := writer.AddDocuments(ctx, input.DocumentName, records); err != nil {
		return nil, fmt.Errorf("add documents: %w", err)
	}

	dim := 0
	if len(vectors) > 0 {
		dim = len(vectors[0].Vector)
	}
	return &IngestOutput{
		DocumentName: input.DocumentName,
		ChunksCount:  len(chunks),
		VectorsCount: len(vectors),
		RecordsCount: len(records),
		VectorDim:    dim,
	}, nil
}

func validateInput(i IngestInput) error {
	if i.DocumentName == "" {
		return errors.New("documentName is required")
	}
	if len(i.Files) == 0 {
		return errors.New("files is empty")
	}
	if i.SourceReader == nil {
		return errors.New("sourceReader is required")
	}
	if i.EmbedderName == "" {
		return errors.New("embedderName is required")
	}
	if i.VectorStore == "" {
		return errors.New("vectorStore is required")
	}
	return nil
}

// ========== 步骤实现（内部函数，纯流程编排） ==========

// stepReadFiles 依次解析每个文件，返回 flatten 的 chunk 列表
func stepReadFiles(files []*document.File, reader port.SourceReader) ([]*document.Chunk, error) {
	out := make([]*document.Chunk, 0, 64)
	for _, f := range files {
		chunks, err := reader.Read(f)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", f.FileName, err)
		}
		out = append(out, chunks...)
	}
	return out, nil
}

// stepEmbed 向量化
func stepEmbed(ctx context.Context, chunks []*document.Chunk, emb port.Embedder) ([]*embedding.EmbeddingVector, error) {
	return emb.Embed(ctx, chunks)
}

// stepBuildRecords 把 embedding 结果转换为写入向量库的记录
func stepBuildRecords(
	vectors []*embedding.EmbeddingVector,
	documentName string,
	extraMeta map[string]any,
) []*store.VectorRecord {
	records := make([]*store.VectorRecord, len(vectors))
	for i, v := range vectors {
		meta := make(map[string]any, len(extraMeta)+1)
		for k, val := range extraMeta {
			meta[k] = val
		}
		meta["name"] = documentName
		chunkData := ""
		if v.Chunk != nil {
			chunkData = v.Chunk.Data
		}
		records[i] = &store.VectorRecord{
			Vector:   utils.ConvertFloat64ToFloat32(v.Vector),
			Text:     chunkData,
			MetaData: meta,
		}
	}
	return records
}
