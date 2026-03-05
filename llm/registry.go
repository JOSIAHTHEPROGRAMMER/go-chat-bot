package llm

import "fmt"

var registry = map[string]Provider{
	"groq":   &GroqProvider{},
	"gemini": &GeminiProvider{},
}

// Get returns the Provider matching the given name.
// Falls back to Groq if the name is unrecognized.
func Get(name string) (Provider, error) {
	p, ok := registry[name]
	if !ok {
		return nil, fmt.Errorf("unknown provider: %q", name)
	}
	return p, nil
}
