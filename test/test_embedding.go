package test

import (
	"com.wyq.01rag/domain/model/document"
	"com.wyq.01rag/domain/model/store"
	embedding "com.wyq.01rag/infra/model/embedding"
	"com.wyq.01rag/infra/storage/chroma"
	"context"
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

func TestChromaStorage(t *testing.T) {
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
	fmt.Println(len(vectors[0].Vector))
	documentName := "test"
	err := chroma.ChromaVectorStorageApp.CreateCollection(context.Background(), documentName, len(vectors[0].Vector))
	if err != nil {
		fmt.Println("fail to crateCollection: " + err.Error())
		return
	}

	// 数据转化成float32试试
	F32 := make([]float32, len(vectors[0].Vector))
	for i, f := range vectors[0].Vector {
		F32[i] = float32(f)
	}

	VectorRecord := store.VectorRecord{
		ID:       "id1",
		Vector:   F32,
		Text:     vectors[0].Chunk.Data,
		MetaData: make(map[string]interface{}),
	}

	err = chroma.ChromaVectorStorageApp.AddDocuments(context.Background(), documentName, []*store.VectorRecord{
		&VectorRecord,
	})

	fmt.Println(err)
}
