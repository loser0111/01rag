package handler

import (
	"com.wyq.01rag/domain/model/document"
	"com.wyq.01rag/domain/model/embedding"
)

// 初始化上下文

func BuildEmbeddingContext(File []*document.File, FileReader document.IFileReader, EmbeddingModelName string) (embedding.EmbeddingContext, error) {
	return embedding.EmbeddingContext{
		FileReader:         FileReader,
		File:               File,
		EmbeddingModelName: EmbeddingModelName,
	}, nil
}
