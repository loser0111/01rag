package handler

import (
	"com.wyq.01rag/domain/model/document"
	"com.wyq.01rag/domain/model/embedding"
	"context"
	"log"
)

type ReadFileHandler struct {
}

func (h *ReadFileHandler) Handle(ctx context.Context, embeddingCtx embedding.EmbeddingContext) error {
	var chunks = make([]*document.Chunk, 0)
	fileReader := embeddingCtx.FileReader
	for _, fileName := range embeddingCtx.File {
		partChunks, err := fileReader.Read(fileName)
		if err != nil {
			log.Println(err)
			embeddingCtx.IsStop = true
			embeddingCtx.Error = err
			return err
		}
		chunks = append(chunks, partChunks...)
	}
	embeddingCtx.Chunks = chunks
	return nil
}
func (h *ReadFileHandler) Name() string {
	return "ReadFile"
}
