package routes

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/JOSIAHTHEPROGRAMMER/portfolio-backend/config"
	"github.com/JOSIAHTHEPROGRAMMER/portfolio-backend/llm"
	"github.com/JOSIAHTHEPROGRAMMER/portfolio-backend/rag"
)

// Request & Response structs
type ChatRequest struct {
	Question string `json:"question"`
}

type ChatResponse struct {
	Answer string `json:"answer"`
}

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

	// 1. Get RAG Context
	docs, err := rag.SearchTopK(req.Question, 3)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Error loading RAG context: %v", err)
		return
	}

	context := rag.GetContextString(docs)
	finalPrompt := fmt.Sprintf(`You are a portfolio assistant.

Context:
%s

Question:
%s`, context, req.Question)

	// 2. Get global model
	model := config.GetCurrentModel()

	// 3. Call appropriate LLM
	var answer string
	switch model {
	case "gemini":
		answer, err = llm.CallGemini(finalPrompt)
	default:
		answer, err = llm.CallGroq(finalPrompt)
	}

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Error calling LLM: %v", err)
		return
	}

	json.NewEncoder(w).Encode(ChatResponse{Answer: answer})
}
