package rag

import (
	"fmt"
	"time"

	"github.com/JOSIAHTHEPROGRAMMER/go-chat-bot/fetcher"
	"github.com/JOSIAHTHEPROGRAMMER/go-chat-bot/llm"
)

// Doc represents a project README with its embedding vector and language stats.
type Doc struct {
	Path      string         `json:"path"`
	Content   string         `json:"content"`
	Embedding []float32      `json:"embedding"`
	Languages map[string]int `json:"languages"`
}

// geminiEmbedDelay keeps us under Gemini's 15 requests/minute free tier limit.
const geminiEmbedDelay = 4 * time.Second

// EmbedAndStore embeds a slice of fetcher.Docs using the registered embedder
// and upserts each one into Qdrant. Returns the count of successfully stored docs.
func EmbedAndStore(rawDocs []fetcher.Doc) (int, error) {
	embedder := llm.GetEmbedder()
	if embedder == nil {
		return 0, fmt.Errorf("no embedder registered — call llm.RegisterEmbedder() before EmbedAndStore()")
	}

	count := 0
	total := len(rawDocs)

	for i, d := range rawDocs {
		vec, err := embedder.Embed(d.Content)
		if err != nil {
			fmt.Printf("embed failed for %s: %v\n", d.Path, err)
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
		count++

		time.Sleep(geminiEmbedDelay)
	}

	fmt.Printf("EmbedAndStore: stored %d/%d docs\n", count, total)
	return count, nil
}
