package routes

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/JOSIAHTHEPROGRAMMER/portfolio-backend/config"
	"github.com/JOSIAHTHEPROGRAMMER/portfolio-backend/llm"
	"github.com/JOSIAHTHEPROGRAMMER/portfolio-backend/planner"
)

type ChatRequest struct {
	Question string `json:"question"`
}

type ChatResponse struct {
	Answer  string `json:"answer"`
	UsedRAG bool   `json:"used_rag"` // useful for debugging and frontend transparency
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

	// Resolve the active LLM provider
	provider, err := llm.Get(config.GetCurrentModel())
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Provider error: %v", err)
		return
	}

	// Let the planner decide whether RAG is needed and build the final prompt
	plan, err := planner.Build(req.Question, provider)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Planner error: %v", err)
		return
	}

	// Call the LLM with the planned prompt
	answer, err := provider.Complete(plan.Prompt)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "LLM error: %v", err)
		return
	}

	json.NewEncoder(w).Encode(ChatResponse{
		Answer:  answer,
		UsedRAG: plan.NeedsRAG,
	})
}
