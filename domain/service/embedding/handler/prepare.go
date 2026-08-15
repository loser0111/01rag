package handler

import (
	"com.wyq.01rag/domain/model/document"
	"com.wyq.01rag/domain/model/embedding"
)

// 初始化上下文

func BuildEmbeddingContext(
	DocumentName string,
	File []*document.File,
	FileReader document.IFileReader,
	EmbeddingModelName string,
	StoreType string,
) (*embedding.EmbeddingContext, error) {
	return &embedding.EmbeddingContext{
		DocumentName:       DocumentName,
		FileReader:         FileReader,
		File:               File,
		EmbeddingModelName: EmbeddingModelName,
		StoreType:          StoreType,
	}, nil
}
