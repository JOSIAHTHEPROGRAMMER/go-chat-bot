package rag

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/JOSIAHTHEPROGRAMMER/portfolio-backend/fetcher"
	"github.com/JOSIAHTHEPROGRAMMER/portfolio-backend/llm"
)

type Doc struct {
	Path      string    `json:"path"`
	Content   string    `json:"content"`
	Embedding []float32 `json:"embedding"`
}

// EmbedAllReadmes fetches all READMEs from GitHub, embeds them, and saves to JSON
func EmbedAllReadmes() ([]Doc, error) {
	// 1. Fetch READMEs
	rawDocs, err := fetcher.FetchAllReadmes()
	if err != nil {
		return nil, err
	}

	var docs []Doc

	// 2️ Embed each README
	for _, d := range rawDocs {
		prompt := "Embed this text as vector (comma-separated floats):\n" + d.Content
		embeddingStr, _ := llm.CallGroq(prompt)
		vector := parseEmbeddingString(embeddingStr)

		docs = append(docs, Doc{
			Path:      d.Path,
			Content:   d.Content,
			Embedding: vector,
		})
	}

	// 3️ Save embeddings to JSON
	dataBytes, _ := json.MarshalIndent(docs, "", "  ")
	os.WriteFile("./data/embeddings.json", dataBytes, 0644)

	fmt.Printf("Generated embeddings for %d documents\n", len(docs))
	return docs, nil
}

// parseEmbeddingString converts "0.1,0.2,0.3" -> []float32
func parseEmbeddingString(s string) []float32 {
	parts := strings.Split(s, ",")
	vec := make([]float32, len(parts))
	for i, p := range parts {
		fmt.Sscanf(strings.TrimSpace(p), "%f", &vec[i])
	}
	return vec
}
