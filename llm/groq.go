package llm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
)

type GroqProvider struct{}

func (g *GroqProvider) Name() string { return "groq" }

// Complete sends a single-turn prompt. Used for classification.
func (g *GroqProvider) Complete(prompt string) (string, error) {
	return g.Chat([]Message{{Role: "user", Content: prompt}})
}

// Chat sends a full message history to Groq and returns the assistant's reply.
func (g *GroqProvider) Chat(messages []Message) (string, error) {
	apiKey := os.Getenv("GROQ_API_KEY")
	model := os.Getenv("GROQ_MODEL")

	if apiKey == "" || model == "" {
		return "", fmt.Errorf("missing GROQ_API_KEY or GROQ_MODEL in env")
	}

	// Groq uses the OpenAI message format natively
	msgs := make([]groqMessage, len(messages))
	for i, m := range messages {
		msgs[i] = groqMessage{Role: m.Role, Content: m.Content}
	}

	reqBody := groqRequest{Model: model, Messages: msgs}
	bodyBytes, _ := json.Marshal(reqBody)

	req, err := http.NewRequest("POST", "https://api.groq.com/openai/v1/chat/completions", bytes.NewBuffer(bodyBytes))
	if err != nil {
		return "", err
	}

	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	var out groqResponse
	json.NewDecoder(res.Body).Decode(&out)

	if len(out.Choices) == 0 {
		return "", fmt.Errorf("no response from Groq")
	}

	return out.Choices[0].Message.Content, nil
}

// -- internal types --

type groqRequest struct {
	Model    string        `json:"model"`
	Messages []groqMessage `json:"messages"`
}

type groqMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type groqResponse struct {
	Choices []struct {
		Message groqMessage `json:"message"`
	} `json:"choices"`
}
