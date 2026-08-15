package storage

import (
	chroma "github.com/tmc/langchaingo/vectorstores/chroma"
)

func init() {
	store, err := chroma.New(
		chroma.WithURL("http://127.0.0.1:8000"),
		chroma.WithCollectionName("rag_demo"),
		// 如果开了token认证
		// chroma.WithAuthToken("xxx"),
	)
	if err != nil {
		log.Fatal(err)
	}
	ctx := context.Background()
	// 后续可以直接使用 AddDocuments / SimilaritySearch
	_ = ctx
	_ = store
}
