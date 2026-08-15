package handler

import (
	"com.wyq.01rag/domain/model/embedding"
	"context"
)

// handler定义

type EmbeddingHandler interface {
	Name() string
	Handle(ctx context.Context, embeddingCtx *embedding.EmbeddingContext) error
}
