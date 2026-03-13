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

type GeminiProvider struct{}

func (g *GeminiProvider) Name() string { return "gemini" }

// Complete sends a single-turn prompt. Used for classification.
func (g *GeminiProvider) Complete(prompt string) (string, error) {
	return g.Chat([]Message{{Role: "user", Content: prompt}})
}

// Chat sends a full message history and returns the complete response.
func (g *GeminiProvider) Chat(messages []Message) (string, error) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	model := os.Getenv("GEMINI_MODEL")

	if apiKey == "" || model == "" {
		return "", fmt.Errorf("missing GEMINI_API_KEY or GEMINI_MODEL in env")
	}

	contents := toGeminiContents(messages)
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

// Stream sends a full message history and writes tokens into out as they arrive.
// Uses Gemini's streamGenerateContent endpoint which returns newline-delimited JSON.
func (g *GeminiProvider) Stream(messages []Message, out chan<- string) error {
	defer close(out)

	apiKey := os.Getenv("GEMINI_API_KEY")
	model := os.Getenv("GEMINI_MODEL")

	if apiKey == "" || model == "" {
		return fmt.Errorf("missing GEMINI_API_KEY or GEMINI_MODEL in env")
	}

	contents := toGeminiContents(messages)
	reqBody := geminiRequest{Contents: contents}
	bodyBytes, _ := json.Marshal(reqBody)

	url := fmt.Sprintf(
		"https://generativelanguage.googleapis.com/v1beta/models/%s:streamGenerateContent?key=%s",
		model, apiKey,
	)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	scanner := bufio.NewScanner(res.Body)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || line == "[" || line == "]" || line == "," {
			continue
		}
		var chunk geminiResponse
		if err := json.Unmarshal([]byte(line), &chunk); err != nil {
			continue
		}
		if len(chunk.Candidates) > 0 && len(chunk.Candidates[0].Content.Parts) > 0 {
			token := chunk.Candidates[0].Content.Parts[0].Text
			if token != "" {
				out <- token
			}
		}
	}

	return scanner.Err()
}

// toGeminiContents converts our Message slice to Gemini's content format.
// Gemini uses "model" instead of "assistant" for assistant turns.
func toGeminiContents(messages []Message) []geminiContent {
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
	return contents
}

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
