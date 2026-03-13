package rag

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

var jinaClient = &http.Client{Timeout: 15 * time.Second}

type jinaRerankRequest struct {
	Model           string   `json:"model"`
	Query           string   `json:"query"`
	Documents       []string `json:"documents"`
	TopN            int      `json:"top_n"`
	ReturnDocuments bool     `json:"return_documents"`
}

type jinaRerankResponse struct {
	Results []struct {
		Index          int     `json:"index"`
		RelevanceScore float64 `json:"relevance_score"`
	} `json:"results"`
}

// Rerank calls the Jina rerank API to sort docs by relevance to the query.
// Returns the top-k docs in relevance order.
// Falls back to original Qdrant ordering if API key is missing or unreachable.
func Rerank(query string, docs []Doc, topK int) []Doc {
	if len(docs) == 0 {
		return docs
	}
	if topK > len(docs) {
		topK = len(docs)
	}

	apiKey := os.Getenv("JINA_API_KEY")

	if apiKey == "" {
		fmt.Println("rerank: JINA_API_KEY not set, skipping")
		return docs[:topK]
	}
	fmt.Printf("rerank: calling Jina with %d candidates, top_k=%d\n", len(docs), topK)

	// Send first 512 chars of each doc — enough context for scoring
	// without hitting token limits or slowing the request.
	documents := make([]string, len(docs))
	for i, d := range docs {
		content := d.Content
		if len(content) > 512 {
			content = content[:512]
		}
		documents[i] = content
	}

	payload := jinaRerankRequest{
		Model:           "jina-reranker-v3",
		Query:           query,
		Documents:       documents,
		TopN:            topK,
		ReturnDocuments: false, // we already have the docs, just need the indices
	}

	b, err := json.Marshal(payload)
	if err != nil {
		fmt.Printf("rerank: marshal failed (%v), using original ordering\n", err)
		return docs[:topK]
	}

	req, err := http.NewRequest("POST", "https://api.jina.ai/v1/rerank", bytes.NewReader(b))
	if err != nil {
		fmt.Printf("rerank: request build failed (%v), using original ordering\n", err)
		return docs[:topK]
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := jinaClient.Do(req)
	if err != nil {
		fmt.Printf("rerank: Jina unreachable (%v), using original ordering\n", err)
		return docs[:topK]
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(resp.Body)
		fmt.Printf("rerank: Jina returned %d: %s, using original ordering\n", resp.StatusCode, string(raw))
		return docs[:topK]
	}

	var result jinaRerankResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		fmt.Printf("rerank: decode failed (%v), using original ordering\n", err)
		return docs[:topK]
	}

	reranked := make([]Doc, 0, len(result.Results))
	for _, r := range result.Results {
		if r.Index >= 0 && r.Index < len(docs) {
			reranked = append(reranked, docs[r.Index])
		}
	}

	if len(reranked) == 0 {
		return docs[:topK]
	}
	fmt.Printf("rerank: Jina ok, reranked %d docs\n", len(reranked))
	return reranked
}
