package store

import "context"

// 向量数据库

type VectorRecord struct {
	ID       string                 `json:"id"`
	Vector   []float32              `json:"vector"`
	Text     string                 `json:"text"`
	MetaData map[string]interface{} `json:"metaData"`
}

// SearchOption 检索入参选项
type SearchOption struct {
	TopK     int                    // 返回几条结果
	MinScore float64                // 最低相似度阈值，过滤低相关结果
	Filter   map[string]interface{} // 元数据过滤条件
}

// SearchResult 检索返回结果
type SearchResult struct {
	VectorRecord
	Score float64 // 相似度分数（余弦距离/内积等，统一归一化）
}

// VectorStore抽象的功能

type VectorStore interface {
	// ========== 集合（知识库）管理：实现多个文档集合隔离查询 ==========
	// CreateCollection 创建向量集合（指定维度、距离算法）
	CreateCollection(ctx context.Context, collName string, dim int) error

	// DropCollection 删除整个集合
	DropCollection(ctx context.Context, collName string) error

	// CollectionExists 判断集合是否存在
	CollectionExists(ctx context.Context, collName string) bool

	// ========== 写入能力：文档向量化后入库 ==========
	// AddDocuments 批量插入向量记录（RAG核心，chunk入库）
	AddDocuments(ctx context.Context, collName string, records []*VectorRecord) error

	// ========== 删除能力：文档更新/清理场景 ==========

	// DeleteByIDs 根据主键ID批量删除向量
	DeleteByIDs(ctx context.Context, collName string, ids []string) error

	// DeleteByFilter 根据元数据条件批量删除（例如删除某个文件全部chunk）
	DeleteByFilter(ctx context.Context, collName string, filter map[string]interface{}) error

	// ========== 核心：相似度检索（RAG查询阶段最重要方法） ==========

	// SimilaritySearch 根据query向量，在指定集合内召回相似文档
	SimilaritySearch(
		ctx context.Context,
		collName string,
		queryVector []float32,
		opt SearchOption,
	) ([]SearchResult, error)

	// ========== 可选辅助扩展（按需取舍） ==========
	// GetByID 根据ID直接查询单条向量与元数据
	GetByID(ctx context.Context, collName string, id string) (*VectorRecord, error)

	// Close 释放连接、资源
	Close() error
}
