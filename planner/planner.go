package planner

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/JOSIAHTHEPROGRAMMER/portfolio-backend/llm"
	"github.com/JOSIAHTHEPROGRAMMER/portfolio-backend/rag"
)

// Plan is the result of the planner's decision for a given question.
type Plan struct {
	NeedsRAG bool   // whether to retrieve context from the vector store
	Prompt   string // the final prompt ready to send to the LLM
}

// classifyResponse is what we expect the LLM to return during classification.
type classifyResponse struct {
	NeedsRAG bool `json:"needs_rag"`
}

// classifyPrompt asks the LLM to decide if RAG is needed.
// Keeping this as a const makes it easy to tune later.
const classifyPrompt = `You are a routing assistant. Given a user question, decide if answering it requires retrieving context from a portfolio knowledge base (GitHub READMEs, project docs).

Respond ONLY with valid JSON in this exact format:
{"needs_rag": true}
or
{"needs_rag": false}

Rules:
- needs_rag = true  → question is about specific projects, skills, experience, or work history
- needs_rag = false → question is general, conversational, or answerable without project context

Question: %s`

// Build classifies the question and returns a Plan with the final prompt.
func Build(question string, provider llm.Provider) (Plan, error) {
	// Step 1: ask the LLM whether RAG context is needed
	classifyOut, err := provider.Complete(fmt.Sprintf(classifyPrompt, question))
	if err != nil {
		return Plan{}, fmt.Errorf("classification failed: %w", err)
	}

	// Strip any markdown fences the LLM might wrap around the JSON
	cleaned := strings.TrimSpace(classifyOut)
	cleaned = strings.TrimPrefix(cleaned, "```json")
	cleaned = strings.TrimPrefix(cleaned, "```")
	cleaned = strings.TrimSuffix(cleaned, "```")

	var classification classifyResponse
	if err := json.Unmarshal([]byte(cleaned), &classification); err != nil {
		// If parsing fails, default to using RAG — safer than missing context
		fmt.Printf("planner: classification parse failed (%v), defaulting to RAG\n", err)
		classification.NeedsRAG = true
	}

	// Step 2: build the final prompt based on the decision
	if !classification.NeedsRAG {
		return Plan{
			NeedsRAG: false,
			Prompt:   fmt.Sprintf("You are a portfolio assistant.\n\nQuestion:\n%s", question),
		}, nil
	}

	// Step 3: retrieve RAG context only if needed
	docs, err := rag.SearchTopK(question, 3)
	if err != nil {
		return Plan{}, fmt.Errorf("RAG search failed: %w", err)
	}

	context := rag.GetContextString(docs)
	return Plan{
		NeedsRAG: true,
		Prompt: fmt.Sprintf(
			"You are a portfolio assistant.\n\nContext:\n%s\n\nQuestion:\n%s",
			context, question,
		),
	}, nil
}
