package port

import (
	"com.wyq.01rag/domain/model/rerank"
	"context"
)

type ReRanker interface {
	Rerank(ctx context.Context, query string, docs []string, topN int) ([]*rerank.RankItem, error)
}
