package handler

import (
	"com.wyq.01rag/domain/model/embedding"
	"com.wyq.01rag/domain/model/store"
	"com.wyq.01rag/infra/storage/manager"
	"com.wyq.01rag/infra/utils"
	"context"
)

type StoreVectorHandler struct {
}

func (h *StoreVectorHandler) Handle(ctx context.Context, embeddingCtx *embedding.EmbeddingContext) error {

	records := make([]*store.VectorRecord, len(embeddingCtx.Vectors))

	for i, vector := range embeddingCtx.Vectors {
		records[i] = &store.VectorRecord{
			Vector: utils.ConvertFloat64ToFloat32(vector.Vector),
			Text:   vector.Chunk.Data,
			MetaData: map[string]interface{}{
				"name": embeddingCtx.DocumentName,
			},
		}
	}
	embeddingCtx.Records = records

	embeddingCtx.VectorStore = manager.StorageManagerApp.GetStoreByName(embeddingCtx.StoreType)

	err := embeddingCtx.VectorStore.AddDocuments(ctx, embeddingCtx.DocumentName, embeddingCtx.Records)
	embeddingCtx.Stop(err)
	if err != nil {
		return err
	}
	return nil
}

func (h *StoreVectorHandler) Name() string {
	return "StoreVector"
}
