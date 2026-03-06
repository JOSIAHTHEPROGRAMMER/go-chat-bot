package rag

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"sort"
	"strings"
)

// LoadEmbeddings reads the persisted embeddings JSON from disk.
// Used at startup to populate the store - not called during queries.
func LoadEmbeddings() ([]Doc, error) {
	dataBytes, err := os.ReadFile("./data/embeddings.json")
	if err != nil {
		return nil, err
	}
	var docs []Doc
	if err := json.Unmarshal(dataBytes, &docs); err != nil {
		return nil, err
	}
	return docs, nil
}

// cosineSimilarity returns how similar two vectors are, between -1 and 1.
func cosineSimilarity(a, b []float32) float32 {
	if len(a) != len(b) {
		return 0
	}
	var dot, normA, normB float32
	for i := range a {
		dot += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}
	if normA == 0 || normB == 0 {
		return 0
	}
	return dot / (float32(math.Sqrt(float64(normA))) * float32(math.Sqrt(float64(normB))))
}

// SearchTopK returns the k docs most semantically similar to the query.
// Fails hard if the store is empty — means LoadFromDisk wasn't called at startup.
func SearchTopK(query string, k int) ([]Doc, error) {
	if store.Size() == 0 {
		return nil, fmt.Errorf("vector store is empty: call rag.LoadFromDisk() at startup")
	}

	queryVec, err := embedder.Embed(query)
	if err != nil {
		return nil, fmt.Errorf("failed to embed query: %w", err)
	}

	type scoredDoc struct {
		doc   Doc
		score float32
	}

	all := store.All()
	scored := make([]scoredDoc, 0, len(all))
	for _, d := range all {
		scored = append(scored, scoredDoc{
			doc:   d,
			score: cosineSimilarity(queryVec, d.Embedding),
		})
	}

	// Sort highest similarity first
	sort.Slice(scored, func(i, j int) bool {
		return scored[i].score > scored[j].score
	})

	top := make([]Doc, 0, k)
	for i := 0; i < k && i < len(scored); i++ {
		top = append(top, scored[i].doc)
	}

	return top, nil
}

// GetContextString concatenates doc content into a single string for prompt injection.
func GetContextString(docs []Doc) string {
	var context strings.Builder
	for _, d := range docs {
		fmt.Fprintf(&context, "File: %s\n%s\n\n", d.Path, d.Content)
	}
	return context.String()
}
