package planner

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/JOSIAHTHEPROGRAMMER/go-chat-bot/config"
	"github.com/JOSIAHTHEPROGRAMMER/go-chat-bot/llm"
	"github.com/JOSIAHTHEPROGRAMMER/go-chat-bot/logger"
	"github.com/JOSIAHTHEPROGRAMMER/go-chat-bot/rag"
	"github.com/JOSIAHTHEPROGRAMMER/go-chat-bot/tools"
)

// PlanType tells the route handler how the question was handled.
type PlanType string

const (
	PlanDirect PlanType = "direct"
	PlanRAG    PlanType = "rag"
	PlanTool   PlanType = "tool"
)

type Plan struct {
	Type     PlanType
	Messages []llm.Message
}

type classifyResponse struct {
	Type  string `json:"type"`
	Tool  string `json:"tool"`
	Input string `json:"input"`
}

// classifyPrompt is intentionally compact — fewer tokens = faster Groq response.
// Four valid outputs only. Any deviation defaults to RAG.
const classifyPrompt = `Output one JSON object only. No markdown. No explanation.

Rules:
- Greeting or off-topic → {"type":"direct"}
- General skills/background/experience → {"type":"rag"}
- Which projects use a language or technology (Go, Python, React, etc.) → {"type":"tool","tool":"filter_by_tech","input":"<technology, lowercase>"}
- Question about a specific named project → {"type":"tool","tool":"get_project","input":"<project name>"}

Only "filter_by_tech" or "get_project" are valid tool names.
filter_by_tech = languages/tech only. get_project = named projects only.
When unsure → {"type":"rag"}

Question: %s`

// Build classifies the question and returns a Plan with a ready-to-send message slice.
func Build(ctx context.Context, question string, history []llm.Message, provider llm.Provider) (Plan, error) {
	raw, err := provider.Complete(fmt.Sprintf(classifyPrompt, question))
	if err != nil {
		return Plan{}, fmt.Errorf("classification failed: %w", err)
	}

	// Strip markdown fences the LLM might add despite being told not to
	cleaned := strings.TrimSpace(raw)
	cleaned = strings.TrimPrefix(cleaned, "```json")
	cleaned = strings.TrimPrefix(cleaned, "```")
	cleaned = strings.TrimSuffix(cleaned, "```")
	cleaned = strings.TrimSpace(cleaned)

	// Extract just the first JSON object in case the LLM adds trailing text
	if idx := strings.Index(cleaned, "}"); idx != -1 {
		cleaned = cleaned[:idx+1]
	}

	var classification classifyResponse
	if err := json.Unmarshal([]byte(cleaned), &classification); err != nil {
		fmt.Printf("planner: parse failed (%v), defaulting to rag. raw=%q\n", err, raw)
		classification.Type = "rag"
	}

	if PlanType(classification.Type) == PlanTool {
		if classification.Tool != "filter_by_tech" && classification.Tool != "get_project" {
			fmt.Printf("planner: invalid tool %q, defaulting to rag\n", classification.Tool)
			return buildRAGPlan(ctx, question, history)
		}
	}

	switch PlanType(classification.Type) {

	case PlanDirect:
		writeLog(ctx, PlanDirect, 0)
		return Plan{
			Type:     PlanDirect,
			Messages: buildMessages("", question, history),
		}, nil

	case PlanTool:
		tool, err := tools.Get(classification.Tool)
		if err != nil {
			fmt.Printf("planner: %v, falling back to rag\n", err)
			return buildRAGPlan(ctx, question, history)
		}

		result, err := tool.Run(classification.Input)
		if err != nil {
			fmt.Printf("planner: tool %q returned no result (%v), falling back to rag\n", classification.Tool, err)
			return buildRAGPlan(ctx, question, history)
		}

		matchCount := countToolMatches(classification.Tool, result)
		writeLog(ctx, PlanTool, matchCount)

		return Plan{
			Type:     PlanTool,
			Messages: buildMessages(result, question, history),
		}, nil

	default:
		return buildRAGPlan(ctx, question, history)
	}
}

func countToolMatches(toolName, result string) int {
	if toolName == "get_project" {
		return 1
	}
	var n int
	if _, err := fmt.Sscanf(extractParenthetical(result), "%d total", &n); err == nil && n > 0 {
		return n
	}
	return strings.Count(result, "### ")
}

func extractParenthetical(s string) string {
	start := strings.Index(s, "(")
	end := strings.Index(s, ")")
	if start == -1 || end == -1 || end <= start {
		return ""
	}
	return s[start+1 : end]
}

// buildRAGPlan fetches top-5 semantically similar docs and builds a RAG plan.
// k=5 balances context richness vs prompt size — each doc is capped at 1500 chars
// in GetContextString so total context stays well within token limits.
func buildRAGPlan(ctx context.Context, question string, history []llm.Message) (Plan, error) {
	docs, err := rag.SearchTopK(question, 5)
	if err != nil {
		return Plan{}, fmt.Errorf("RAG search failed: %w", err)
	}

	writeLog(ctx, PlanRAG, len(docs))
	return Plan{
		Type:     PlanRAG,
		Messages: buildMessages(rag.GetContextString(docs), question, history),
	}, nil
}

// buildMessages assembles the full message slice for provider.Chat().
// System prompt is sent as a proper system role — both Groq and Gemini support this.
func buildMessages(contextStr, question string, history []llm.Message) []llm.Message {
	messages := []llm.Message{
		{Role: "system", Content: config.SystemPrompt},
	}

	messages = append(messages, history...)

	userContent := question
	if contextStr != "" {
		userContent = fmt.Sprintf("Context:\n%s\nQuestion: %s", contextStr, question)
	}

	messages = append(messages, llm.Message{Role: "user", Content: userContent})
	return messages
}

func writeLog(ctx context.Context, planType PlanType, docCount int) {
	if rl := logger.FromContext(ctx); rl != nil {
		rl.PlanType = string(planType)
		rl.DocCount = docCount
	}
}
