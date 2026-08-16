package rerank

type RankItem struct {
	Index int
	Score float64
	Doc   string
	Meta  map[string]any
}
