package test

import (
	"com.wyq.01rag/domain/model/document"
	"com.wyq.01rag/domain/service/embedding/handler"
	"com.wyq.01rag/infra/storage/file_reader"
	"context"
	"log"
	"testing"
)

func TestEmbeddingFlow(t *testing.T) {
	var (
		ctx                   = context.Background()
		files                 = []*document.File{{"wangyongqing.md", "E:/learn/01agent/RAG实现方案调研报告.md"}}
		readFileHandler       = &handler.ReadFileHandler{}
		modelEmbeddingHandler = &handler.ModelEmbeddingHandler{}
		storeVectorHandler    = &handler.StoreVectorHandler{}
	)

	ebdCtx, err := handler.BuildEmbeddingContext("test", files, file_reader.NewMarkdownFileReader(1024), "qwen3-embedding:0.6b", "chroma")
	if err != nil {
		log.Fatal(err)
	}
	err = readFileHandler.Handle(ctx, ebdCtx)
	if err != nil {
		log.Fatal(err)
	}
	err = modelEmbeddingHandler.Handle(ctx, ebdCtx)
	if err != nil {
		log.Fatal(err)
	}
	err = storeVectorHandler.Handle(ctx, ebdCtx)
	if err != nil {
		log.Fatal(err)
	}
}
