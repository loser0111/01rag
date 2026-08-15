package embedding

import (
	"com.wyq.01rag/constant"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

var EmbeddingManagerApp *EmbeddingManager

type EmbeddingManager struct {
	EmbeddingModels map[string]Embedding
}

func (manager *EmbeddingManager) FindByName(name string) Embedding {
	if embedding, ok := manager.EmbeddingModels[name]; ok {
		return embedding
	}
	return nil
}

type EmbeddingModelType struct {
	Type   string
	Model  string
	Url    string
	ApiKey string
}

func init() {
	// 读取config文件夹下的json文件
	EmbeddingManagerApp = &EmbeddingManager{
		EmbeddingModels: make(map[string]Embedding),
	}
	allModes := make([]EmbeddingModelType, 0)
	configPath := filepath.Join(GetProjectRoot(), "config", "embedding_model_config.json")
	fmt.Print(configPath)
	data, err := os.ReadFile(configPath)
	if err != nil {
		panic(err.Error())
	}
	err = json.Unmarshal(data, &allModes)
	if err != nil {
		panic(err.Error())
	}
	for _, m := range allModes {
		switch m.Type {
		case constant.EmbeddingModelTypeOllamaLocal:
			t := &OllamaEmbedding{
				ModelName: m.Model,
				Url:       m.Url,
			}
			EmbeddingManagerApp.EmbeddingModels[m.Model] = t
		case constant.EmbeddingModelTypeHttp:
			t := &OpenApiEmbedding{
				ModelName: m.Model,
				Url:       m.Url,
				ApiKey:    m.ApiKey,
			}
			EmbeddingManagerApp.EmbeddingModels[m.Model] = t
		}
	}
	s, _ := json.Marshal(EmbeddingManagerApp)
	fmt.Println("Add Models: ", string(s))
}
func GetProjectRoot() string {
	_, b, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(b), "../../..")
}
