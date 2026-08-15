package manager

import (
	"com.wyq.01rag/domain/model/store"
	"com.wyq.01rag/infra/storage/chroma"
)

var StorageManagerApp *StorageManager

type StorageManager struct {
	AllStore map[string]store.VectorStore
}

// 注意初始化的顺序，Manager需要最后初始化

func init() {
	StorageManagerApp = &StorageManager{
		AllStore: map[string]store.VectorStore{
			"chroma": chroma.ChromaVectorStorageApp,
		},
	}
}

func (sm *StorageManager) GetStoreRageByName(Name string) store.VectorStore {
	return sm.AllStore[Name]
}
