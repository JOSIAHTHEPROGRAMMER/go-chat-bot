package rag

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"sort"

	"github.com/JOSIAHTHEPROGRAMMER/portfolio-backend/llm"
)

// Load embeddings from JSON
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

// Cosine similarity
func cosineSimilarity(a, b []float32) float32 {
	if len(a) != len(b) {
		return 0
	}
	var dot, normA, normB float32
	for i := 0; i < len(a); i++ {
		dot += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}
	if normA == 0 || normB == 0 {
		return 0
	}
	return dot / (float32(math.Sqrt(float64(normA))) * float32(math.Sqrt(float64(normB))))
}

// SearchTopK returns the top-k closest docs
func SearchTopK(query string, k int) ([]Doc, error) {
	docs, err := LoadEmbeddings()
	if err != nil {
		return nil, err
	}

	// Generate query embedding
	embeddingStr, _ := llm.CallGroq("Embed this text as vector (comma-separated floats):\n" + query)
	queryVec := parseEmbeddingString(embeddingStr)

	// Compute similarity
	type scoredDoc struct {
		Doc   Doc
		Score float32
	}
	var scored []scoredDoc
	for _, d := range docs {
		score := cosineSimilarity(queryVec, d.Embedding)
		scored = append(scored, scoredDoc{Doc: d, Score: score})
	}

	// Sort descending
	sort.Slice(scored, func(i, j int) bool {
		return scored[i].Score > scored[j].Score
	})

	// Return top-k
	var top []Doc
	for i := 0; i < k && i < len(scored); i++ {
		top = append(top, scored[i].Doc)
	}

	return top, nil
}

// Get concatenated content for context
func GetContextString(docs []Doc) string {
	context := ""
	for _, d := range docs {
		context += fmt.Sprintf("File: %s\n%s\n\n", d.Path, d.Content)
	}
	return context
}
