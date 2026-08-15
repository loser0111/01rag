package main

import (
	_ "com.wyq.01rag/infra"
	"com.wyq.01rag/test"
	"fmt"
)

func init() {

}

func main() {
	fmt.Println("01RAG")

	fmt.Println("============TEST=============")
	// test.TestEmbedding(nil)

	test.TestChromaStorage(nil)
}
