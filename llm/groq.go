package llm

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
)

type GroqProvider struct{}

func (g *GroqProvider) Name() string { return "groq" }

// Complete sends a single-turn prompt. Used for classification.
func (g *GroqProvider) Complete(prompt string) (string, error) {
	return g.Chat([]Message{{Role: "user", Content: prompt}})
}

// Chat sends a full message history and returns the complete response.
func (g *GroqProvider) Chat(messages []Message) (string, error) {
	apiKey := os.Getenv("GROQ_API_KEY")
	model := os.Getenv("GROQ_MODEL")

	if apiKey == "" || model == "" {
		return "", fmt.Errorf("missing GROQ_API_KEY or GROQ_MODEL in env")
	}

	msgs := make([]groqMessage, len(messages))
	for i, m := range messages {
		msgs[i] = groqMessage{Role: m.Role, Content: m.Content}
	}

	reqBody := groqRequest{Model: model, Messages: msgs, Stream: false}
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

// Stream sends a full message history and writes tokens into out as they arrive.
// Uses the OpenAI-compatible SSE format that Groq supports.
func (g *GroqProvider) Stream(messages []Message, out chan<- string) error {
	defer close(out)

	apiKey := os.Getenv("GROQ_API_KEY")
	model := os.Getenv("GROQ_MODEL")

	if apiKey == "" || model == "" {
		return fmt.Errorf("missing GROQ_API_KEY or GROQ_MODEL in env")
	}

	msgs := make([]groqMessage, len(messages))
	for i, m := range messages {
		msgs[i] = groqMessage{Role: m.Role, Content: m.Content}
	}

	reqBody := groqRequest{Model: model, Messages: msgs, Stream: true}
	bodyBytes, _ := json.Marshal(reqBody)

	req, err := http.NewRequest("POST", "https://api.groq.com/openai/v1/chat/completions", bytes.NewBuffer(bodyBytes))
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	// Groq streams as SSE lines: "data: {...}" or "data: [DONE]"
	scanner := bufio.NewScanner(res.Body)
	for scanner.Scan() {
		line := scanner.Text()

		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		payload := strings.TrimPrefix(line, "data: ")
		if payload == "[DONE]" {
			break
		}

		var chunk groqStreamChunk
		if err := json.Unmarshal([]byte(payload), &chunk); err != nil {
			continue
		}

		if len(chunk.Choices) > 0 {
			token := chunk.Choices[0].Delta.Content
			if token != "" {
				out <- token
			}
		}
	}

	return scanner.Err()
}

// -- internal types --

type groqRequest struct {
	Model    string        `json:"model"`
	Messages []groqMessage `json:"messages"`
	Stream   bool          `json:"stream"`
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

type groqStreamChunk struct {
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
	} `json:"choices"`
}
