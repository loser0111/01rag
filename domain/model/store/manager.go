package store

type StorageManager interface {
	GetStoreByName(Name string) VectorStore
}
