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

	// Chat handles a multi-turn conversation. Used for the actual answer call.
	// The system prompt should be prepended by the caller as the first message.
	Chat(messages []Message) (string, error)

	Name() string
}
