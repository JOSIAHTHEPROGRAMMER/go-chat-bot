package rag

import (
	"fmt"
	"strings"
)

// SearchTopK returns the k docs most semantically similar to the query.
// Delegates to Qdrant - no in-memory cosine similarity needed anymore.
func SearchTopK(query string, k int) ([]Doc, error) {
	queryVec, err := embedder.Embed(query)
	if err != nil {
		return nil, fmt.Errorf("failed to embed query: %w", err)
	}

	docs, err := client.Search(queryVec, k)
	if err != nil {
		return nil, fmt.Errorf("qdrant search failed: %w", err)
	}

	return docs, nil
}

// GetContextString concatenates doc content into a single string for prompt injection.
func GetContextString(docs []Doc) string {
	var sb strings.Builder
	for _, d := range docs {
		fmt.Fprintf(&sb, "File: %s\n%s\n\n", d.Path, d.Content)
	}
	return sb.String()
}
