package test

import (
	"context"
	"fmt"
	"log"
	"testing"

	"com.wyq.01rag/application/ingest"
	"com.wyq.01rag/application/query"
	"com.wyq.01rag/domain/model/document"
	"com.wyq.01rag/domain/model/store"
	"com.wyq.01rag/infra/storage/file_reader"
	"com.wyq.01rag/internal/app"
)

// TestIngestFlow_New 演示【DDD 新路径】：Bootstrap → IngestUseCase。
// 特点：不直接 import infra 里的具体实现（chroma/ollama 单例），
// 一切通过构造注入，mock 方便。
func TestIngestFlow_New(t *testing.T) {
	ctx := context.Background()

	// 1) 装配：Composition Root（真实 main 会放这里调用，之后只拿 app 用）
	app_, err := app.BootstrapFromEnv()
	if err != nil {
		log.Fatalf("bootstrap failed: %v", err)
	}

	// 2) 入参：Application 层 DTO
	in := ingest.IngestInput{
		DocumentName: "test_md_via_usecase",
		Files: []*document.File{
			{FileName: "RAG实现方案调研报告.md", FilePath: "E:/learn/01agent/RAG实现方案调研报告.md"},
		},
		SourceReader:  file_reader.NewMarkdownFileReader(1024),
		EmbedderName:  "qwen3-embedding:0.6b",
		VectorStore:   "chroma",
		MetadataPatch: map[string]any{"source": "dd_demo"},
	}

	// 3) 执行用例：读 → 向量化 → 入库
	out, err := app_.Ingest.Execute(ctx, in)
	if err != nil {
		t.Fatalf("IngestUseCase.Execute failed: %v", err)
	}
	fmt.Printf("[ingest ok] doc=%s chunks=%d vectors=%d dim=%d\n",
		out.DocumentName, out.ChunksCount, out.VectorsCount, out.VectorDim)

	// 4) 顺便跑一个 query usecase，验证全链路
	results, err := app_.Query.Execute(ctx, query.SearchInput{
		Question:     "RAG 系统中向量库的作用是什么",
		CollName:     in.DocumentName,
		EmbedderName: in.EmbedderName,
		VectorStore:  in.VectorStore,
		TopK:         3,
		MinScore:     0.0,
		Filter:       nil,
	})
	if err != nil {
		t.Fatalf("QueryUseCase.Execute failed: %v", err)
	}
	fmt.Printf("[query ok] hits=%d\n", len(results))
	for i, r := range results {
		txt := r.Text
		if len(txt) > 80 {
			txt = txt[:80]
		}
		fmt.Printf("  [%d] score=%.4f | %s\n", i+1, r.Score, txt)
	}
}

// 额外提供给外部代码（不使用 *testing.T）调用的入口。
// main.go 仍然依赖的旧入口仍可以在 test_embedding.go 找到。

// DemoQuery_New 最小化示例：演示 QueryUseCase 的直接调用
func DemoQuery_New(
	ctx context.Context,
	collName string,
	question string,
	_ []store.SearchOption, // 保留命名兼容旧外部调用；SearchOption 字段直接映射到 query.SearchInput
) ([]store.SearchResult, error) {
	app_, err := app.BootstrapFromEnv()
	if err != nil {
		return nil, err
	}
	return app_.Query.Execute(ctx, query.SearchInput{
		Question:     question,
		CollName:     collName,
		EmbedderName: "qwen3-embedding:0.6b",
		VectorStore:  "chroma",
		TopK:         5,
		MinScore:     0.0,
	})
}
