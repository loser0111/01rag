package main

import (
	"context"
	"fmt"
	"log"

	"com.wyq.01rag/application/ingest"
	"com.wyq.01rag/application/query"
	"com.wyq.01rag/domain/model/document"
	"com.wyq.01rag/infra/storage/file_reader"
	"com.wyq.01rag/internal/app"
)

func main() {
	fmt.Println("01RAG (DDD 重构版)")
	fmt.Println("==================")

	ctx := context.Background()

	// 【DDD Composition Root】显式装配，替代原来基于 init() + 包级单例的隐式构造
	app_, err := app.BootstrapFromEnv()
	if err != nil {
		log.Fatalf("启动失败 (bootstrap): %v", err)
	}
	fmt.Printf("  已加载 embedding 模型配置: %d 条\n", len(app_.Cfg.EmbeddingModels))
	fmt.Printf("  向量库: chroma @ %s (space=%s)\n",
		app_.Cfg.Chroma.Url, app_.Cfg.Chroma.Space)

	// 演示路径 A：IngestUseCase + QueryUseCase 全链路（DDD 推荐路径）
	fmt.Println()
	fmt.Println("==== UseCase Demo ====")
	in := ingest.IngestInput{
		DocumentName: "demo_doc",
		Files: []*document.File{
			{FileName: "RAG实现方案调研报告.md", FilePath: "E:/learn/01agent/RAG实现方案调研报告.md"},
		},
		SourceReader:  file_reader.NewMarkdownFileReader(1024),
		EmbedderName:  "qwen3-embedding:0.6b",
		VectorStore:   "chroma",
		MetadataPatch: map[string]any{"source": "main_demo"},
	}
	out, err := app_.Ingest.Execute(ctx, in)
	if err != nil {
		log.Fatalf("Ingest 失败: %v", err)
	}
	fmt.Printf("[Ingest OK] doc=%s chunks=%d vectors=%d dim=%d\n",
		out.DocumentName, out.ChunksCount, out.VectorsCount, out.VectorDim)

	results, err := app_.Query.Execute(ctx, query.SearchInput{
		Question:     "什么是 RAG",
		CollName:     in.DocumentName,
		EmbedderName: in.EmbedderName,
		VectorStore:  in.VectorStore,
		TopK:         3,
		MinScore:     0.0,
	})
	if err != nil {
		log.Fatalf("Query 失败: %v", err)
	}
	fmt.Printf("[Query OK] hits=%d\n", len(results))
	for i, r := range results {
		txt := r.Doc
		if len(txt) > 100 {
			txt = txt[:100]
		}
		fmt.Printf("  [%d] score=%.4f | %s\n", i+1, r.Score, txt)
	}
}
