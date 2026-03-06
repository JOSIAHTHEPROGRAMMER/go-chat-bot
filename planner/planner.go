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

// Plan is the planner's decision for a given question.
type Plan struct {
	Type     PlanType
	Messages []llm.Message // full message slice ready to pass to provider.Chat()
}

// classifyResponse is what we expect the LLM to return during classification.
type classifyResponse struct {
	Type  string `json:"type"`
	Tool  string `json:"tool"`
	Input string `json:"input"`
}

const classifyPrompt = `You are a routing assistant. Your only job is to output one JSON object and nothing else.

STEP 1 - Read the question carefully.
STEP 2 - Pick exactly one of the four outputs below. No other output is valid.

OUTPUT A - greetings, small talk, or questions unrelated to the developer portfolio:
{"type":"direct"}

OUTPUT B - general questions about the developer's skills, background, or experience:
{"type":"rag"}

OUTPUT C - question asks which projects use a specific programming language or technology.
A technology is something like: JavaScript, Go, Python, React, TypeScript, CSS, SQL.
A technology is NOT a project name.
Use one technology only - if multiple are mentioned pick the most specific one.
{"type":"tool","tool":"filter_by_tech","input":"<single technology name, lowercase>"}

OUTPUT D - question asks about a specific named project (e.g. tell me about Neura, what is CaribCart):
{"type":"tool","tool":"get_project","input":"<exact project name, preserve original casing>"}

STRICT RULES:
- Output only raw JSON, no markdown, no explanation, no extra text
- The only valid values for "tool" are exactly: "filter_by_tech" or "get_project" - nothing else
- "filter_by_tech" is ONLY for programming languages and technologies, never for project names
- "get_project" is ONLY for named projects, never for languages or technologies
- Never combine multiple technologies into one input string
- If you are unsure, use OUTPUT B

Question: %s`

// Build classifies the question and returns a Plan with a ready-to-send message slice.
// history contains prior turns in the conversation, may be empty for the first message.
func Build(ctx context.Context, question string, history []llm.Message, provider llm.Provider) (Plan, error) {
	raw, err := provider.Complete(fmt.Sprintf(classifyPrompt, question))
	if err != nil {
		return Plan{}, fmt.Errorf("classification failed: %w", err)
	}

	// Strip any markdown fences the LLM might add despite being told not to
	cleaned := strings.TrimSpace(raw)
	cleaned = strings.TrimPrefix(cleaned, "```json")
	cleaned = strings.TrimPrefix(cleaned, "```")
	cleaned = strings.TrimSuffix(cleaned, "```")
	cleaned = strings.TrimSpace(cleaned)

	var classification classifyResponse
	if err := json.Unmarshal([]byte(cleaned), &classification); err != nil {
		fmt.Printf("planner: parse failed (%v), defaulting to rag\n", err)
		classification.Type = "rag"
	}

	// Validate tool name before dispatching - prevents hallucinated tool names from erroring
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

		writeLog(ctx, PlanTool, 1)
		return Plan{
			Type:     PlanTool,
			Messages: buildMessages(result, question, history),
		}, nil

	default:
		return buildRAGPlan(ctx, question, history)
	}
}

// buildRAGPlan runs similarity search and builds a RAG-based plan.
func buildRAGPlan(ctx context.Context, question string, history []llm.Message) (Plan, error) {
	docs, err := rag.SearchTopK(question, 3)
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
// Structure: system prompt, history, current user message with optional context.
func buildMessages(context, question string, history []llm.Message) []llm.Message {
	// Neither Groq nor Gemini support a dedicated system role in all configurations,
	// so we prepend it as a user turn followed by an assistant acknowledgement.
	messages := []llm.Message{
		{Role: "user", Content: config.SystemPrompt},
		{Role: "assistant", Content: "Understood. I will follow these guidelines."},
	}

	messages = append(messages, history...)

	userContent := question
	if context != "" {
		userContent = fmt.Sprintf("Context:\n%s\n\nQuestion:\n%s", context, question)
	}

	messages = append(messages, llm.Message{Role: "user", Content: userContent})
	return messages
}

// writeLog records the plan decision into the request-scoped logger.
// Safe to call with a context that has no logger attached.
func writeLog(ctx context.Context, planType PlanType, docCount int) {
	if rl := logger.FromContext(ctx); rl != nil {
		rl.PlanType = string(planType)
		rl.DocCount = docCount
	}
}
