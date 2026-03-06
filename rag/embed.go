package rag

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/JOSIAHTHEPROGRAMMER/portfolio-backend/fetcher"
	"github.com/JOSIAHTHEPROGRAMMER/portfolio-backend/llm"
)

// embedder is the package-level embedder instance.
// Swap this out if you add a second embedding provider.
var embedder llm.Embedder = &llm.GeminiEmbedder{}

type Doc struct {
	Path      string    `json:"path"`
	Content   string    `json:"content"`
	Embedding []float32 `json:"embedding"`
}

// EmbedAllReadmes fetches all READMEs from GitHub, embeds them, and saves to JSON.
func EmbedAllReadmes() ([]Doc, error) {
	rawDocs, err := fetcher.FetchAllReadmes()
	if err != nil {
		return nil, err
	}

	var docs []Doc

	for _, d := range rawDocs {
		vec, err := embedder.Embed(d.Content)
		if err != nil {
			fmt.Printf("embedding failed for %s: %v\n", d.Path, err)
			continue
		}

		doc := Doc{
			Path:      d.Path,
			Content:   d.Content,
			Embedding: vec,
		}

		docs = append(docs, doc)

		// Keep the in-memory store current as we go
		store.Set(doc)
	}

	// Persist to disk so restarts don't need to re-embed
	dataBytes, err := json.MarshalIndent(docs, "", "  ")
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile("./data/embeddings.json", dataBytes, 0644); err != nil {
		return nil, err
	}

	fmt.Printf("Embedded and stored %d documents\n", len(docs))
	return docs, nil
}
