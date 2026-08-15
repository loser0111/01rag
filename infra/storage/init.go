package storage

import (
	_ "com.wyq.01rag/infra/storage/chroma"
	_ "com.wyq.01rag/infra/storage/manager"
	"fmt"
)

func init() {
	fmt.Println(" storage init")
}
