package embedding

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"com.wyq.01rag/constant"
	"com.wyq.01rag/domain/port"
	"com.wyq.01rag/internal/config"
)

// ==================== Registry ====================

// EmbeddingManager 实现 port.EmbedderRegistry，构造时显式接收 cfg，不再靠包级 init 读文件
type EmbeddingManager struct {
	EmbeddingModels map[string]port.Embedder
}

func (m *EmbeddingManager) FindByName(name string) (port.Embedder, error) {
	if m == nil || m.EmbeddingModels == nil {
		return nil, fmt.Errorf("embedding manager not initialized")
	}
	if e, ok := m.EmbeddingModels[name]; ok {
		return e, nil
	}
	return nil, fmt.Errorf("embedder %q not found", name)
}

// LegacyFindByName 兼容旧 API 调用风格：单返回值，找不到时返回 nil（不给 error 信息）
// 新代码/DDD bootstrap 请用 FindByName（带 error）。
func (m *EmbeddingManager) LegacyFindByName(name string) Embedding {
	if m == nil || m.EmbeddingModels == nil {
		return nil
	}
	e, ok := m.EmbeddingModels[name]
	if !ok {
		return nil
	}
	// port.Embedder 和 legacy Embedding 的方法名不同，但两者都由 *OllamaEmbedding / *OpenApiEmbedding 同时实现，
	// 这里做一次动态断言把它转成 legacy 接口返回给旧调用方。
	if legacy, ok := e.(Embedding); ok {
		return legacy
	}
	return nil
}

// NewEmbeddingManager 显式构造（从强类型 config 构建）
func NewEmbeddingManager(models []config.EmbeddingModelConfig) (*EmbeddingManager, error) {
	mgr := &EmbeddingManager{
		EmbeddingModels: make(map[string]port.Embedder, len(models)),
	}
	for _, m := range models {
		switch m.Type {
		case constant.EmbeddingModelTypeOllamaLocal:
			mgr.EmbeddingModels[m.Model] = &OllamaEmbedding{
				modelName: m.Model,
				Url:       m.Url,
			}
		case constant.EmbeddingModelTypeHttp, constant.EmbeddingModelTypeOpenAPI:
			mgr.EmbeddingModels[m.Model] = &OpenApiEmbedding{
				modelName: m.Model,
				Url:       m.Url,
				ApiKey:    m.ApiKey,
			}
		default:
			return nil, fmt.Errorf("unknown embedding model type %q (model=%s)", m.Type, m.Model)
		}
	}
	return mgr, nil
}

// ==================== 兼容旧 API（保留单例 + package init，仅给 legacy 代码用） ====================
// 新代码请通过 Bootstrap 调用 NewEmbeddingManager。

var EmbeddingManagerApp *EmbeddingManager

type EmbeddingModelType struct {
	Type   string
	Model  string
	Url    string
	ApiKey string
}

// compat init：仍通过 runtime.Caller 找 config 目录；失败打 log 不 panic（旧代码 panic 太重）
func init() {
	compatRoot := compatProjectRoot()
	allModels := make([]EmbeddingModelType, 0)
	data, err := os.ReadFile(filepath.Join(compatRoot, "config", "embedding_model_config.json"))
	if err != nil {
		fmt.Println("[compat] EmbeddingManager init skipped:", err)
		return
	}
	if err := json.Unmarshal(data, &allModels); err != nil {
		fmt.Println("[compat] EmbeddingManager init skipped:", err)
		return
	}
	cfgModels := make([]config.EmbeddingModelConfig, len(allModels))
	for i, m := range allModels {
		cfgModels[i] = config.EmbeddingModelConfig{
			Type:   m.Type,
			Model:  m.Model,
			Url:    m.Url,
			ApiKey: m.ApiKey,
		}
	}
	mgr, err := NewEmbeddingManager(cfgModels)
	if err != nil {
		fmt.Println("[compat] EmbeddingManager init skipped:", err)
		return
	}
	EmbeddingManagerApp = mgr
}

func compatProjectRoot() string {
	_, b, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(b), "../../..")
}
