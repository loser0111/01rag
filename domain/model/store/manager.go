package store

type StorageManager interface {
	GetStoreRageByName(Name string) VectorStore
}
