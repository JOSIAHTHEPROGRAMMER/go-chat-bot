package llm

// Provider is the shared contract for all LLM backends.
type Provider interface {
	Complete(prompt string) (string, error)
	Name() string
}
