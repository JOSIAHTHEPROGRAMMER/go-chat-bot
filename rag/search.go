package rag

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"sort"
)

// LoadEmbeddings reads the persisted embeddings from disk.
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

// cosineSimilarity returns the cosine similarity between two equal-length vectors.
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

// SearchTopK returns the k most semantically similar docs to the query.
func SearchTopK(query string, k int) ([]Doc, error) {
	docs, err := LoadEmbeddings()
	if err != nil {
		return nil, err
	}

	queryVec, err := embedder.Embed(query)
	if err != nil {
		return nil, fmt.Errorf("failed to embed query: %w", err)
	}

	type scoredDoc struct {
		doc   Doc
		score float32
	}

	scored := make([]scoredDoc, 0, len(docs))
	for _, d := range docs {
		scored = append(scored, scoredDoc{
			doc:   d,
			score: cosineSimilarity(queryVec, d.Embedding),
		})
	}

	sort.Slice(scored, func(i, j int) bool {
		return scored[i].score > scored[j].score
	})

	top := make([]Doc, 0, k)
	for i := 0; i < k && i < len(scored); i++ {
		top = append(top, scored[i].doc)
	}

	return top, nil
}

// GetContextString concatenates doc content for prompt injection.
func GetContextString(docs []Doc) string {
	context := ""
	for _, d := range docs {
		context += fmt.Sprintf("File: %s\n%s\n\n", d.Path, d.Content)
	}
	return context
}
