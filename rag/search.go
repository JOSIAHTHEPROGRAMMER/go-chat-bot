package rag

import (
	"fmt"
	"sort"
	"strings"

	"github.com/JOSIAHTHEPROGRAMMER/go-chat-bot/llm"
)

// maxDocChars is the maximum characters taken from each README for the LLM prompt.
// 1500 chars captures the description, stack, and key features in most READMEs.
const maxDocChars = 1500

// SearchTopK embeds the query, fetches top-N candidates from Qdrant,
// reranks with Jina, and returns the top-K most relevant docs.
//
// Pipeline: query → embed → Qdrant top-10 → Jina rerank → top-5
func SearchTopK(query string, k int) ([]Doc, error) {
	embedder := llm.GetEmbedder()
	if embedder == nil {
		return nil, fmt.Errorf("no embedder registered")
	}

	vec, err := embedder.Embed(query)
	if err != nil {
		return nil, fmt.Errorf("embed query: %w", err)
	}

	// Fetch 2x candidates so reranker has room to reorder
	candidateK := k * 2
	if candidateK < 10 {
		candidateK = 10
	}

	docs, err := client.Search(vec, candidateK)
	if err != nil {
		return nil, fmt.Errorf("qdrant search: %w", err)
	}

	if len(docs) == 0 {
		return nil, nil
	}

	reranked := Rerank(query, docs, k)
	return reranked, nil
}

// GetContextString formats docs into a context block for the LLM prompt.
// Includes GitHub language stats per doc so the LLM knows exactly what
// language each project uses — prevents hallucinating Go for Python projects etc.
func GetContextString(docs []Doc) string {
	if len(docs) == 0 {
		return ""
	}

	var sb strings.Builder
	for _, d := range docs {
		fmt.Fprintf(&sb, "--- Source: %s ---\n", d.Path)

		// Inject language stats before the README content so they're always visible
		// even if the README content gets truncated.
		if len(d.Languages) > 0 {
			fmt.Fprintf(&sb, "Languages: %s\n", formatLanguages(d.Languages))
		}

		content := strings.TrimSpace(d.Content)
		if len(content) > maxDocChars {
			content = content[:maxDocChars] + "..."
		}
		fmt.Fprintf(&sb, "%s\n\n", content)
	}
	return sb.String()
}

// formatLanguages returns a sorted, human-readable language list.
// e.g. "Go (12400 bytes), JavaScript (4500 bytes)"
// Sorted by byte count descending so primary language appears first.
func formatLanguages(langs map[string]int) string {
	type langEntry struct {
		name  string
		bytes int
	}

	entries := make([]langEntry, 0, len(langs))
	for name, bytes := range langs {
		entries = append(entries, langEntry{name, bytes})
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].bytes > entries[j].bytes
	})

	parts := make([]string, len(entries))
	for i, e := range entries {
		parts[i] = e.name
	}
	return strings.Join(parts, ", ")
}
