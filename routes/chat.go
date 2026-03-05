package routes

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/JOSIAHTHEPROGRAMMER/portfolio-backend/config"
	"github.com/JOSIAHTHEPROGRAMMER/portfolio-backend/llm"
	"github.com/JOSIAHTHEPROGRAMMER/portfolio-backend/rag"
)

type ChatRequest struct {
	Question string `json:"question"`
}

type ChatResponse struct {
	Answer string `json:"answer"`
}

// ChatHandler handles incoming chat requests, retrieves relevant context using RAG, and generates a response using the current LLM provider.
func ChatHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		fmt.Fprint(w, "Only POST allowed")
		return
	}

	var req ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, "Invalid JSON")
		return
	}

	// 1. Retrieve RAG context
	docs, err := rag.SearchTopK(req.Question, 3)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "RAG error: %v", err)
		return
	}

	context := rag.GetContextString(docs)
	finalPrompt := fmt.Sprintf("You are a portfolio assistant.\n\nContext:\n%s\n\nQuestion:\n%s", context, req.Question)

	// 2. Resolve provider -  no switch, no string matching here
	provider, err := llm.Get(config.GetCurrentModel())
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Provider error: %v", err)
		return
	}

	// 3. Call LLM
	answer, err := provider.Complete(finalPrompt)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "LLM error: %v", err)
		return
	}

	json.NewEncoder(w).Encode(ChatResponse{Answer: answer})
}
