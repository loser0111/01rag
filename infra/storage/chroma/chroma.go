package chroma

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"com.wyq.01rag/constant"
	"com.wyq.01rag/domain/model/store"

	"github.com/google/uuid"
)

var ChromaVectorStorageApp store.VectorStore

// DistanceSpace 距离空间类型（决定 score 归一化公式）
type DistanceSpace string

const (
	DistanceSpaceL2     DistanceSpace = "l2"
	DistanceSpaceCosine DistanceSpace = "cosine"
	DistanceSpaceIP     DistanceSpace = "ip"
)

// ScoreNormalizer 将 chroma 返回的原始 distance → 归一化 score (越大越相似)
type ScoreNormalizer func(rawDistance float64) float64

var (
	// L2Normalizer 欧氏距离 (d ∈ [0, +∞)) → score ∈ (0,1]，精确匹配时=1
	L2Normalizer ScoreNormalizer = func(d float64) float64 { return 1.0 / (1.0 + d) }
	// CosineNormalizer 余弦距离 (d ∈ [0, 2]) → 余弦相似度 ∈ [-1,1]
	CosineNormalizer ScoreNormalizer = func(d float64) float64 { return 1.0 - d }
	// IPNormalizer 内积空间（Chroma 返回的 raw 已经是 -dot，距离越小=点积越大）→ score=-d
	IPNormalizer ScoreNormalizer = func(d float64) float64 { return -d }
)

// ResolveScoreNormalizer 根据空间类型返回归一化函数；未知/空时按 L2 保守处理
func ResolveScoreNormalizer(space DistanceSpace) ScoreNormalizer {
	switch space {
	case DistanceSpaceCosine:
		return CosineNormalizer
	case DistanceSpaceIP:
		return IPNormalizer
	case DistanceSpaceL2, "":
		return L2Normalizer
	default:
		return L2Normalizer
	}
}

type ChromaVectorStorage struct {
	Url        string
	HttpClient *http.Client
	Tenant     string
	Database   string
	ApiKey     string
	Name2Id    sync.Map
	// Space 距离空间：L2 / cosine / ip，用来选择 score 归一化公式
	Space          DistanceSpace
	scoreNormalize ScoreNormalizer
}

// ====== 内部响应/请求结构体（按 chroma_api_file.json 定义） ======

// CollectionResponse 对应 schema: Collection
type CollectionResponse struct {
	ID                string          `json:"id"`
	Name              string          `json:"name"`
	ConfigurationJSON json.RawMessage `json:"configuration_json"`
	Tenant            string          `json:"tenant"`
	Database          string          `json:"database"`
	LogPosition       int64           `json:"log_position"`
	Version           int32           `json:"version"`
	Dimension         *int32          `json:"dimension,omitempty"`
	Metadata          map[string]any  `json:"metadata,omitempty"`
}

// CreateCollectionBody 对应 schema: CreateCollectionPayload
type CreateCollectionBody struct {
	Name          string         `json:"name"`
	Configuration any            `json:"configuration,omitempty"`
	GetOrCreate   bool           `json:"get_or_create,omitempty"`
	Metadata      map[string]any `json:"metadata,omitempty"`
	Schema        any            `json:"schema,omitempty"`
}

// QueryResponse 对应 schema: QueryResponse
type QueryResponse struct {
	Ids        [][]string          `json:"ids"`
	Include    []string            `json:"include"`
	Distances  *[][]float64        `json:"distances,omitempty"`
	Documents  *[][]string         `json:"documents,omitempty"`
	Metadatas  *[][]map[string]any `json:"metadatas,omitempty"`
	Embeddings *[][][]float32      `json:"embeddings,omitempty"`
	Uris       *[][]string         `json:"uris,omitempty"`
}

// GetResponse 对应 schema: GetResponse
type GetResponse struct {
	Ids        []string          `json:"ids"`
	Include    []string          `json:"include"`
	Documents  *[]string         `json:"documents,omitempty"`
	Embeddings *[][]float32      `json:"embeddings,omitempty"`
	Metadatas  *[]map[string]any `json:"metadatas,omitempty"`
	Uris       *[]string         `json:"uris,omitempty"`
}

// ErrorResponse 对应 schema: ErrorResponse
type ErrorResponse struct {
	Error   string `json:"error,omitempty"`
	Message string `json:"message,omitempty"`
}

// ====== 初始化 ======

// NewChromaVectorStorage 显式构造函数（依赖注入），空间类型缺省为 L2
func NewChromaVectorStorage(
	url string,
	tenant string,
	database string,
	apiKey string,
	space DistanceSpace,
) *ChromaVectorStorage {
	if url == "" {
		url = "http://127.0.0.1:8000/api/v2"
	}
	if tenant == "" {
		tenant = constant.DefaultChromaTenantName
	}
	if database == "" {
		database = constant.DefaultChromaDataBaseName
	}
	if space == "" {
		space = DistanceSpaceL2
	}
	return &ChromaVectorStorage{
		Url: url,
		HttpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		Tenant:         tenant,
		Database:       database,
		ApiKey:         apiKey,
		Space:          space,
		scoreNormalize: ResolveScoreNormalizer(space),
		Name2Id:        sync.Map{},
	}
}

// init 保留默认单例以兼容旧调用（不再 panic，启动时默认值不保证可连通）
func init() {
	fmt.Println("chroma DB init (compat singleton)")
	ChromaVectorStorageApp = NewChromaVectorStorage("", "", "", "", DistanceSpaceL2)
}

// ====== 内部辅助方法 ======

// basePath 返回 tenants/{tenant}/databases/{database} 基础路径
func (c *ChromaVectorStorage) basePath() string {
	return fmt.Sprintf("%s/tenants/%s/databases/%s", c.Url, c.Tenant, c.Database)
}

// doRequest 发送 HTTP 请求，自动附加 ApiKey Header，并解析错误响应
func (c *ChromaVectorStorage) doRequest(
	ctx context.Context,
	method string,
	url string,
	body any,
) (*http.Response, error) {
	var reader io.Reader
	if body != nil {
		buf, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal request body: %w", err)
		}
		reader = bytes.NewReader(buf)
	}
	req, err := http.NewRequestWithContext(ctx, method, url, reader)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")
	if c.ApiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.ApiKey)
		req.Header.Set("X-Chroma-Token", c.ApiKey)
	}
	resp, err := c.HttpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http do: %w", err)
	}
	if resp.StatusCode >= 400 {
		defer resp.Body.Close()
		buf, _ := io.ReadAll(resp.Body)
		var errResp ErrorResponse
		_ = json.Unmarshal(buf, &errResp)
		msg := errResp.Message
		if msg == "" {
			msg = errResp.Error
		}
		if msg == "" {
			msg = string(buf)
		}
		return resp, fmt.Errorf("chroma api %s %s status=%d err=%s", method, url, resp.StatusCode, msg)
	}
	return resp, nil
}

// resolveCollectionId 通过 collectionName 获取 collection id（优先缓存，其次通过 get_or_create 创建/查找）
func (c *ChromaVectorStorage) resolveCollectionId(
	ctx context.Context,
	documentName string,
) (string, error) {
	if val, ok := c.Name2Id.Load(documentName); ok {
		return val.(string), nil
	}
	createBody := CreateCollectionBody{
		Name:        documentName,
		GetOrCreate: true,
		Metadata:    map[string]any{},
	}
	url := c.basePath() + "/collections"
	resp, err := c.doRequest(ctx, http.MethodPost, url, createBody)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var coll CollectionResponse
	if err := json.NewDecoder(resp.Body).Decode(&coll); err != nil {
		return "", fmt.Errorf("decode collection response: %w", err)
	}
	if coll.ID == "" {
		return "", errors.New("chroma returned empty collection id")
	}
	c.Name2Id.Store(documentName, coll.ID)
	return coll.ID, nil
}

// collectionRecordsPath 构造 collection 下 records 相关操作的 URL
func (c *ChromaVectorStorage) collectionRecordsPath(collectionID, action string) string {
	return fmt.Sprintf("%s/collections/%s/%s", c.basePath(), collectionID, action)
}

// ====== 集合管理 ======

func (c *ChromaVectorStorage) CreateCollection(
	ctx context.Context,
	documentName string,
	dim int,
) error {
	_, err := c.resolveCollectionId(ctx, documentName)
	if err != nil {
		log.Println("create collection err:", err)
		return err
	}
	return nil
}

func (c *ChromaVectorStorage) DropCollection(
	ctx context.Context,
	documentName string,
) error {
	id, err := c.resolveCollectionId(ctx, documentName)
	if err != nil {
		return err
	}
	url := fmt.Sprintf("%s/collections/%s", c.basePath(), id)
	resp, err := c.doRequest(ctx, http.MethodDelete, url, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	c.Name2Id.Delete(documentName)
	return nil
}

func (c *ChromaVectorStorage) CollectionExists(
	ctx context.Context,
	documentName string,
) bool {
	_, err := c.resolveCollectionId(ctx, documentName)
	return err == nil
}

// ====== 写入 ======

func (c *ChromaVectorStorage) AddDocuments(
	ctx context.Context,
	documentName string,
	records []*store.VectorRecord,
) error {
	id, err := c.resolveCollectionId(ctx, documentName)
	if err != nil {
		return err
	}
	var ids []string
	var embeddings [][]float32
	var documents []string
	var metadatas []map[string]interface{}

	for _, r := range records {
		// ID 为空时自动生成 UUID，调用方无需手动管理
		recordID := r.ID
		if recordID == "" {
			recordID = uuid.New().String()
			r.ID = recordID
		}
		ids = append(ids, recordID)
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
	url := c.collectionRecordsPath(id, "upsert")
	resp, err := c.doRequest(ctx, http.MethodPost, url, payload)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

// ====== 删除 ======

func (c *ChromaVectorStorage) DeleteByIDs(
	ctx context.Context,
	documentName string,
	ids []string,
) error {
	id, err := c.resolveCollectionId(ctx, documentName)
	if err != nil {
		return err
	}
	payload := map[string]any{"ids": ids}
	url := c.collectionRecordsPath(id, "delete")
	resp, err := c.doRequest(ctx, http.MethodPost, url, payload)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

func (c *ChromaVectorStorage) DeleteByFilter(
	ctx context.Context,
	documentName string,
	filter map[string]interface{},
) error {
	id, err := c.resolveCollectionId(ctx, documentName)
	if err != nil {
		return err
	}
	payload := map[string]any{"where": filter}
	url := c.collectionRecordsPath(id, "delete")
	resp, err := c.doRequest(ctx, http.MethodPost, url, payload)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

// ====== 查询 ======

func (c *ChromaVectorStorage) SimilaritySearch(
	ctx context.Context,
	documentName string,
	queryVector []float32,
	opt store.SearchOption,
) ([]store.SearchResult, error) {
	id, err := c.resolveCollectionId(ctx, documentName)
	if err != nil {
		return nil, err
	}

	payload := map[string]any{
		"query_embeddings": [][]float32{queryVector},
		"n_results":        opt.TopK,
		"include":          []string{"documents", "metadatas", "embeddings", "distances"},
	}
	if opt.Filter != nil && len(opt.Filter) > 0 {
		payload["where"] = opt.Filter
	}
	url := c.collectionRecordsPath(id, "query")
	resp, err := c.doRequest(ctx, http.MethodPost, url, payload)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var res QueryResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, fmt.Errorf("decode query response: %w", err)
	}

	var results []store.SearchResult
	if len(res.Ids) == 0 {
		return results, nil
	}

	idsBatch := res.Ids[0]
	var docsBatch []string
	if res.Documents != nil && len(*res.Documents) > 0 {
		docsBatch = (*res.Documents)[0]
	}
	var metasBatch []map[string]any
	if res.Metadatas != nil && len(*res.Metadatas) > 0 {
		metasBatch = (*res.Metadatas)[0]
	}
	var embBatch [][]float32
	if res.Embeddings != nil && len(*res.Embeddings) > 0 {
		embBatch = (*res.Embeddings)[0]
	}
	var distBatch []float64
	if res.Distances != nil && len(*res.Distances) > 0 {
		distBatch = (*res.Distances)[0]
	}

	for i := range idsBatch {
		var distance float64
		if i < len(distBatch) {
			distance = distBatch[i]
		}
		// 按距离空间归一化 score（当前实例 Space 决定，缺省 L2 → 1/(1+d)）
		score := c.scoreNormalize(distance)

		if opt.MinScore > 0 && score < opt.MinScore {
			continue
		}

		var text string
		if i < len(docsBatch) {
			text = docsBatch[i]
		}
		var meta map[string]any
		if i < len(metasBatch) {
			meta = metasBatch[i]
		}
		var vec []float32
		if i < len(embBatch) {
			vec = embBatch[i]
		}

		results = append(results, store.SearchResult{
			VectorRecord: store.VectorRecord{
				ID:       idsBatch[i],
				Text:     text,
				MetaData: meta,
				Vector:   vec,
			},
			Score: score,
		})
	}
	return results, nil
}

// ====== 按 ID 获取 ======

func (c *ChromaVectorStorage) GetByID(
	ctx context.Context,
	documentName string,
	id string,
) (*store.VectorRecord, error) {
	collID, err := c.resolveCollectionId(ctx, documentName)
	if err != nil {
		return nil, err
	}
	payload := map[string]any{
		"ids":     []string{id},
		"include": []string{"documents", "metadatas", "embeddings"},
	}
	url := c.collectionRecordsPath(collID, "get")
	resp, err := c.doRequest(ctx, http.MethodPost, url, payload)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var data GetResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("decode get response: %w", err)
	}
	if len(data.Ids) == 0 {
		return nil, errors.New("record not found")
	}
	var text string
	if data.Documents != nil && len(*data.Documents) > 0 {
		text = (*data.Documents)[0]
	}
	var meta map[string]any
	if data.Metadatas != nil && len(*data.Metadatas) > 0 {
		meta = (*data.Metadatas)[0]
	}
	var vec []float32
	if data.Embeddings != nil && len(*data.Embeddings) > 0 {
		vec = (*data.Embeddings)[0]
	}
	return &store.VectorRecord{
		ID:       data.Ids[0],
		Text:     text,
		MetaData: meta,
		Vector:   vec,
	}, nil
}

func (c *ChromaVectorStorage) Close() error {
	return nil
}
