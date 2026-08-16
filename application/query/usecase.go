// Package query 【Application 层 - 知识检索用例】
//
// 同样只依赖 domain/port 接口，不依赖 infra。
// 负责：问题 → 向量化 → 向量检索 → 返回排序后的 SearchResult
package query

import (
	"com.wyq.01rag/domain/model/rerank"
	"context"
	"errors"
	"fmt"

	"com.wyq.01rag/domain/model/document"
	"com.wyq.01rag/domain/model/store"
	"com.wyq.01rag/domain/port"
	"com.wyq.01rag/infra/utils"
)

// SearchInput 检索用例入参
type SearchInput struct {
	Question     string
	CollName     string
	EmbedderName string
	VectorStore  string
	TopK         int
	MinScore     float64
	Filter       map[string]interface{}
}

// UseCase 检索用例
type UseCase struct {
	embedders port.EmbedderRegistry
	stores    port.VectorStoreRegistry
	reranker  port.ReRanker
}

func NewQueryUseCase(
	embedders port.EmbedderRegistry,
	stores port.VectorStoreRegistry,
	reranker port.ReRanker,
) *UseCase {
	return &UseCase{embedders: embedders, stores: stores}
}

func (uc *UseCase) Execute(ctx context.Context, input SearchInput) ([]*rerank.RankItem, error) {
	if er := uc.validateSearchInput(input); er != nil {
		return nil, er
	}
	topK := input.TopK
	if topK <= 0 {
		topK = 5
	}

	embed, err := uc.embedders.FindByName(input.EmbedderName)
	if err != nil {
		return nil, fmt.Errorf("resolve embedder %q: %w", input.EmbedderName, err)
	}
	searcher, err := uc.stores.GetSearcher(input.VectorStore)
	if err != nil {
		return nil, fmt.Errorf("resolve vector store %q: %w", input.VectorStore, err)
	}

	qVec, err := embed.EmbedOne(ctx, &document.Chunk{Data: input.Question})
	if err != nil {
		return nil, fmt.Errorf("embed question: %w", err)
	}

	searchResult, err := searcher.SimilaritySearch(ctx, input.CollName, utils.ConvertFloat64ToFloat32(qVec.Vector), store.SearchOption{
		TopK:     topK,
		MinScore: input.MinScore,
		Filter:   input.Filter,
	})
	if err != nil {
		return nil, fmt.Errorf("search: %w", err)
	}

	// 执行reRank
	docs := make([]string, 0)
	for _, SearchResult := range searchResult {
		docs = append(docs, SearchResult.Text)
	}
	rankItems, err := uc.reranker.Rerank(ctx, input.Question, docs, input.TopK)

	return rankItems, nil
}
func (uc *UseCase) validateSearchInput(input SearchInput) error {
	if input.Question == "" {
		return errors.New("question is required")
	}
	if input.CollName == "" {
		return errors.New("collection name is required")
	}
	if input.EmbedderName == "" {
		return errors.New("embedderName is required")
	}
	if input.VectorStore == "" {
		return errors.New("vectorStore is required")
	}
	return nil
}
