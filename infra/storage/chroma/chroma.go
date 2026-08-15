package chroma

import (
	"bytes"
	"com.wyq.01rag/domain/model/store"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
)

var ChromaVectorStorageApp store.VectorStore

// 本地的chroma使用的是UUID访问的向量集合

type ChromaVectorStorage struct {
	Url        string
	HttpClient *http.Client
	Name2UUId  sync.Map
}

// 初始化连接的client
func init() {
	fmt.Println("chroma DB init")
	ChromaVectorStorageApp = &ChromaVectorStorage{
		Url: "http://127.0.0.1:8000/api/v2",
		HttpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

// 通过documentId获取UUID
func (c *ChromaVectorStorage) getCollectionUUId(ctx context.Context, documentName string) (collectionUUID string, err error) {
	if val, ok := c.Name2UUId.Load(documentName); ok {
		return val.(string), nil
	}
	reqBody := map[string]any{
		"name":          documentName,
		"metadata":      map[string]any{},
		"get_or_create": true,
	}
	body, _ := json.Marshal(reqBody)
	fmt.Println(c.Url + "/collections")
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.Url+"/collections", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HttpClient.Do(req)
	if err != nil {
		return "", err
	}
	fmt.Printf("%#v\n", resp)
	var result struct {
		UUID string `json:"uuid"`
		Name string `json:"name"`
	}
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return "", err
	}

	c.Name2UUId.Store(documentName, result.UUID)
	return result.UUID, nil
}

func (c *ChromaVectorStorage) CreateCollection(ctx context.Context, documentName string, dim int) error {
	_, err := c.getCollectionUUId(ctx, documentName)
	if err != nil {
		log.Println("create collection err:", err)
		return err
	}
	return nil
}

func (c *ChromaVectorStorage) DropCollection(ctx context.Context, documentName string) error {
	uuid, err := c.getCollectionUUId(ctx, documentName)
	if err != nil {
		return err
	}
	url := fmt.Sprintf("%s/collections/%s", c.Url, uuid)
	req, _ := http.NewRequestWithContext(ctx, http.MethodDelete, url, nil)
	resp, err := c.HttpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	c.Name2UUId.Delete(documentName)
	return nil
}

func (c *ChromaVectorStorage) CollectionExists(ctx context.Context, documentName string) bool {
	_, err := c.getCollectionUUId(ctx, documentName)
	return err == nil
}

func (c *ChromaVectorStorage) AddDocuments(ctx context.Context, documentName string, records []*store.VectorRecord) error {
	uuid, err := c.getCollectionUUId(ctx, documentName)
	if err != nil {
		return err
	}
	var ids []string
	var embeddings [][]float32
	var documents []string
	var metadatas []map[string]interface{}

	for _, r := range records {
		ids = append(ids, r.ID)
		embeddings = append(embeddings, r.Vector)
		documents = append(documents, r.Text)
		metadatas = append(metadatas, r.MetaData)
	}

	payload := map[string]any{
		"ids":        ids,
		"embeddings": embeddings,
		"documents":  documents,
		"metadatas":  metadatas,
	}
	body, _ := json.Marshal(payload)
	url := fmt.Sprintf("%s/collections/%s/upsert", c.Url, uuid)

	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.HttpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return errors.New("upsert failed")
	}
	return nil
}

func (c *ChromaVectorStorage) DeleteByIDs(ctx context.Context, documentName string, ids []string) error {
	uuid, err := c.getCollectionUUId(ctx, documentName)
	if err != nil {
		return err
	}
	payload := map[string]any{"ids": ids}
	body, _ := json.Marshal(payload)
	url := fmt.Sprintf("%s/collections/%s/delete", c.Url, uuid)

	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.HttpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

func (c *ChromaVectorStorage) DeleteByFilter(ctx context.Context, documentName string, filter map[string]interface{}) error {
	uuid, err := c.getCollectionUUId(ctx, documentName)
	if err != nil {
		return err
	}
	payload := map[string]any{"where": filter}
	body, _ := json.Marshal(payload)
	url := fmt.Sprintf("%s/collections/%s/delete", c.Url, uuid)

	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.HttpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

func (c *ChromaVectorStorage) SimilaritySearch(ctx context.Context, documentName string, queryVector []float32, opt store.SearchOption) ([]store.SearchResult, error) {
	uuid, err := c.getCollectionUUId(ctx, documentName)
	if err != nil {
		return nil, err
	}

	payload := map[string]any{
		"query_embeddings": [][]float32{queryVector},
		"n_results":        opt.TopK,
		"where":            opt.Filter,
	}
	body, _ := json.Marshal(payload)
	url := fmt.Sprintf("%s/collections/%s/query", c.Url, uuid)

	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.HttpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// 解析Chroma返回结构
	type chromaResp struct {
		Ids        [][]string         `json:"ids"`
		Documents  [][]string         `json:"documents"`
		Metadatas  [][]map[string]any `json:"metadatas"`
		Embeddings [][][]float32      `json:"embeddings"`
		Distances  [][]float64        `json:"distances"`
	}
	var res chromaResp
	err = json.NewDecoder(resp.Body).Decode(&res)
	if err != nil {
		return nil, err
	}

	var results []store.SearchResult
	if len(res.Ids) == 0 {
		return results, nil
	}

	// Chroma distance: Cosine Distance (0~2)
	// 转换为余弦相似度 score = 1 - distance
	for i := range res.Ids[0] {
		distance := res.Distances[0][i]
		score := 1.0 - distance

		if opt.MinScore > 0 && score < opt.MinScore {
			continue
		}

		results = append(results, store.SearchResult{
			VectorRecord: store.VectorRecord{
				ID:       res.Ids[0][i],
				Text:     res.Documents[0][i],
				MetaData: res.Metadatas[0][i],
				Vector:   res.Embeddings[0][i],
			},
			Score: score,
		})
	}
	return results, nil
}

func (c *ChromaVectorStorage) GetByID(ctx context.Context, documentName string, id string) (*store.VectorRecord, error) {
	uuid, err := c.getCollectionUUId(ctx, documentName)
	if err != nil {
		return nil, err
	}
	url := fmt.Sprintf("%s/collections/%s/get?ids=%s", c.Url, uuid, id)
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	resp, err := c.HttpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	type getResp struct {
		Ids        []string         `json:"ids"`
		Documents  []string         `json:"documents"`
		Metadatas  []map[string]any `json:"metadatas"`
		Embeddings [][]float32      `json:"embeddings"`
	}
	var data getResp
	_ = json.NewDecoder(resp.Body).Decode(&data)
	if len(data.Ids) == 0 {
		return nil, errors.New("record not found")
	}
	return &store.VectorRecord{
		ID:       data.Ids[0],
		Text:     data.Documents[0],
		MetaData: data.Metadatas[0],
		Vector:   data.Embeddings[0],
	}, nil
}

func (c *ChromaVectorStorage) Close() error {
	return nil
}
