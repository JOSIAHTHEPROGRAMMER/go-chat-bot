package rag

import (
	"fmt"
	"time"

	"github.com/JOSIAHTHEPROGRAMMER/go-chat-bot/fetcher"
	"github.com/JOSIAHTHEPROGRAMMER/go-chat-bot/llm"
)

// embedder is the single embedding provider used across the rag package.
// Swap this line if you ever add a second embedding backend.
var embedder llm.Embedder = &llm.GeminiEmbedder{}

// Doc represents a project README with its embedding vector and language stats.
type Doc struct {
	Path      string         `json:"path"`
	Content   string         `json:"content"`
	Embedding []float32      `json:"embedding"`
	Languages map[string]int `json:"languages"` // from GitHub languages API e.g. {"JavaScript": 45123}
}

// geminiEmbedDelay is the minimum time between Gemini embedding calls.
// The free tier allows 15 requests per minute for text-embedding-004,
// so 4 seconds between calls keeps us safely under the limit.
const geminiEmbedDelay = 4 * time.Second

// EmbedAllReadmes fetches READMEs, embeds them, and upserts into Qdrant.
func EmbedAllReadmes() ([]Doc, error) {
	rawDocs, err := fetcher.FetchAllReadmes()
	if err != nil {
		return nil, err
	}

	var docs []Doc
	total := len(rawDocs)

	for i, d := range rawDocs {
		vec, err := embedder.Embed(d.Content)
		if err != nil {
			fmt.Printf("embedding failed for %s: %v\n", d.Path, err)
			// Still wait before next call even on failure - we hit the rate limit either way
			time.Sleep(geminiEmbedDelay)
			continue
		}

		doc := Doc{
			Path:      d.Path,
			Content:   d.Content,
			Embedding: vec,
			Languages: d.Languages,
		}

		if err := Set(doc); err != nil {
			fmt.Printf("qdrant upsert failed for %s: %v\n", d.Path, err)
			continue
		}

		fmt.Printf("embedded %s (%d/%d)\n", d.Path, i+1, total)
		docs = append(docs, doc)

		// Pause to stay within Gemini's 15 requests per minute limit
		time.Sleep(geminiEmbedDelay)
	}

	fmt.Printf("Embedded and stored %d documents\n", len(docs))
	return docs, nil
}
