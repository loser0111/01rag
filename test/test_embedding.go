package test

import (
	"context"
	"fmt"
	"strconv"
	"testing"

	"com.wyq.01rag/domain/model/document"
	"com.wyq.01rag/domain/model/store"
	embedding "com.wyq.01rag/infra/model/embedding"
	"com.wyq.01rag/infra/storage/chroma"
)

func TestEmbedding(t *testing.T) {
	embeddingModel := embedding.EmbeddingManagerApp.LegacyFindByName("qwen3-embedding:0.6b")
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
	embeddingModel := embedding.EmbeddingManagerApp.LegacyFindByName("qwen3-embedding:0.6b")
	chunks := []*document.Chunk{
		&document.Chunk{
			Data: "我的名字叫王永庆",
		},
		&document.Chunk{
			Data: "我的邮箱是zhuwe0111@163.com",
		},
		&document.Chunk{
			Data: "我的QQ账号是2907354711",
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
	//err := chroma.ChromaVectorStorageApp.CreateCollection(context.Background(), documentName, len(vectors[0].Vector))
	//if err != nil {
	//	fmt.Println("fail to crateCollection: " + err.Error())
	//	return
	//}

	records := make([]*store.VectorRecord, 0)
	for i := 0; i < len(vectors); i++ {
		records = append(records, &store.VectorRecord{
			ID:     "id" + strconv.Itoa(i),
			Vector: f64ToF32(vectors[i].Vector),
			Text:   vectors[i].Chunk.Data,
			MetaData: map[string]interface{}{
				"name": documentName,
			},
		})
	}

	err := chroma.ChromaVectorStorageApp.AddDocuments(context.Background(), documentName, records)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(err)
	// 召回测试

	searchQuery := "我的邮箱是什么"

	chunk := &document.Chunk{
		Data: searchQuery,
	}

	vector, er := embeddingModel.EmbeddingOneChunk(chunk)
	if er != nil {
		fmt.Println(er)
	}

	result, er := chroma.ChromaVectorStorageApp.SimilaritySearch(context.Background(), documentName, f64ToF32(vector.Vector), store.SearchOption{
		TopK:     3,
		MinScore: 0.0,
	})
	if er != nil {
		fmt.Println(er)
	}
	fmt.Printf("召回数量: %d\n", len(result))
	for i, v := range result {
		fmt.Printf("[%d] score=%.4f | text=%s\n", i+1, v.Score, v.Text)
	}
}

func f64ToF32(arr []float64) []float32 {
	// 数据转化成float32试试
	F32 := make([]float32, len(arr))
	for i, f := range arr {
		F32[i] = float32(f)
	}
	return F32
}
