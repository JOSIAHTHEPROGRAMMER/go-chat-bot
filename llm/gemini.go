package llm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
)

type GeminiProvider struct{}

func (g *GeminiProvider) Name() string { return "gemini" }

// Complete sends a single-turn prompt. Used for classification.
func (g *GeminiProvider) Complete(prompt string) (string, error) {
	return g.Chat([]Message{{Role: "user", Content: prompt}})
}

// Chat sends a full message history to Gemini and returns the assistant's reply.
// Gemini uses "model" instead of "assistant" for the assistant role.
func (g *GeminiProvider) Chat(messages []Message) (string, error) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	model := os.Getenv("GEMINI_MODEL")

	if apiKey == "" || model == "" {
		return "", fmt.Errorf("missing GEMINI_API_KEY or GEMINI_MODEL in env")
	}

	// Gemini uses "model" for assistant turns - map accordingly
	contents := make([]geminiContent, len(messages))
	for i, m := range messages {
		role := m.Role
		if role == "assistant" {
			role = "model"
		}
		contents[i] = geminiContent{
			Role:  role,
			Parts: []geminiPart{{Text: m.Content}},
		}
	}

	reqBody := geminiRequest{Contents: contents}
	bodyBytes, _ := json.Marshal(reqBody)

	url := fmt.Sprintf(
		"https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s",
		model, apiKey,
	)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	var out geminiResponse
	json.NewDecoder(res.Body).Decode(&out)

	if len(out.Candidates) == 0 || len(out.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("no response from Gemini")
	}

	return out.Candidates[0].Content.Parts[0].Text, nil
}

// -- internal types --

type geminiRequest struct {
	Contents []geminiContent `json:"contents"`
}

type geminiContent struct {
	Role  string       `json:"role"`
	Parts []geminiPart `json:"parts"`
}

type geminiPart struct {
	Text string `json:"text"`
}

type geminiResponse struct {
	Candidates []struct {
		Content geminiContent `json:"content"`
	} `json:"candidates"`
}
