package manager

import (
	"fmt"
	"sync"

	"com.wyq.01rag/domain/model/store"
	"com.wyq.01rag/domain/port"
	"com.wyq.01rag/infra/storage/chroma"
	"com.wyq.01rag/internal/config"
)

// StorageManager 向量存储注册表：同时实现 legacy 的 store.StorageManager 和 DDD 的 port.VectorStoreRegistry
type StorageManager struct {
	AllStore map[string]store.VectorStore
	once     sync.Once
}

// NewStorageManager 显式构造。如果 cfg.Chroma 有配置，则默认注册名为 "chroma" 的实现。
func NewStorageManager(cfg config.ChromaConfig) *StorageManager {
	mgr := &StorageManager{
		AllStore: make(map[string]store.VectorStore),
	}
	space := chroma.DistanceSpace(cfg.Space)
	mgr.AllStore["chroma"] = chroma.NewChromaVectorStorage(
		cfg.Url,
		cfg.Tenant,
		cfg.Database,
		cfg.ApiKey,
		space,
	)
	return mgr
}

// Register 允许在 bootstrap 阶段动态注册更多存储（如 Milvus/Memory/FAISS）
func (sm *StorageManager) Register(name string, s store.VectorStore) {
	if sm.AllStore == nil {
		sm.AllStore = make(map[string]store.VectorStore)
	}
	sm.AllStore[name] = s
}

func (sm *StorageManager) GetStoreByName(Name string) store.VectorStore {
	if sm == nil || sm.AllStore == nil {
		return nil
	}
	return sm.AllStore[Name]
}

// ==================== port.VectorStoreRegistry ====================

func (sm *StorageManager) resolve(name string) (store.VectorStore, error) {
	s := sm.GetStoreByName(name)
	if s == nil {
		return nil, fmt.Errorf("vector store %q not registered", name)
	}
	return s, nil
}

func (sm *StorageManager) GetWriter(name string) (port.VectorWriter, error) {
	return sm.resolve(name)
}

func (sm *StorageManager) GetSearcher(name string) (port.VectorSearcher, error) {
	return sm.resolve(name)
}

func (sm *StorageManager) GetCollectionManager(name string) (port.CollectionManager, error) {
	return sm.resolve(name)
}

// ==================== Compat：保留包级单例给 legacy 代码使用 ====================
// （init 仍然存在；新代码请通过 Bootstrap 调 NewStorageManager。）

var StorageManagerApp *StorageManager

func init() {
	StorageManagerApp = NewStorageManager(config.ChromaConfig{
		Url:      "http://127.0.0.1:8000/api/v2",
		Tenant:   "default_tenant",
		Database: "default_database",
		Space:    "l2",
	})
}
