package llm

// Embedder produces a fixed-dimension vector for a given input text.
type Embedder interface {
	Embed(text string) ([]float32, error)
}
