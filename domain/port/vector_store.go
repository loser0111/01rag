package port

import (
	"context"

	"com.wyq.01rag/domain/model/store"
)

// CollectionManager 集合管理端口（RAG 中一般一个"知识库"是一个 collection）
type CollectionManager interface {
	CreateCollection(ctx context.Context, collName string, dim int) error
	DropCollection(ctx context.Context, collName string) error
	CollectionExists(ctx context.Context, collName string) bool
}

// VectorWriter 向量写入端口（KnowledgeIngestion BC）
type VectorWriter interface {
	// AddDocuments 向集合中 upsert 向量（ID 为空由实现决定是否自动生成，返回 records 会回填生成的 ID）
	AddDocuments(ctx context.Context, collName string, records []*store.VectorRecord) error
	DeleteByIDs(ctx context.Context, collName string, ids []string) error
	DeleteByFilter(ctx context.Context, collName string, filter map[string]interface{}) error
}

// VectorSearcher 向量检索端口（QuestionAnswering BC）
type VectorSearcher interface {
	SimilaritySearch(
		ctx context.Context,
		collName string,
		queryVector []float32,
		opt store.SearchOption,
	) ([]store.SearchResult, error)
	GetByID(ctx context.Context, collName string, id string) (*store.VectorRecord, error)
}

// VectorStoreRegistry 按类型获取 VectorWriter/Searcher/Manager（一般返回同一实例）
type VectorStoreRegistry interface {
	GetWriter(name string) (VectorWriter, error)
	GetSearcher(name string) (VectorSearcher, error)
	GetCollectionManager(name string) (CollectionManager, error)
}
