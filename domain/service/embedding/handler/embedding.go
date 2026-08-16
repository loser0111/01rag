package handler

import (
	"com.wyq.01rag/domain/model/embedding"
	infra_embedding "com.wyq.01rag/infra/model/embedding"
	"context"
	"errors"
)

type ModelEmbeddingHandler struct {
}

func (h *ModelEmbeddingHandler) Handle(ctx context.Context, embeddingCtx *embedding.EmbeddingContext) error {
	if err := Check(ctx, embeddingCtx); err != nil {
		embeddingCtx.Stop(err)
		return err
	}

	// 调用模型
	embeddingModel := infra_embedding.EmbeddingManagerApp.LegacyFindByName(embeddingCtx.EmbeddingModelName)
	if embeddingModel == nil {
		embeddingCtx.Stop(errors.New("model not found"))
		return embeddingCtx.Error
	}
	vectors, err := embeddingModel.Embedding(embeddingCtx.Chunks)
	if err != nil {
		embeddingCtx.Stop(err)
		return err
	}
	// 数据存储
	embeddingCtx.Vectors = vectors
	return nil
}

func Check(ctx context.Context, embeddingCtx *embedding.EmbeddingContext) error {
	var err error = nil
	if len(embeddingCtx.Chunks) == 0 {
		err = errors.New("not find Valid Chunks to embedding")
	}
	if len(embeddingCtx.EmbeddingModelName) == 0 {
		err = errors.New("you do not config model for embedding ")
	}
	return err
}
func (h *ModelEmbeddingHandler) Name() string {
	return "ModelEmbedding"
}
