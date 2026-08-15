package test

import (
	"com.wyq.01rag/domain/model/document"
	embedding "com.wyq.01rag/infra/model/embedding"
	"fmt"
	"testing"
)

func TestEmbedding(t *testing.T) {
	embeddingModel := embedding.EmbeddingManagerApp.FindByName("qwen3-embedding:0.6b")
	chunks := []*document.Chunk{
		&document.Chunk{
			Data: "这是一段文字",
		},
	}
	fmt.Printf("%#v, %#v", chunks, embeddingModel)

	vectors, er := embeddingModel.Embedding(chunks)
	if er != nil {
		fmt.Println(er)
	}
	fmt.Printf("this is vector: %#v\n", vectors)
}
