// Package config 强类型配置：启动时 LoadConfig 一次性读到内存，之后所有构造函数都接收 cfg struct。
// 不使用 init() 读取文件；读文件失败返回 error，让 Composition Root 决定降级/退出。
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// EmbeddingModelConfig 单个 embedding 模型的连接配置（对应 embedding_model_config.json 里每项）
type EmbeddingModelConfig struct {
	Type   string `json:"type"`
	Model  string `json:"model"`
	Url    string `json:"url"`
	ApiKey string `json:"apiKey,omitempty"`
}

// ChromaConfig chroma 连接配置
type ChromaConfig struct {
	Url      string `json:"url"`
	Tenant   string `json:"tenant,omitempty"`
	Database string `json:"database,omitempty"`
	ApiKey   string `json:"apiKey,omitempty"`
	Space    string `json:"space,omitempty"` // "l2" / "cosine" / "ip"，缺省 l2
}

// AppConfig 应用级配置，之后可扩展 LLM/日志/服务端口等
type AppConfig struct {
	EmbeddingModels []EmbeddingModelConfig `json:"embedding_models"`
	Chroma          ChromaConfig           `json:"chroma"`
}

// GetProjectRoot 返回项目根（基于此文件在 internal/config 下，向上 2 级即可到 01rag）
func GetProjectRoot() string {
	_, b, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(b), "../..")
}

// LoadConfig 从 config 目录读取强类型配置。path 为空时用默认路径。
func LoadConfig() (*AppConfig, error) {
	cfg := &AppConfig{}
	root := GetProjectRoot()

	// embedding 模型配置
	data, err := os.ReadFile(filepath.Join(root, "config", "embedding_model_config.json"))
	if err != nil {
		return nil, fmt.Errorf("read embedding_model_config.json: %w", err)
	}
	if err := json.Unmarshal(data, &cfg.EmbeddingModels); err != nil {
		return nil, fmt.Errorf("parse embedding_model_config.json: %w", err)
	}

	// chroma 连接配置（不存在时提供默认本地值，兼容老流程）
	chromaPath := filepath.Join(root, "config", "chroma_config.json")
	if raw, err := os.ReadFile(chromaPath); err == nil {
		if err := json.Unmarshal(raw, &cfg.Chroma); err != nil {
			return nil, fmt.Errorf("parse chroma_config.json: %w", err)
		}
	} else {
		cfg.Chroma = ChromaConfig{
			Url:      "http://127.0.0.1:8000/api/v2",
			Tenant:   "default_tenant",
			Database: "default_database",
			Space:    "l2",
		}
	}
	return cfg, nil
}
