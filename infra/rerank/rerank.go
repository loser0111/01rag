package rerank

type RerankRequest struct {
	Model           string   `json:"model"`
	Query           string   `json:"query"`
	Documents       []string `json:"documents"`
	TopN            int      `json:"top_n"`
	ReturnDocuments bool     `json:"return_documents"`
}

type RerankTokens struct {
	InputTokens int `json:"input_tokens"`
}
type RerankResult struct {
	Index          int     `json:"index"`
	RelevanceScore float64 `json:"relevance_score"`
	Document       string  `json:"document"`
}
type RerankMeta struct {
	Model  string       `json:"model"`
	Tokens RerankTokens `json:"tokens"`
}
type RerankResponse struct {
	ID      string          `json:"id"`
	Results []*RerankResult `json:"results"`
	Mata    RerankMeta      `json:"mata"`
}
