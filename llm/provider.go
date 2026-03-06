package llm

// Message represents a single turn in a conversation.
type Message struct {
	Role    string `json:"role"`    // "user" or "assistant"
	Content string `json:"content"`
}

// Provider is the shared contract for all LLM backends.
type Provider interface {
	// Complete handles a single-turn prompt. Used for classification calls.
	Complete(prompt string) (string, error)

	// Chat handles a multi-turn conversation. Returns the full response at once.
	Chat(messages []Message) (string, error)

	// Stream handles a multi-turn conversation, sending tokens into out as they arrive.
	// The caller is responsible for closing nothing - the provider closes out when done.
	Stream(messages []Message, out chan<- string) error

	Name() string
}
