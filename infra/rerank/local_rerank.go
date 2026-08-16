package rerank

import (
	"bytes"
	"com.wyq.01rag/domain/model/rerank"
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
)

type LocalReranker struct {
	Url   string
	Model string
}

func NewLocalReranker(url, model string) *LocalReranker {
	return &LocalReranker{
		Url:   url,
		Model: model,
	}
}

func (ranker *LocalReranker) Rerank(ctx context.Context, query string, docs []string, topN int) ([]*rerank.RankItem, error) {
	req := &RerankRequest{
		Model:     ranker.Model,
		Query:     query,
		Documents: docs,
		TopN:      topN,
	}
	reqBody, err := json.Marshal(req)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, ranker.Url, bytes.NewReader(reqBody))
	if err != nil {
		log.Println(err)
		return nil, err
	}
	httpResp, err := DoHttpRequest(ctx, httpReq)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	defer httpResp.Body.Close()
	buf, _ := io.ReadAll(httpResp.Body)
	var reRankResp RerankResponse
	_ = json.Unmarshal(buf, &reRankResp)

	return Convert2RankItem(reRankResp.Results), nil
}

func Convert2RankItem(results []*RerankResult) []*rerank.RankItem {
	if len(results) == 0 {
		return nil
	}
	rankItems := make([]*rerank.RankItem, len(results))
	for i, result := range results {
		rankItems[i] = &rerank.RankItem{
			Index: result.Index,
			Score: result.RelevanceScore,
		}
	}
	return rankItems
}

func DoHttpRequest(ctx context.Context, httpReq *http.Request) (*http.Response, error) {
	httpReq.Header.Add("Content-Type", "application/json")
	httpClient := &http.Client{}
	httpResp, err := httpClient.Do(httpReq)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	if httpResp.StatusCode != http.StatusOK {
		log.Println(httpResp.Status)
		return nil, err
	}
	return httpResp, nil
}
