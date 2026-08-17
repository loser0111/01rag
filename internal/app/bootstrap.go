// Package app 【Composition Root / Bootstrap】
//
// 这里是 DDD 的"装配根"：main/test 只调用一次 app.Bootstrap(cfg)，
// 拿到 *App 就可以调用已经 DI 好的 IngestUseCase / QueryUseCase。
// 其他地方一律不允许直接 import infra 里的具体实现。
//
// 此文件"知道"所有具体实现（chroma、ollama、markdown reader），但不写业务逻辑；
// 只负责 new 出来、连起来、返回。
package app

import (
	"fmt"

	"com.wyq.01rag/application/ingest"
	"com.wyq.01rag/application/query"
	"com.wyq.01rag/domain/port"
	infra_embed "com.wyq.01rag/infra/model/embedding"
	infra_rerank "com.wyq.01rag/infra/rerank"
	infra_store "com.wyq.01rag/infra/storage/manager"
	"com.wyq.01rag/internal/config"
)

// App 对外暴露已经 DI 好的用例。UI / CLI / test 只依赖这个对象。
type App struct {
	Cfg       *config.AppConfig
	Ingest    *ingest.UseCase
	Query     *query.UseCase
	Embedders port.EmbedderRegistry
	Stores    port.VectorStoreRegistry
}

// Bootstrap 从强类型配置装配全应用。
// 调用顺序 main.go: cfg,err := config.LoadConfig(); app := Bootstrap(cfg)
func Bootstrap(cfg *config.AppConfig) (*App, error) {
	if cfg == nil {
		return nil, fmt.Errorf("cfg is nil")
	}

	// 1) Embedding Registry
	embMgr, err := infra_embed.NewEmbeddingManager(cfg.EmbeddingModels)
	if err != nil {
		return nil, fmt.Errorf("init embedders: %w", err)
	}

	// 2) Vector Store Registry
	storeMgr := infra_store.NewStorageManager(cfg.Chroma)

	// 3) Vector Store Registry
	rerankMgr := infra_rerank.NewLocalReranker("http://localhost:8001/rerank", "BAAI/bge-reranker-v2-m3")

	// 4) Application UseCases
	ingestUC := ingest.NewIngestUseCase(embMgr, storeMgr)
	queryUC := query.NewQueryUseCase(embMgr, storeMgr, rerankMgr)

	return &App{
		Cfg:       cfg,
		Ingest:    ingestUC,
		Query:     queryUC,
		Embedders: embMgr,
		Stores:    storeMgr,
	}, nil
}

// BootstrapFromEnv 方便用在 test：从 config 文件自动读取（跟 main 启动用的是同一套）
func BootstrapFromEnv() (*App, error) {
	cfg, err := config.LoadConfig()
	if err != nil {
		return nil, err
	}
	return Bootstrap(cfg)
}
