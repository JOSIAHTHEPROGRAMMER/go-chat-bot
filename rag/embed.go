package rag

import (
	"fmt"

	"github.com/JOSIAHTHEPROGRAMMER/portfolio-backend/fetcher"
	"github.com/JOSIAHTHEPROGRAMMER/portfolio-backend/llm"
)

// embedder is the single embedding provider used across the rag package.
// Swap this line if you ever add a second embedding backend.
var embedder llm.Embedder = &llm.GeminiEmbedder{}

// Doc represents a project README with its embedding vector.
type Doc struct {
	Path      string    `json:"path"`
	Content   string    `json:"content"`
	Embedding []float32 `json:"embedding"`
}

// EmbedAllReadmes fetches READMEs, embeds them, and upserts them into Qdrant.
func EmbedAllReadmes() ([]Doc, error) {
	rawDocs, err := fetcher.FetchAllReadmes()
	if err != nil {
		return nil, err
	}

	var docs []Doc

	for _, d := range rawDocs {
		vec, err := embedder.Embed(d.Content)
		if err != nil {
			// Log and skip rather than aborting the whole batch
			fmt.Printf("embedding failed for %s: %v\n", d.Path, err)
			continue
		}

		doc := Doc{
			Path:      d.Path,
			Content:   d.Content,
			Embedding: vec,
		}

		// Upsert into Qdrant - idempotent, safe to call on refresh
		if err := Set(doc); err != nil {
			fmt.Printf("qdrant upsert failed for %s: %v\n", d.Path, err)
			continue
		}

		docs = append(docs, doc)
	}

	fmt.Printf("Embedded and stored %d documents\n", len(docs))
	return docs, nil
}
