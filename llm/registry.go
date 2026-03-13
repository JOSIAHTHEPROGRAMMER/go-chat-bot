package llm

import "fmt"

var registry = map[string]Provider{
	"groq":   &GroqProvider{},
	"gemini": &GeminiProvider{},
}

var activeEmbedder Embedder

func Get(name string) (Provider, error) {
	p, ok := registry[name]
	if !ok {
		return nil, fmt.Errorf("unknown provider: %q", name)
	}
	return p, nil
}

func Register(p Provider) {
	registry[p.Name()] = p
}

func RegisterEmbedder(e Embedder) {
	activeEmbedder = e
}

func GetEmbedder() Embedder {
	return activeEmbedder
}
